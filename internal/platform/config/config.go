// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package config defines openfaithmap-api's install (static, var/conf/install.yml) and runtime
// (hot-reloadable, var/conf/runtime.yml) configuration types. Both embed witchcraft-go-server's
// base config types, matching go-oikumenea's own internal/platform/config convention (see
// docs/architecture/conventions.md).
//
// M1 and M2 added real settings (DATABASE_URL, OIKUMENEA_BASE_URL, REGISTRATION_ROOT_UNIT_ID,
// REGISTRATION_CONGREGATION_ADMIN_ROLE_ID, OIKUMENEA_INSECURE_SKIP_VERIFY) but cmd/openfaithmap-api
// reads every one of them straight from the environment via requireEnv, bypassing this type — env
// vars get no schema, no validation, and no ECV encryption path for the secrets among them. Worth
// folding the rest into Install when a third module needs configuration; see
// docs/architecture/conventions.md.
//
// Environment (M10.2) is the one deliberate exception: D-DirectTokenVerification's amendment makes
// it the sole input to GuardSymmetricIssuers, and requires it be a real, schema-validated field
// rather than an env var — "never derive dev-ness from the presence of a secret" is the whole point,
// and a requireEnv read next to a HMAC-key env var would invite exactly that shortcut.
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
}

// Runtime is the hot-reloadable configuration (var/conf/runtime.yml).
type Runtime struct {
	wconfig.Runtime `yaml:",inline"`
}
