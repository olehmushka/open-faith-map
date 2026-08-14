// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package uaedr

import (
	"compress/flate"
	"context"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

// httpStreamingCursor is the opaque, non-literal cursor value fetchHTTP hands back — HTTP mode has
// no reopen-and-reskip semantics (see domain.Connector's own doc comment), so this exists only to
// tell RunConnector's loop "not exhausted yet, call Fetch again"; the actual resume position lives
// entirely in httpModeState.stream, never in this string.
const httpStreamingCursor = "streaming"

// httpModeState is SourceURL-mode's run-scoped state, held on the Connector instance across Fetch
// calls. runLock is a pure semaphore (TryLock'd, never blocked on) — it exists to reject a second
// concurrent RunConnector call on the same source with a clear error rather than let two goroutines
// race on the same *http.Response/*xml.Decoder. mu guards the two fields below it, which are
// mutated both by fetchHTTP's own normal-completion path and by Close's out-of-band cleanup path.
type httpModeState struct {
	runLock     sync.Mutex
	mu          sync.Mutex
	runLockHeld bool
	stream      *httpStream
}

// httpStream is one run's live HTTP response, held open across fetchHTTP calls.
type httpStream struct {
	resp  *http.Response
	unzip io.ReadCloser // non-nil only when SourceURL ends in .zip
	dec   *xml.Decoder
}

// fetchHTTP is SourceURL mode's Fetch implementation: stateful, one held-open stream per run. A
// nil cursor starts a fresh run (acquires runLock, opens the stream); any non-nil cursor is treated
// as "continue the already-open stream", never as a real resume offset — see domain.Connector's own
// doc comment for why re-fetching a multi-hundred-MB source per batch isn't viable. A mid-run crash
// (process restart) cannot resume: the next call after restart finds no matching in-memory stream
// and returns a clear error rather than silently restarting from record zero under the same cursor.
func (c *Connector) fetchHTTP(ctx context.Context, cursor *string) (batch []domain.RawRecord, nextCursor *string, err error) {
	if cursor == nil {
		if !c.http.runLock.TryLock() {
			return nil, nil, fmt.Errorf("uaedr: another run is already streaming %s", c.SourceURL)
		}
		c.http.mu.Lock()
		c.http.runLockHeld = true
		c.http.mu.Unlock()

		st, openErr := c.openHTTPStream(ctx)
		if openErr != nil {
			c.releaseRun()
			return nil, nil, openErr
		}
		c.http.mu.Lock()
		c.http.stream = st
		c.http.mu.Unlock()
	}

	c.http.mu.Lock()
	st := c.http.stream
	c.http.mu.Unlock()
	if st == nil {
		return nil, nil, fmt.Errorf("uaedr: cursor has no matching in-memory HTTP stream (process restarted or the run already ended) — HTTP mode cannot resume mid-stream; start a fresh run")
	}

	batch, done, ferr := st.readBatch(ctx, batchSize)
	if done || ferr != nil {
		closeErr := st.close()
		c.http.mu.Lock()
		c.http.stream = nil
		c.http.mu.Unlock()
		c.releaseRun()
		if ferr != nil {
			return batch, nil, ferr
		}
		if closeErr != nil {
			return batch, nil, fmt.Errorf("uaedr: close stream: %w", closeErr)
		}
		return batch, nil, nil
	}
	next := httpStreamingCursor
	return batch, &next, nil
}

// releaseRun clears runLockHeld and unlocks runLock at most once — safe to call from fetchHTTP's
// own normal-completion path or Close's out-of-band cleanup path regardless of which runs first;
// whichever runs second sees runLockHeld already false and is a no-op, so a double-unlock panic on
// an already-unlocked sync.Mutex is impossible even under concurrent Fetch/Close.
func (c *Connector) releaseRun() {
	c.http.mu.Lock()
	held := c.http.runLockHeld
	c.http.runLockHeld = false
	c.http.mu.Unlock()
	if held {
		c.http.runLock.Unlock()
	}
}

// Close implements domain.ConnectorCloser — the only path guaranteed to release runLock and close
// the stream even when a run ends via a mid-stream Fetch error or ctx cancellation, cases where
// fetchHTTP's own normal-completion cleanup above never runs. No-op in FilePath mode and whenever
// no stream is currently open (already closed by fetchHTTP itself, or never opened).
func (c *Connector) Close() error {
	if c.SourceURL == "" {
		return nil
	}
	c.http.mu.Lock()
	st := c.http.stream
	c.http.stream = nil
	c.http.mu.Unlock()
	c.releaseRun()
	if st == nil {
		return nil
	}
	return st.close()
}

// openHTTPStream issues the GET and wires up decompression (if SourceURL is a .zip) plus the same
// charset-aware xml.Decoder fetchFile uses. Never buffers the response — resp.Body is read forward
// only, through newStreamingZipEntryReader (if applicable) straight into xml.NewDecoder.
func (c *Connector) openHTTPStream(ctx context.Context) (*httpStream, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.SourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("uaedr: build request for %s: %w", c.SourceURL, err)
	}
	req.Header.Set("User-Agent", c.Citation().UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("uaedr: GET %s: %w", c.SourceURL, err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("uaedr: GET %s: unexpected status %s: %s", c.SourceURL, resp.Status, body)
	}

	var body io.Reader = resp.Body
	var unzip io.ReadCloser
	if strings.HasSuffix(strings.ToLower(c.SourceURL), ".zip") {
		unzip, err = newStreamingZipEntryReader(resp.Body)
		if err != nil {
			_ = resp.Body.Close()
			return nil, err
		}
		body = unzip
	}

	dec := xml.NewDecoder(body)
	dec.CharsetReader = charsetReader
	return &httpStream{resp: resp, unzip: unzip, dec: dec}, nil
}

// readBatch is fetchFile's token-walk loop, reused against an already-open, already-positioned
// decoder — no skip/seen bookkeeping, since HTTP mode never reopens from scratch. done=true means
// the source is exhausted (xml.Decoder hit EOF), matching fetchFile's nil-nextCursor signal.
func (st *httpStream) readBatch(ctx context.Context, size int) (batch []domain.RawRecord, done bool, err error) {
	for {
		if cerr := ctx.Err(); cerr != nil {
			return batch, false, cerr
		}
		tok, terr := st.dec.Token()
		if terr == io.EOF {
			return batch, true, nil
		}
		if terr != nil {
			return batch, false, fmt.Errorf("uaedr: decode: %w", terr)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "SUBJECT" {
			continue
		}

		var s subject
		if derr := st.dec.DecodeElement(&s, &start); derr != nil {
			return batch, false, fmt.Errorf("uaedr: decode SUBJECT: %w", derr)
		}
		if !strings.EqualFold(strings.TrimSpace(s.OPF), religiousOrgOPF) {
			continue
		}

		payload, merr := json.Marshal(s)
		if merr != nil {
			return batch, false, fmt.Errorf("uaedr: marshal raw payload: %w", merr)
		}
		batch = append(batch, domain.RawRecord{
			SourceRecordID: s.EDRPOU,
			RawPayload:     payload,
			FetchedAt:      time.Now(),
		})
		if len(batch) >= size {
			return batch, false, nil
		}
	}
}

func (st *httpStream) close() error {
	var errs []error
	if st.unzip != nil {
		if e := st.unzip.Close(); e != nil {
			errs = append(errs, e)
		}
	}
	if e := st.resp.Body.Close(); e != nil {
		errs = append(errs, e)
	}
	return errors.Join(errs...)
}

// newStreamingZipEntryReader parses ONE zip local file header by hand (signature "PK\x03\x04" at
// offset 0, general-purpose flag at offset 6, compression method at offset 8, filename/extra-field
// lengths at offsets 26/28 — the fixed 30-byte local file header layout) and hands the remaining
// stream straight to compress/flate (DEFLATE) or a length-bounded pass-through (STORE) — no
// archive/zip, no io.ReaderAt, true forward-only streaming: flate's own end-of-stream marker
// terminates decoding, independent of whatever size the zip metadata declared.
//
// Assumes a single-entry zip, matching the real uo.zip export (verified in an earlier session, not
// inferred) — this reads only the FIRST entry; it never reads the central directory (which sits at
// the end of the file, unreachable without seeking or downloading everything, exactly what this
// function exists to avoid). If the export is ever repacked multi-entry, this silently reads only
// that first entry — a documented assumption, not a silent one. STORE is only handled when its
// length is knowable up front (no data-descriptor bit set, nonzero declared size); DEFLATE has no
// such restriction, and DEFLATE is what the real export actually uses (its ~10x compression ratio
// rules out STORE).
func newStreamingZipEntryReader(r io.Reader) (io.ReadCloser, error) {
	var header [30]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("uaedr: read zip local file header: %w", err)
	}
	if string(header[0:4]) != "PK\x03\x04" {
		return nil, fmt.Errorf("uaedr: not a zip local file header (got % x)", header[0:4])
	}
	flags := binary.LittleEndian.Uint16(header[6:8])
	method := binary.LittleEndian.Uint16(header[8:10])
	compressedSize := binary.LittleEndian.Uint32(header[18:22])
	nameLen := binary.LittleEndian.Uint16(header[26:28])
	extraLen := binary.LittleEndian.Uint16(header[28:30])

	if _, err := io.CopyN(io.Discard, r, int64(nameLen)+int64(extraLen)); err != nil {
		return nil, fmt.Errorf("uaedr: skip zip filename/extra fields: %w", err)
	}

	const methodDeflate, methodStore = 8, 0
	switch method {
	case methodDeflate:
		return flate.NewReader(r), nil
	case methodStore:
		const dataDescriptorBit = 0x0008
		if flags&dataDescriptorBit != 0 || compressedSize == 0 {
			return nil, errors.New("uaedr: zip entry is STORE-compressed with an unknown streamed length (data-descriptor flag set) — not supported")
		}
		return io.NopCloser(io.LimitReader(r, int64(compressedSize))), nil
	default:
		return nil, fmt.Errorf("uaedr: unsupported zip compression method %d (only DEFLATE=8 and STORE=0 are handled)", method)
	}
}
