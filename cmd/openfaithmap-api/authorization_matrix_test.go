// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M10.9's table-driven authorization matrix: {anonymous, congregation-admin@own,
// congregation-admin@other, registration-operator, platform-moderator, instance-admin} against a
// representative, real, HTTP-level sample of every guarded endpoint across all 7 Conjure contracts
// — not the full ~70-endpoint x 6-subject cartesian product (many writes need complex pre-existing
// fixtures this test doesn't build), but every one of the four gate shapes
// (anonymous / authenticated-only / target-scoped religionorg.manage / root-scoped unit.lifecycle /
// instance-admin plane) proven for real, per contract, with both a real denial and — wherever a
// fixture is cheap to build — a real success.
//
// Requires a live openfaithmap-api reachable over HTTP, its own Postgres, and DEV_ISSUER_HMAC_KEY
// set on the SERVER to the same value passed here (docker-compose.override.yml, never committed —
// D-DirectTokenVerification's amendment ships the committed config with this unset). openfaithmap-api
// publishes no host port for its app listener (D-HeadlessTopology), so this must run from inside a
// container on the compose network, not the host:
//
//	docker run --rm --network open-faith-map_default -v "$PWD":/src -w /src golang:1.26-bookworm \
//	  sh -c 'DATABASE_URL="postgres://postgres:postgres@postgres:5432/postgres?sslmode=disable" \
//	  OPENFAITHMAP_API_BASE_URL="https://open-faith-map-openfaithmap-api-1:3000" \
//	  DEV_ISSUER_HMAC_KEY="<matches the override>" \
//	  go test ./cmd/openfaithmap-api/... -run TestAuthorizationMatrix -v'
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	authzadapters "github.com/olehmushka/open-faith-map/internal/authz/adapters"
	directoryapplication "github.com/olehmushka/open-faith-map/internal/directory/application"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/devtoken"
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
)

// subjects, seeded fresh per test run and torn down after.
type matrixSubjects struct {
	anonymous      string // always "" — no Authorization header
	congAdminOwn   string
	congAdminOther string
	operator       string
	moderator      string
	instanceAdmin  string

	unitA string // congAdminOwn's own unit
	unitB string // congAdminOther's own unit — "other" relative to unitA

	instanceAdminPersonID string // M11.1's getAccountStatus needs a real personId path segment
}

