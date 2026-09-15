// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

// M15 (DS-OFM-15): action/target constants for this module's internal/auditlog.Record calls.
//
// Both literals deliberately match a same-named constant elsewhere: auditActionCreateChildOrg
// matches internal/congregationimport/application's own constant (both modules call the identical
// religion.CreateChildOrg primitive), and auditActionGrantUnitRole matches
// internal/core/application's own constant (both call the identical authz.Service.GrantUnitRole
// primitive) — one shared action-name vocabulary per underlying primitive, even though each module
// owns its own private const block, so the audit viewer can filter by action across modules.
const (
	auditActionCreateChildOrg = "CREATE_CHILD_ORG"
	auditActionGrantUnitRole  = "GRANT_UNIT_ROLE"

	auditTargetUnit           = "UNIT"
	auditTargetRoleAssignment = "ROLE_ASSIGNMENT"
)
