// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"time"

	identityadapters "github.com/olehmushka/open-faith-map/internal/identity/adapters"
	identityapplication "github.com/olehmushka/open-faith-map/internal/identity/application"
	identitybootstrap "github.com/olehmushka/open-faith-map/internal/identity/bootstrap"
	identitymiddleware "github.com/olehmushka/open-faith-map/internal/identity/middleware"
	platformdb "github.com/olehmushka/open-faith-map/internal/platform/db"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
)

// registerIdentity builds M10.2's (D-DirectTokenVerification, D-SeedBootstrap) JWT-validation
// middleware and runs the boot-time first-admin seed. Additive only, matching M10.1's own
// precedent — the authenticator is built, Bind-wired, and unit-tested (internal/identity/middleware),
// but NOT attached via server.WithMiddleware here: identity_persons/authz_role_assignments are empty
// except the boot-seeded admin until M10.6 cuts the six consumer modules over from go-oikumenea's own
// Whoami/Authorize SDK calls. Wiring it live before then would 401 every existing authenticated flow.
// Registers no HTTP routes — identity has no Conjure surface until M10.7.
func registerIdentity(ctx context.Context, info witchcraft.InitInfo, deps *Deps) error {
	identityStore := identityadapters.NewStore(deps.Pool)
	identitySvc := identityapplication.NewService(identityStore)

	issuers := []identitymiddleware.IssuerConfig{
		{
			Issuer: "https://accounts.google.com", Type: identitymiddleware.IssuerOIDC,
			Audiences: []string{requireEnv("GOOGLE_OAUTH_CLIENT_ID")},
		},
	}
	if devHMACKey := os.Getenv("DEV_ISSUER_HMAC_KEY"); devHMACKey != "" {
		issuers = append(issuers, identitymiddleware.IssuerConfig{
			Issuer: identitymiddleware.ReservedLocalIssuer + ":dev", Type: identitymiddleware.IssuerHS256, HMACKey: devHMACKey,
		})
	}
	if err := identitymiddleware.GuardSymmetricIssuers(issuers, deps.Install.Environment); err != nil {
		return werror.WrapWithContextParams(ctx, err, "identity: symmetric issuer guard")
	}
	if err := identitymiddleware.GuardReservedIssuer(issuers); err != nil {
		return werror.WrapWithContextParams(ctx, err, "identity: reserved issuer guard")
	}
	if err := identitymiddleware.GuardIssuerAudience(issuers); err != nil {
		return werror.WrapWithContextParams(ctx, err, "identity: issuer audience guard")
	}

	jitEnabled := os.Getenv("IDENTITY_JIT_ENABLED") == "true"
	validator := identitymiddleware.NewValidator(identitymiddleware.Config{
		Issuers: issuers, ClockSkew: 60 * time.Second,
		JITEnabled: jitEnabled, JITClaim: os.Getenv("IDENTITY_JIT_CLAIM"), JITMatch: os.Getenv("IDENTITY_JIT_MATCH"),
	})
	authenticator := identitymiddleware.NewUnbound()
	authenticator.Bind(validator, identitySvc, identitySvc, jitEnabled)
	if err := authenticator.MustBeBound(); err != nil {
		return werror.WrapWithContextParams(ctx, err, "identity: authenticator not bound")
	}

	// Boot-time first-admin seed. Refused outside local/dev on an unset or placeholder value
	// (ValidateSeedForEnvironment), same fail-closed shape as GuardSymmetricIssuers; a genuinely
	// unset seed in local/dev is fine (a fresh checkout with no admin configured yet) and simply
	// skips seeding. Serialized under the boot-seed advisory lock so a restart or future multi-
	// replica boot can't race the seed.
	adminSeed := identitybootstrap.AdminSeed{
		Issuer:      os.Getenv("BOOTSTRAP_ADMIN_ISSUER"),
		Subject:     os.Getenv("BOOTSTRAP_ADMIN_SUBJECT"),
		Email:       os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
		DisplayName: os.Getenv("BOOTSTRAP_ADMIN_DISPLAY_NAME"),
	}
	if err := identitybootstrap.ValidateSeedForEnvironment(adminSeed, deps.Install.Environment); err != nil {
		return werror.WrapWithContextParams(ctx, err, "identity: bootstrap admin seed")
	}
	if adminSeed.Issuer != "" && adminSeed.Subject != "" {
		seedErr := platformdb.WithAdvisoryLock(ctx, deps.Pool, platformdb.LockBootSeed, func(ctx context.Context) error {
			_, err := identitybootstrap.Run(ctx, deps.Pool, adminSeed)
			return err
		})
		if seedErr != nil {
			return werror.WrapWithContextParams(ctx, seedErr, "identity: bootstrap admin seed run")
		}
	}
	return nil
}
