// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package config defines openfaithmap-api's install (static, var/conf/install.yml) and runtime
// (hot-reloadable, var/conf/runtime.yml) configuration types. Both embed witchcraft-go-server's
// base config types, matching go-oikumenea's own internal/platform/config convention (see
// docs/architecture/conventions.md).
//
// M1 and M2 originally added DATABASE_URL and GOOGLE_OAUTH_CLIENT_ID as requireEnv reads,
// bypassing this type entirely. U12 (resolved 2026-08-26) folded both into Install proper — the
// go-oikumenea-era settings this comment used to list (OIKUMENEA_BASE_URL,
// REGISTRATION_ROOT_UNIT_ID, REGISTRATION_CONGREGATION_ADMIN_ROLE_ID,
// OIKUMENEA_INSECURE_SKIP_VERIFY) are gone entirely since M10 absorbed the core in-process, so
// there was nothing left to fold in beyond these two.
//
// Environment (M10.2) set the precedent this follows: a real, schema-validated Install field
// rather than an env var, with an ECV-encryption path for the secret among them (DatabaseURL) in a
// real deployment — see var/conf/install.yml's own comments.
package config

import (
	wconfig "github.com/palantir/witchcraft-go-server/v2/config"
)

// Install is the static, operator-supplied configuration (var/conf/install.yml).
type Install struct {
	wconfig.Install `yaml:",inline"`

	// Environment gates GuardSymmetricIssuers (internal/identity/middleware): HS256 issuers are
	// refused at boot unless this is exactly "local" or "dev" — any other value, including empty or
	// unrecognized, fails closed. One of: local | dev | staging | prod.
	Environment string `yaml:"environment"`

	// DatabaseURL is the Postgres connection string (initServer dials it directly into a pgxpool).
	// A secret in any real deployment — ECV-encrypt this field, don't pass it as a plain env var.
	DatabaseURL string `yaml:"database-url"`

	// GoogleOAuthClientID is the sole accepted audience for the Google OIDC issuer
	// (registerIdentity). Not a secret — OAuth client IDs are public by design — but schema-
	// validated like every other install setting rather than a bare requireEnv read.
	GoogleOAuthClientID string `yaml:"google-oauth-client-id"`
}

// Runtime is the hot-reloadable configuration (var/conf/runtime.yml).
type Runtime struct {
	wconfig.Runtime `yaml:",inline"`
}
