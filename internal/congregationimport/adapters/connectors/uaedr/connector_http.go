// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package uaedr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	upstream "github.com/olehmushka/go-uaedr"
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
// race on the same upstream.Reader. mu guards the two fields below it, which are mutated both by
// fetchHTTP's own normal-completion path and by Close's out-of-band cleanup path.
type httpModeState struct {
	runLock     sync.Mutex
	mu          sync.Mutex
	runLockHeld bool
	stream      *httpStream
}

// httpStream is one run's live upstream.Reader, held open across fetchHTTP calls, plus the
// io.Closer go-uaedr's OpenHTTP hands back (releases the response body and, when SourceURL ends in
// .zip, the streaming zip decompressor — both internal to go-uaedr, this package never touches
// them directly).
type httpStream struct {
	reader *upstream.Reader
	closer io.Closer
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

// openHTTPStream delegates the GET + decompression (if SourceURL is a .zip) + charset-aware XML
// decoding entirely to go-uaedr's OpenHTTP — never buffers the response, since a real export can
// run several hundred MB. This connector's own job is just wiring: pass its httpClient and citation
// user-agent through, wrap the result in httpStream.
func (c *Connector) openHTTPStream(ctx context.Context) (*httpStream, error) {
	r, closer, err := upstream.OpenHTTP(ctx, c.SourceURL, upstream.OpenHTTPOptions{
		HTTPClient: c.httpClient,
		UserAgent:  c.Citation().UserAgent,
	})
	if err != nil {
		return nil, err
	}
	return &httpStream{reader: r, closer: closer}, nil
}

// readBatch pulls up to size religious-org records from an already-open, already-positioned
// upstream.Reader — no skip/seen bookkeeping, since HTTP mode never reopens from scratch. done=true
// means the source is exhausted (io.EOF), matching fetchFile's nil-nextCursor signal.
func (st *httpStream) readBatch(ctx context.Context, size int) (batch []domain.RawRecord, done bool, err error) {
	for {
		if cerr := ctx.Err(); cerr != nil {
			return batch, false, cerr
		}
		s, terr := st.reader.Next()
		if terr == io.EOF {
			return batch, true, nil
		}
		if terr != nil {
			return batch, false, fmt.Errorf("uaedr: decode: %w", terr)
		}
		if !s.IsReligiousOrg() {
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
	return st.closer.Close()
}
