// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package uaedr

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// buildTestZip returns a single-entry zip (name/method as given) containing content — real bytes
// via archive/zip.Writer, not hand-rolled, so the streaming reader is tested against a genuinely
// valid zip, not a reader's own idea of one.
func buildTestZip(t *testing.T, name string, method uint16, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: method})
	if err != nil {
		t.Fatalf("CreateHeader: %v", err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatalf("write entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip Close: %v", err)
	}
	return buf.Bytes()
}

func TestNewStreamingZipEntryReaderDeflate(t *testing.T) {
	want := []byte(strings.Repeat("<SUBJECT><NAME>test</NAME></SUBJECT>", 1000)) // compressible
	zb := buildTestZip(t, "uo.xml", zip.Deflate, want)

	rc, err := newStreamingZipEntryReader(bytes.NewReader(zb))
	if err != nil {
		t.Fatalf("newStreamingZipEntryReader: %v", err)
	}
	defer func() { _ = rc.Close() }()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read decompressed: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("decompressed content mismatch: got %d bytes, want %d bytes", len(got), len(want))
	}
}

// buildRawStoreLocalFileHeader hand-builds a zip local file header with the STORE method and the
// data-descriptor bit (general-purpose flag bit 3) explicitly clear, i.e. sizes declared up front
// in the header itself — the one shape newStreamingZipEntryReader can stream STORE from without a
// central directory. archive/zip.Writer cannot produce this: empirically (verified against this
// Go toolchain), it always sets the data-descriptor bit for a streamed Write, even against a
// seekable *os.File target, so this case is built by hand against the exact fixed-offset layout
// newStreamingZipEntryReader itself parses.
func buildRawStoreLocalFileHeader(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteString("PK\x03\x04")
	writeUint16LE(&buf, 20)                   // version needed — arbitrary, unread by the parser
	writeUint16LE(&buf, 0)                    // general-purpose flag — bit 3 (data descriptor) clear
	writeUint16LE(&buf, zip.Store)            // compression method
	writeUint16LE(&buf, 0)                    // last mod time — arbitrary
	writeUint16LE(&buf, 0)                    // last mod date — arbitrary
	writeUint32LE(&buf, 0)                    // CRC-32 — unread by the parser
	writeUint32LE(&buf, uint32(len(content))) // compressed size == uncompressed size for STORE
	writeUint32LE(&buf, uint32(len(content))) // uncompressed size
	writeUint16LE(&buf, uint16(len(name)))    // filename length
	writeUint16LE(&buf, 0)                    // extra field length
	buf.WriteString(name)
	buf.Write(content)
	return buf.Bytes()
}

func writeUint16LE(buf *bytes.Buffer, v uint16) {
	buf.WriteByte(byte(v))
	buf.WriteByte(byte(v >> 8))
}

func writeUint32LE(buf *bytes.Buffer, v uint32) {
	buf.WriteByte(byte(v))
	buf.WriteByte(byte(v >> 8))
	buf.WriteByte(byte(v >> 16))
	buf.WriteByte(byte(v >> 24))
}

func TestNewStreamingZipEntryReaderStore(t *testing.T) {
	want := []byte("<SUBJECT><NAME>stored, not deflated</NAME></SUBJECT>")
	zb := buildRawStoreLocalFileHeader(t, "uo.xml", want)

	rc, err := newStreamingZipEntryReader(bytes.NewReader(zb))
	if err != nil {
		t.Fatalf("newStreamingZipEntryReader: %v", err)
	}
	defer func() { _ = rc.Close() }()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read stored content: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("stored content mismatch: got %q, want %q", got, want)
	}
}

// TestNewStreamingZipEntryReaderStoreWithDataDescriptorRejected confirms the one documented gap in
// newStreamingZipEntryReader's own doc comment: STORE with the data-descriptor bit set (unknown
// length up front) is refused with a clear error rather than silently reading garbage or hanging.
// This is also, empirically, what Go's own archive/zip.Writer produces for a streamed STORE
// entry (verified via buildTestZip below) — not just a hypothetical shape.
func TestNewStreamingZipEntryReaderStoreWithDataDescriptorRejected(t *testing.T) {
	zb := buildTestZip(t, "uo.xml", zip.Store, []byte("some content"))
	_, err := newStreamingZipEntryReader(bytes.NewReader(zb))
	if err == nil {
		t.Fatal("expected an error for STORE with the data-descriptor bit set, got nil")
	}
}

func TestNewStreamingZipEntryReaderBadSignature(t *testing.T) {
	_, err := newStreamingZipEntryReader(bytes.NewReader(bytes.Repeat([]byte{0x00}, 30)))
	if err == nil {
		t.Fatal("expected an error for a non-zip byte stream, got nil")
	}
}

