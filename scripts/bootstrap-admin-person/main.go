// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Command bootstrap-admin-person is a one-time, local-dev-only operator tool (M1): go-oikumenea's
// JIT provisioning is link-on-match only — "on no match, reject. JIT never creates a person."
// (go-oikumenea's D-JIT) — so a brand-new instance has no Person/Account for a human's Google
// identity to link onto on first login. This mints a token for go-oikumenea's HS256 bootstrap-admin
// identity (configured in deploy/oikumenea-install.yml, never used for anything else — the same one
// scripts/bootstrap-service-principal uses), then creates a Person and a login-less "shell" Account
// carrying the given email, so that person's first Google sign-in JIT-links onto it (the
// `account-email` match arm, already enabled in deploy/oikumenea-install.yml's `idp.jit`).
//
// Run once per fresh go-oikumenea instance, on the compose network (oikumenea-app publishes no host
// port — D-HeadlessTopology):
//
//	docker run --rm --network open-faith-map_default -v "$PWD":/src -w /src golang:1.26-bookworm \
//	  go run ./scripts/bootstrap-admin-person -email you@example.com -display-name "Your Name"
//
// Idempotent in effect (a re-run reuses the existing person/account on conflict), matching
// scripts/bootstrap-service-principal's registerOrReuse pattern.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	client "github.com/olehmushka/go-oikumenea/clients/go"
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/identityfederation"
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/person"
	"github.com/palantir/pkg/bearertoken"
)

// Bootstrap-admin identity + HS256 key, matching deploy/oikumenea-install.yml exactly (same
// constants as scripts/bootstrap-service-principal — local-dev only, safe to keep in source).
const (
	bootstrapIssuer  = "https://local-dev.oikumenea.test"
	bootstrapSubject = "local-admin"
	bootstrapAud     = "oikumenea"
	bootstrapHMACKey = "local-dev-insecure-signing-key-change-me"
)

func main() {
	baseURL := flag.String("base-url", "https://oikumenea-app:8443", "go-oikumenea base URL (compose-internal)")
	email := flag.String("email", "", "the Google account email to pre-link (required)")
	displayName := flag.String("display-name", "", "display name for the Person record (required)")
	code := flag.String("code", "", "stable person.code, used to find this person again on a re-run (defaults to \"admin-<email>\")")
	insecure := flag.Bool("insecure-skip-verify", true, "skip TLS verification (self-signed local-dev cert)")
	flag.Parse()

	if *email == "" || *displayName == "" {
		fmt.Fprintln(os.Stderr, "missing required -email and -display-name")
		os.Exit(2)
	}
	if *code == "" {
		*code = "admin-" + *email
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

	p, err := createOrReusePerson(ctx, c, *code, *displayName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create person:", err)
		os.Exit(1)
	}
	fmt.Printf("person: id=%s code=%s displayName=%s\n", p.Id, *code, p.DisplayName)

	acct, err := createOrReuseShellAccount(ctx, c, p.Id, *email)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create shell account:", err)
		os.Exit(1)
	}
	fmt.Printf("account: id=%s email=%s\n", acct.Id, *email)

	fmt.Println("done — the next Google sign-in with this email will JIT-link onto this account")
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

// createOrReusePerson creates the Person, or finds the existing one by code on a PersonConflict —
// a fresh instance creates, a re-run against an already-bootstrapped instance reuses it.
func createOrReusePerson(ctx context.Context, c *client.Client, code, displayName string) (person.Person, error) {
	p, err := c.Person.CreatePerson(ctx, person.CreatePersonRequest{
		Code:        &code,
		DisplayName: displayName,
	})
	if err == nil {
		return p, nil
	}
	if !person.IsPersonConflict(err) {
		return person.Person{}, err
	}
	page, err := c.Person.ListPersons(ctx, nil, nil, &code, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return person.Person{}, fmt.Errorf("list after conflict: %w", err)
	}
	for _, existing := range page.Persons {
		if existing.Code != nil && *existing.Code == code {
			return existing, nil
		}
	}
	return person.Person{}, fmt.Errorf("createPerson reported a conflict but no person with code %q was found", code)
}

// createOrReuseShellAccount creates a login-less account carrying email, or fetches the person's
// existing active account on an AccountConflict (the person already has one — a re-run).
func createOrReuseShellAccount(ctx context.Context, c *client.Client, personID, email string) (identityfederation.Account, error) {
	acct, err := c.IdentityFederation.CreateAccount(ctx, identityfederation.CreateAccountRequest{
		PersonId: personID,
		Email:    &email,
		// Identity omitted: this is a login-less shell — the person's first Google sign-in attaches
		// the (issuer, subject), not this script (D-JIT's account-email arm).
	})
	if err == nil {
		return acct, nil
	}
	if !identityfederation.IsAccountConflict(err) {
		return identityfederation.Account{}, err
	}
	return c.IdentityFederation.GetAccountByPerson(ctx, personID)
}
