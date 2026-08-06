// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package coreintegration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/connector"
	"github.com/olehmushka/open-faith-map/internal/coreintegration"
	"github.com/palantir/pkg/bearertoken"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
)

// TestServiceClient_ResolvesAndEnforcesGrant proves the M1 service-principal MECHANISM end-to-end
// against a real, running go-oikumenea instance (docker compose up postgres oikumenea-migrate
// oikumenea-init-role oikumenea-app, then scripts/bootstrap-service-principal to register the
// principal this test authenticates as, and to grant it connector.read). Skipped unless pointed at a
// real instance — this is not a unit test, it makes a real network call and mints a real Google ID
// token.
//
// Calls ListConnectors (connector.read), not a religion.read-gated endpoint, even though
// religion.read is the grant OpenFaithMap's docs actually describe needing: every religion-module
// read endpoint currently uses RequireAnywhere, a person-shaped PEP path that denies any machine
// subject outright, regardless of grants — see scripts/bootstrap-service-principal's comment for the
// full finding. connector.read is the permission that is actually machine-reachable
// (RequireServiceOrPerson) today, so it is what proves the mechanism (Google ID token -> principal
// resolution -> grant lookup -> PDP allow -> real data) actually works.
//
//	docker run --rm --network open-faith-map_default -v "$PWD":/src -w /src golang:1.26-bookworm \
//	  env COREINTEGRATION_BASE_URL=https://oikumenea-app:8443 \
//	      GOOGLE_APPLICATION_CREDENTIALS=/src/var/<key>.json \
//	  go test ./internal/coreintegration/... -run TestServiceClient_ResolvesAndEnforcesGrant -v
func TestServiceClient_ResolvesAndEnforcesGrant(t *testing.T) {
	baseURL := os.Getenv("COREINTEGRATION_BASE_URL")
	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if baseURL == "" || credsFile == "" {
		t.Skip("set COREINTEGRATION_BASE_URL and GOOGLE_APPLICATION_CREDENTIALS to run against a live go-oikumenea instance")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	c, err := coreintegration.NewServiceClient(ctx, coreintegration.Config{
		BaseURL:            baseURL,
		CredentialsFile:    credsFile,
		Audience:           "openfaithmap-api",
		InsecureSkipVerify: true, // self-signed local-dev cert
	})
	if err != nil {
		t.Fatalf("NewServiceClient: %v", err)
	}

	// The unified SDK façade (client.Client) omits Connector — it wasn't updated when the
	// connector/wiring modules landed (a real, separate SDK gap) — so build it directly off the
	// exposed HTTPClient, per the façade's own "advanced/custom calls" escape hatch. A single Token()
	// call is enough for one test request (no need for the refreshing provider NewServiceClient uses
	// internally).
	ts, err := idtoken.NewTokenSource(ctx, "openfaithmap-api", option.WithCredentialsFile(credsFile)) //nolint:staticcheck
	if err != nil {
		t.Fatalf("build Google ID token source: %v", err)
	}
	tok, err := ts.Token()
	if err != nil {
		t.Fatalf("mint Google ID token: %v", err)
	}
	connectorClient := connector.NewConnectorServiceClientWithAuth(connector.NewConnectorServiceClient(c.HTTPClient), bearertoken.Token(tok.AccessToken))

	// GET /me/capabilities is deliberately person-only ("a machine/service subject gets an empty
	// set... cosmetic UI gating only" — internal/authorization/transport/service.go) — the wrong
	// endpoint to prove a principal's grant took effect; it always reports empty for a machine
	// subject regardless of real grants. ListConnectors is real connector.read-gated data on a
	// RequireServiceOrPerson path; the PDP re-decides it per request, so a successful response is the
	// actual proof the principal's (issuer, subject) resolved and its grant was enforced end-to-end.
	page, err := connectorClient.ListConnectors(ctx, nil, nil)
	if err != nil {
		t.Fatalf("ListConnectors (connector.read-gated): %v", err)
	}
	t.Logf("resolved as the service principal, connector.read enforced by the PDP: %d connectors", len(page.Connectors))
}
