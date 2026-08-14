// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Command bootstrap-service-principal is a one-time, local-dev-only operator tool (M1,
// D-CoreDependency): it mints a token for go-oikumenea's HS256 bootstrap-admin identity (configured
// in deploy/oikumenea-install.yml, never used for anything else), then uses it to register
// OpenFaithMap's real GCP service-account identity as a go-oikumenea service principal
// (registerServicePrincipal), granting religion.read (the real, documented need — currently
// unexercisable, see the grant loop below) and connector.read (proof-only, machine-reachable today).
// core-integration.md's "audit.write" is deliberately not requested — that permission doesn't exist.
//
// Run once per fresh go-oikumenea instance, on the compose network (oikumenea-app publishes no host
// port — D-HeadlessTopology):
//
//	docker run --rm --network open-faith-map_default -v "$PWD":/src -w /src golang:1.26-bookworm \
//	  go run ./scripts/bootstrap-service-principal -subject <google-service-account-numeric-sub>
//
// Idempotent in effect (ServicePrincipalConflict on a re-run is treated as already-done), but not
// idempotent in the sense of updating an existing registration — see registerOrReuse.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	client "github.com/olehmushka/go-oikumenea/clients/go"
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/authorization"
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/identityfederation"
	"github.com/palantir/pkg/bearertoken"
)

// Bootstrap-admin identity + HS256 key, matching deploy/oikumenea-install.yml exactly. Local-dev
// only (GuardSymmetricIssuers refuses this issuer type outside environment: local) — safe to keep in
// source, same as go-oikumenea's own published local-dev example.
const (
	bootstrapIssuer  = "https://local-dev.oikumenea.test"
	bootstrapSubject = "local-admin"
	bootstrapAud     = "oikumenea"
	bootstrapHMACKey = "local-dev-insecure-signing-key-change-me"

	// serviceAudience is the target_audience OpenFaithMap's own Google ID tokens carry (see
	// internal/coreintegration) — must match the "openfaithmap-api" entry in the Google issuer's
	// `audiences` list in deploy/oikumenea-install.yml.
	googleIssuer = "https://accounts.google.com"

	principalCode = "openfaithmap-api"
	principalName = "OpenFaithMap API"
)

func main() {
	baseURL := flag.String("base-url", "https://oikumenea-app:8443", "go-oikumenea base URL (compose-internal)")
	subject := flag.String("subject", "", "the GCP service account's numeric OAuth2 client_id (its ID token 'sub')")
	insecure := flag.Bool("insecure-skip-verify", true, "skip TLS verification (self-signed local-dev cert)")
	flag.Parse()

	if *subject == "" {
		fmt.Fprintln(os.Stderr, "missing required -subject (the service account's client_id from its key JSON)")
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
		fmt.Fprintln(os.Stderr, "whoami (bootstrap-admin token rejected — is deploy/oikumenea-install.yml's bootstrap-admin block applied?):", err)
		os.Exit(1)
	}
	fmt.Printf("authenticated as bootstrap-admin personId=%s\n", who.PersonId)

	principal, err := registerOrReuse(ctx, c, *subject)
	if err != nil {
		fmt.Fprintln(os.Stderr, "register service principal:", err)
		os.Exit(1)
	}
	fmt.Printf("service principal: id=%s code=%s issuer=%s subject=%s\n", principal.Id, principal.Code, principal.Issuer, principal.Subject)

	// religion.read: the real, documented need (core-integration.md — discovery-cache refresh). Used
	// to be structurally unexercisable by any principal (every religion-module read endpoint was
	// gated with RequireAnywhere, a person-shaped PEP path that denies a machine subject outright,
	// regardless of grants) — fixed go-oikumenea-side (GH-33, "gate instance-wide reads with
	// RequireServiceOrPerson"), so this grant is now real.
	//
	// country.read: same fix, same shape, one module over — go-oikumenea's GeoService.ListCountries
	// (and ListPlaces/ResolveCoordinate) had the identical RequireAnywhere gap. Added live
	// (2026-08-14) for congregationimport's matchCountry (application/countrymatch.go): resolving a
	// connector's CountryHint to a real country RID at ingest time, under this same service
	// principal, the same way checkExcluded already calls Religion.GetTaxon. Fixed go-oikumenea-side
	// as GH-37 ("gate country.read reads with RequireServiceOrPerson", image 0.0.5+) — bump
	// docker-compose.yml's oikumenea-app image and re-run this bootstrap script before relying on it.
	//
	// docs/modules/core-integration.md's authorization-touchpoints table also lists "audit.write" —
	// that grant does not exist: go-oikumenea's audit module is READ-ONLY from the API ("there is no
	// write endpoint; writes happen in-process", docs/modules/audit.md) and its permission catalog
	// defines only audit.read. Both discrepancies need a core-integration.md correction pass, not
	// attempted here.
	//
	// connector.read is added alongside them SOLELY to prove the service-principal mechanism itself
	// (registration -> grant -> Google ID token -> PDP allow) end-to-end against a permission that is
	// actually machine-reachable today — see internal/coreintegration's integration test.
	for _, perm := range []string{"religion.read", "country.read", "connector.read"} {
		if err := grantOrReuse(ctx, c, principal.Id, perm); err != nil {
			fmt.Fprintf(os.Stderr, "grant %s: %v\n", perm, err)
			os.Exit(1)
		}
		fmt.Printf("granted %s (instance-wide)\n", perm)
	}

	fmt.Println("done")
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

// registerOrReuse registers OpenFaithMap's service principal, or looks up the existing one on a
// ServicePrincipalConflict (code already taken) — a fresh instance registers, a re-run of this
// script against an already-bootstrapped instance reuses it.
func registerOrReuse(ctx context.Context, c *client.Client, subject string) (identityfederation.ServicePrincipal, error) {
	sp, err := c.IdentityFederation.RegisterServicePrincipal(ctx, identityfederation.RegisterServicePrincipalRequest{
		Code:    principalCode,
		Name:    principalName,
		Issuer:  googleIssuer,
		Subject: subject,
	})
	if err == nil {
		return sp, nil
	}
	if !isConflict(err) {
		return identityfederation.ServicePrincipal{}, err
	}
	page, err := c.IdentityFederation.ListServicePrincipals(ctx, nil, nil)
	if err != nil {
		return identityfederation.ServicePrincipal{}, fmt.Errorf("list after conflict: %w", err)
	}
	for _, p := range page.Principals {
		if p.Code == principalCode {
			return p, nil
		}
	}
	return identityfederation.ServicePrincipal{}, errors.New("registerServicePrincipal reported a conflict but no matching principal was found on the page")
}

func grantOrReuse(ctx context.Context, c *client.Client, principalID, permission string) error {
	_, err := c.Authorization.GrantPrincipalPermission(ctx, authorization.GrantPrincipalPermissionRequest{
		PrincipalId: principalID,
		Permission:  permission,
	})
	if err == nil || isConflict(err) {
		return nil
	}
	return err
}

func isConflict(err error) bool {
	return err != nil && (identityfederation.IsServicePrincipalConflict(err) || authorization.IsPrincipalGrantConflict(err))
}
