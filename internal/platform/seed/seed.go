// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package seed names the fixed, deterministic RIDs migrations/0022_core_seed.sql inserts
// (D-SeedBootstrap): the shared root unit, the canonical graph, and the three base roles. These
// values are identical across every deployment that runs the migration unmodified — that's the
// point (docs/architecture/decisions.md's D-SeedBootstrap: "structural RIDs deterministic, identity
// RIDs never"). Before M10.6 these were environment variables (REGISTRATION_ROOT_UNIT_ID,
// REGISTRATION_CONGREGATION_ADMIN_ROLE_ID) produced by manual bootstrap-script runs; owning the seed
// migration makes them Go constants instead — one of the three required env vars D-SeedBootstrap's
// own text says disappear.
package seed

const (
	// RootUnitID is the single shared root unit every top-level religious organization registers
	// under (migrations/0022_core_seed.sql:24).
	RootUnitID = "01989e26-ce01-8101-8301-0196a3b0bdca"

	// RegistrationOperatorRoleID is the "registration-operator" base role
	// (migrations/0022_core_seed.sql:35-36).
	RegistrationOperatorRoleID = "01989e26-ce01-8101-8201-0196a3b0bdca"
	// CongregationAdminRoleID is the "congregation-admin" base role, granted per-congregation at
	// registration-approval time (migrations/0022_core_seed.sql:37-38).
	CongregationAdminRoleID = "01989e26-ce02-8101-8201-029daab7c4d1"
	// PlatformModeratorRoleID is the "platform-moderator" base role (migrations/0022_core_seed.sql:39-40).
	PlatformModeratorRoleID = "01989e26-ce03-8101-8201-03a4b1becbd8"
)
