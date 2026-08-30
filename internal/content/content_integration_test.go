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
	"bytes"
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
	locationapplication "github.com/olehmushka/open-faith-map/internal/location/application"
	locationdomain "github.com/olehmushka/open-faith-map/internal/location/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
	religionadapters "github.com/olehmushka/open-faith-map/internal/religion/adapters"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
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
	closurePort := directoryadapters.NewRepository(pool)
	pdp := authzdomain.NewPDP(closurePort)
	authzStore := authzadapters.NewRepository(pool)
	authzSvc := authz.NewService(pdp, authzStore, pool)
	religionSvc := religionapplication.NewService(pool, directorySvc, authzSvc)
	contentStore := contentadapters.NewRepository(pool)
	contentSvc := application.NewService(contentStore, authzSvc, religionSvc, "m14-7-test-preview-hmac-key", application.Config{
		RootUnitID: seed.RootUnitID,
	})

	var personIDs, unitIDs, siteIDs, assignmentIDs, documentIDs, religionSiteIDs, locationIDs, blockTypeIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		// M14.13: content_block_types.code is uniquely indexed, so a leftover row from a prior run
		// would collide on the next — hard-deleted here (UpdateBlockType only ever retires, never
		// deletes, since a real catalog row can be referenced by real content_blocks rows).
		for _, id := range blockTypeIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.content_block_types WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete block type %s: %v", id, err)
			}
		}
		// Reverse creation order: content_documents.parent_document_id has no ON DELETE CASCADE
		// (app-enforced 3-level cap, not a DB one — see checkParentDepth), and M14.10 is the first
		// case in this file to create a real parent/child chain, so a child must be deleted before
		// its parent or the FK rejects the parent's delete. Reverse order also happens to be a
		// no-op for every pre-existing (non-nested) document in this file.
		for i := len(documentIDs) - 1; i >= 0; i-- {
			id := documentIDs[i]
			// ON DELETE CASCADE from content_blocks; content_documents.site_id has no cascade, so
			// this must run before the content_sites delete below.
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.content_documents WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete document %s: %v", id, err)
			}
		}
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
		// M14.11: religion_sites before location_locations (FK), both before directory_units below —
		// same ordering internal/religion/religion_integration_test.go's own cleanup already uses.
		for _, id := range religionSiteIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.religion_sites WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete religion site %s: %v", id, err)
			}
		}
		for _, id := range locationIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.location_locations WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete location %s: %v", id, err)
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
	// M14.12: theme is now D-CuratedTheme's fixed vocabulary, not a free-form object — "color":"blue"
	// would be rejected by the write-time schema gate (see TestContentIntegrationTheme below for
	// that gate's own coverage).
	updated, err := contentSvc.UpdateSiteTheme(adminCtx, site.ID, []byte(`{"accent":"indigo","mode":"light"}`))
	if err != nil {
		t.Fatalf("UpdateSiteTheme by congregation-admin: %v", err)
	}
	var theme struct {
		Accent string `json:"accent"`
	}
	if err := json.Unmarshal(updated.Theme, &theme); err != nil || theme.Accent != "indigo" {
		t.Errorf("UpdateSiteTheme result theme = %s, want accent=indigo", updated.Theme)
	}

	// --- M14.12: UpdateSiteTheme rejects a value outside D-CuratedTheme's fixed vocabulary with a
	// typed ThemeInvalidError naming the field — a raw hex, never a curated token name.
	_, err = contentSvc.UpdateSiteTheme(adminCtx, site.ID, []byte(`{"accent":"#ff00ff"}`))
	var themeInvalidErr *contentdomain.ThemeInvalidError
	if !errors.As(err, &themeInvalidErr) {
		t.Errorf("UpdateSiteTheme(raw hex accent) error = %v, want ThemeInvalidError", err)
	} else if themeInvalidErr.Field != "accent" {
		t.Errorf("UpdateSiteTheme(raw hex accent) Field = %q, want %q", themeInvalidErr.Field, "accent")
	}

	// --- M14.12: an accent/mode pair that fails WCAG AA contrast is rejected with a typed
	// ThemeContrastFailedError naming the pair, even though both values are individually curated.
	_, err = contentSvc.UpdateSiteTheme(adminCtx, site.ID, []byte(`{"accent":"indigo","mode":"dark"}`))
	var contrastErr *contentdomain.ThemeContrastFailedError
	if !errors.As(err, &contrastErr) {
		t.Errorf("UpdateSiteTheme(indigo/dark) error = %v, want ThemeContrastFailedError", err)
	} else if contrastErr.Accent != "indigo" || contrastErr.Mode != "dark" {
		t.Errorf("UpdateSiteTheme(indigo/dark) = %+v, want {Accent:indigo Mode:dark}", contrastErr)
	}

	// --- GetSite is the public read — no auth required, works with a plain background context.
	publicSite, err := contentSvc.GetSite(context.Background(), unit.ID)
	if err != nil {
		t.Fatalf("GetSite (public): %v", err)
	}
	if publicSite.ID != site.ID {
		t.Errorf("GetSite (public) = %+v, want id %s", publicSite, site.ID)
	}

	// --- M14.9: GetSiteBySlug is the public read the tenant-subdomain proxy resolves a Host header
	// through — same anonymous shape as GetSite, keyed by slug instead of unit RID.
	siteBySlug, err := contentSvc.GetSiteBySlug(context.Background(), site.Slug)
	if err != nil {
		t.Fatalf("GetSiteBySlug (public): %v", err)
	}
	if siteBySlug.ID != site.ID {
		t.Errorf("GetSiteBySlug (public) = %+v, want id %s", siteBySlug, site.ID)
	}
	if _, err := contentSvc.GetSiteBySlug(context.Background(), "no-such-slug-m149"); !errors.Is(err, contentdomain.ErrSiteNotFound) {
		t.Errorf("GetSiteBySlug (unknown slug) error = %v, want ErrSiteNotFound", err)
	}

	// --- M14.11: GetSiteChrome degrades gracefully before unit has any religion_sites row —
	// congregationName falls back to the site's own slug, address/schedules empty, no error.
	chromeBeforeReligionSite, err := contentSvc.GetSiteChrome(context.Background(), site.ID)
	if err != nil {
		t.Fatalf("GetSiteChrome (no religion site): %v", err)
	}
	if chromeBeforeReligionSite.CongregationName != site.Slug || chromeBeforeReligionSite.Address != nil || len(chromeBeforeReligionSite.Schedules) != 0 {
		t.Errorf("GetSiteChrome (no religion site) = %+v, want CongregationName=%s, Address=nil, no schedules", chromeBeforeReligionSite, site.Slug)
	}

	// --- M14.11: UpdateSiteChrome is content.manage-gated the same way UpdateSiteTheme is, and
	// persists logoUrl/socialLinks wholesale.
	if _, err := contentSvc.UpdateSiteChrome(otherCtx, site.ID, nil, contentdomain.SocialLinks{}); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("UpdateSiteChrome by non-manager error = %v, want ErrForbidden", err)
	}
	logoURL := "https://example.org/logo.png"
	fbURL := "https://facebook.com/example"
	chromedSite, err := contentSvc.UpdateSiteChrome(adminCtx, site.ID, &logoURL, contentdomain.SocialLinks{Facebook: &fbURL})
	if err != nil {
		t.Fatalf("UpdateSiteChrome by congregation-admin: %v", err)
	}
	if chromedSite.LogoURL == nil || *chromedSite.LogoURL != logoURL || chromedSite.SocialLinks.Facebook == nil || *chromedSite.SocialLinks.Facebook != fbURL {
		t.Errorf("UpdateSiteChrome result = %+v, want LogoURL=%s Facebook=%s", chromedSite, logoURL, fbURL)
	}

	// --- M14.11: GetSiteChrome composes live name/address/schedules from religion_sites/
	// religion_service_schedules once the unit has a real religion site — proving content never
	// copies that data, it reads it live at request time (docs/modules/content.md's invariant).
	loc := locationapplication.NewService(pool)
	var countryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.refdata_countries WHERE code = 'UA'`).Scan(&countryID); err != nil {
		t.Fatalf("lookup UA country: %v", err)
	}
	var churchSiteTypeID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.religion_site_types WHERE code = 'church' LIMIT 1`).Scan(&churchSiteTypeID); err != nil {
		t.Fatalf("lookup church site type: %v", err)
	}
	chromeLoc, err := loc.CreateLocation(ctx, locationdomain.LocationInput{Latitude: 50.4501, Longitude: 30.5234, CountryID: countryID})
	if err != nil {
		t.Fatalf("CreateLocation (m14.11 chrome test): %v", err)
	}
	locationIDs = append(locationIDs, chromeLoc.ID)
	if _, err := pool.Exec(ctx, `UPDATE openfaithmap.location_locations SET locality = 'Kyiv', admin_area_1 = 'Kyiv City', street = 'Khreshchatyk St', house_number = '1' WHERE id = $1`, chromeLoc.ID); err != nil {
		t.Fatalf("set location address: %v", err)
	}
	religionSite, err := religionSvc.CreateSite(ctx, religionadapters.CreateSiteInput{OrgUnitID: unit.ID, LocationID: chromeLoc.ID, SiteTypeID: churchSiteTypeID, IsPrimary: true})
	if err != nil {
		t.Fatalf("religion CreateSite (m14.11 chrome test): %v", err)
	}
	religionSiteIDs = append(religionSiteIDs, religionSite.ID)
	var mainServiceTypeID string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.religion_service_types WHERE code = 'main' AND tradition_taxon_id IS NULL`).Scan(&mainServiceTypeID); err != nil {
		t.Fatalf("lookup main service type: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO openfaithmap.religion_service_schedules (site_id, service_type_id, day_of_week, start_time, end_time, timezone, language)
		VALUES ($1, $2, 0, '10:00', '11:30', 'Europe/Kyiv', 'uk')`, religionSite.ID, mainServiceTypeID); err != nil {
		t.Fatalf("insert service schedule (m14.11 chrome test): %v", err)
	}

	chromeWithReligionSite, err := contentSvc.GetSiteChrome(context.Background(), site.ID)
	if err != nil {
		t.Fatalf("GetSiteChrome (with religion site): %v", err)
	}
	if chromeWithReligionSite.CongregationName != unit.Name {
		t.Errorf("GetSiteChrome.CongregationName = %q, want %q", chromeWithReligionSite.CongregationName, unit.Name)
	}
	if chromeWithReligionSite.Address == nil || *chromeWithReligionSite.Address == "" {
		t.Errorf("GetSiteChrome.Address = %v, want a non-empty coarsened address", chromeWithReligionSite.Address)
	}
	if len(chromeWithReligionSite.Schedules) != 1 {
		t.Fatalf("GetSiteChrome.Schedules = %+v, want exactly one row", chromeWithReligionSite.Schedules)
	}
	sch := chromeWithReligionSite.Schedules[0]
	if sch.DayOfWeek == nil || *sch.DayOfWeek != 0 || sch.StartTime == nil || *sch.StartTime != "10:00" || sch.EndTime == nil || *sch.EndTime != "11:30" || sch.Language == nil || *sch.Language != "uk" {
		t.Errorf("GetSiteChrome.Schedules[0] = %+v, want DayOfWeek=0 StartTime=10:00 EndTime=11:30 Language=uk", sch)
	}
	if chromeWithReligionSite.LogoURL == nil || *chromeWithReligionSite.LogoURL != logoURL {
		t.Errorf("GetSiteChrome.LogoURL = %v, want %s (content_sites' own column, unaffected by the religion read)", chromeWithReligionSite.LogoURL, logoURL)
	}

	// --- M14.11: a `hidden`-precision religion site hides the address (CoarsenAddress's own
	// ok=false case) but NOT the congregation's own name — a site showing its own name on its own
	// subdomain is not the discovery-search leak D-DiscoveryAddressPrecision guards against.
	if _, err := pool.Exec(ctx, `UPDATE openfaithmap.religion_sites SET public_precision = 'hidden' WHERE id = $1`, religionSite.ID); err != nil {
		t.Fatalf("mark religion site hidden: %v", err)
	}
	chromeHidden, err := contentSvc.GetSiteChrome(context.Background(), site.ID)
	if err != nil {
		t.Fatalf("GetSiteChrome (hidden precision): %v", err)
	}
	if chromeHidden.Address != nil {
		t.Errorf("GetSiteChrome (hidden precision).Address = %v, want nil", chromeHidden.Address)
	}
	if chromeHidden.CongregationName != unit.Name {
		t.Errorf("GetSiteChrome (hidden precision).CongregationName = %q, want %q (name still shown)", chromeHidden.CongregationName, unit.Name)
	}

	// --- M14.9/D-TenantSubdomains: a reserved slug is rejected at CreateSite, server-side —
	// content_sites.slug is a hostname component as of this milestone.
	if _, err := contentSvc.CreateSite(adminCtx, contentdomain.CreateSiteInput{CongregationUnitRID: unit.ID, Slug: "admin"}); !errors.As(err, new(*contentdomain.SlugReservedError)) {
		t.Errorf("CreateSite(slug=admin) error = %v, want *SlugReservedError", err)
	}

	// --- M14.9/U16: content.manage is now its own permission (migrations/0026_content_manage_permission.sql),
	// not a byproduct of registration-operator's religionorg.manage subtree grant on root. Two
	// things to prove:
	//   1. A congregation-admin granted on unit is denied on an unrelated unit B — genuine
	//      cross-tenant isolation (docs/modules/content.md's named "denial path needs test
	//      coverage" gap).
	//   2. A registration-operator granted the exact same *unit-scoped* shape congregation-admin
	//      holds (not their real subtree grant) is still denied — proving content.manage itself
	//      gates this, not incidental scope.
	unitB, err := directorySvc.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "M14.9 Unrelated Congregation"}, seed.RootUnitID, directorydomain.CanonicalGraphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge (unitB): %v", err)
	}
	unitIDs = append(unitIDs, unitB.ID)
	if _, err := contentSvc.CreateSite(adminCtx, contentdomain.CreateSiteInput{CongregationUnitRID: unitB.ID, Slug: "m149-cross-tenant"}); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("CreateSite by unit-A congregation-admin on unrelated unit B error = %v, want ErrForbidden", err)
	}

	operatorID := insertPerson("M14.9 Registration Operator")
	var operatorAssignmentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope)
		VALUES ($1, $2, $3, 'unit') RETURNING id`,
		operatorID, seed.RegistrationOperatorRoleID, unit.ID,
	).Scan(&operatorAssignmentID); err != nil {
		t.Fatalf("grant registration-operator (unit-scoped, M14.9 test): %v", err)
	}
	assignmentIDs = append(assignmentIDs, operatorAssignmentID)
	operatorCtx := authz.NewContext(ctx, authz.Subject{PersonID: operatorID})
	if _, err := contentSvc.UpdateSiteTheme(operatorCtx, site.ID, []byte(`{"color":"green"}`)); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("UpdateSiteTheme by registration-operator (no content.manage grant) error = %v, want ErrForbidden", err)
	}

	// --- M14.1: PutBlocks enforces D-PublicSiteCSP's URL scheme/embed-host allowlist at write
	// time, with a typed BlockUrlNotAllowedError naming the offending field.
	doc, err := contentSvc.CreateDocument(adminCtx, site.ID, contentdomain.CreateDocumentInput{
		Kind: contentdomain.KindPage, Locale: "en", Slug: "m14-1-blocks-test",
	})
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	documentIDs = append(documentIDs, doc.ID)

	_, err = contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "button", Position: 0, Data: json.RawMessage(`{"label":"x","href":"javascript:alert(1)"}`)},
	})
	var urlErr *contentdomain.BlockUrlNotAllowedError
	if !errors.As(err, &urlErr) {
		t.Errorf("PutBlocks(button, javascript: href) error = %v, want BlockUrlNotAllowedError", err)
	} else if urlErr.Field != "href" {
		t.Errorf("PutBlocks(button, javascript: href) error field = %q, want %q", urlErr.Field, "href")
	}

	if _, err := contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "button", Position: 0, Data: json.RawMessage(`{"label":"x","href":"https://example.org"}`)},
	}); err != nil {
		t.Errorf("PutBlocks(button, https: href) error = %v, want nil", err)
	}

	_, err = contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "social_embed", Position: 0, Data: json.RawMessage(`{"platform":"facebook","url":"https://evil.example.com/post/1"}`)},
	})
	var socialErr *contentdomain.BlockUrlNotAllowedError
	if !errors.As(err, &socialErr) {
		t.Errorf("PutBlocks(social_embed, host mismatch) error = %v, want BlockUrlNotAllowedError", err)
	} else if socialErr.Field != "url" {
		t.Errorf("PutBlocks(social_embed, host mismatch) error field = %q, want %q", socialErr.Field, "url")
	}

	// --- M14.2: paragraph.text is now a richText node array (D-RichTextNodes). A bolded run with
	// an inline link round-trips through PutBlocks/GetBlocks.
	richParagraph, err := contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "paragraph", Position: 10, Data: json.RawMessage(`{"text":[
			{"type":"text","text":"Hello "},
			{"type":"text","text":"bold","marks":[{"type":"bold"}]},
			{"type":"text","text":" and a "},
			{"type":"text","text":"link","marks":[{"type":"link","href":"https://example.org"}]}
		]}`)},
	})
	if err != nil {
		t.Fatalf("PutBlocks(paragraph, richText): %v", err)
	}
	if len(richParagraph) != 1 {
		t.Fatalf("PutBlocks(paragraph, richText) returned %d blocks, want 1", len(richParagraph))
	}

	// A link mark's href goes through the exact same URL-scheme allowlist as any other URL field.
	_, err = contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "paragraph", Position: 10, Data: json.RawMessage(`{"text":[
			{"type":"text","text":"click","marks":[{"type":"link","href":"javascript:alert(1)"}]}
		]}`)},
	})
	var richTextErr *contentdomain.BlockUrlNotAllowedError
	if !errors.As(err, &richTextErr) {
		t.Errorf("PutBlocks(paragraph, javascript: link) error = %v, want BlockUrlNotAllowedError", err)
	} else if richTextErr.Field != "text" {
		t.Errorf("PutBlocks(paragraph, javascript: link) error field = %q, want %q", richTextErr.Field, "text")
	}

	// A `list` block round-trips; a bad-scheme link nested inside a list item is rejected the same
	// way, naming the block's own field ("content").
	if _, err := contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "list", Position: 10, Data: json.RawMessage(`{"content":[
			{"type":"list","style":"bullet","items":[
				{"type":"listItem","content":[{"type":"text","text":"First item"}]},
				{"type":"listItem","content":[{"type":"text","text":"Second item"}]}
			]}
		]}`)},
	}); err != nil {
		t.Errorf("PutBlocks(list, valid): %v", err)
	}

	_, err = contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "list", Position: 10, Data: json.RawMessage(`{"content":[
			{"type":"list","style":"bullet","items":[
				{"type":"listItem","content":[{"type":"text","text":"bad","marks":[{"type":"link","href":"javascript:alert(1)"}]}]}
			]}
		]}`)},
	})
	var listErr *contentdomain.BlockUrlNotAllowedError
	if !errors.As(err, &listErr) {
		t.Errorf("PutBlocks(list, javascript: link nested in item) error = %v, want BlockUrlNotAllowedError", err)
	} else if listErr.Field != "content" {
		t.Errorf("PutBlocks(list, javascript: link nested in item) error field = %q, want %q", listErr.Field, "content")
	}

	// --- M14.3: a Google Drive share link normalizes to its direct-content form at write time,
	// with the original preserved in a new "originalUrl" field (D-ExternalMediaOnly, DS-OFM-17).
	driveBlocks, err := contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "image", Position: 20, Data: json.RawMessage(
			`{"url":"https://drive.google.com/file/d/1uEpd4IzXgwCzfxwivCTzbM5_ViMPLjZY/view?usp=sharing","alt":"A photo"}`,
		)},
	})
	if err != nil {
		t.Fatalf("PutBlocks(image, drive share link): %v", err)
	}
	var driveData map[string]any
	if err := json.Unmarshal(driveBlocks[0].Data, &driveData); err != nil {
		t.Fatalf("unmarshal drive block data: %v", err)
	}
	if want := "https://drive.google.com/uc?export=view&id=1uEpd4IzXgwCzfxwivCTzbM5_ViMPLjZY"; driveData["url"] != want {
		t.Errorf("PutBlocks(image, drive share link) url = %v, want %q", driveData["url"], want)
	}
	if want := "https://drive.google.com/file/d/1uEpd4IzXgwCzfxwivCTzbM5_ViMPLjZY/view?usp=sharing"; driveData["originalUrl"] != want {
		t.Errorf("PutBlocks(image, drive share link) originalUrl = %v, want %q", driveData["originalUrl"], want)
	}

	// A Dropbox share link normalizes to raw=1, original preserved.
	dropboxBlocks, err := contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "image", Position: 20, Data: json.RawMessage(
			`{"url":"https://www.dropbox.com/s/abcd1234/photo.jpg?dl=0","alt":"A photo"}`,
		)},
	})
	if err != nil {
		t.Fatalf("PutBlocks(image, dropbox share link): %v", err)
	}
	var dropboxData map[string]any
	if err := json.Unmarshal(dropboxBlocks[0].Data, &dropboxData); err != nil {
		t.Fatalf("unmarshal dropbox block data: %v", err)
	}
	if want := "https://www.dropbox.com/s/abcd1234/photo.jpg?raw=1"; dropboxData["url"] != want {
		t.Errorf("PutBlocks(image, dropbox share link) url = %v, want %q", dropboxData["url"], want)
	}
	if dropboxData["originalUrl"] != "https://www.dropbox.com/s/abcd1234/photo.jpg?dl=0" {
		t.Errorf("PutBlocks(image, dropbox share link) originalUrl = %v, want original share URL", dropboxData["originalUrl"])
	}

	// The long-form OneDrive URL normalizes by pure string substitution (redir -> download); a
	// short 1drv.ms link is left unchanged — resolving it would require following a redirect,
	// i.e. a server-side fetch of an admin-supplied URL, which this arc never does.
	onedriveBlocks, err := contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "image", Position: 20, Data: json.RawMessage(
			`{"url":"https://onedrive.live.com/redir?resid=ABC123&authkey=xyz","alt":"A photo"}`,
		)},
	})
	if err != nil {
		t.Fatalf("PutBlocks(image, onedrive long-form link): %v", err)
	}
	var onedriveData map[string]any
	if err := json.Unmarshal(onedriveBlocks[0].Data, &onedriveData); err != nil {
		t.Fatalf("unmarshal onedrive block data: %v", err)
	}
	if want := "https://onedrive.live.com/download?resid=ABC123&authkey=xyz"; onedriveData["url"] != want {
		t.Errorf("PutBlocks(image, onedrive long-form link) url = %v, want %q", onedriveData["url"], want)
	}

	shortOnedriveBlocks, err := contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "image", Position: 20, Data: json.RawMessage(
			`{"url":"https://1drv.ms/i/s!AbCdEfG","alt":"A photo"}`,
		)},
	})
	if err != nil {
		t.Fatalf("PutBlocks(image, onedrive short link): %v", err)
	}
	var shortOnedriveData map[string]any
	if err := json.Unmarshal(shortOnedriveBlocks[0].Data, &shortOnedriveData); err != nil {
		t.Fatalf("unmarshal short onedrive block data: %v", err)
	}
	if shortOnedriveData["url"] != "https://1drv.ms/i/s!AbCdEfG" {
		t.Errorf("PutBlocks(image, onedrive short link) url = %v, want unchanged", shortOnedriveData["url"])
	}
	if _, hasOriginal := shortOnedriveData["originalUrl"]; hasOriginal {
		t.Errorf("PutBlocks(image, onedrive short link) originalUrl = %v, want absent (not normalized)", shortOnedriveData["originalUrl"])
	}

	// A plain, already-direct URL from an unrecognized host passes through unchanged, with no
	// originalUrl set.
	plainBlocks, err := contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "image", Position: 20, Data: json.RawMessage(
			`{"url":"https://example.org/photo.jpg","alt":"A photo"}`,
		)},
	})
	if err != nil {
		t.Fatalf("PutBlocks(image, plain url): %v", err)
	}
	var plainData map[string]any
	if err := json.Unmarshal(plainBlocks[0].Data, &plainData); err != nil {
		t.Fatalf("unmarshal plain block data: %v", err)
	}
	if _, hasOriginal := plainData["originalUrl"]; hasOriginal {
		t.Errorf("PutBlocks(image, plain url) originalUrl = %v, want absent", plainData["originalUrl"])
	}

	// alt is now schema-required on image and each gallery image.
	_, err = contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "image", Position: 20, Data: json.RawMessage(`{"url":"https://example.org/photo.jpg"}`)},
	})
	var noAltErr *contentdomain.BlockDataInvalidError
	if !errors.As(err, &noAltErr) {
		t.Errorf("PutBlocks(image, no alt) error = %v, want BlockDataInvalidError", err)
	} else if noAltErr.Field != "alt" {
		t.Errorf("PutBlocks(image, no alt) Field = %q, want %q", noAltErr.Field, "alt")
	}

	_, err = contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "gallery", Position: 20, Data: json.RawMessage(
			`{"images":[{"url":"https://example.org/photo.jpg"}]}`,
		)},
	})
	var noGalleryAltErr *contentdomain.BlockDataInvalidError
	if !errors.As(err, &noGalleryAltErr) {
		t.Errorf("PutBlocks(gallery, no alt) error = %v, want BlockDataInvalidError", err)
	} else if noGalleryAltErr.Field != "images" {
		t.Errorf("PutBlocks(gallery, no alt) Field = %q, want %q", noGalleryAltErr.Field, "images")
	}

	// paragraphText extracts a paragraph block's plain text, tolerating key-order/whitespace
	// differences the richText validation pipeline introduces by re-marshaling (raw string equality
	// against block.Data is too fragile for these assertions).
	paragraphText := func(t *testing.T, data json.RawMessage) string {
		t.Helper()
		var v struct {
			Text []struct {
				Text string `json:"text"`
			} `json:"text"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			t.Fatalf("unmarshal paragraph data: %v", err)
		}
		if len(v.Text) != 1 {
			t.Fatalf("paragraph data = %s, want exactly one text run", data)
		}
		return v.Text[0].Text
	}

	// --- M14.6: forward revisions. doc is still DRAFT at this point (never transitioned above) —
	// GetPublicBlocks must 404, same "draft is never public" invariant as before this milestone.
	if _, err := contentSvc.GetPublicBlocks(context.Background(), doc.ID); !errors.Is(err, contentdomain.ErrDocumentNotFound) {
		t.Errorf("GetPublicBlocks(draft doc) error = %v, want ErrDocumentNotFound", err)
	}

	// Reset to a known single-block draft so the assertions below aren't tangled up in every block
	// type exercised above.
	if _, err := contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "paragraph", Position: 0, Data: json.RawMessage(`{"text":[{"type":"text","text":"Published text"}]}`)},
	}); err != nil {
		t.Fatalf("PutBlocks(reset before publish): %v", err)
	}

	if _, err := contentSvc.TransitionDocument(adminCtx, doc.ID, contentdomain.ActionPublish); err != nil {
		t.Fatalf("TransitionDocument(PUBLISH): %v", err)
	}
	publishedFirst, err := contentSvc.GetPublicBlocks(context.Background(), doc.ID)
	if err != nil {
		t.Fatalf("GetPublicBlocks (after first publish): %v", err)
	}
	if len(publishedFirst) != 1 || paragraphText(t, publishedFirst[0].Data) != "Published text" {
		t.Fatalf("GetPublicBlocks (after first publish) = %+v, want the just-published paragraph", publishedFirst)
	}

	// Editing the draft heavily after publish must never move what the public site serves — the
	// milestone's core acceptance criterion.
	if _, err := contentSvc.PutBlocks(adminCtx, doc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "paragraph", Position: 0, Data: json.RawMessage(`{"text":[{"type":"text","text":"Changed after publish, not yet republished"}]}`)},
	}); err != nil {
		t.Fatalf("PutBlocks(edit after publish): %v", err)
	}
	stillPublished, err := contentSvc.GetPublicBlocks(context.Background(), doc.ID)
	if err != nil {
		t.Fatalf("GetPublicBlocks (after draft edit, before republish): %v", err)
	}
	if got := paragraphText(t, stillPublished[0].Data); got != "Published text" {
		t.Fatalf("GetPublicBlocks (after draft edit, before republish) = %q, want unchanged published text", got)
	}
	draftAfterEdit, err := contentSvc.GetBlocks(adminCtx, doc.ID)
	if err != nil {
		t.Fatalf("GetBlocks (draft after edit): %v", err)
	}
	if got := paragraphText(t, draftAfterEdit[0].Data); got != "Changed after publish, not yet republished" {
		t.Fatalf("GetBlocks (draft after edit) = %q, want the new, unpublished text", got)
	}

	// --- ListRevisions/RestoreRevision are content.manage-gated like every other admin endpoint.
	if _, err := contentSvc.ListRevisions(otherCtx, doc.ID); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("ListRevisions by non-manager error = %v, want ErrForbidden", err)
	}
	revisions, err := contentSvc.ListRevisions(adminCtx, doc.ID)
	if err != nil {
		t.Fatalf("ListRevisions: %v", err)
	}
	if len(revisions) != 1 {
		t.Fatalf("ListRevisions after one publish = %d entries, want 1", len(revisions))
	}
	checkpointID := revisions[0].ID

	if _, err := contentSvc.RestoreRevision(otherCtx, doc.ID, checkpointID); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("RestoreRevision by non-manager error = %v, want ErrForbidden", err)
	}

	// A revision id that belongs to a different document is never trusted just because it's a
	// well-formed id — Content:RevisionNotFound, not silently accepted or leaked cross-document.
	otherDoc, err := contentSvc.CreateDocument(adminCtx, site.ID, contentdomain.CreateDocumentInput{
		Kind: contentdomain.KindPage, Locale: "en", Slug: "m14-6-other-doc",
	})
	if err != nil {
		t.Fatalf("CreateDocument (other doc for cross-document revision check): %v", err)
	}
	documentIDs = append(documentIDs, otherDoc.ID)
	if _, err := contentSvc.RestoreRevision(adminCtx, otherDoc.ID, checkpointID); !errors.Is(err, contentdomain.ErrRevisionNotFound) {
		t.Errorf("RestoreRevision(otherDoc, doc's own revision) error = %v, want ErrRevisionNotFound", err)
	}

	// Restore loads the checkpoint into the draft ONLY (owner decision, 2026-08-28) — it must not
	// touch what's published.
	restored, err := contentSvc.RestoreRevision(adminCtx, doc.ID, checkpointID)
	if err != nil {
		t.Fatalf("RestoreRevision: %v", err)
	}
	if got := paragraphText(t, restored[0].Data); got != "Published text" {
		t.Errorf("RestoreRevision result = %q, want the checkpoint's original text", got)
	}
	draftAfterRestore, err := contentSvc.GetBlocks(adminCtx, doc.ID)
	if err != nil {
		t.Fatalf("GetBlocks (draft after restore): %v", err)
	}
	if got := paragraphText(t, draftAfterRestore[0].Data); got != "Published text" {
		t.Errorf("GetBlocks (draft after restore) = %q, want the restored text", got)
	}
	stillPublishedAfterRestore, err := contentSvc.GetPublicBlocks(context.Background(), doc.ID)
	if err != nil {
		t.Fatalf("GetPublicBlocks (after restore, no republish): %v", err)
	}
	if got := paragraphText(t, stillPublishedAfterRestore[0].Data); got != "Published text" {
		t.Errorf("GetPublicBlocks (after restore, no republish) = %q, want unchanged (restore never auto-publishes)", got)
	}

	// --- Retention: the 50-most-recent cap prunes older checkpoints on every publish beyond it
	// (owner decision, 2026-08-28). One checkpoint already exists from the publish above; 55 more
	// republish cycles push the total to 56, so the cap must have trimmed it back to 50.
	for i := 0; i < 55; i++ {
		if _, err := contentSvc.TransitionDocument(adminCtx, doc.ID, contentdomain.ActionUnlist); err != nil {
			t.Fatalf("retention test: unlist iteration %d: %v", i, err)
		}
		if _, err := contentSvc.TransitionDocument(adminCtx, doc.ID, contentdomain.ActionPublish); err != nil {
			t.Fatalf("retention test: publish iteration %d: %v", i, err)
		}
	}
	prunedRevisions, err := contentSvc.ListRevisions(adminCtx, doc.ID)
	if err != nil {
		t.Fatalf("ListRevisions (after 56 total publishes): %v", err)
	}
	if len(prunedRevisions) != 50 {
		t.Errorf("ListRevisions (after 56 total publishes) = %d entries, want the 50-revision cap", len(prunedRevisions))
	}

	// --- M14.7: Preview. CreatePreviewLink is content.manage-gated like every other draft-adjacent
	// read on this service.
	if _, err := contentSvc.CreatePreviewLink(otherCtx, site.ID); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("CreatePreviewLink by non-manager error = %v, want ErrForbidden", err)
	}
	previewToken, err := contentSvc.CreatePreviewLink(adminCtx, site.ID)
	if err != nil {
		t.Fatalf("CreatePreviewLink: %v", err)
	}
	if previewToken == "" {
		t.Fatalf("CreatePreviewLink returned an empty token")
	}

	// A document that has never been published — GetPublicBlocks/ListPublicDocuments would never
	// show this to an anonymous caller (the whole "draft is never public" invariant); preview is the
	// one deliberate, token-gated exception (D-ContentRevisions: "a draft is content, not a special
	// code path").
	previewDoc, err := contentSvc.CreateDocument(adminCtx, site.ID, contentdomain.CreateDocumentInput{
		Kind: contentdomain.KindPage, Locale: "en", Slug: "m14-7-preview-only",
	})
	if err != nil {
		t.Fatalf("CreateDocument (preview-only doc): %v", err)
	}
	documentIDs = append(documentIDs, previewDoc.ID)
	if _, err := contentSvc.PutBlocks(adminCtx, previewDoc.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "paragraph", Position: 0, Data: json.RawMessage(`{"text":[{"type":"text","text":"Preview only, never published"}]}`)},
	}); err != nil {
		t.Fatalf("PutBlocks (preview-only doc): %v", err)
	}

	previewBlocks, err := contentSvc.GetPreviewBlocks(context.Background(), previewDoc.ID, previewToken)
	if err != nil {
		t.Fatalf("GetPreviewBlocks (valid token, never-published doc): %v", err)
	}
	if len(previewBlocks) != 1 || paragraphText(t, previewBlocks[0].Data) != "Preview only, never published" {
		t.Fatalf("GetPreviewBlocks = %+v, want the draft paragraph", previewBlocks)
	}
	// Confirm the ordinary public read still 404s the same never-published document — the preview
	// carve-out changes nothing about GetPublicBlocks itself.
	if _, err := contentSvc.GetPublicBlocks(context.Background(), previewDoc.ID); !errors.Is(err, contentdomain.ErrDocumentNotFound) {
		t.Errorf("GetPublicBlocks (preview-only doc, no token) error = %v, want ErrDocumentNotFound", err)
	}

	previewDocs, err := contentSvc.ListPreviewDocuments(context.Background(), site.ID, previewToken, nil, nil)
	if err != nil {
		t.Fatalf("ListPreviewDocuments: %v", err)
	}
	foundPreviewOnly := false
	for _, d := range previewDocs {
		if d.ID == previewDoc.ID {
			foundPreviewOnly = true
		}
	}
	if !foundPreviewOnly {
		t.Errorf("ListPreviewDocuments = %+v, want it to include the never-published doc %s", previewDocs, previewDoc.ID)
	}
	publicDocsOnly, err := contentSvc.ListPublicDocuments(context.Background(), site.ID, nil, nil)
	if err != nil {
		t.Fatalf("ListPublicDocuments: %v", err)
	}
	for _, d := range publicDocsOnly {
		if d.ID == previewDoc.ID {
			t.Errorf("ListPublicDocuments unexpectedly included the never-published doc %s", previewDoc.ID)
		}
	}

	// Missing/malformed/garbage tokens are rejected, never treated as "no token = fall back to
	// public" or as a lookup failure that leaks whether the document/site exists.
	if _, err := contentSvc.GetPreviewBlocks(context.Background(), previewDoc.ID, ""); !errors.Is(err, contentdomain.ErrPreviewTokenInvalid) {
		t.Errorf("GetPreviewBlocks (empty token) error = %v, want ErrPreviewTokenInvalid", err)
	}
	if _, err := contentSvc.GetPreviewBlocks(context.Background(), previewDoc.ID, "not-a-real-token"); !errors.Is(err, contentdomain.ErrPreviewTokenInvalid) {
		t.Errorf("GetPreviewBlocks (garbage token) error = %v, want ErrPreviewTokenInvalid", err)
	}
	if _, err := contentSvc.ListPreviewDocuments(context.Background(), site.ID, "not-a-real-token", nil, nil); !errors.Is(err, contentdomain.ErrPreviewTokenInvalid) {
		t.Errorf("ListPreviewDocuments (garbage token) error = %v, want ErrPreviewTokenInvalid", err)
	}

	// A token minted for a different site is rejected — the token's own scope is checked against the
	// document/site actually being read, never trusted just because it verifies.
	unitC, err := directorySvc.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "M14.7 Other Preview Congregation"}, seed.RootUnitID, directorydomain.CanonicalGraphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge (unitC): %v", err)
	}
	unitIDs = append(unitIDs, unitC.ID)
	var otherSiteAssignmentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope)
		VALUES ($1, $2, $3, 'unit') RETURNING id`,
		adminID, seed.CongregationAdminRoleID, unitC.ID,
	).Scan(&otherSiteAssignmentID); err != nil {
		t.Fatalf("grant congregation-admin on unitC: %v", err)
	}
	assignmentIDs = append(assignmentIDs, otherSiteAssignmentID)
	otherSite, err := contentSvc.CreateSite(adminCtx, contentdomain.CreateSiteInput{CongregationUnitRID: unitC.ID, Slug: "m14-7-other-site"})
	if err != nil {
		t.Fatalf("CreateSite (unitC): %v", err)
	}
	siteIDs = append(siteIDs, otherSite.ID)
	otherSiteToken, err := contentSvc.CreatePreviewLink(adminCtx, otherSite.ID)
	if err != nil {
		t.Fatalf("CreatePreviewLink (unitC): %v", err)
	}
	if _, err := contentSvc.GetPreviewBlocks(context.Background(), previewDoc.ID, otherSiteToken); !errors.Is(err, contentdomain.ErrPreviewTokenInvalid) {
		t.Errorf("GetPreviewBlocks (token scoped to a different site) error = %v, want ErrPreviewTokenInvalid", err)
	}
	if _, err := contentSvc.ListPreviewDocuments(context.Background(), site.ID, otherSiteToken, nil, nil); !errors.Is(err, contentdomain.ErrPreviewTokenInvalid) {
		t.Errorf("ListPreviewDocuments (token scoped to a different site) error = %v, want ErrPreviewTokenInvalid", err)
	}

	// ---- M14.10: nav items + page-route resolution ----

	// PutNavItems/ListNavItems are content.manage-gated like every other write/draft-adjacent read
	// on this service.
	if _, err := contentSvc.PutNavItems(otherCtx, site.ID, nil); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("PutNavItems by non-manager error = %v, want ErrForbidden", err)
	}
	if _, err := contentSvc.ListNavItems(otherCtx, site.ID); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("ListNavItems by non-manager error = %v, want ErrForbidden", err)
	}

	// A 3-level published page tree, so path resolution/breadcrumbs can be proved at every depth.
	topPage, err := contentSvc.CreateDocument(adminCtx, site.ID, contentdomain.CreateDocumentInput{
		Kind: contentdomain.KindPage, Locale: "en", Slug: "m14-10-top",
	})
	if err != nil {
		t.Fatalf("CreateDocument (top page): %v", err)
	}
	documentIDs = append(documentIDs, topPage.ID)
	if _, err := contentSvc.PutBlocks(adminCtx, topPage.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "paragraph", Position: 0, Data: json.RawMessage(`{"text":[{"type":"text","text":"Top"}]}`)},
	}); err != nil {
		t.Fatalf("PutBlocks (top page): %v", err)
	}
	if _, err := contentSvc.TransitionDocument(adminCtx, topPage.ID, contentdomain.ActionPublish); err != nil {
		t.Fatalf("TransitionDocument(PUBLISH, top page): %v", err)
	}

	childPage, err := contentSvc.CreateDocument(adminCtx, site.ID, contentdomain.CreateDocumentInput{
		Kind: contentdomain.KindPage, Locale: "en", Slug: "m14-10-child", ParentDocumentID: &topPage.ID,
	})
	if err != nil {
		t.Fatalf("CreateDocument (child page): %v", err)
	}
	documentIDs = append(documentIDs, childPage.ID)
	if _, err := contentSvc.PutBlocks(adminCtx, childPage.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "paragraph", Position: 0, Data: json.RawMessage(`{"text":[{"type":"text","text":"Child"}]}`)},
	}); err != nil {
		t.Fatalf("PutBlocks (child page): %v", err)
	}
	if _, err := contentSvc.TransitionDocument(adminCtx, childPage.ID, contentdomain.ActionPublish); err != nil {
		t.Fatalf("TransitionDocument(PUBLISH, child page): %v", err)
	}

	grandchildPage, err := contentSvc.CreateDocument(adminCtx, site.ID, contentdomain.CreateDocumentInput{
		Kind: contentdomain.KindPage, Locale: "en", Slug: "m14-10-grandchild", ParentDocumentID: &childPage.ID,
	})
	if err != nil {
		t.Fatalf("CreateDocument (grandchild page): %v", err)
	}
	documentIDs = append(documentIDs, grandchildPage.ID)
	if _, err := contentSvc.PutBlocks(adminCtx, grandchildPage.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "paragraph", Position: 0, Data: json.RawMessage(`{"text":[{"type":"text","text":"Grandchild"}]}`)},
	}); err != nil {
		t.Fatalf("PutBlocks (grandchild page): %v", err)
	}
	if _, err := contentSvc.TransitionDocument(adminCtx, grandchildPage.ID, contentdomain.ActionPublish); err != nil {
		t.Fatalf("TransitionDocument(PUBLISH, grandchild page): %v", err)
	}

	// A never-published (DRAFT) top-level page — for the DRAFT-leaf-404 and DRAFT-nav-target-
	// omitted cases below.
	draftPage, err := contentSvc.CreateDocument(adminCtx, site.ID, contentdomain.CreateDocumentInput{
		Kind: contentdomain.KindPage, Locale: "en", Slug: "m14-10-draft",
	})
	if err != nil {
		t.Fatalf("CreateDocument (draft page): %v", err)
	}
	documentIDs = append(documentIDs, draftPage.ID)

	// --- GetPublicDocumentByPath: happy path at depth 1/2/3, ancestors returned root-first.
	got1, ancestors1, _, err := contentSvc.GetPublicDocumentByPath(context.Background(), site.ID, "en", []string{"m14-10-top"})
	if err != nil {
		t.Fatalf("GetPublicDocumentByPath (depth 1): %v", err)
	}
	if got1.ID != topPage.ID || len(ancestors1) != 0 {
		t.Errorf("GetPublicDocumentByPath (depth 1) = doc %s, %d ancestors, want %s, 0 ancestors", got1.ID, len(ancestors1), topPage.ID)
	}

	got2, ancestors2, _, err := contentSvc.GetPublicDocumentByPath(context.Background(), site.ID, "en", []string{"m14-10-top", "m14-10-child"})
	if err != nil {
		t.Fatalf("GetPublicDocumentByPath (depth 2): %v", err)
	}
	if got2.ID != childPage.ID || len(ancestors2) != 1 || ancestors2[0].ID != topPage.ID {
		t.Errorf("GetPublicDocumentByPath (depth 2) = doc %s, ancestors %+v, want %s, [%s]", got2.ID, ancestors2, childPage.ID, topPage.ID)
	}

	got3, ancestors3, _, err := contentSvc.GetPublicDocumentByPath(context.Background(), site.ID, "en", []string{"m14-10-top", "m14-10-child", "m14-10-grandchild"})
	if err != nil {
		t.Fatalf("GetPublicDocumentByPath (depth 3): %v", err)
	}
	if got3.ID != grandchildPage.ID || len(ancestors3) != 2 || ancestors3[0].ID != topPage.ID || ancestors3[1].ID != childPage.ID {
		t.Errorf("GetPublicDocumentByPath (depth 3) = doc %s, ancestors %+v, want %s, [%s %s]", got3.ID, ancestors3, grandchildPage.ID, topPage.ID, childPage.ID)
	}

	// A DRAFT leaf 404s exactly like a missing document does.
	if _, _, _, err := contentSvc.GetPublicDocumentByPath(context.Background(), site.ID, "en", []string{"m14-10-draft"}); !errors.Is(err, contentdomain.ErrDocumentNotFound) {
		t.Errorf("GetPublicDocumentByPath (draft leaf) error = %v, want ErrDocumentNotFound", err)
	}

	// A wrong MIDDLE segment 404s — proves positional ancestor matching, not last-segment-only
	// resolution (a naive implementation would resolve this by grandchildPage's own slug alone,
	// since slugs are unique per site+kind+locale, not per parent).
	if _, _, _, err := contentSvc.GetPublicDocumentByPath(context.Background(), site.ID, "en", []string{"m14-10-top", "wrong-slug", "m14-10-grandchild"}); !errors.Is(err, contentdomain.ErrDocumentNotFound) {
		t.Errorf("GetPublicDocumentByPath (wrong middle segment) error = %v, want ErrDocumentNotFound", err)
	}

	// More than 3 segments 404s outright, before any lookup.
	if _, _, _, err := contentSvc.GetPublicDocumentByPath(context.Background(), site.ID, "en", []string{"a", "b", "c", "d"}); !errors.Is(err, contentdomain.ErrDocumentNotFound) {
		t.Errorf("GetPublicDocumentByPath (>3 segments) error = %v, want ErrDocumentNotFound", err)
	}

	// --- M14.14: locale switching (translation groups, closes DS-OFM-7).

	ukPage, err := contentSvc.CreateDocument(adminCtx, site.ID, contentdomain.CreateDocumentInput{
		Kind: contentdomain.KindPage, Locale: "uk", Slug: "m14-14-uk",
	})
	if err != nil {
		t.Fatalf("CreateDocument (m14.14 uk page, new group): %v", err)
	}
	documentIDs = append(documentIDs, ukPage.ID)
	if _, err := contentSvc.PutBlocks(adminCtx, ukPage.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "paragraph", Position: 0, Data: json.RawMessage(`{"text":[{"type":"text","text":"UK"}]}`)},
	}); err != nil {
		t.Fatalf("PutBlocks (m14.14 uk page): %v", err)
	}
	if _, err := contentSvc.TransitionDocument(adminCtx, ukPage.ID, contentdomain.ActionPublish); err != nil {
		t.Fatalf("TransitionDocument(PUBLISH, m14.14 uk page): %v", err)
	}

	// Joins ukPage's translation group, left as DRAFT — must not appear in translations until published.
	enPage, err := contentSvc.CreateDocument(adminCtx, site.ID, contentdomain.CreateDocumentInput{
		Kind: contentdomain.KindPage, Locale: "en", Slug: "m14-14-en", TranslationGroupID: &ukPage.TranslationGroupID,
	})
	if err != nil {
		t.Fatalf("CreateDocument (m14.14 en page, joins group): %v", err)
	}
	documentIDs = append(documentIDs, enPage.ID)

	_, _, translationsDraftSibling, err := contentSvc.GetPublicDocumentByPath(context.Background(), site.ID, "uk", []string{"m14-14-uk"})
	if err != nil {
		t.Fatalf("GetPublicDocumentByPath (m14.14, en sibling still draft): %v", err)
	}
	if len(translationsDraftSibling) != 1 || translationsDraftSibling[0].Locale != "uk" || translationsDraftSibling[0].Href != "/uk/m14-14-uk" {
		t.Errorf("GetPublicDocumentByPath translations (en sibling draft) = %+v, want exactly [{uk /uk/m14-14-uk}]", translationsDraftSibling)
	}

	if _, err := contentSvc.PutBlocks(adminCtx, enPage.ID, []contentdomain.BlockInput{
		{BlockTypeCode: "paragraph", Position: 0, Data: json.RawMessage(`{"text":[{"type":"text","text":"EN"}]}`)},
	}); err != nil {
		t.Fatalf("PutBlocks (m14.14 en page): %v", err)
	}
	if _, err := contentSvc.TransitionDocument(adminCtx, enPage.ID, contentdomain.ActionPublish); err != nil {
		t.Fatalf("TransitionDocument(PUBLISH, m14.14 en page): %v", err)
	}

	_, _, translationsBothPublished, err := contentSvc.GetPublicDocumentByPath(context.Background(), site.ID, "en", []string{"m14-14-en"})
	if err != nil {
		t.Fatalf("GetPublicDocumentByPath (m14.14, both published): %v", err)
	}
	gotLocales := map[string]string{}
	for _, tr := range translationsBothPublished {
		gotLocales[tr.Locale] = tr.Href
	}
	if len(gotLocales) != 2 || gotLocales["uk"] != "/uk/m14-14-uk" || gotLocales["en"] != "/en/m14-14-en" {
		t.Errorf("GetPublicDocumentByPath translations (both published) = %+v, want {uk:/uk/m14-14-uk en:/en/m14-14-en}", translationsBothPublished)
	}

	// CreateDocument rejects a locale already present in the target translation group — no DB
	// constraint backs this, so it's the app-level checkTranslationGroup guard being proved here.
	if _, err := contentSvc.CreateDocument(adminCtx, site.ID, contentdomain.CreateDocumentInput{
		Kind: contentdomain.KindPage, Locale: "uk", Slug: "m14-14-uk-dup", TranslationGroupID: &ukPage.TranslationGroupID,
	}); !errors.As(err, new(*contentdomain.TranslationLocaleTakenError)) {
		t.Errorf("CreateDocument (duplicate locale in group) error = %v, want *TranslationLocaleTakenError", err)
	}

	// CreateDocument rejects a translationGroupId belonging to a different site (otherSite/unitC,
	// created above) — translation_group_id has no FK to site, so this guard is app-level too.
	if _, err := contentSvc.CreateDocument(adminCtx, otherSite.ID, contentdomain.CreateDocumentInput{
		Kind: contentdomain.KindPage, Locale: "fr", Slug: "m14-14-cross-site", TranslationGroupID: &ukPage.TranslationGroupID,
	}); !errors.As(err, new(*contentdomain.TranslationGroupNotFoundError)) {
		t.Errorf("CreateDocument (cross-site translation group) error = %v, want *TranslationGroupNotFoundError", err)
	}

	// --- PutNavItems: full CRUD round trip, mixing an internal target with an external URL.
	navItems, err := contentSvc.PutNavItems(adminCtx, site.ID, []contentdomain.NavItemInput{
		{Label: "Top", TargetDocumentID: &topPage.ID, SortOrder: 0},
		{Label: "Our Friends", TargetURL: strPtr("https://example.org/friends"), SortOrder: 1},
		{Label: "Grandchild", TargetDocumentID: &grandchildPage.ID, SortOrder: 2},
	})
	if err != nil {
		t.Fatalf("PutNavItems: %v", err)
	}
	if len(navItems) != 3 {
		t.Fatalf("PutNavItems returned %d items, want 3", len(navItems))
	}
	listedNavItems, err := contentSvc.ListNavItems(adminCtx, site.ID)
	if err != nil {
		t.Fatalf("ListNavItems: %v", err)
	}
	if len(listedNavItems) != 3 {
		t.Errorf("ListNavItems = %d items, want 3", len(listedNavItems))
	}

	// A full replace really replaces — the next call's items are the only ones left afterward,
	// nothing from the previous call lingers.
	if _, err := contentSvc.PutNavItems(adminCtx, site.ID, []contentdomain.NavItemInput{
		{Label: "Top", TargetDocumentID: &topPage.ID, SortOrder: 0},
		{Label: "Draft (never public)", TargetDocumentID: &draftPage.ID, SortOrder: 1},
		{Label: "External", TargetURL: strPtr("https://example.org/external"), SortOrder: 2},
	}); err != nil {
		t.Fatalf("PutNavItems (replace): %v", err)
	}

	// --- ListPublicNavItems: the internal item resolves to a real hierarchical href; the item
	// targeting the still-DRAFT page is silently omitted (never a broken link); the external item
	// passes through with External=true and its raw target URL as Href.
	publicNavItems, err := contentSvc.ListPublicNavItems(context.Background(), site.ID)
	if err != nil {
		t.Fatalf("ListPublicNavItems: %v", err)
	}
	if len(publicNavItems) != 2 {
		t.Fatalf("ListPublicNavItems = %+v, want 2 items (draft target omitted)", publicNavItems)
	}
	var sawTop, sawExternal bool
	for _, item := range publicNavItems {
		switch item.Label {
		case "Top":
			sawTop = true
			if item.External {
				t.Errorf("ListPublicNavItems Top item External = true, want false")
			}
			if want := "/en/m14-10-top"; item.Href != want {
				t.Errorf("ListPublicNavItems Top item Href = %q, want %q", item.Href, want)
			}
		case "External":
			sawExternal = true
			if !item.External {
				t.Errorf("ListPublicNavItems External item External = false, want true")
			}
			if item.Href != "https://example.org/external" {
				t.Errorf("ListPublicNavItems External item Href = %q, want the raw target URL", item.Href)
			}
		case "Draft (never public)":
			t.Errorf("ListPublicNavItems unexpectedly included the draft-target item")
		}
	}
	if !sawTop || !sawExternal {
		t.Errorf("ListPublicNavItems = %+v, want both Top and External items present", publicNavItems)
	}

	// A nested target resolves to its full hierarchical href, not just its own slug.
	if _, err := contentSvc.PutNavItems(adminCtx, site.ID, []contentdomain.NavItemInput{
		{Label: "Grandchild", TargetDocumentID: &grandchildPage.ID, SortOrder: 0},
	}); err != nil {
		t.Fatalf("PutNavItems (nested target): %v", err)
	}
	publicNavItemsNested, err := contentSvc.ListPublicNavItems(context.Background(), site.ID)
	if err != nil {
		t.Fatalf("ListPublicNavItems (nested target): %v", err)
	}
	if len(publicNavItemsNested) != 1 {
		t.Fatalf("ListPublicNavItems (nested target) = %+v, want 1 item", publicNavItemsNested)
	}
	if want := "/en/m14-10-top/m14-10-child/m14-10-grandchild"; publicNavItemsNested[0].Href != want {
		t.Errorf("ListPublicNavItems (nested target) Href = %q, want %q", publicNavItemsNested[0].Href, want)
	}

	// --- Validation: duplicate sortOrder, ambiguous target (both/neither set), cross-site target,
	// non-PAGE-kind target — each rejected up front with its own typed error.
	if _, err := contentSvc.PutNavItems(adminCtx, site.ID, []contentdomain.NavItemInput{
		{Label: "A", TargetURL: strPtr("https://example.org/a"), SortOrder: 0},
		{Label: "B", TargetURL: strPtr("https://example.org/b"), SortOrder: 0},
	}); !errors.As(err, new(*contentdomain.DuplicateNavItemSortOrderError)) {
		t.Errorf("PutNavItems (duplicate sortOrder) error = %v, want *DuplicateNavItemSortOrderError", err)
	}

	if _, err := contentSvc.PutNavItems(adminCtx, site.ID, []contentdomain.NavItemInput{
		{Label: "Neither", SortOrder: 0},
	}); !errors.As(err, new(*contentdomain.NavTargetAmbiguousError)) {
		t.Errorf("PutNavItems (neither target set) error = %v, want *NavTargetAmbiguousError", err)
	}
	if _, err := contentSvc.PutNavItems(adminCtx, site.ID, []contentdomain.NavItemInput{
		{Label: "Both", TargetDocumentID: &topPage.ID, TargetURL: strPtr("https://example.org"), SortOrder: 0},
	}); !errors.As(err, new(*contentdomain.NavTargetAmbiguousError)) {
		t.Errorf("PutNavItems (both targets set) error = %v, want *NavTargetAmbiguousError", err)
	}

	// A targetDocumentId belonging to a DIFFERENT site (otherSite/unitC, created above in the
	// M14.7 preview section) is rejected — content.manage on THIS site doesn't imply trust of a
	// document id just because it resolves to a real PAGE somewhere else.
	crossSiteDoc, err := contentSvc.CreateDocument(adminCtx, otherSite.ID, contentdomain.CreateDocumentInput{
		Kind: contentdomain.KindPage, Locale: "en", Slug: "m14-10-cross-site-target",
	})
	if err != nil {
		t.Fatalf("CreateDocument (otherSite, cross-site nav target check): %v", err)
	}
	documentIDs = append(documentIDs, crossSiteDoc.ID)
	if _, err := contentSvc.PutNavItems(adminCtx, site.ID, []contentdomain.NavItemInput{
		{Label: "Cross-site", TargetDocumentID: &crossSiteDoc.ID, SortOrder: 0},
	}); !errors.As(err, new(*contentdomain.NavTargetInvalidError)) {
		t.Errorf("PutNavItems (cross-site target) error = %v, want *NavTargetInvalidError", err)
	}

	// A targetDocumentId pointing at a POST (not a PAGE) is rejected the same way.
	postDoc, err := contentSvc.CreateDocument(adminCtx, site.ID, contentdomain.CreateDocumentInput{
		Kind: contentdomain.KindPost, Locale: "en", Slug: "m14-10-a-post",
	})
	if err != nil {
		t.Fatalf("CreateDocument (POST, for non-PAGE-kind nav target check): %v", err)
	}
	documentIDs = append(documentIDs, postDoc.ID)
	var navInvalidErr *contentdomain.NavTargetInvalidError
	if _, err := contentSvc.PutNavItems(adminCtx, site.ID, []contentdomain.NavItemInput{
		{Label: "A Post", TargetDocumentID: &postDoc.ID, SortOrder: 0},
	}); !errors.As(err, &navInvalidErr) {
		t.Errorf("PutNavItems (POST-kind target) error = %v, want *NavTargetInvalidError", err)
	} else if navInvalidErr.TargetDocumentID != postDoc.ID {
		t.Errorf("PutNavItems (POST-kind target) TargetDocumentID = %q, want %q", navInvalidErr.TargetDocumentID, postDoc.ID)
	}

	// ---- M14.13: block-type/pattern catalog admin (content.catalog.manage, platform-moderator) ----

	catalogBlockTypeSchema := []byte(`{"type":"object","required":["value"],"additionalProperties":false,"properties":{"value":{"type":"string"}}}`)

	// --- Every catalog write is denied for a plain congregation-admin (adminCtx already holds
	// content.manage on unit, which must not imply content.catalog.manage — a different, platform-wide
	// authority).
	if _, err := contentSvc.CreateBlockType(adminCtx, contentdomain.CreateBlockTypeInput{
		Code: "m1413_test_block", Name: "M14.13 Test Block", JSONSchema: catalogBlockTypeSchema, UISchema: []byte(`{}`), SortOrder: 200,
	}); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("CreateBlockType by non-moderator error = %v, want ErrForbidden", err)
	}
	if _, err := contentSvc.CreatePattern(adminCtx, contentdomain.CreatePatternInput{
		Name: "M14.13 Test Pattern", Blocks: []contentdomain.BlockInput{}, SortOrder: 200,
	}); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("CreatePattern by non-moderator error = %v, want ErrForbidden", err)
	}
	if _, err := contentSvc.ListAllBlockTypesForCatalog(adminCtx); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("ListAllBlockTypesForCatalog by non-moderator error = %v, want ErrForbidden", err)
	}

	// --- Grant a real platform-moderator standing on the shared root unit (mirrors
	// internal/moderation/moderation_integration_test.go's own grant-then-call pattern exactly).
	moderatorID := insertPerson("M14.13 Content Test Moderator")
	var modAssignmentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.authz_role_assignments (subject_person_id, role_id, target_unit_id, scope)
		VALUES ($1, $2, $3, 'unit') RETURNING id`,
		moderatorID, seed.PlatformModeratorRoleID, seed.RootUnitID,
	).Scan(&modAssignmentID); err != nil {
		t.Fatalf("grant platform-moderator: %v", err)
	}
	assignmentIDs = append(assignmentIDs, modAssignmentID)
	modCtx := authz.NewContext(ctx, authz.Subject{PersonID: moderatorID})

	// --- CreateBlockType succeeds for a real moderator; the new type immediately appears in the
	// public ListBlockTypes (active by default) and the admin-side ListAllBlockTypesForCatalog.
	newBlockType, err := contentSvc.CreateBlockType(modCtx, contentdomain.CreateBlockTypeInput{
		Code: "m1413_test_block", Name: "M14.13 Test Block", JSONSchema: catalogBlockTypeSchema, UISchema: []byte(`{}`), SortOrder: 200,
	})
	if err != nil {
		t.Fatalf("CreateBlockType by moderator: %v", err)
	}
	blockTypeIDs = append(blockTypeIDs, newBlockType.ID)
	if newBlockType.Status != contentdomain.BlockTypeActive {
		t.Errorf("CreateBlockType result Status = %q, want ACTIVE", newBlockType.Status)
	}
	publicTypes, err := contentSvc.ListBlockTypes(context.Background())
	if err != nil {
		t.Fatalf("ListBlockTypes (public) after CreateBlockType: %v", err)
	}
	foundPublic := false
	for _, bt := range publicTypes {
		if bt.Code == "m1413_test_block" {
			foundPublic = true
		}
	}
	if !foundPublic {
		t.Errorf("ListBlockTypes (public) = %+v, want to contain m1413_test_block", publicTypes)
	}

	// --- A duplicate code is rejected with a typed BlockTypeCodeTakenError, not a generic error.
	var codeTakenErr *contentdomain.BlockTypeCodeTakenError
	if _, err := contentSvc.CreateBlockType(modCtx, contentdomain.CreateBlockTypeInput{
		Code: "m1413_test_block", Name: "Duplicate", JSONSchema: catalogBlockTypeSchema, UISchema: []byte(`{}`), SortOrder: 201,
	}); !errors.As(err, &codeTakenErr) {
		t.Errorf("CreateBlockType (duplicate code) error = %v, want *BlockTypeCodeTakenError", err)
	} else if codeTakenErr.Code != "m1413_test_block" {
		t.Errorf("CreateBlockType (duplicate code) Code = %q, want m1413_test_block", codeTakenErr.Code)
	}

	// --- A json_schema that fails to compile is rejected up front, before ever reaching the store.
	if _, err := contentSvc.CreateBlockType(modCtx, contentdomain.CreateBlockTypeInput{
		Code: "m1413_test_broken", Name: "Broken", JSONSchema: []byte(`{"type":"not-a-real-type"}`), UISchema: []byte(`{}`), SortOrder: 202,
	}); err == nil {
		t.Errorf("CreateBlockType (uncompilable schema) error = nil, want a compile error")
	}

	// --- UpdateBlockType retires the type by status alone — json_schema/ui_schema are structurally
	// unsettable (domain.UpdateBlockTypeInput has no such field), the owner's "locked after creation"
	// decision.
	retiredStatus := contentdomain.BlockTypeRetired
	retired, err := contentSvc.UpdateBlockType(modCtx, newBlockType.ID, contentdomain.UpdateBlockTypeInput{Status: &retiredStatus})
	if err != nil {
		t.Fatalf("UpdateBlockType (retire) by moderator: %v", err)
	}
	if retired.Status != contentdomain.BlockTypeRetired || !bytes.Equal(retired.JSONSchema, newBlockType.JSONSchema) {
		t.Errorf("UpdateBlockType (retire) result = %+v, want Status=RETIRED, unchanged JSONSchema", retired)
	}
	// A retired type is excluded from the public active-only list but still visible to the moderator
	// catalog read.
	publicTypesAfterRetire, err := contentSvc.ListBlockTypes(context.Background())
	if err != nil {
		t.Fatalf("ListBlockTypes (public) after retire: %v", err)
	}
	for _, bt := range publicTypesAfterRetire {
		if bt.Code == "m1413_test_block" {
			t.Errorf("ListBlockTypes (public) after retire still contains m1413_test_block")
		}
	}
	catalogTypes, err := contentSvc.ListAllBlockTypesForCatalog(modCtx)
	if err != nil {
		t.Fatalf("ListAllBlockTypesForCatalog by moderator: %v", err)
	}
	foundRetired := false
	for _, bt := range catalogTypes {
		if bt.Code == "m1413_test_block" && bt.Status == contentdomain.BlockTypeRetired {
			foundRetired = true
		}
	}
	if !foundRetired {
		t.Errorf("ListAllBlockTypesForCatalog = %+v, want a RETIRED m1413_test_block", catalogTypes)
	}

	// --- UpdateBlockType against a nonexistent id is ErrBlockTypeNotFound.
	if _, err := contentSvc.UpdateBlockType(modCtx, "00000000-0000-0000-0000-000000000000", contentdomain.UpdateBlockTypeInput{Status: &retiredStatus}); !errors.Is(err, contentdomain.ErrBlockTypeNotFound) {
		t.Errorf("UpdateBlockType (nonexistent id) error = %v, want ErrBlockTypeNotFound", err)
	}

	// --- Patterns: create/update/delete round trip, moderator-gated the same way; ListPatterns is
	// the public read the admin editor's insert-a-pattern UI calls (no auth, mirroring ListBlockTypes).
	patternBlocks := []contentdomain.BlockInput{
		{BlockTypeCode: "heading", Position: 0, Data: json.RawMessage(`{"level":2,"text":[{"type":"text","text":"M14.13 Test Pattern"}]}`)},
	}
	newPattern, err := contentSvc.CreatePattern(modCtx, contentdomain.CreatePatternInput{
		Name: "M14.13 Test Pattern", Description: "for integration test", Blocks: patternBlocks, SortOrder: 999,
	})
	if err != nil {
		t.Fatalf("CreatePattern by moderator: %v", err)
	}
	patterns, err := contentSvc.ListPatterns(context.Background())
	if err != nil {
		t.Fatalf("ListPatterns (public): %v", err)
	}
	foundPattern := false
	for _, p := range patterns {
		if p.ID == newPattern.ID {
			foundPattern = true
		}
	}
	if !foundPattern {
		t.Errorf("ListPatterns (public) = %+v, want to contain %s", patterns, newPattern.ID)
	}

	updatedName := "M14.13 Test Pattern (updated)"
	updatedPattern, err := contentSvc.UpdatePattern(modCtx, newPattern.ID, contentdomain.UpdatePatternInput{Name: &updatedName})
	if err != nil {
		t.Fatalf("UpdatePattern by moderator: %v", err)
	}
	if updatedPattern.Name != updatedName || len(updatedPattern.Blocks) != len(patternBlocks) || updatedPattern.Blocks[0].BlockTypeCode != patternBlocks[0].BlockTypeCode {
		t.Errorf("UpdatePattern result = %+v, want Name=%q, unchanged Blocks", updatedPattern, updatedName)
	}

	var patternNotFoundErr *contentdomain.PatternNotFoundError
	if _, err := contentSvc.UpdatePattern(modCtx, "00000000-0000-0000-0000-000000000000", contentdomain.UpdatePatternInput{Name: &updatedName}); !errors.As(err, &patternNotFoundErr) {
		t.Errorf("UpdatePattern (nonexistent id) error = %v, want *PatternNotFoundError", err)
	}

	// --- DeletePattern is moderator-gated too, and not-found on an already-deleted/unknown id.
	if err := contentSvc.DeletePattern(adminCtx, newPattern.ID); !errors.Is(err, contentdomain.ErrForbidden) {
		t.Errorf("DeletePattern by non-moderator error = %v, want ErrForbidden", err)
	}
	if err := contentSvc.DeletePattern(modCtx, newPattern.ID); err != nil {
		t.Fatalf("DeletePattern by moderator: %v", err)
	}
	if err := contentSvc.DeletePattern(modCtx, newPattern.ID); !errors.As(err, &patternNotFoundErr) {
		t.Errorf("DeletePattern (already deleted) error = %v, want *PatternNotFoundError", err)
	}
	patternsAfterDelete, err := contentSvc.ListPatterns(context.Background())
	if err != nil {
		t.Fatalf("ListPatterns (public) after delete: %v", err)
	}
	for _, p := range patternsAfterDelete {
		if p.ID == newPattern.ID {
			t.Errorf("ListPatterns (public) after delete still contains %s", newPattern.ID)
		}
	}
}

func strPtr(s string) *string { return &s }
