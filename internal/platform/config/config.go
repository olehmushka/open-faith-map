// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package config defines openfaithmap-api's install (static, var/conf/install.yml) and runtime
// (hot-reloadable, var/conf/runtime.yml) configuration types. Both embed witchcraft-go-server's
// base config types, matching go-oikumenea's own internal/platform/config convention (see
// docs/architecture/conventions.md).
//
// Still empty of OpenFaithMap-specific fields, but no longer because there is nothing to configure:
// M1 and M2 added real settings (DATABASE_URL, OIKUMENEA_BASE_URL, REGISTRATION_ROOT_UNIT_ID,
// REGISTRATION_CONGREGATION_ADMIN_ROLE_ID, OIKUMENEA_INSECURE_SKIP_VERIFY) and cmd/openfaithmap-api
// reads every one of them straight from the environment via requireEnv, bypassing this type. That
// deviates from the witchcraft install-config convention this package exists to follow — env vars
// get no schema, no validation, and no ECV encryption path for the secrets among them. Worth
// folding into Install when a third module needs configuration; see docs/architecture/conventions.md.
package config

import (
	wconfig "github.com/palantir/witchcraft-go-server/v2/config"
)

// Install is the static, operator-supplied configuration (var/conf/install.yml).
type Install struct {
	wconfig.Install `yaml:",inline"`
}

// Runtime is the hot-reloadable configuration (var/conf/runtime.yml).
type Runtime struct {
	wconfig.Runtime `yaml:",inline"`
}
