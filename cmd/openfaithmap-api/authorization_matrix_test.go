// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M10.9's table-driven authorization matrix: {anonymous, congregation-admin@own,
// congregation-admin@other, registration-operator, platform-moderator, instance-admin} against a
// representative, real, HTTP-level sample of every guarded endpoint across all 7 Conjure contracts
// — not the full ~70-endpoint x 6-subject cartesian product (many writes need complex pre-existing
// fixtures this test doesn't build), but every one of the four gate shapes
// (anonymous / authenticated-only / target-scoped religionorg.manage / root-scoped moderation.standing /
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

	// M11.3: every non-anonymous subject also carries its own identity_sessions row id, sent as
	// X-Session-Id alongside the bearer — the per-request session check (D-SessionTracking) applies
	// to dev-issued tokens the same as real Google ID tokens, no issuer-based carve-out (confirmed
	// decision, docs/milestones.md's M11.3 row).
	congAdminOwnSession   string
	congAdminOtherSession string
	operatorSession       string
	moderatorSession      string
	instanceAdminSession  string

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
				status, _ := doReq(t, client, apiBase, c.method, c.path, "", "", c.body)
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
				anonStatus, _ := doReq(t, client, apiBase, c.method, c.path, "", "", nil)
				if anonStatus != http.StatusUnauthorized {
					t.Errorf("%s: anonymous got %d, want 401", c.name, anonStatus)
				}
				authStatus, body := doReq(t, client, apiBase, c.method, c.path, subj.congAdminOwn, subj.congAdminOwnSession, nil)
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
			anonStatus, _ := doReq(t, client, apiBase, http.MethodPost, "/core/v1/units/children", "", "", body)
			assertStatus(t, "anonymous", anonStatus, http.StatusUnauthorized)

			otherStatus, otherBody := doReq(t, client, apiBase, http.MethodPost, "/core/v1/units/children", subj.congAdminOther, subj.congAdminOtherSession, body)
			assertStatus(t, "congAdminOther (wrong unit)", otherStatus, http.StatusForbidden)
			assertErrorName(t, otherBody, "Core:Forbidden")

			ownStatus, _ := doReq(t, client, apiBase, http.MethodPost, "/core/v1/units/children", subj.congAdminOwn, subj.congAdminOwnSession, body)
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
			anonStatus, _ := doReq(t, client, apiBase, http.MethodPost, "/vouching/v1/vouches", "", "", deniedBody)
			assertStatus(t, "anonymous", anonStatus, http.StatusUnauthorized)

			deniedStatus, _ := doReq(t, client, apiBase, http.MethodPost, "/vouching/v1/vouches", subj.congAdminOwn, subj.congAdminOwnSession, deniedBody)
			assertStatus(t, "congAdminOwn (no standing on unitB, the guarantor unit)", deniedStatus, http.StatusForbidden)

			allowedBody := map[string]any{
				"claimantPersonId": subj.instanceAdmin, "congregationUnitId": subj.unitB,
				"guarantorCongregationUnitId": subj.unitA,
			}
			allowedStatus, allowedRespBody := doReq(t, client, apiBase, http.MethodPost, "/vouching/v1/vouches", subj.congAdminOwn, subj.congAdminOwnSession, allowedBody)
			if allowedStatus != http.StatusOK && allowedStatus != http.StatusCreated {
				t.Errorf("congAdminOwn (own standing on unitA, the guarantor unit): status = %d, body: %s", allowedStatus, allowedRespBody)
			}
		})

		t.Run("registration_approveRequest_denied", func(t *testing.T) {
			reqID := submitThrowawayRequest(t, client, apiBase, subj)
			status, body := doReq(t, client, apiBase, http.MethodPost, "/registration/v1/requests/"+reqID+"/approve", subj.congAdminOwn, subj.congAdminOwnSession, map[string]any{})
			assertStatus(t, "congAdminOwn (not an operator)", status, http.StatusForbidden)
			assertErrorName(t, body, "Registration:Forbidden")
		})

		t.Run("congregationimport_runJurisdictionSync", func(t *testing.T) {
			anonStatus, _ := doReq(t, client, apiBase, http.MethodPost, "/congregation-import/v1/jurisdiction-sync/runs", "", "", map[string]any{"sourceCode": "wikidata-catholic"})
			assertStatus(t, "anonymous", anonStatus, http.StatusUnauthorized)

			status, body := doReq(t, client, apiBase, http.MethodPost, "/congregation-import/v1/jurisdiction-sync/runs", subj.congAdminOwn, subj.congAdminOwnSession, map[string]any{"sourceCode": "wikidata-catholic"})
			assertStatus(t, "congAdminOwn (not an operator)", status, http.StatusForbidden)
			assertErrorName(t, body, "CongregationImport:Forbidden")
		})

		t.Run("congregationimport_listTaxonAliases", func(t *testing.T) {
			anonStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/congregation-import/v1/taxon-aliases", "", "", nil)
			assertStatus(t, "anonymous", anonStatus, http.StatusUnauthorized)

			deniedStatus, deniedBody := doReq(t, client, apiBase, http.MethodGet, "/congregation-import/v1/taxon-aliases", subj.congAdminOwn, subj.congAdminOwnSession, nil)
			assertStatus(t, "congAdminOwn (not an operator)", deniedStatus, http.StatusForbidden)
			assertErrorName(t, deniedBody, "CongregationImport:Forbidden")

			allowedStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/congregation-import/v1/taxon-aliases", subj.operator, subj.operatorSession, nil)
			assertStatus(t, "operator", allowedStatus, http.StatusOK)
		})
	})

	// ---- category: unit.lifecycle, target-scoped (M12.1) ----
	t.Run("target_scoped_unit_lifecycle", func(t *testing.T) {
		createBody := map[string]any{"parentUnitId": subj.unitA, "code": "", "name": "Matrix Test M12.1 Child"}

		anonStatus, _ := doReq(t, client, apiBase, http.MethodPost, "/core/v1/units", "", "", createBody)
		assertStatus(t, "anonymous", anonStatus, http.StatusUnauthorized)

		// congAdminOwn holds religionorg.manage on unitA (M11's own seed), not unit.lifecycle — a
		// real proof the two permission codes are genuinely distinct gates, not one conflated check.
		deniedStatus, deniedBody := doReq(t, client, apiBase, http.MethodPost, "/core/v1/units", subj.congAdminOwn, subj.congAdminOwnSession, createBody)
		assertStatus(t, "congAdminOwn (holds religionorg.manage, not unit.lifecycle)", deniedStatus, http.StatusForbidden)
		assertErrorName(t, deniedBody, "Core:Forbidden")

		createStatus, createRespBody := doReq(t, client, apiBase, http.MethodPost, "/core/v1/units", subj.instanceAdmin, subj.instanceAdminSession, createBody)
		if createStatus != http.StatusOK && createStatus != http.StatusCreated {
			t.Fatalf("createUnit(instanceAdmin): status = %d, body: %s", createStatus, createRespBody)
		}
		var created struct {
			Id string `json:"id"`
		}
		if err := json.Unmarshal(createRespBody, &created); err != nil || created.Id == "" {
			t.Fatalf("parse createUnit response: %v (body: %s)", err, createRespBody)
		}

		updateStatus, updateBody := doReq(t, client, apiBase, http.MethodPost, "/core/v1/unit-lifecycle/"+created.Id, subj.instanceAdmin, subj.instanceAdminSession, map[string]any{"name": "Matrix Test M12.1 Child Renamed"})
		assertStatus(t, "instanceAdmin updateUnit", updateStatus, http.StatusOK)
		var updated struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(updateBody, &updated); err != nil || updated.Name != "Matrix Test M12.1 Child Renamed" {
			t.Errorf("updateUnit response = %s, want name %q", updateBody, "Matrix Test M12.1 Child Renamed")
		}

		stateStatus, stateBody := doReq(t, client, apiBase, http.MethodPost, "/core/v1/unit-lifecycle/"+created.Id+"/state", subj.instanceAdmin, subj.instanceAdminSession, map[string]any{"state": "suspended"})
		assertStatus(t, "instanceAdmin setUnitState", stateStatus, http.StatusOK)
		var suspended struct {
			State string `json:"state"`
		}
		if err := json.Unmarshal(stateBody, &suspended); err != nil || suspended.State != "suspended" {
			t.Errorf("setUnitState response = %s, want state %q", stateBody, "suspended")
		}

		// The root unit refuses every state change regardless of who's asking, instanceAdmin included.
		rootStateStatus, rootStateBody := doReq(t, client, apiBase, http.MethodPost, "/core/v1/unit-lifecycle/"+seed.RootUnitID+"/state", subj.instanceAdmin, subj.instanceAdminSession, map[string]any{"state": "suspended"})
		assertStatus(t, "instanceAdmin setUnitState(root)", rootStateStatus, http.StatusConflict)
		assertErrorName(t, rootStateBody, "Core:RootUnitProtected")

		deleteAnonStatus, _ := doReq(t, client, apiBase, http.MethodDelete, "/core/v1/units/"+created.Id, "", "", nil)
		assertStatus(t, "anonymous deleteUnit", deleteAnonStatus, http.StatusUnauthorized)

		deleteStatus, deleteBody := doReq(t, client, apiBase, http.MethodDelete, "/core/v1/units/"+created.Id, subj.instanceAdmin, subj.instanceAdminSession, nil)
		if deleteStatus != http.StatusOK && deleteStatus != http.StatusNoContent {
			t.Errorf("deleteUnit(instanceAdmin): status = %d, body: %s", deleteStatus, deleteBody)
		}
	})

	// ---- category: moderation.standing on root (platform-moderator) ----
	t.Run("root_scoped_moderation_standing", func(t *testing.T) {
		t.Run("moderation_listReports", func(t *testing.T) {
			anonStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/moderation/v1/reports", "", "", nil)
			assertStatus(t, "anonymous", anonStatus, http.StatusUnauthorized)

			deniedStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/moderation/v1/reports", subj.congAdminOwn, subj.congAdminOwnSession, nil)
			assertStatus(t, "congAdminOwn (not a moderator)", deniedStatus, http.StatusForbidden)

			allowedStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/moderation/v1/reports", subj.moderator, subj.moderatorSession, nil)
			assertStatus(t, "platform-moderator", allowedStatus, http.StatusOK)
		})

		t.Run("vouching_listVouches", func(t *testing.T) {
			deniedStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/vouching/v1/vouches", subj.operator, subj.operatorSession, nil)
			assertStatus(t, "operator (not a moderator)", deniedStatus, http.StatusForbidden)

			allowedStatus, _ := doReq(t, client, apiBase, http.MethodGet, "/vouching/v1/vouches", subj.moderator, subj.moderatorSession, nil)
			assertStatus(t, "platform-moderator", allowedStatus, http.StatusOK)
		})
	})

	// ---- category: instance-admin plane (CoreSuperAdminService — all 14 endpoints share one
	// route-group gate, so this is the highest-value single check in the whole matrix: every
	// subject except instance-admin must be refused by the SAME middleware, not per-handler logic) ----
	t.Run("instance_admin_plane", func(t *testing.T) {
		endpoints := []struct {
			name, method, path string
			// adminOK is every status code that counts as "instanceAdmin reached the real handler,
			// past the route-group gate" — usually just 200, but revokeSession's synthetic sessionId
			// legitimately 404s past the gate rather than succeeding, unlike every other subject
			// below, who are all refused at 403 before the handler ever runs.
			adminOK []int
			// adminBody is sent only on the instanceAdmin call — the anonymous/denied checks below
			// are refused at the route-group gate before any handler ever decodes a body, so nil is
			// fine for those; nil here for every endpoint that doesn't need one.
			adminBody any
		}{
			{"searchPersons", http.MethodGet, "/core/v1/super-admin/persons", []int{http.StatusOK}, nil},
			{"listRoles", http.MethodGet, "/core/v1/super-admin/roles", []int{http.StatusOK}, nil},
			{"listInstanceAdmins", http.MethodGet, "/core/v1/super-admin/instance-admins", []int{http.StatusOK}, nil},
			// M11.1 — keeps the representative sample current with the new endpoints on this service.
			{"getAccountStatus", http.MethodGet, "/core/v1/super-admin/persons/" + subj.instanceAdminPersonID + "/account-status", []int{http.StatusOK}, nil},
			// M11.2 — same reasoning: listAuditLog must be refused by the same route-group gate too.
			{"listAuditLog", http.MethodGet, "/core/v1/super-admin/audit-log", []int{http.StatusOK}, nil},
			// M11.3 — same reasoning: listSessions/revokeSession must be refused by the same
			// route-group gate too. revokeSession uses a syntactically valid but nonexistent
			// sessionId — proving instanceAdmin reaches the handler (404) is enough; a real fixture
			// isn't needed to prove the GATE is the same for every subject.
			{"listSessions", http.MethodGet, "/core/v1/super-admin/persons/" + subj.instanceAdminPersonID + "/sessions", []int{http.StatusOK}, nil},
			{"revokeSession", http.MethodDelete, "/core/v1/super-admin/persons/" + subj.instanceAdminPersonID + "/sessions/00000000-0000-8000-8000-000000000000", []int{http.StatusOK, http.StatusNotFound}, nil},
			// M11.6 — same reasoning: invitePerson must be refused by the same route-group gate too.
			// The email deliberately reuses an already-seeded subject's own address (never a fresh
			// one) so instanceAdmin's call reliably 409s on Core:AccountAlreadyExists past the gate —
			// proving the handler was reached with no new person/account/invite row left behind for
			// every repeated run of this test.
			{"invitePerson", http.MethodPost, "/core/v1/super-admin/invites", []int{http.StatusConflict}, map[string]any{"email": "matrix-instance-admin@example.com", "displayName": "Matrix Test Invitee"}},
			// M11.7 — same reasoning: bulkGrantUnitRole must be refused by the same route-group gate
			// too. An empty personIds list is side-effect-free (never touches the DB) and reliably
			// 400s on Core:EmptyPersonIdsList past the gate, proving the handler was reached with no
			// fixture needed.
			{"bulkGrantUnitRole", http.MethodPost, "/core/v1/super-admin/bulk-role-assignments", []int{http.StatusBadRequest},
				map[string]any{"personIds": []string{}, "roleId": "00000000-0000-8000-8000-000000000000", "unitId": "00000000-0000-8000-8000-000000000000"}},
		}
		for _, ep := range endpoints {
			t.Run(ep.name, func(t *testing.T) {
				anonStatus, _ := doReq(t, client, apiBase, ep.method, ep.path, "", "", nil)
				assertStatus(t, "anonymous", anonStatus, http.StatusUnauthorized)

				for _, s := range []struct {
					name, token, sessionID string
				}{
					{"congAdminOwn", subj.congAdminOwn, subj.congAdminOwnSession},
					{"congAdminOther", subj.congAdminOther, subj.congAdminOtherSession},
					{"operator", subj.operator, subj.operatorSession},
					{"moderator", subj.moderator, subj.moderatorSession},
				} {
					status, body := doReq(t, client, apiBase, ep.method, ep.path, s.token, s.sessionID, nil)
					assertStatus(t, s.name, status, http.StatusForbidden)
					assertErrorName(t, body, "Authz:InstanceAdminRequired")
				}

				adminStatus, _ := doReq(t, client, apiBase, ep.method, ep.path, subj.instanceAdmin, subj.instanceAdminSession, ep.adminBody)
				if !containsStatus(ep.adminOK, adminStatus) {
					t.Errorf("instanceAdmin: status = %d, want one of %v", adminStatus, ep.adminOK)
				}
			})
		}
	})

	// ---- category: resolveInvite (M11.6) — the one CoreService endpoint reachable with NO bearer
	// at all (internal/identity/middleware's anonymousRoutes), the opposite direction from every
	// other check in this file. Proves the bypass is real (anonymous does NOT 401) and that an
	// authenticated caller still gets a normal typed response too (the bypass doesn't special-case
	// away the endpoint's own logic) — a bogus token both ways returns Core:InviteNotFound, not 401.
	t.Run("resolveInvite_anonymous_bypass", func(t *testing.T) {
		anonStatus, anonBody := doReq(t, client, apiBase, http.MethodPost, "/core/v1/public/invites/resolve", "", "", map[string]any{"token": "not-a-real-token"})
		assertStatus(t, "anonymous", anonStatus, http.StatusNotFound)
		assertErrorName(t, anonBody, "Core:InviteNotFound")

		authedStatus, authedBody := doReq(t, client, apiBase, http.MethodPost, "/core/v1/public/invites/resolve", subj.congAdminOwn, subj.congAdminOwnSession, map[string]any{"token": "not-a-real-token"})
		assertStatus(t, "congAdminOwn (endpoint is public either way)", authedStatus, http.StatusNotFound)
		assertErrorName(t, authedBody, "Core:InviteNotFound")
	})
}

