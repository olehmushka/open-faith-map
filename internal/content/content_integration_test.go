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
	closurePort := directoryadapters.NewRepository(pool)
	pdp := authzdomain.NewPDP(closurePort)
	authzStore := authzadapters.NewRepository(pool)
	authzSvc := authz.NewService(pdp, authzStore, pool)
	contentStore := contentadapters.NewRepository(pool)
	contentSvc := application.NewService(contentStore, authzSvc, "m14-7-test-preview-hmac-key")

	var personIDs, unitIDs, siteIDs, assignmentIDs, documentIDs []string
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range documentIDs {
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
}
