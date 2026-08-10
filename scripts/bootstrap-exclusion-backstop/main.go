// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Command bootstrap-exclusion-backstop is a one-time, local-dev-only operator tool (M4.1): closes
// D-Exclusions' org-level backstop, which was "designed, not real" under D-FlatRoot because there
// was no per-excluded-body root unit to attach a religion_org_policies row to. For each of the three
// named exclusions (domain.ExcludedTaxonCodes), this creates a placeholder Unit beneath the shared
// root and attaches go-oikumenea's excludes_child_creation policy to it, so createChildOrg beneath
// any of them is rejected by go-oikumenea itself — a second enforcement layer behind the existing
// taxon-level gate (checkNotExcluded), not a replacement for it.
//
// Idempotent in effect: re-running against an already-bootstrapped instance finds and reuses the
// existing placeholder units (by code) and skips re-adding a policy that's already attached.
//
// Run once per fresh go-oikumenea instance, on the compose network (oikumenea-app publishes no host
// port — D-HeadlessTopology):
//
//	docker run --rm --network open-faith-map_default -v "$PWD":/src -w /src golang:1.26-bookworm \
//	  go run ./scripts/bootstrap-exclusion-backstop -root-unit-id <REGISTRATION_ROOT_UNIT_ID>
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	client "github.com/olehmushka/go-oikumenea/clients/go"
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/religion"
	"github.com/olehmushka/open-faith-map/internal/registration/domain"
	"github.com/palantir/pkg/bearertoken"
)

const (
	bootstrapIssuer  = "https://local-dev.oikumenea.test"
	bootstrapSubject = "local-admin"
	bootstrapAud     = "oikumenea"
	bootstrapHMACKey = "local-dev-insecure-signing-key-change-me"

	excludesChildCreationPolicyCode = "excludes_child_creation"
)

// excludedBodies maps each D-Exclusions taxon code to the placeholder unit's own code/name — kept
// here rather than derived, since the display name isn't part of domain.ExcludedTaxonCodes.
var excludedBodies = []struct {
	taxonCode string
	unitCode  string
	unitName  string
}{
	{"russian_orthodox_church", "excluded-russian_orthodox_church", "Russian Orthodox Church (excluded — D-Exclusions)"},
	{"jehovahs_witnesses", "excluded-jehovahs_witnesses", "Jehovah's Witnesses (excluded — D-Exclusions)"},
	{"lds_church", "excluded-lds_church", "LDS Church (excluded — D-Exclusions)"},
}

func main() {
	baseURL := flag.String("base-url", "https://oikumenea-app:8443", "go-oikumenea base URL (compose-internal)")
	rootUnitID := flag.String("root-unit-id", "", "the shared root unit id (REGISTRATION_ROOT_UNIT_ID — required)")
	insecure := flag.Bool("insecure-skip-verify", true, "skip TLS verification (self-signed local-dev cert)")
	flag.Parse()

	if *rootUnitID == "" {
		fmt.Fprintln(os.Stderr, "missing required -root-unit-id")
		os.Exit(2)
	}
	if len(excludedBodies) != len(domain.ExcludedTaxonCodes) {
		fmt.Fprintln(os.Stderr, "excludedBodies is out of sync with domain.ExcludedTaxonCodes — update both together")
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

	rootUnit, err := c.Tenant.GetUnit(ctx, *rootUnitID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "get root unit (to resolve its owning org for ListUnits' required org param):", err)
		os.Exit(1)
	}

	excludesKindID, err := findPolicyKindByCode(ctx, c, excludesChildCreationPolicyCode)
	if err != nil {
		fmt.Fprintln(os.Stderr, "find excludes_child_creation policy kind:", err)
		os.Exit(1)
	}

	for _, body := range excludedBodies {
		if !domain.ExcludedTaxonCodes[body.taxonCode] {
			fmt.Fprintf(os.Stderr, "excludedBodies has %q, not present in domain.ExcludedTaxonCodes — update both together\n", body.taxonCode)
			os.Exit(2)
		}

		taxonID, err := findTaxonByCode(ctx, c, body.taxonCode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "find taxon %q: %v\n", body.taxonCode, err)
			os.Exit(1)
		}

		unitID, err := createOrReuseUnit(ctx, c, rootUnit.OrgId, *rootUnitID, body.unitCode, body.unitName, taxonID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create placeholder unit for %q: %v\n", body.taxonCode, err)
			os.Exit(1)
		}
		fmt.Printf("placeholder unit for %q: id=%s code=%s\n", body.taxonCode, unitID, body.unitCode)

		if err := ensureExclusionPolicy(ctx, c, unitID, excludesKindID); err != nil {
			fmt.Fprintf(os.Stderr, "attach excludes_child_creation policy to %s: %v\n", unitID, err)
			os.Exit(1)
		}
		fmt.Printf("  excludes_child_creation policy: attached\n")
	}

	fmt.Println("\ndone — createChildOrg beneath any of these three placeholder units now returns")
	fmt.Println("Religion:ChildCreationExcluded (second enforcement layer behind the taxon-level gate).")
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