func containsStatus(statuses []int, got int) bool {
	for _, s := range statuses {
		if s == got {
			return true
		}
	}
	return false
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

// doReq issues one HTTP request as (token, sessionID) — both empty for anonymous. sessionID is
// sent as X-Session-Id (M11.3, D-SessionTracking): every authenticated request now needs one,
// dev-issued tokens included (confirmed decision, docs/milestones.md's M11.3 row) — see
// seedSubjects, which inserts a real identity_sessions row per minted subject.
func doReq(t *testing.T, client *http.Client, base, method, path, token, sessionID string, body any) (int, []byte) {
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
	if sessionID != "" {
		req.Header.Set("X-Session-Id", sessionID)
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
	status, body := doReq(t, client, apiBase, http.MethodPost, "/registration/v1/requests", subj.congAdminOwn, subj.congAdminOwnSession, map[string]any{
		"taxonId":          mustTaxonIDViaHTTP(t, client, apiBase, subj.congAdminOwn, subj.congAdminOwnSession),
		"congregationName": "Matrix Test Congregation",
		"countryId":        mustCountryIDViaHTTP(t, client, apiBase, subj.congAdminOwn, subj.congAdminOwnSession),
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

func mustTaxonIDViaHTTP(t *testing.T, client *http.Client, apiBase, token, sessionID string) string {
	t.Helper()
	status, body := doReq(t, client, apiBase, http.MethodGet, "/core/v1/taxa?query=christianity&limit=1", token, sessionID, nil)
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

func mustCountryIDViaHTTP(t *testing.T, client *http.Client, apiBase, token, sessionID string) string {
	t.Helper()
	status, body := doReq(t, client, apiBase, http.MethodGet, "/core/v1/countries", token, sessionID, nil)
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

	authzStore := authzadapters.NewRepository(pool)

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
	sessionIDs := map[string]string{}
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
		// M11.3: every non-anonymous request now needs a valid, unrevoked X-Session-Id alongside
		// the bearer, dev-issued tokens included (no issuer-based carve-out) — insert the backing
		// identity_sessions row directly rather than going through registerSession's HTTP endpoint
		// (which itself requires an already-authenticated request). Cascades away with the account
		// on cleanup below (identity_sessions.account_id ON DELETE CASCADE) — no separate teardown.
		var sessionID string
		if err := pool.QueryRow(ctx, `INSERT INTO openfaithmap.identity_sessions (account_id, issuer) VALUES ($1, $2) RETURNING id`, accountID, devtoken.Issuer).Scan(&sessionID); err != nil {
			t.Fatalf("insert session for %s: %v", sp.label, err)
		}
		sessionIDs[sp.label] = sessionID

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
		// M12.1's own new subtest is the first one in this file to make a real audit-logged write as
		// instanceAdmin — identity_audit_log.actor_person_id is ON DELETE SET NULL
		// (migrations/0016_core_audit.sql:27), and that SET NULL is itself an UPDATE the
		// identity_audit_log_reject_mutation append-only trigger blocks unless disabled first, same
		// dance core_self_service_integration_test.go's own cleanup already uses.
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log DISABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: disable reject_mutation: %v", err)
		}
		for _, id := range personIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_audit_log WHERE actor_person_id = $1`, id); err != nil {
				t.Errorf("cleanup: delete audit rows for %s: %v", id, err)
			}
		}
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log ENABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: re-enable reject_mutation: %v", err)
		}
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

		congAdminOwnSession:   sessionIDs["congAdminOwn"],
		congAdminOtherSession: sessionIDs["congAdminOther"],
		operatorSession:       sessionIDs["operator"],
		moderatorSession:      sessionIDs["moderator"],
		instanceAdminSession:  sessionIDs["instanceAdmin"],

		unitA: unitA.ID,
		unitB: unitB.ID,

		instanceAdminPersonID: personIDByLabel["instanceAdmin"],
	}
}

func insertRoleAssignment(ctx context.Context, pool *pgxpool.Pool, store *authzadapters.Repository, personID, roleID, unitID string) (string, error) {
	// InsertRoleAssignment now returns the row's id directly (M11.2 — the audit log needs a real
	// target_id, including on the idempotent-conflict path), so no separate lookup query is needed.
	// Always scope="unit" here (M12.2 added scope/graphID params) — this matrix's own seed points are
	// all exact-unit grants; TestAuthorizationMatrix's subtree case (M12.2) calls
	// store.InsertRoleAssignment directly instead of through this helper.
	return store.InsertRoleAssignment(ctx, personID, roleID, unitID, "unit", "", personID, nil)
}
