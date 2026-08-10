// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Command bootstrap-registration-org is a one-time, local-dev-only operator tool (M2): creates the
// single shared church-domain root organization every congregation is registered as a unit beneath
// (a project-specific simplification — one flat OpenFaithMap organization, not one root per
// denomination), a "registration-operator" Role (the permissions an M2 registration approval needs:
// religionorg.manage, site.manage, schedule.manage, assignment.grant/revoke, person.create/update,
// membership.create/update, position.create/update — plus assignment.read, needed since M2.3 so this
// role's own holder can pass go-oikumenea's Authorize check, which itself requires the caller already
// hold assignment.read reaching the target unit, no self-exemption by design), a "congregation-admin"
// Role (granted to a submitter on THEIR OWN new unit at approval time by internal/registration —
// created here, not there, because role.create is instance-scope and a registration-operator does
// not hold it), and (M5) a "platform-moderator" Role (D-PlatformModerator): unit.lifecycle +
// assignment.read, subtree-scoped on the same shared root unit — a deliberately separate role from
// registration-operator, not a reuse, so the two stay distinguishable at the PDP even though both
// are granted on the same unit (docs/architecture/decisions.md's D-PlatformModerator explains why:
// approving registrations and adjudicating reports are different jobs with different escalation
// paths).
//
// It prints the root unit ID and role IDs, and the exact SQL to run next: go-oikumenea has no
// API-reachable way to grant the FIRST unit-scoped role assignment on a brand-new unit — not even
// for an instance admin (assignment.grant is unit-scoped; instance-admin only auto-passes
// instance-scope checks — see internal/authorization/application/service.go's GrantAssignment,
// whose only ungated path is the internal system/bootstrap-seed one, never reachable via the API).
// This mirrors exactly what go-oikumenea's own D-Bootstrap does for the first instance admin
// ("the first admin cannot be granted from inside the API, so it must be seeded out-of-band...
// recovery/break-glass is operator-owned DB access") — one level down, for the first assignment on
// a brand-new unit.
//
// Idempotent in effect: re-running against an already-bootstrapped instance finds and reuses the
// existing org/roles rather than erroring.
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

	operatorRoleCode          = "registration-operator"
	congregationAdminRoleCode = "congregation-admin"
	platformModeratorRoleCode = "platform-moderator"
)

