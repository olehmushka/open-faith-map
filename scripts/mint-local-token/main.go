// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Command mint-local-token is a local-dev-only operator tool: mints an HS256 token for go-oikumenea's
// local-dev issuer (deploy/oikumenea-install.yml), for an ARBITRARY (email, subject) pair rather than
// the fixed "local-admin" bootstrap identity scripts/bootstrap-admin-person, bootstrap-service-principal,
// and bootstrap-registration-org each mint for themselves.
//
// This exists to authenticate as a second, genuinely non-admin test person headlessly — for example
// scripts/bootstrap-admin-person's own moderator-test@example.com shell account — so that a
// "non-moderator refused, moderator allowed" or "guarantor with standing vs. without" proof can be run
// against a real docker compose stack without a real browser Google OAuth session. go-oikumenea's
// HS256 validator (../go-oikumenea's internal/identityfederation/middleware/validator.go) has no
// subject allowlist: any (issuer, subject) signed with the known local-dev key is accepted, and JIT
// (idp.jit, match: account-email) links it onto an EXISTING person's account by email, never creating
// one — so -email here must already have a shell account (run scripts/bootstrap-admin-person first).
// Instance-admin status is a separate DB fact keyed by person_id, not by token subject, so the person
// this resolves to is a real, non-admin, PDP-checked identity distinct from local-admin.
//
// Prints only the signed token to stdout (no API calls, no side effects) so it composes with curl or
// any client — e.g.:
//
//	go run ./scripts/mint-local-token -email moderator-test@example.com
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Matches deploy/oikumenea-install.yml exactly — the same constants
// scripts/bootstrap-admin-person/bootstrap-service-principal/bootstrap-registration-org already use,
// local-dev only, safe to keep in source (see deploy/oikumenea-install.yml's own comment: anyone who
// knows this key can mint a token for any subject on this issuer, fine for a throwaway local stack).
const (
	localIssuer  = "https://local-dev.oikumenea.test"
	localAud     = "oikumenea"
	localHMACKey = "local-dev-insecure-signing-key-change-me"
)

func main() {
	email := flag.String("email", "", "email of an existing shell account to JIT-link onto (required — see scripts/bootstrap-admin-person)")
	subject := flag.String("subject", "", "the token's sub claim (defaults to \"test-<email>\" — any value not already bound to another identity)")
	ttl := flag.Duration("ttl", 5*time.Minute, "token lifetime")
	flag.Parse()

	if *email == "" {
		fmt.Fprintln(os.Stderr, "missing required -email")
		os.Exit(2)
	}
	if *subject == "" {
		*subject = "test-" + *email
	}

	tok, err := mintToken(*subject, *email, *ttl)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mint token:", err)
		os.Exit(1)
	}
	fmt.Println(tok)
}

func mintToken(subject, email string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":            localIssuer,
		"sub":            subject,
		"aud":            localAud,
		"iat":            now.Unix(),
		"exp":            now.Add(ttl).Unix(),
		"email":          email,
		"email_verified": true,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(localHMACKey))
}