func TestAuthorizationMatrix(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	apiBase := os.Getenv("OPENFAITHMAP_API_BASE_URL")
	hmacKey := os.Getenv("DEV_ISSUER_HMAC_KEY")
	if dsn == "" || apiBase == "" || hmacKey == "" {
		t.Skip("set DATABASE_URL, OPENFAITHMAP_API_BASE_URL, and DEV_ISSUER_HMAC_KEY (matching the target server's own) to run the authorization matrix against a live stack")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	subj := seedSubjects(t, ctx, pool, hmacKey)
	client := insecureClient()

	// ---- category: genuinely anonymous (8 endpoints) — reachable with no token at all ----
	t.Run("anonymous_endpoints", func(t *testing.T) {
		cases := []struct {
			name, method, path string
			body               any
		}{
			{"content_getPublicSite", http.MethodGet, "/content/v1/public/units/" + subj.unitA + "/site", nil},
			{"content_listPublicDocuments", http.MethodGet, "/content/v1/public/sites/nonexistent-site/documents", nil},
			{"content_getPublicBlocks", http.MethodGet, "/content/v1/public/documents/nonexistent-doc/blocks", nil},
			{"content_listBlockTypes", http.MethodGet, "/content/v1/public/block-types", nil},
			{"discovery_search", http.MethodGet, "/discovery/v1/search", nil},
			{"moderation_fileReport", http.MethodPost, "/moderation/v1/reports", map[string]any{
				"targetKind": "CONGREGATION", "targetRef": subj.unitA, "reasonCode": "SPAM",
			}},
			{"moderation_checkExclusion", http.MethodPost, "/moderation/v1/exclusion-check", map[string]any{
				"taxonId": mustTaxonID(t, ctx, pool),
			}},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				status, _ := doReq(t, client, apiBase, c.method, c.path, "", c.body)
				if status == http.StatusUnauthorized {
					t.Errorf("%s: got 401 with no token — should be reachable anonymously", c.name)
				}
			})
		}
	})

	// ---- category: authenticated-only, no extra PDP check beyond the header ----
	t.Run("authenticated_only", func(t *testing.T) {
		cases := []struct {
			name, method, path string
		}{
			{"core_whoami", http.MethodGet, "/core/v1/whoami"},
			{"core_listUnits", http.MethodGet, "/core/v1/units?query=&limit=5"},
			{"core_listTaxa", http.MethodGet, "/core/v1/taxa?limit=5"},
			{"core_listCountries", http.MethodGet, "/core/v1/countries"},
			{"core_listOrgKinds", http.MethodGet, "/core/v1/org-kinds"},
			{"registration_listRequests", http.MethodGet, "/registration/v1/requests"},
			{"congregationimport_listCandidates", http.MethodGet, "/congregation-import/v1/candidates"},
			{"congregationimport_listRuns", http.MethodGet, "/congregation-import/v1/runs"},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				anonStatus, _ := doReq(t, client, apiBase, c.method, c.path, "", nil)
				if anonStatus != http.StatusUnauthorized {
					t.Errorf("%s: anonymous got %d, want 401", c.name, anonStatus)
				}
				authStatus, body := doReq(t, client, apiBase, c.method, c.path, subj.congAdminOwn, nil)
				if authStatus == http.StatusUnauthorized || authStatus == http.StatusForbidden {
					t.Errorf("%s: authenticated (non-privileged) subject got %d, want a real response — body: %s", c.name, authStatus, body)
				}
			})
		}
	})

	// ---- category: religionorg.manage, target-scoped ----
	t.Run("target_scoped_religionorg_manage", func(t *testing.T) {
		t.Run("core_createChildOrg", func(t *testing.T) {
			body := map[string]any{"parentUnitId": subj.unitA, "name": "Matrix Test Child", "orgKindId": nil}
			anonStatus, _ := doReq(t, client, apiBase, http.MethodPost, "/core/v1/units/children", "", body)
			assertStatus(t, "anonymous", anonStatus, http.StatusUnauthorized)

			otherStatus, otherBody := doReq(t, client, apiBase, http.MethodPost, "/core/v1/units/children", subj.congAdminOther, body)
			assertStatus(t, "congAdminOther (wrong unit)", otherStatus, http.StatusForbidden)
			assertErrorName(t, otherBody, "Core:Forbidden")

			ownStatus, _ := doReq(t, client, apiBase, http.MethodPost, "/core/v1/units/children", subj.congAdminOwn, body)
			if ownStatus != http.StatusOK && ownStatus != http.StatusCreated {
				t.Errorf("congAdminOwn (own unit): status = %d, want 200/201", ownStatus)
			}
		})

		t.Run("vouching_createVouch", func(t *testing.T) {
			// guarantorCongregationUnitId is the unit the CALLER proves standing on — deliberately
			// independent of congregationUnitId, the claim being vouched for (vouching.conjure.yml's
			// own header comment). congAdminOwn holds their grant on unitA only, so a request
			// claiming unitB as the guarantor unit must be denied even though they're a real
			// congregation-admin somewhere else entirely; the same request with unitA as the
			// guarantor unit (their own) must succeed.
			deniedBody := map[string]any{
				"claimantPersonId": subj.instanceAdmin, "congregationUnitId": subj.unitB,
				"guarantorCongregationUnitId": subj.unitB,
			}
			anonStatus, _ := doReq(t, client, apiBase, http.MethodPost, "/vouching/v1/vouches", "", deniedBody)
			assertStatus(t, "anonymous", anonStatus, http.StatusUnauthorized)

			deniedStatus, _ := doReq(t, client, apiBase, http.MethodPost, "/vouching/v1/vouches", subj.congAdminOwn, deniedBody)
			assertStatus(t, "congAdminOwn (no standing on unitB, the guarantor unit)", deniedStatus, http.StatusForbidden)

			allowedBody := map[string]any{
				"claimantPersonId": subj.instanceAdmin, "congregationUnitId": subj.unitB,
				"guarantorCongregationUnitId": subj.unitA,
			}
			allowedStatus, allowedRespBody := doReq(t, client, apiBase, http.MethodPost, "/vouching/v1/vouches", subj.congAdminOwn, allowedBody)
			if allowedStatus != http.StatusOK && allowedStatus != http.StatusCreated {
				t.Errorf("congAdminOwn (own standing on unitA, the guarantor unit): status = %d, body: %s", allowedStatus, allowedRespBody)
			}
		})

		t.Run("registration_approveRequest_denied", func(t *testing.T) {
			reqID := submitThrowawayRequest(t, client, apiBase, subj)
			status, body := doReq(t, client, apiBase, http.MethodPost, "/registration/v1/requests/"+reqID+"/approve", subj.congAdminOwn, map[string]any{})
			assertStatus(t, "congAdminOwn (not an operator)", status, http.StatusForbidden)
			assertErrorName(t, body, "Registration:Forbidden")
		})

		t.Run("congregationimport_runJurisdictionSync", func(t *testing.T) {
			anonStatus, _ := doReq(t, client, apiBase, http.MethodPost, "/congregation-import/v1/jurisdiction-sync/runs", "", map[string]any{"sourceCode": "wikidata-catholic"})
			assertStatus(t, "anonymous", anonStatus, http.StatusUnauthorized)

			status, body := doReq(t, client, apiBase, http.MethodPost, "/congregation-import/v1/jurisdiction-sync/runs", subj.congAdminOwn, map[string]any{"sourceCode": "wikidata-catholic"})
			assertStatus(t, "congAdminOwn (not an operator)", status, http.StatusForbidden)
			assertErrorName(t, body, "CongregationImport:Forbidden")
		})

		t.Run("congregationimport_listTaxonAliases", func(t *testing.T) {
			anonStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/congregation-import/v1/taxon-aliases", "", nil)
			assertStatus(t, "anonymous", anonStatus, http.StatusUnauthorized)

			deniedStatus, deniedBody := doReq(t, client, apiBase, http.MethodGet, "/congregation-import/v1/taxon-aliases", subj.congAdminOwn, nil)
			assertStatus(t, "congAdminOwn (not an operator)", deniedStatus, http.StatusForbidden)
			assertErrorName(t, deniedBody, "CongregationImport:Forbidden")

			allowedStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/congregation-import/v1/taxon-aliases", subj.operator, nil)
			assertStatus(t, "operator", allowedStatus, http.StatusOK)
		})
	})

	// ---- category: unit.lifecycle on root (platform-moderator) ----
	t.Run("root_scoped_unit_lifecycle", func(t *testing.T) {
		t.Run("moderation_listReports", func(t *testing.T) {
			anonStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/moderation/v1/reports", "", nil)
			assertStatus(t, "anonymous", anonStatus, http.StatusUnauthorized)

			deniedStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/moderation/v1/reports", subj.congAdminOwn, nil)
			assertStatus(t, "congAdminOwn (not a moderator)", deniedStatus, http.StatusForbidden)

			allowedStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/moderation/v1/reports", subj.moderator, nil)
			assertStatus(t, "platform-moderator", allowedStatus, http.StatusOK)
		})

		t.Run("vouching_listVouches", func(t *testing.T) {
			deniedStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/vouching/v1/vouches", subj.operator, nil)
			assertStatus(t, "operator (not a moderator)", deniedStatus, http.StatusForbidden)

			allowedStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/vouching/v1/vouches", subj.moderator, nil)
			assertStatus(t, "platform-moderator", allowedStatus, http.StatusOK)
		})
	})

	// ---- category: instance-admin plane (CoreSuperAdminService — all 11 endpoints share one
	// route-group gate, so this is the highest-value single check in the whole matrix: every
	// subject except instance-admin must be refused by the SAME middleware, not per-handler logic) ----
	t.Run("instance_admin_plane", func(t *testing.T) {
		endpoints := []struct{ name, path string }{
			{"searchPersons", "/core/v1/super-admin/persons"},
			{"listRoles", "/core/v1/super-admin/roles"},
			{"listInstanceAdmins", "/core/v1/super-admin/instance-admins"},
			// M11.1 — keeps the representative sample current with the new endpoints on this service.
			{"getAccountStatus", "/core/v1/super-admin/persons/" + subj.instanceAdminPersonID + "/account-status"},
		}
		for _, ep := range endpoints {
			t.Run(ep.name, func(t *testing.T) {
				anonStatus, _ := doReq(t, client, apiBase, http.MethodGet, ep.path, "", nil)
				assertStatus(t, "anonymous", anonStatus, http.StatusUnauthorized)

				for _, s := range []struct {
					name  string
					token string
				}{
					{"congAdminOwn", subj.congAdminOwn}, {"congAdminOther", subj.congAdminOther},
					{"operator", subj.operator}, {"moderator", subj.moderator},
				} {
					status, body := doReq(t, client, apiBase, http.MethodGet, ep.path, s.token, nil)
					assertStatus(t, s.name, status, http.StatusForbidden)
					assertErrorName(t, body, "Authz:InstanceAdminRequired")
				}

				adminStatus, _ := doReq(t, client, apiBase, http.MethodGet, ep.path, subj.instanceAdmin, nil)
				assertStatus(t, "instanceAdmin", adminStatus, http.StatusOK)
			})
		}
	})
}

