// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package config defines openfaithmap-api's install (static, var/conf/install.yml) and runtime
// (hot-reloadable, var/conf/runtime.yml) configuration types. Both embed witchcraft-go-server's
// base config types, matching go-oikumenea's own internal/platform/config convention (see
// docs/architecture/conventions.md). Empty today — no go-oikumenea integration, database, or IDP
// wiring exists yet (that lands at M1+, see docs/milestones.md).
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
