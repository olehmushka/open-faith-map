// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package uaedr

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// buildTestZip returns a single-entry zip (name/method as given) containing content — real bytes
// via archive/zip.Writer, not hand-rolled, so this package's own HTTP wiring is tested against a
// genuinely valid zip. The streaming zip-parsing logic itself now lives in go-uaedr, which has its
// own equivalent coverage (zip_test.go) — not duplicated here.
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

// TestFetchHTTPEndToEnd serves a small crafted zip over httptest and drives fetchHTTP through a
// full run via the same Fetch entrypoint RunConnector uses — proving this connector's own wiring of
// go-uaedr's OpenHTTP (streaming unzip + XML decode + OPF filter, all upstream) works end-to-end,
// not just each piece in isolation.
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
// rejected rather than racing on the shared upstream.Reader. TryLock happens synchronously at the
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
