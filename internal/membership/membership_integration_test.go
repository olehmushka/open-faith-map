// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves CreatePosition/FillPosition's conflict/already-filled error paths against a real Postgres
// instance — see internal/directory/directory_integration_test.go's own header comment for the
// invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/membership/... -run TestMembershipIntegration -v
package membership_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	directoryapp "github.com/olehmushka/open-faith-map/internal/directory/application"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
	"github.com/olehmushka/open-faith-map/internal/membership/application"
	"github.com/olehmushka/open-faith-map/internal/membership/domain"
)

func TestMembershipIntegration(t *testing.T) {
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

	dir := directoryapp.NewService(pool)
	mem := application.NewService(pool)

	unit, err := dir.CreateUnit(ctx, directorydomain.Unit{Name: "M10.5 membership test unit"})
	if err != nil {
		t.Fatalf("CreateUnit: %v", err)
	}
	var personID string
	t.Cleanup(func() {
		bg := context.Background()
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.membership_memberships WHERE unit_id = $1`, unit.ID); err != nil {
			t.Errorf("cleanup: delete memberships: %v", err)
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.membership_positions WHERE unit_id = $1`, unit.ID); err != nil {
			t.Errorf("cleanup: delete positions: %v", err)
		}
		if personID != "" {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, personID); err != nil {
				t.Errorf("cleanup: delete person: %v", err)
			}
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_units WHERE id = $1`, unit.ID); err != nil {
			t.Errorf("cleanup: delete unit: %v", err)
		}
	})

	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
		VALUES ('M10.5 Test Person', 'M10.5', 'Test Person') RETURNING id`,
	).Scan(&personID); err != nil {
		t.Fatalf("insert test person: %v", err)
	}
	pos, err := mem.CreatePosition(ctx, unit.ID, "admin", "Congregation Admin")
	if err != nil {
		t.Fatalf("CreatePosition: %v", err)
	}

	if _, err := mem.CreatePosition(ctx, unit.ID, "admin", "Congregation Admin (dup)"); err != domain.ErrPositionConflict {
		t.Errorf("CreatePosition(dup code) error = %v, want ErrPositionConflict", err)
	}

	positions, err := mem.ListPositionsByUnit(ctx, unit.ID)
	if err != nil {
		t.Fatalf("ListPositionsByUnit: %v", err)
	}
	if len(positions) != 1 || positions[0].ID != pos.ID {
		t.Errorf("ListPositionsByUnit = %+v, want exactly [pos]", positions)
	}

	if _, err := mem.FillPosition(ctx, personID, unit.ID, pos.ID); err != nil {
		t.Fatalf("FillPosition: %v", err)
	}
	if _, err := mem.FillPosition(ctx, personID, unit.ID, pos.ID); err != domain.ErrPositionAlreadyFilled {
		t.Errorf("FillPosition(already filled) error = %v, want ErrPositionAlreadyFilled", err)
	}

	// --- ListMembershipsByUnit (M10.7): the just-filled membership is returned.
	memberships, err := mem.ListMembershipsByUnit(ctx, unit.ID)
	if err != nil {
		t.Fatalf("ListMembershipsByUnit: %v", err)
	}
	if len(memberships) != 1 || memberships[0].PersonID != personID || memberships[0].PositionID != pos.ID {
		t.Errorf("ListMembershipsByUnit = %+v, want exactly one membership for person=%s position=%s", memberships, personID, pos.ID)
	}
}
