// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M10.6's content cutover — CreateSite/UpdateSiteTheme denied for a caller with no
// religionorg.manage grant on the site's congregation unit, allowed for a real congregation-admin
// grant — against a real Postgres instance. Same invocation shape as
// internal/registration/registration_integration_test.go:
//
//	DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/content/... -run TestContentIntegration -v
//
// The live HTTP surface is NOT exercised here — see registration_integration_test.go's own header
// comment for why. This test drives application.Service directly, with authz.NewContext standing in
// for what the identity middleware will inject once it's live.
package content_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/authz"
	authzadapters "github.com/olehmushka/open-faith-map/internal/authz/adapters"
	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	contentadapters "github.com/olehmushka/open-faith-map/internal/content/adapters"
	"github.com/olehmushka/open-faith-map/internal/content/application"
	contentdomain "github.com/olehmushka/open-faith-map/internal/content/domain"
	directoryadapters "github.com/olehmushka/open-faith-map/internal/directory/adapters"
	directoryapplication "github.com/olehmushka/open-faith-map/internal/directory/application"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
)

func TestContentIntegration(t *testing.T) {
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

	directorySvc := directoryapplication.NewService(pool)
	closurePort := directoryadapters.NewStore(pool)
	pdp := authzdomain.NewPDP(closurePort)
	authzStore := authzadapters.NewRepository(pool)
	authzSvc := authz.NewService(pdp, authzStore, pool)
	contentStore := contentadapters.NewRepository(pool)
	contentSvc := application.NewService(contentStore, authzSvc)

	var personIDs, unitIDs, siteIDs, assignmentIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range siteIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.content_sites WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete site %s: %v", id, err)
			}
		}
		for _, id := range assignmentIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete role assignment %s: %v", id, err)
			}
		}
		for _, id := range unitIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_closure WHERE ancestor_id = $1 OR descendant_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete closure rows for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_edges WHERE parent_id = $1 OR child_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete edges for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_units WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete unit %s: %v", id, err)
			}
		}
		for _, id := range personIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete person %s: %v", id, err)
			}
		}
	})

	insertPerson := func(name string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
			VALUES ($1, $1, 'Test') RETURNING id`, name,
		).Scan(&id); err != nil {
			t.Fatalf("insert person %s: %v", name, err)
		}
		personIDs = append(personIDs, id)
		return id
	}
	adminID := insertPerson("M10.6 Content Test Admin")
	otherID := insertPerson("M10.6 Content Test Other")

	unit, err := directorySvc.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "M10.6 Content Test Congregation"}, seed.RootUnitID, directorydomain.CanonicalGraphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge: %v", err)
	}
	unitIDs = append(unitIDs, unit.ID)

	// --- CreateSite is denied for a caller with no religionorg.manage grant on the unit.
	otherCtx := authz.NewContext(ctx, authz.Subject{PersonID: otherID})
	if _, err := contentSvc.CreateSite(otherCtx, contentdomain.CreateSiteInput{CongregationUnitRID: unit.ID, Slug: "m106-test"}); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("CreateSite by non-manager error = %v, want ErrForbidden", err)
	}

	// --- Grant adminID congregation-admin on unit (mirrors what registration's Approve grants a
	// real submitter).
	var assignmentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope)
		VALUES ($1, $2, $3, 'unit') RETURNING id`,
		adminID, seed.CongregationAdminRoleID, unit.ID,
	).Scan(&assignmentID); err != nil {
		t.Fatalf("grant congregation-admin: %v", err)
	}
	assignmentIDs = append(assignmentIDs, assignmentID)

	// --- CreateSite succeeds for the real congregation admin.
	adminCtx := authz.NewContext(ctx, authz.Subject{PersonID: adminID})
	site, err := contentSvc.CreateSite(adminCtx, contentdomain.CreateSiteInput{CongregationUnitRID: unit.ID, Slug: "m106-test"})
	if err != nil {
		t.Fatalf("CreateSite by congregation-admin: %v", err)
	}
	siteIDs = append(siteIDs, site.ID)
	if site.CongregationUnitRID != unit.ID {
		t.Errorf("CreateSite result = %+v, want CongregationUnitRID %s", site, unit.ID)
	}

	// --- UpdateSiteTheme resolves the target unit from the site itself, and gates the same way:
	// denied for the non-manager, allowed for the admin.
	if _, err := contentSvc.UpdateSiteTheme(otherCtx, site.ID, []byte(`{"color":"red"}`)); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("UpdateSiteTheme by non-manager error = %v, want ErrForbidden", err)
	}
	updated, err := contentSvc.UpdateSiteTheme(adminCtx, site.ID, []byte(`{"color":"blue"}`))
	if err != nil {
		t.Fatalf("UpdateSiteTheme by congregation-admin: %v", err)
	}
	var theme struct {
		Color string `json:"color"`
	}
	if err := json.Unmarshal(updated.Theme, &theme); err != nil || theme.Color != "blue" {
		t.Errorf("UpdateSiteTheme result theme = %s, want color=blue", updated.Theme)
	}

	// --- GetSite is the public read — no auth required, works with a plain background context.
	publicSite, err := contentSvc.GetSite(context.Background(), unit.ID)
	if err != nil {
		t.Fatalf("GetSite (public): %v", err)
	}
	if publicSite.ID != site.ID {
		t.Errorf("GetSite (public) = %+v, want id %s", publicSite, site.ID)
	}
}