func assertStatus(t *testing.T, subject string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s: status = %d, want %d", subject, got, want)
	}
}

func assertErrorName(t *testing.T, body []byte, want string) {
	t.Helper()
	var parsed struct {
		ErrorName string `json:"errorName"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Errorf("parse error body: %v (body: %s)", err, body)
		return
	}
	if parsed.ErrorName != want {
		t.Errorf("errorName = %q, want %q (body: %s)", parsed.ErrorName, want, body)
	}
}

func insecureClient() *http.Client {
	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, // dev-only self-signed cert, D-HeadlessTopology
	}
}

func doReq(t *testing.T, client *http.Client, base, method, path, token string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, base+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return resp.StatusCode, respBody
}

// mustTaxonID reads one real, seeded taxon id for request bodies that need one.
func mustTaxonID(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(ctx, `SELECT id FROM openfaithmap.religion_taxa WHERE code = 'christianity'`).Scan(&id); err != nil {
		t.Fatalf("look up christianity taxon id: %v", err)
	}
	return id
}

// submitThrowawayRequest submits a real PENDING registration request as congAdminOwn (submitRequest
// has no gate — any authenticated person may submit), for tests that need one to exist. Not
// cleaned up individually — swept by seedSubjects' own teardown via the submitter's person id.
func submitThrowawayRequest(t *testing.T, client *http.Client, apiBase string, subj matrixSubjects) string {
	t.Helper()
	status, body := doReq(t, client, apiBase, http.MethodPost, "/registration/v1/requests", subj.congAdminOwn, map[string]any{
		"taxonId":          mustTaxonIDViaHTTP(t, client, apiBase, subj.congAdminOwn),
		"congregationName": "Matrix Test Congregation",
		"countryId":        mustCountryIDViaHTTP(t, client, apiBase, subj.congAdminOwn),
		"coordinate":       map[string]any{"latitude": 50.45, "longitude": 30.52},
	})
	if status != http.StatusOK && status != http.StatusCreated {
		t.Fatalf("submitRequest: status = %d, body: %s", status, body)
	}
	var parsed struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse submitRequest response: %v (body: %s)", err, body)
	}
	return parsed.ID
}

func mustTaxonIDViaHTTP(t *testing.T, client *http.Client, apiBase, token string) string {
	t.Helper()
	status, body := doReq(t, client, apiBase, http.MethodGet, "/core/v1/taxa?query=christianity&limit=1", token, nil)
	if status != http.StatusOK {
		t.Fatalf("listTaxa: status = %d, body: %s", status, body)
	}
	var parsed struct {
		Taxa []struct {
			ID string `json:"id"`
		} `json:"taxa"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Taxa) == 0 {
		t.Fatalf("parse listTaxa response: %v (body: %s)", err, body)
	}
	return parsed.Taxa[0].ID
}