func TestNewStreamingZipEntryReaderTruncated(t *testing.T) {
	_, err := newStreamingZipEntryReader(bytes.NewReader([]byte("PK\x03\x04short")))
	if err == nil {
		t.Fatal("expected an error for a truncated local file header, got nil")
	}
}

// TestFetchHTTPEndToEnd serves a small crafted zip over httptest and drives fetchHTTP through a
// full run via the same Fetch entrypoint RunConnector uses — proving the streaming unzip + XML
// decode + OPF filter chain works end-to-end, not just each piece in isolation.
func TestFetchHTTPEndToEnd(t *testing.T) {
	xmlBody := `<?xml version="1.0" encoding="utf-8"?><SUBJECTS>` +
		`<SUBJECT><RECORD>1</RECORD><NAME>Church A</NAME><OPF>РЕЛІГІЙНА ОРГАНІЗАЦІЯ</OPF><EDRPOU>111</EDRPOU></SUBJECT>` +
		`<SUBJECT><RECORD>2</RECORD><NAME>Some LLC</NAME><OPF>ТОВАРИСТВО</OPF><EDRPOU>222</EDRPOU></SUBJECT>` +
		`<SUBJECT><RECORD>3</RECORD><NAME>Church B</NAME><OPF>релігійна організація</OPF><EDRPOU>333</EDRPOU></SUBJECT>` +
		`</SUBJECTS>`
	zb := buildTestZip(t, "uo.xml", zip.Deflate, []byte(xmlBody))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zb)
	}))
	defer srv.Close()

	c, err := New("", srv.URL+"/uo.zip", srv.Client())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	var got []string
	var cursor *string
	for {
		batch, next, ferr := c.Fetch(ctx, cursor)
		if ferr != nil {
			t.Fatalf("Fetch: %v", ferr)
		}
		for _, r := range batch {
			got = append(got, r.SourceRecordID)
		}
		if next == nil {
			break
		}
		cursor = next
	}

	if len(got) != 2 || got[0] != "111" || got[1] != "333" {
		t.Fatalf("got %v, want [111 333] (the LLC record must be filtered out by OPF)", got)
	}

	// The run released its lock/state on exhaustion — a fresh run must be startable again.
	if _, _, err := c.Fetch(ctx, nil); err != nil {
		t.Fatalf("second run after exhaustion: %v", err)
	}
}

// TestFetchHTTPConcurrentGuard proves a second concurrent run on the same connector instance is
// rejected rather than racing on the shared decoder/response. TryLock happens synchronously at the
// top of fetchHTTP, before the HTTP request is even built, so blocking the first goroutine's
// request mid-flight (via the started/release handshake below) is enough to deterministically
// observe the second call's rejection — no timing-based sleep needed.
func TestFetchHTTPConcurrentGuard(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once

	xmlBody := `<?xml version="1.0" encoding="utf-8"?><SUBJECTS>` +
		`<SUBJECT><NAME>Church</NAME><OPF>РЕЛІГІЙНА ОРГАНІЗАЦІЯ</OPF><EDRPOU>1</EDRPOU></SUBJECT></SUBJECTS>`
	zb := buildTestZip(t, "uo.xml", zip.Deflate, []byte(xmlBody))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		<-release
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zb)
	}))
	defer srv.Close()

	c, err := New("", srv.URL+"/uo.zip", srv.Client())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		_, _, ferr := c.Fetch(ctx, nil)
		done <- ferr
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the first Fetch's request to reach the server")
	}

	_, _, err = c.Fetch(ctx, nil)
	if err == nil {
		t.Fatal("expected the second concurrent Fetch(cursor=nil) call to be rejected, got nil error")
	}

	close(release)
	if ferr := <-done; ferr != nil {
		t.Fatalf("first Fetch call: %v", ferr)
	}
}

func TestNewMutualExclusion(t *testing.T) {
	if _, err := New("", "", nil); err == nil {
		t.Fatal("expected an error when neither filePath nor sourceURL is set")
	}
	if _, err := New("/tmp/uo.xml", "https://example.com/uo.zip", nil); err == nil {
		t.Fatal("expected an error when both filePath and sourceURL are set")
	}
	if _, err := New("/tmp/uo.xml", "", nil); err != nil {
		t.Fatalf("filePath-only should be valid: %v", err)
	}
	if _, err := New("", "https://example.com/uo.zip", nil); err != nil {
		t.Fatalf("sourceURL-only should be valid: %v", err)
	}
}
