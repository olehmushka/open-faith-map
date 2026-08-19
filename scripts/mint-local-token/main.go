// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Command mint-local-token is a local-dev-only operator tool: mints an HS256 token for
// openfaithmap-api's own synthetic local-dev issuer (internal/platform/devtoken), for an ARBITRARY
// (email, subject) pair. Target server must be booted with DEV_ISSUER_HMAC_KEY set to the same
// -hmac-key value (docker-compose.override.yml, never committed — D-DirectTokenVerification's
// amendment ships the committed config with this unset).
//
// This exists to authenticate as a test person headlessly — e.g. a real congregation-admin/
// registration-operator/platform-moderator/instance-admin grant, or a plain authenticated-but-
// ungranted person — so a "denied for X, allowed for Y" proof can be run against a real docker
// compose stack without a real browser Google OAuth session. The token alone resolves to nothing:
// -subject must already have a matching identity_external_identities row (issuer=devtoken.Issuer)
// pointing at a real identity_persons row, either via IDENTITY_JIT_ENABLED's link-on-match or a row
// inserted directly — the same requirement a real IdP's JIT path has.
//
// Prints only the signed token to stdout (no API calls, no side effects) so it composes with curl or
// any client — e.g.:
//
//	go run ./scripts/mint-local-token -subject test-operator -email operator@example.com \
//	  -hmac-key "$DEV_ISSUER_HMAC_KEY"
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/olehmushka/open-faith-map/internal/platform/devtoken"
)

func main() {
	email := flag.String("email", "", "email claim on the minted token (required)")
	subject := flag.String("subject", "", "the token's sub claim — must already resolve via a real identity_external_identities row (required)")
	hmacKey := flag.String("hmac-key", "", "must match the target server's own DEV_ISSUER_HMAC_KEY (required)")
	ttl := flag.Duration("ttl", 5*time.Minute, "token lifetime")
	flag.Parse()

	if *email == "" {
		fmt.Fprintln(os.Stderr, "missing required -email")
		os.Exit(2)
	}
	if *subject == "" {
		fmt.Fprintln(os.Stderr, "missing required -subject")
		os.Exit(2)
	}
	if *hmacKey == "" {
		fmt.Fprintln(os.Stderr, "missing required -hmac-key")
		os.Exit(2)
	}

	tok, err := devtoken.Mint(*subject, *email, *ttl, *hmacKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mint token:", err)
		os.Exit(1)
	}
	fmt.Println(tok)
}
