// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package seed names the fixed, deterministic RIDs migrations/0015_core_seed.sql inserts
// (D-SeedBootstrap): the shared root unit, the canonical graph, and the three base roles. These
// values are identical across every deployment that runs the migration unmodified — that's the
// point (docs/architecture/decisions.md's D-SeedBootstrap: "structural RIDs deterministic, identity
// RIDs never"). Before M10.6 these were environment variables (REGISTRATION_ROOT_UNIT_ID,
// REGISTRATION_CONGREGATION_ADMIN_ROLE_ID) produced by manual bootstrap-script runs; owning the seed
// migration makes them Go constants instead — one of the three required env vars D-SeedBootstrap's
// own text says disappear.
//
// The constants below are fixed UUIDs — legitimate as test fixtures (hardcoding a value the
// deterministic seed migration guarantees is fine; that's not the same as production code doing
// it), and kept unchanged for the many *_integration_test.go files that already reference them
// directly. Production wiring (cmd/openfaithmap-api) uses Resolve instead (resolve.go): it looks
// these same rows up by their stable `code` column at boot, rather than hardcoding the UUID a
// second time in Go — directory_units and authz_roles both already carry a unique-while-active code
// index for exactly this.
package seed

const (
	// RootUnitID is the single shared root unit every top-level religious organization registers
	// under (migrations/0015_core_seed.sql, code='root').
	RootUnitID = "01989e26-ce01-8101-8301-0196a3b0bdca"

	// RegistrationOperatorRoleID is the "registration-operator" base role
	// (migrations/0015_core_seed.sql, code='registration-operator').
	RegistrationOperatorRoleID = "01989e26-ce01-8101-8201-0196a3b0bdca"
	// CongregationAdminRoleID is the "congregation-admin" base role, granted per-congregation at
	// registration-approval time (migrations/0015_core_seed.sql, code='congregation-admin').
	CongregationAdminRoleID = "01989e26-ce02-8101-8201-029daab7c4d1"
	// PlatformModeratorRoleID is the "platform-moderator" base role (migrations/0015_core_seed.sql,
	// code='platform-moderator').
	PlatformModeratorRoleID = "01989e26-ce03-8101-8201-03a4b1becbd8"
)
