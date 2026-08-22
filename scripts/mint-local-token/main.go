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
// By default, prints only the signed token to stdout — no API calls, no side effects — so it
// composes with curl or any client:
//
//	go run ./scripts/mint-local-token -subject test-operator -email operator@example.com \
//	  -hmac-key "$DEV_ISSUER_HMAC_KEY"
//
// M11.3, D-SessionTracking: every authenticated request now also needs a valid, unrevoked
// X-Session-Id header, dev-issued tokens included (no issuer-based carve-out — confirmed decision,
// docs/milestones.md's M11.3 row). Pass -database-url and -account-id to have this tool ALSO insert
// the backing identity_sessions row and print the session id on a second stdout line — a deliberate,
// opt-in expansion of this tool's previously side-effect-free contract, since without it the minted
// token now 401s on every real endpoint:
//
//	go run ./scripts/mint-local-token -subject test-operator -email operator@example.com \
//	  -hmac-key "$DEV_ISSUER_HMAC_KEY" \
//	  -database-url "postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  -account-id "<the operator's identity_accounts.id>"
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	identityadapters "github.com/olehmushka/open-faith-map/internal/identity/adapters"
	"github.com/olehmushka/open-faith-map/internal/platform/devtoken"
)

func main() {
	email := flag.String("email", "", "email claim on the minted token (required)")
	subject := flag.String("subject", "", "the token's sub claim — must already resolve via a real identity_external_identities row (required)")
	hmacKey := flag.String("hmac-key", "", "must match the target server's own DEV_ISSUER_HMAC_KEY (required)")
	ttl := flag.Duration("ttl", 5*time.Minute, "token lifetime")
	databaseURL := flag.String("database-url", "", "if set (together with -account-id), also inserts a real identity_sessions row and prints its id — required for the token to pass M11.3's per-request session check")
	accountID := flag.String("account-id", "", "the identity_accounts.id -subject's external identity is linked to; required together with -database-url")
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
	if (*databaseURL == "") != (*accountID == "") {
		fmt.Fprintln(os.Stderr, "-database-url and -account-id must be given together")
		os.Exit(2)
	}

	tok, err := devtoken.Mint(*subject, *email, *ttl, *hmacKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mint token:", err)
		os.Exit(1)
	}
	fmt.Println(tok)

	if *databaseURL == "" {
		fmt.Fprintln(os.Stderr, "warning: no -database-url/-account-id given — this token has no backing identity_sessions row and will 401 on every real endpoint (M11.3, D-SessionTracking)")
		return
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect to database:", err)
		os.Exit(1)
	}
	defer pool.Close()

	sess, err := identityadapters.NewStore(pool).InsertSession(ctx, *accountID, devtoken.Issuer, "mint-local-token")
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert session:", err)
		os.Exit(1)
	}
	fmt.Println("X-Session-Id: " + sess.ID)
}