func mustCountryIDViaHTTP(t *testing.T, client *http.Client, apiBase, token string) string {
	t.Helper()
	status, body := doReq(t, client, apiBase, http.MethodGet, "/core/v1/countries", token, nil)
	if status != http.StatusOK {
		t.Fatalf("listCountries: status = %d, body: %s", status, body)
	}
	var parsed struct {
		Countries []struct {
			ID   string `json:"id"`
			Code string `json:"code"`
		} `json:"countries"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse listCountries response: %v (body: %s)", err, body)
	}
	for _, c := range parsed.Countries {
		if c.Code == "UA" {
			return c.ID
		}
	}
	t.Fatal("UA not found in listCountries response")
	return ""
}

// seedSubjects creates 5 real throwaway persons (+2 units, +grants), mints a devtoken for each, and
// registers cleanup. Anonymous needs no person at all — an empty token string.
func seedSubjects(t *testing.T, ctx context.Context, pool *pgxpool.Pool, hmacKey string) matrixSubjects {
	t.Helper()

	dirSvc := directoryapplication.NewService(pool)
	unitA, err := dirSvc.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "Matrix Test Unit A", State: directorydomain.StateActive}, seed.RootUnitID, "")
	if err != nil {
		t.Fatalf("create unitA: %v", err)
	}
	unitB, err := dirSvc.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "Matrix Test Unit B", State: directorydomain.StateActive}, seed.RootUnitID, "")
	if err != nil {
		t.Fatalf("create unitB: %v", err)
	}

	authzStore := authzadapters.NewStore(pool)

	type personSpec struct {
		label       string
		subject     string
		email       string
		roleID      string // "" = no role assignment
		targetUnit  string
		instanceAdm bool
	}
	specs := []personSpec{
		{label: "congAdminOwn", subject: "matrix-cong-admin-own", email: "matrix-cong-admin-own@example.com", roleID: seed.CongregationAdminRoleID, targetUnit: unitA.ID},
		{label: "congAdminOther", subject: "matrix-cong-admin-other", email: "matrix-cong-admin-other@example.com", roleID: seed.CongregationAdminRoleID, targetUnit: unitB.ID},
		{label: "operator", subject: "matrix-operator", email: "matrix-operator@example.com", roleID: seed.RegistrationOperatorRoleID, targetUnit: seed.RootUnitID},
		{label: "moderator", subject: "matrix-moderator", email: "matrix-moderator@example.com", roleID: seed.PlatformModeratorRoleID, targetUnit: seed.RootUnitID},
		{label: "instanceAdmin", subject: "matrix-instance-admin", email: "matrix-instance-admin@example.com", instanceAdm: true},
	}

	tokens := map[string]string{}
	personIDByLabel := map[string]string{}
	var personIDs, assignmentIDs, instanceAdminIDs []string

	for _, sp := range specs {
		var personID string
		if err := pool.QueryRow(ctx, `INSERT INTO openfaithmap.identity_persons (display_name) VALUES ($1) RETURNING id`, "Matrix Test "+sp.label).Scan(&personID); err != nil {
			t.Fatalf("insert person %s: %v", sp.label, err)
		}
		personIDs = append(personIDs, personID)
		personIDByLabel[sp.label] = personID

		var accountID string
		if err := pool.QueryRow(ctx, `INSERT INTO openfaithmap.identity_accounts (person_id, email) VALUES ($1, $2) RETURNING id`, personID, sp.email).Scan(&accountID); err != nil {
			t.Fatalf("insert account %s: %v", sp.label, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO openfaithmap.identity_external_identities (account_id, issuer, subject) VALUES ($1, $2, $3)`, accountID, devtoken.Issuer, sp.subject); err != nil {
			t.Fatalf("insert external identity %s: %v", sp.label, err)
		}

		if sp.roleID != "" {
			assignmentID, err := insertRoleAssignment(ctx, pool, authzStore, personID, sp.roleID, sp.targetUnit)
			if err != nil {
				t.Fatalf("grant role to %s: %v", sp.label, err)
			}
			assignmentIDs = append(assignmentIDs, assignmentID)
		}
		if sp.instanceAdm {
			id, err := authzStore.InsertInstanceAdmin(ctx, personID, personID)
			if err != nil {
				t.Fatalf("grant instance admin to %s: %v", sp.label, err)
			}
			instanceAdminIDs = append(instanceAdminIDs, id)
		}

		tok, err := devtoken.Mint(sp.subject, sp.email, time.Hour, hmacKey)
		if err != nil {
			t.Fatalf("mint token for %s: %v", sp.label, err)
		}
		tokens[sp.label] = tok
	}

	unitIDs := []string{unitA.ID, unitB.ID}
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range instanceAdminIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_instance_admins WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete instance admin %s: %v", id, err)
			}
		}
		for _, id := range assignmentIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete role assignment %s: %v", id, err)
			}
		}
		for _, id := range personIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE subject_person_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete residual role assignments for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.registration_requests WHERE submitted_by_person_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete registration requests submitted by %s: %v", id, err)
			}
		}
		for _, id := range unitIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE target_unit_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete role assignments targeting unit %s: %v", id, err)
			}
		}
		// Sweep by name prefix, not just unitA/unitB's own tracked ids — core_createChildOrg's own
		// success case creates one more real unit under unitA whose id this test never captures.
		// Closure/edges before the units themselves (FK-ordered), matching every other integration
		// test's own established cleanup discipline in this repo.
		const namePrefix = "Matrix Test %"
		// core_createChildOrg's own success case also gives the new unit a religion_org_profiles
		// row (SetOrgProfile, called unconditionally inside CreateChildOrg) — clear it and its own
		// possible classification row before the unit itself, same FK-ordering discipline as above.
		if _, err := pool.Exec(bg, `
			DELETE FROM openfaithmap.religion_org_classifications
			WHERE unit_id IN (SELECT id FROM openfaithmap.directory_units WHERE name LIKE $1)`, namePrefix); err != nil {
			t.Errorf("cleanup: delete org classifications for Matrix Test units: %v", err)
		}
		if _, err := pool.Exec(bg, `
			DELETE FROM openfaithmap.religion_org_profiles
			WHERE unit_id IN (SELECT id FROM openfaithmap.directory_units WHERE name LIKE $1)`, namePrefix); err != nil {
			t.Errorf("cleanup: delete org profiles for Matrix Test units: %v", err)
		}
		if _, err := pool.Exec(bg, `
			DELETE FROM openfaithmap.directory_unit_closure
			WHERE ancestor_id IN (SELECT id FROM openfaithmap.directory_units WHERE name LIKE $1)
			   OR descendant_id IN (SELECT id FROM openfaithmap.directory_units WHERE name LIKE $1)`, namePrefix); err != nil {
			t.Errorf("cleanup: delete closure rows for Matrix Test units: %v", err)
		}
		if _, err := pool.Exec(bg, `
			DELETE FROM openfaithmap.directory_unit_edges
			WHERE parent_id IN (SELECT id FROM openfaithmap.directory_units WHERE name LIKE $1)
			   OR child_id IN (SELECT id FROM openfaithmap.directory_units WHERE name LIKE $1)`, namePrefix); err != nil {
			t.Errorf("cleanup: delete edges for Matrix Test units: %v", err)
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_units WHERE name LIKE $1`, namePrefix); err != nil {
			t.Errorf("cleanup: delete Matrix Test units: %v", err)
		}
		for _, id := range personIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_external_identities WHERE account_id IN (SELECT id FROM openfaithmap.identity_accounts WHERE person_id = $1)`, id); err != nil {
				t.Errorf("cleanup: delete external identities for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_accounts WHERE person_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete account for %s: %v", id, err)
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete person %s: %v", id, err)
			}
		}
	})

	return matrixSubjects{
		anonymous:      "",
		congAdminOwn:   tokens["congAdminOwn"],
		congAdminOther: tokens["congAdminOther"],
		operator:       tokens["operator"],
		moderator:      tokens["moderator"],
		instanceAdmin:  tokens["instanceAdmin"],
		unitA:          unitA.ID,
		unitB:          unitB.ID,

		instanceAdminPersonID: personIDByLabel["instanceAdmin"],
	}
}

func insertRoleAssignment(ctx context.Context, pool *pgxpool.Pool, store *authzadapters.Store, personID, roleID, unitID string) (string, error) {
	if err := store.InsertRoleAssignment(ctx, personID, roleID, unitID, personID); err != nil {
		return "", err
	}
	// InsertRoleAssignment (M10.6) doesn't return the row's id — look it up by the unique
	// (person, role, unit, active) index this repo's own conflict-as-success idempotency relies on.
	var id string
	err := pool.QueryRow(ctx, `
		SELECT id FROM openfaithmap.authz_role_assignments
		WHERE subject_person_id = $1 AND role_id = $2 AND target_unit_id = $3 AND revoked_at IS NULL`,
		personID, roleID, unitID).Scan(&id)
	return id, err
}