func main() {
	baseURL := flag.String("base-url", "https://oikumenea-app:8443", "go-oikumenea base URL (compose-internal)")
	orgCode := flag.String("org-code", "openfaithmap", "the single shared root organization's code")
	orgName := flag.String("org-name", "OpenFaithMap", "the single shared root organization's display name")
	operatorPersonCode := flag.String("operator-person-code", "", "person.code of the person to grant the registration-operator role to (required — see scripts/bootstrap-admin-person)")
	moderatorPersonCode := flag.String("moderator-person-code", "", "person.code of the person to grant the platform-moderator role to (M5 — defaults to -operator-person-code, a local-dev simplification; D-PlatformModerator's two roles stay separate regardless of who holds them)")
	insecure := flag.Bool("insecure-skip-verify", true, "skip TLS verification (self-signed local-dev cert)")
	flag.Parse()

	if *operatorPersonCode == "" {
		fmt.Fprintln(os.Stderr, "missing required -operator-person-code")
		os.Exit(2)
	}
	if *moderatorPersonCode == "" {
		*moderatorPersonCode = *operatorPersonCode
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

	moderatorPersonID := operatorPersonID
	if *moderatorPersonCode != *operatorPersonCode {
		moderatorPersonID, err = findPersonByCode(ctx, c, *moderatorPersonCode)
		if err != nil {
			fmt.Fprintln(os.Stderr, "find moderator person:", err)
			os.Exit(1)
		}
	}
	fmt.Printf("moderator person: id=%s code=%s\n", moderatorPersonID, *moderatorPersonCode)

	rootUnitID, err := createOrReuseRootOrg(ctx, c, *orgCode, *orgName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create root org:", err)
		os.Exit(1)
	}
	fmt.Printf("root unit: id=%s\n", rootUnitID)

	operatorRole, err := createOrReuseRole(ctx, c, authorization.CreateRoleRequest{
		Code: operatorRoleCode,
		Name: "Registration operator",
		Permissions: []string{
			"religionorg.manage",
			"site.manage",
			"schedule.manage",
			"assignment.grant",
			"assignment.revoke",
			"assignment.read",
			"person.create",
			"person.update",
			"membership.create",
			"membership.update",
			"position.create",
			"position.update",
			// unit.read + unit.edges.manage: M4.1, for browsing/creating jurisdiction units
			// (Tenant.ListUnits/UnitAncestors) and re-parenting a congregation's unit onto one
			// (Tenant.AddEdge/RemoveEdge on the canonical graph). unit.edges.manage is the broad
			// fallback, not a per-graph code — go-oikumenea's own D-EdgePerms only seeds dedicated
			// unit.edges.<graph>.manage permissions for command/operational, not canonical.
			"unit.read",
			"unit.edges.manage",
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "create registration-operator role:", err)
		os.Exit(1)
	}
	fmt.Printf("role: id=%s code=%s\n", operatorRole.Id, operatorRole.Code)

	adminRole, err := createOrReuseRole(ctx, c, authorization.CreateRoleRequest{
		Code: congregationAdminRoleCode,
		Name: "Congregation admin",
		Permissions: []string{
			"unit.read",
			"person.read",
			"membership.read",
			"position.read",
			"role.read",
			"person.create",
			"person.update",
			"membership.create",
			"membership.update",
			"position.create",
			"position.update",
			"religionorg.manage",
			"site.manage",
			"schedule.manage",
			// assignment.read: M3, mirroring M2.3's identical fix for registration-operator above.
			// Authorize (the target-scoped PDP check internal/content's requireManage — and
			// registration's own IsOperator — both call) requires the CALLER to already hold
			// assignment.read reaching the target unit, no self-exemption by design. Without this, a
			// congregation admin holding religionorg.manage on their own unit still gets
			// PermissionDenied from every content.manage check, the exact bug class M2.3 found and
			// fixed once already — this is that same fix, applied to the role M2.3 didn't touch.
			"assignment.read",
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "create congregation-admin role:", err)
		os.Exit(1)
	}
	fmt.Printf("role: id=%s code=%s (granted per-congregation at approval time, not by this script)\n", adminRole.Id, adminRole.Code)

	moderatorRole, err := createOrReuseRole(ctx, c, authorization.CreateRoleRequest{
		Code: platformModeratorRoleCode,
		Name: "Platform moderator",
		Permissions: []string{
			// unit.lifecycle: platform-moderator's own PDP marker permission
			// (internal/moderation/application/authorize.go's moderatePermission) — deliberately NOT
			// religionorg.manage (registration-operator/congregation-admin's permission), so the two
			// roles stay distinguishable at go-oikumenea's PDP even though both are granted on this
			// same root unit (D-PlatformModerator).
			"unit.lifecycle",
			// assignment.read: same reason registration-operator/congregation-admin both needed it
			// (M2.3/M3) — Authorize requires the CALLER to already hold assignment.read reaching the
			// target unit, no self-exemption by design. Without this, a platform-moderator holding
			// unit.lifecycle on the root unit still gets PermissionDenied from its own moderation
			// checks.
			"assignment.read",
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "create platform-moderator role:", err)
		os.Exit(1)
	}
	fmt.Printf("role: id=%s code=%s\n", moderatorRole.Id, moderatorRole.Code)

	fmt.Println()
	fmt.Println("done — go-oikumenea has no API path to grant the first assignment on a brand-new")
	fmt.Println("unit (see this file's header comment). Run this SQL directly against the shared")
	fmt.Println("Postgres instance (oikumenea schema) to finish bootstrapping the operator's and")
	fmt.Println("moderator's authority (idempotent: ON CONFLICT DO NOTHING, keyed on the same")
	fmt.Println("active-uniqueness index the table itself enforces):")
	fmt.Println()
	fmt.Printf(`INSERT INTO oikumenea.authz_role_assignments
  (subject_person_id, role_id, target_unit_id, scope, graph_id, granted_by)
SELECT v.subject_person_id, v.role_id, v.target_unit_id, 'subtree', g.id, NULL
FROM (VALUES
  ('%s'::text, '%s'::text, '%s'::text),
  ('%s'::text, '%s'::text, '%s'::text)
) AS v(subject_person_id, role_id, target_unit_id)
CROSS JOIN oikumenea.tenant_graphs g WHERE g.code = 'canonical'
ON CONFLICT (subject_person_id, role_id, target_unit_id, scope, graph_id) WHERE revoked_at IS NULL DO NOTHING;
UPDATE oikumenea.authz_epoch SET epoch = epoch + 1 WHERE singleton;
`, operatorPersonID, operatorRole.Id, rootUnitID, moderatorPersonID, moderatorRole.Id, rootUnitID)
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

// createOrReuseRootOrg creates the single shared root org, or finds its existing root unit by
// walking organizations/units by code on a Conflict — a fresh instance creates, a re-run reuses.
func createOrReuseRootOrg(ctx context.Context, c *client.Client, orgCode, orgName string) (string, error) {
	profile, createErr := c.Religion.CreateRootOrg(ctx, religion.CreateRootOrgRequest{
		Code: orgCode,
		Name: orgName,
	})
	if createErr == nil {
		return profile.UnitId, nil
	}
	// The tenant-layer org-code conflict doesn't map to a typed Religion:Conflict at this service
	// boundary (surfaces as a plain 500) — always fall back to look-up-by-code rather than branching
	// on error type; if the real cause was something else, the lookup below fails informatively too.
	orgPage, err := c.Tenant.ListOrganizations(ctx, nil, nil, nil, nil, nil)
	if err != nil {
		return "", fmt.Errorf("createRootOrg failed (%v) and the look-up-existing fallback also failed: %w", createErr, err)
	}
	var orgID string
	for _, org := range orgPage.Organizations {
		if org.Code == orgCode {
			orgID = org.Id
			break
		}
	}
	if orgID == "" {
		return "", fmt.Errorf("createRootOrg failed (%v) and no existing organization with code %q was found", createErr, orgCode)
	}
	rootsOnly := true
	unitPage, err := c.Tenant.ListUnits(ctx, orgID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &rootsOnly, nil, nil)
	if err != nil {
		return "", fmt.Errorf("list root units for org %s: %w", orgID, err)
	}
	if len(unitPage.Units) == 0 {
		return "", fmt.Errorf("organization %s (code %q) has no root unit", orgID, orgCode)
	}
	return unitPage.Units[0].Id, nil
}

// createOrReuseRole creates the given role, or finds the existing one by code on a RoleConflict and
// reconciles its permission set — a fresh instance creates, a re-run against an already-bootstrapped
// instance reuses the role but still picks up any permission this script's own list has since gained
// (e.g. M2.3 adding assignment.read to registration-operator), which a bare lookup-and-return would
// otherwise silently leave stale forever.
func createOrReuseRole(ctx context.Context, c *client.Client, req authorization.CreateRoleRequest) (authorization.Role, error) {
	r, err := c.Authorization.CreateRole(ctx, req)
	if err == nil {
		return r, nil
	}
	if !authorization.IsRoleConflict(err) {
		return authorization.Role{}, err
	}
	page, err := c.Authorization.ListRoles(ctx, nil, nil)
	if err != nil {
		return authorization.Role{}, fmt.Errorf("list roles after conflict: %w", err)
	}
	for _, existing := range page.Roles {
		if existing.Code == req.Code {
			return reconcilePermissions(ctx, c, existing, req.Permissions)
		}
	}
	return authorization.Role{}, fmt.Errorf("createRole reported a conflict but no role with code %q was found", req.Code)
}

// permissionsToAdd returns wanted's permissions missing from have as their union with have (existing
// permissions are always kept — UpdateRole's Permissions field fully replaces the set, so a partial
// list would silently revoke anything granted on the instance outside this script), and whether
// anything was actually missing. Pure so it's unit-testable without a live go-oikumenea client.
func permissionsToAdd(have, wanted []string) (union []string, changed bool) {
	seen := make(map[string]bool, len(have))
	union = append(union, have...)
	for _, p := range have {
		seen[p] = true
	}
	for _, p := range wanted {
		if !seen[p] {
			union = append(union, p)
			seen[p] = true
			changed = true
		}
	}
	return union, changed
}

// reconcilePermissions ensures existing already holds every permission in wanted, calling UpdateRole
// with the merged union only if something is missing — a re-run against an already-reconciled
// instance makes zero UpdateRole calls, not just a harmless one.
func reconcilePermissions(ctx context.Context, c *client.Client, existing authorization.Role, wanted []string) (authorization.Role, error) {
	union, changed := permissionsToAdd(existing.Permissions, wanted)
	if !changed {
		return existing, nil
	}
	updated, err := c.Authorization.UpdateRole(ctx, existing.Id, authorization.UpdateRoleRequest{Permissions: &union})
	if err != nil {
		return authorization.Role{}, fmt.Errorf("updateRole %s (code %q) to add missing permissions: %w", existing.Id, existing.Code, err)
	}
	fmt.Printf("role %s (code %q): added missing permissions, now: %v\n", updated.Id, updated.Code, updated.Permissions)
	return updated, nil
}
