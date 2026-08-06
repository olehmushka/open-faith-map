// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package coreintegration

import (
	"context"
	"fmt"

	oikumenea "github.com/olehmushka/go-oikumenea/clients/go"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
)

// NewServiceClient builds a go-oikumenea SDK client authenticated as OpenFaithMap's service
// principal (D-ServiceIdentities): every call mints a fresh Google-signed ID token for the
// configured GCP service account, targeted at cfg.Audience, via the IAM-backed self-signed-JWT
// exchange idtoken.NewTokenSource performs against Google's token endpoint — no OAuth2
// client-credentials grant, no Keycloak (deploy/oikumenea-install.yml's header comment has the full
// rationale). go-oikumenea validates the resulting token as an ordinary Google OIDC token and
// resolves it to the principal registered by scripts/bootstrap-service-principal, purely by its
// (issuer, subject) — see docs/modules/core-integration.md.
func NewServiceClient(ctx context.Context, cfg Config) (*oikumenea.Client, error) {
	// WithCredentialsFile is deprecated in favor of type-specific options because of the risk of
	// loading an untrusted/attacker-supplied credential file; cfg.CredentialsFile is always an
	// operator-supplied local path (deploy config), never external input, so that risk doesn't apply
	// here.
	ts, err := idtoken.NewTokenSource(ctx, cfg.Audience, option.WithCredentialsFile(cfg.CredentialsFile)) //nolint:staticcheck
	if err != nil {
		return nil, fmt.Errorf("coreintegration: build Google ID token source: %w", err)
	}

	opts := []oikumenea.Option{}
	if cfg.InsecureSkipVerify {
		opts = append(opts, oikumenea.WithInsecureSkipVerify())
	}

	c, err := oikumenea.NewWithTokenProvider(cfg.BaseURL, tokenProvider(ts), opts...)
	if err != nil {
		return nil, fmt.Errorf("coreintegration: dial go-oikumenea: %w", err)
	}
	return c, nil
}

// tokenProvider adapts an oauth2.TokenSource (Google ID tokens) to conjure-go-runtime's
// httpclient.TokenProvider (func(context.Context) (string, error)). The ID token JWT is carried in
// the oauth2.Token's AccessToken field — the documented idtoken.NewTokenSource convention.
func tokenProvider(ts oauth2.TokenSource) func(context.Context) (string, error) {
	return func(context.Context) (string, error) {
		tok, err := ts.Token()
		if err != nil {
			return "", fmt.Errorf("coreintegration: mint Google ID token: %w", err)
		}
		return tok.AccessToken, nil
	}
}
