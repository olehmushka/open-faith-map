// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package uaedr

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestFetchFileMultiBatchResume is a regression test for a real bug (found live, 2026-08-14,
// against the real ~2M-record ЄДР export): fetchFile's returned cursor used to be
// cursorOf(skip + seen), double-counting the skip prefix on every call after the first (seen
// already includes it, since each reopened pass re-decodes from byte zero). The error compounds
// across calls until the inflated cursor races past the file's true record count and hits a real
// io.EOF far too early — indistinguishable from a clean, complete run unless the actual total is
// independently known. This fixture forces more than one batchSize=500 batch (batchSize is an
// unexported package const, not something a test can shrink), so a single-batch run — which never
// exercises skip > 0 at all, exactly how this bug went unnoticed for so long — cannot catch it.
func TestFetchFileMultiBatchResume(t *testing.T) {
	const wantTotal = 1200 // > 2*batchSize, forcing 3 Fetch calls (500, 500, 200)

	dir := t.TempDir()
	path := filepath.Join(dir, "uo.xml")
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="utf-8"?><SUBJECTS>`)
	for i := 0; i < wantTotal; i++ {
		fmt.Fprintf(&sb, `<SUBJECT><NAME>Church %d</NAME><OPF>РЕЛІГІЙНА ОРГАНІЗАЦІЯ</OPF><EDRPOU>%d</EDRPOU></SUBJECT>`, i, i)
	}
	sb.WriteString(`</SUBJECTS>`)
	if err := os.WriteFile(path, []byte(sb.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	c := &Connector{FilePath: path}
	ctx := context.Background()
	var cursor *string
	seenIDs := make(map[string]bool)
	calls := 0
	for {
		batch, next, err := c.Fetch(ctx, cursor)
		if err != nil {
			t.Fatalf("Fetch (call %d): %v", calls+1, err)
		}
		calls++
		for _, r := range batch {
			if seenIDs[r.SourceRecordID] {
				t.Fatalf("EDRPOU %s yielded more than once (call %d) — the cursor is re-serving already-yielded records", r.SourceRecordID, calls)
			}
			seenIDs[r.SourceRecordID] = true
		}
		if next == nil {
			break
		}
		cursor = next
		if calls > 10 {
			t.Fatal("too many Fetch calls — cursor likely stuck, not advancing correctly")
		}
	}

	if len(seenIDs) != wantTotal {
		t.Fatalf("got %d distinct records across %d calls, want %d — the multi-batch reopen-and-reskip cursor is not accounting for every record (the exact shape of the real double-counting bug)", len(seenIDs), calls, wantTotal)
	}
	if calls < 3 {
		t.Fatalf("only needed %d Fetch calls for %d records at batchSize=500 — this fixture should force at least 3, otherwise it never exercises skip > 0 and cannot catch the regression", calls, wantTotal)
	}
	for i := 0; i < wantTotal; i++ {
		if !seenIDs[strconv.Itoa(i)] {
			t.Fatalf("EDRPOU %d was never yielded", i)
		}
	}
}
