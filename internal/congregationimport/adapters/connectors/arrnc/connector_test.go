// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package arrnc

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// writeFixture writes an n-row CSV in this source's real shape (no header, 5 columns) and returns
// its path.
func writeFixture(t *testing.T, n int) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "registro-culto-export.csv")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	w := csv.NewWriter(f)
	for i := 0; i < n; i++ {
		if err := w.Write([]string{
			fmt.Sprintf("IGLESIA %d", i),
			fmt.Sprintf("CALLE %d", i),
			"CAPITAL FEDERAL",
			"Buenos Aires",
			strconv.Itoa(i % 7), // several rows deliberately share a CI, mirroring the real "filial" shape
		}); err != nil {
			t.Fatal(err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestFetchMultiBatch is this design's own equivalent of uaedr's TestFetchFileMultiBatchResume —
// but the failure mode it guards against is different by construction: this connector loads the
// whole file once and slices an already-materialized slice, so there is no reopen-and-reskip step
// to double-count (uaedr's real bug). What CAN go wrong here is an off-by-one at a batch boundary
// (a row dropped or yielded twice right at offset==batchSize), so that's what this asserts.
func TestFetchMultiBatch(t *testing.T) {
	const wantTotal = 1200 // > 2*batchSize, forcing 3 Fetch calls (500, 500, 200)
	path := writeFixture(t, wantTotal)

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
				t.Fatalf("SourceRecordID %s yielded more than once (call %d) — a batch-boundary bug", r.SourceRecordID, calls)
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
		t.Fatalf("got %d distinct records across %d calls, want %d — a row was dropped or double-counted at a batch boundary", len(seenIDs), calls, wantTotal)
	}
	if calls != 3 {
		t.Fatalf("got %d Fetch calls for %d records at batchSize=500, want exactly 3 (500,500,200)", calls, wantTotal)
	}
}

// TestFetchSingleBatchExhaustion checks the common case (fewer rows than one batch) returns a nil
// cursor on the very first call.
func TestFetchSingleBatchExhaustion(t *testing.T) {
	path := writeFixture(t, 5)
	c := &Connector{FilePath: path}
	batch, next, err := c.Fetch(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) != 5 {
		t.Fatalf("got %d records, want 5", len(batch))
	}
	if next != nil {
		t.Fatalf("got non-nil next cursor %q for a fully-exhausted single batch", *next)
	}
}

// TestSourceRecordID is the regression test for this package's own real, live finding: CI
// (column 4) is shared across every branch ("filial") row of one institute, so it cannot be the
// SourceRecordID — content (name+address+locality+province) must be, instead.
func TestSourceRecordID(t *testing.T) {
	a := cultRow{Name: "IGLESIA NUEVA APOSTOLICA - FILIAL 1", Address: "CALLE 1", Locality: "BANFIELD", Province: "Buenos Aires", CI: "4"}
	b := cultRow{Name: "IGLESIA NUEVA APOSTOLICA - FILIAL 2", Address: "CALLE 2", Locality: "CORDOBA", Province: "Córdoba", CI: "4"} // same CI, real distinct branch
	if sourceRecordID(a) == sourceRecordID(b) {
		t.Fatal("two rows with the same CI but different content got the same SourceRecordID — this would collapse distinct real branch locations into one candidate")
	}

	// Same content, different casing — real files mix casing (see christianfilter's own diacritics
	// finding); the ID must still be identical, so a re-run of the connector upserts in place.
	c := cultRow{Name: "Iglesia Nueva Apostolica - Filial 1", Address: "Calle 1", Locality: "Banfield", Province: "buenos aires", CI: "4"}
	if sourceRecordID(a) != sourceRecordID(c) {
		t.Fatal("identical content differing only by case got different SourceRecordIDs — a re-run would duplicate instead of upserting")
	}
}

func TestNewValidation(t *testing.T) {
	if _, err := New("", "", nil); err == nil {
		t.Fatal("New with neither filePath nor sourceURL should error")
	}
	if _, err := New("a", "b", nil); err == nil {
		t.Fatal("New with both filePath and sourceURL should error")
	}
	if _, err := New("a", "", nil); err != nil {
		t.Fatalf("New with only filePath should succeed: %v", err)
	}
}
