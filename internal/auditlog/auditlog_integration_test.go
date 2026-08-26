// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves the audit-log store's keyset pagination and actor/target/date filters against a real
// Postgres instance — see internal/authz/authz_integration_test.go's own header comment for the
// invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/auditlog/... -run TestAuditLogIntegration -v
package auditlog_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/auditlog/adapters"
	"github.com/olehmushka/open-faith-map/internal/auditlog/domain"
)

func TestAuditLogIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL to run against a live Postgres instance")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	store := adapters.NewRepository(pool)

	var actorID string
	t.Cleanup(func() {
		bg := context.Background()
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log DISABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: disable reject_mutation: %v", err)
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_audit_log WHERE actor_person_id = $1`, actorID); err != nil {
			t.Errorf("cleanup: delete audit rows: %v", err)
		}
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log ENABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: re-enable reject_mutation: %v", err)
		}
		if actorID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, actorID); err != nil {
				t.Errorf("cleanup: delete actor person: %v", err)
			}
		}
	})

	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M11.2 Pagination Test Actor', 'M11.2', 'Pagination') RETURNING id`).Scan(&actorID); err != nil {
		t.Fatalf("insert actor person: %v", err)
	}

	// Five entries, explicit ascending created_at (1s apart — well above timestamptz precision) so
	// ordering is deterministic rather than relying on insert-speed timestamps.
	const n = 5
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		createdAt := base.Add(time.Duration(i) * time.Second)
		targetKind := "TEST_TARGET"
		if i%2 == 0 {
			targetKind = "TEST_TARGET_EVEN"
		}
		if err := pool.QueryRow(ctx, `
			INSERT INTO openfaithmap.identity_audit_log (actor_person_id, action, target_kind, target_id, created_at)
			VALUES ($1, 'TEST_ACTION', $2, $3, $4) RETURNING id`,
			actorID, targetKind, "target-"+string(rune('a'+i)), createdAt).Scan(&ids[i]); err != nil {
			t.Fatalf("insert entry %d: %v", i, err)
		}
	}

	// --- Keyset pagination: page size 2 over 5 rows, newest-first (created_at DESC), no gaps/dupes.
	var got []string
	var after *domain.PageCursor
	for {
		page, err := store.ListEntries(ctx, domain.Filter{ActorPersonID: actorID}, 3, after) // pageSize+1 = 3 for page size 2
		if err != nil {
			t.Fatalf("ListEntries: %v", err)
		}
		pageSize := 2
		if len(page) > pageSize {
			page = page[:pageSize]
		}
		for _, e := range page {
			got = append(got, e.ID)
		}
		if len(page) == 0 {
			break
		}
		last := page[len(page)-1]
		after = &domain.PageCursor{CreatedAt: last.CreatedAt, ID: last.ID}
		if len(page) < pageSize {
			break
		}
	}
	want := []string{ids[4], ids[3], ids[2], ids[1], ids[0]} // newest (i=4) first
	if len(got) != len(want) {
		t.Fatalf("paginated through %v, want %v (length mismatch)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("page position %d = %s, want %s (full: got=%v want=%v)", i, got[i], want[i], got, want)
		}
	}

	// --- Filter by target_kind narrows correctly.
	evenOnly, err := store.ListEntries(ctx, domain.Filter{ActorPersonID: actorID, TargetKind: "TEST_TARGET_EVEN"}, 100, nil)
	if err != nil {
		t.Fatalf("ListEntries (target_kind filter): %v", err)
	}
	if len(evenOnly) != 3 { // i=0,2,4
		t.Errorf("ListEntries(target_kind=TEST_TARGET_EVEN) returned %d entries, want 3", len(evenOnly))
	}

	// --- Filter by date range narrows correctly (from=base+1s, to=base+3s inclusive => i=1,2,3).
	from := base.Add(1 * time.Second)
	to := base.Add(3 * time.Second)
	ranged, err := store.ListEntries(ctx, domain.Filter{ActorPersonID: actorID, From: &from, To: &to}, 100, nil)
	if err != nil {
		t.Fatalf("ListEntries (date range filter): %v", err)
	}
	if len(ranged) != 3 {
		t.Errorf("ListEntries(from=%v, to=%v) returned %d entries, want 3", from, to, len(ranged))
	}

	// --- Filter by target_id narrows to exactly one.
	single, err := store.ListEntries(ctx, domain.Filter{ActorPersonID: actorID, TargetID: "target-a"}, 100, nil)
	if err != nil {
		t.Fatalf("ListEntries (target_id filter): %v", err)
	}
	if len(single) != 1 || single[0].ID != ids[0] {
		t.Errorf("ListEntries(target_id=target-a) = %+v, want exactly entry %s", single, ids[0])
	}
}
