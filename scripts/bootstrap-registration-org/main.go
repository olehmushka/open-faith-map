// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Command bootstrap-registration-org is a one-time, local-dev-only operator tool (M2): creates the
// single shared church-domain root organization every congregation is registered as a unit beneath
// (a project-specific simplification — one flat OpenFaithMap organization, not one root per
// denomination), and a "registration-operator" Role carrying the permissions an M2 registration
// approval needs (religionorg.manage, site.manage, schedule.manage, assignment.grant/revoke,
// person.create/update, membership.create/update, position.create/update).
//
// It prints the new unit ID and role ID, and the exact SQL to run next: go-oikumenea has no
// API-reachable way to grant the FIRST unit-scoped role assignment on a brand-new unit — not even
// for an instance admin (assignment.grant is unit-scoped; instance-admin only auto-passes
// instance-scope checks — see internal/authorization/application/service.go's GrantAssignment,
// whose only ungated path is the internal system/bootstrap-seed one, never reachable via the API).
// This mirrors exactly what go-oikumenea's own D-Bootstrap does for the first instance admin
// ("the first admin cannot be granted from inside the API, so it must be seeded out-of-band...
// recovery/break-glass is operator-owned DB access") — one level down, for the first assignment on
// a brand-new unit.
//
// Run once per fresh go-oikumenea instance, on the compose network (oikumenea-app publishes no host
// port — D-HeadlessTopology):
//
//	docker run --rm --network open-faith-map_default -v "$PWD":/src -w /src golang:1.26-bookworm \
//	  go run ./scripts/bootstrap-registration-org -operator-person-code admin-you@example.com
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	client "github.com/olehmushka/go-oikumenea/clients/go"
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/authorization"
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/religion"
	"github.com/palantir/pkg/bearertoken"
)

const (
	bootstrapIssuer  = "https://local-dev.oikumenea.test"
	bootstrapSubject = "local-admin"
	bootstrapAud     = "oikumenea"
	bootstrapHMACKey = "local-dev-insecure-signing-key-change-me"

	roleCode = "registration-operator"
)

func main() {
	baseURL := flag.String("base-url", "https://oikumenea-app:8443", "go-oikumenea base URL (compose-internal)")
	orgCode := flag.String("org-code", "openfaithmap", "the single shared root organization's code")
	orgName := flag.String("org-name", "OpenFaithMap", "the single shared root organization's display name")
	operatorPersonCode := flag.String("operator-person-code", "", "person.code of the person to grant the registration-operator role to (required — see scripts/bootstrap-admin-person)")
	insecure := flag.Bool("insecure-skip-verify", true, "skip TLS verification (self-signed local-dev cert)")
	flag.Parse()

	if *operatorPersonCode == "" {
		fmt.Fprintln(os.Stderr, "missing required -operator-person-code")
		os.Exit(2)
	}

	ctx := context.Background()

	adminToken, err := mintBootstrapAdminToken()
	if err != nil {
		fmt.Fprintln(os.Stderr, "mint bootstrap-admin token:", err)
		os.Exit(1)
	}

	opts := []client.Option{}
	if *insecure {
		opts = append(opts, client.WithInsecureSkipVerify())
	}
	c, err := client.New(*baseURL, bearertoken.Token(adminToken), opts...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial go-oikumenea:", err)
		os.Exit(1)
	}

	who, err := c.IdentityFederation.Whoami(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "whoami (bootstrap-admin token rejected):", err)
		os.Exit(1)
	}
	fmt.Printf("authenticated as bootstrap-admin personId=%s\n", who.PersonId)

	operatorPersonID, err := findPersonByCode(ctx, c, *operatorPersonCode)
	if err != nil {
		fmt.Fprintln(os.Stderr, "find operator person:", err)
		os.Exit(1)
	}
	fmt.Printf("operator person: id=%s code=%s\n", operatorPersonID, *operatorPersonCode)

	profile, err := c.Religion.CreateRootOrg(ctx, religion.CreateRootOrgRequest{
		Code: *orgCode,
		Name: *orgName,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "createRootOrg (if this is Religion:Conflict, the org may already exist — this script does not look up an existing org by code; pass a different -org-code or check manually):", err)
		os.Exit(1)
	}
	fmt.Printf("root unit: id=%s\n", profile.UnitId)

	role, err := createOrReuseRole(ctx, c)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create role:", err)
		os.Exit(1)
	}
	fmt.Printf("role: id=%s code=%s\n", role.Id, role.Code)

	fmt.Println()
	fmt.Println("done — go-oikumenea has no API path to grant the first assignment on a brand-new")
	fmt.Println("unit (see this file's header comment). Run this SQL directly against the shared")
	fmt.Println("Postgres instance (oikumenea schema) to finish bootstrapping the operator's authority:")
	fmt.Println()
	fmt.Printf(`INSERT INTO oikumenea.authz_role_assignments
  (subject_person_id, role_id, target_unit_id, scope, graph_id, granted_by)
SELECT '%s', '%s', '%s', 'subtree', g.id, NULL
FROM oikumenea.tenant_graphs g WHERE g.code = 'canonical';
UPDATE oikumenea.authz_epoch SET epoch = epoch + 1 WHERE singleton;
`, operatorPersonID, role.Id, profile.UnitId)
}

func mintBootstrapAdminToken() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": bootstrapIssuer,
		"sub": bootstrapSubject,
		"aud": bootstrapAud,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(bootstrapHMACKey))
}

// findPersonByCode searches for the operator's Person by their stable code (set by
// scripts/bootstrap-admin-person as "admin-<email>").
func findPersonByCode(ctx context.Context, c *client.Client, code string) (string, error) {
	page, err := c.Person.ListPersons(ctx, nil, nil, &code, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return "", err
	}
	for _, p := range page.Persons {
		if p.Code != nil && *p.Code == code {
			return p.Id, nil
		}
	}
	return "", fmt.Errorf("no person found with code %q (run scripts/bootstrap-admin-person first)", code)
}

// createOrReuseRole creates the registration-operator role, or finds the existing one by code on a
// RoleConflict — a fresh instance creates, a re-run against an already-bootstrapped instance reuses.
func createOrReuseRole(ctx context.Context, c *client.Client) (authorization.Role, error) {
	r, err := c.Authorization.CreateRole(ctx, authorization.CreateRoleRequest{
		Code: roleCode,
		Name: "Registration operator",
		Permissions: []string{
			"religionorg.manage",
			"site.manage",
			"schedule.manage",
			"assignment.grant",
			"assignment.revoke",
			"person.create",
			"person.update",
			"membership.create",
			"membership.update",
			"position.create",
			"position.update",
		},
	})
	if err == nil {
		return r, nil
	}
	if !authorization.IsRoleConflict(err) {
		return authorization.Role{}, err
	}
	page, err := c.Authorization.ListRoles(ctx, nil, nil)
	if err != nil {
		return authorization.Role{}, fmt.Errorf("list after conflict: %w", err)
	}
	for _, existing := range page.Roles {
		if existing.Code == roleCode {
			return existing, nil
		}
	}
	return authorization.Role{}, fmt.Errorf("createRole reported a conflict but no role with code %q was found", roleCode)
}