// findTaxonByCode searches ListTaxa's free-text query for an exact Code match, mirroring
// bootstrap-registration-org's createOrReuseRootOrg find-by-code pattern (no exact-code filter
// exists on the endpoint itself).
func findTaxonByCode(ctx context.Context, c *client.Client, code string) (string, error) {
	page, err := c.Religion.ListTaxa(ctx, nil, nil, nil, nil, &code, nil, nil)
	if err != nil {
		return "", err
	}
	for _, t := range page.Taxa {
		if t.Code == code {
			return t.Id, nil
		}
	}
	return "", fmt.Errorf("no taxon found with code %q", code)
}

func findPolicyKindByCode(ctx context.Context, c *client.Client, code string) (string, error) {
	kinds, err := c.Religion.ListPolicyKinds(ctx)
	if err != nil {
		return "", err
	}
	for _, k := range kinds.PolicyKinds {
		if k.Code == code {
			return k.Id, nil
		}
	}
	return "", fmt.Errorf("no policy kind found with code %q", code)
}

// createOrReuseUnit creates the excluded body's placeholder unit as a child of root, or finds the
// existing one by code on a Conflict — a fresh instance creates, a re-run reuses.
func createOrReuseUnit(ctx context.Context, c *client.Client, orgID, rootUnitID, unitCode, unitName, primaryTaxonID string) (string, error) {
	profile, createErr := c.Religion.CreateChildOrg(ctx, rootUnitID, religion.CreateChildOrgRequest{
		Code:           unitCode,
		Name:           unitName,
		PrimaryTaxonId: &primaryTaxonID,
	})
	if createErr == nil {
		return profile.UnitId, nil
	}
	// Mirrors bootstrap-registration-org's createOrReuseRootOrg: a unit-code conflict doesn't map to
	// a typed Religion:Conflict at this service boundary (surfaces as a plain 500) — always fall back
	// to look-up-by-code rather than branching on error type; if the real cause was something else,
	// the lookup below fails informatively too.
	canonical := "canonical"
	unitPage, err := c.Tenant.ListUnits(ctx, orgID, &unitCode, nil, nil, nil, nil, nil, nil, nil, nil, &canonical, &rootUnitID, nil, nil, nil)
	if err != nil {
		return "", fmt.Errorf("createChildOrg failed (%v) and the look-up-existing fallback also failed: %w", createErr, err)
	}
	for _, u := range unitPage.Units {
		if u.Code != nil && *u.Code == unitCode {
			return u.Id, nil
		}
	}
	return "", fmt.Errorf("createChildOrg failed (%v) and no unit with code %q was found among root's children", createErr, unitCode)
}

// ensureExclusionPolicy attaches the excludes_child_creation policy, treating an already-attached
// policy (Conflict) as success rather than an error.
func ensureExclusionPolicy(ctx context.Context, c *client.Client, unitID, policyKindID string) error {
	reason := "D-Exclusions permanent list"
	_, err := c.Religion.AddOrgPolicy(ctx, unitID, religion.AddOrgPolicyRequest{
		PolicyKindId: policyKindID,
		Reason:       &reason,
	})
	if err == nil || religion.IsConflict(err) {
		return nil
	}
	return err
}
