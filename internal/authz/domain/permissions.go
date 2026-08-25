// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"slices"
)

// Permission is a closed vocabulary of action codes (D-InProcessAuthz): a permission no code path
// checks is dead weight, and a code path checking a permission nobody can grant is a silent hole.
// Keeping the catalog in Go means the compiler is the integrity check, not a DB row.
//
// Trimmed relative to go-oikumenea's own ~140-entry catalog to exactly what this repo seeds
// (migrations/0022_core_seed.sql's three base roles) plus the minimal instance-admin-plane set —
// extend opportunistically as later M10.x milestones need new codes, not ahead of them.
type Permission string

const (
	// Unit-scoped — grantable via a role assignment (migrations/0022_core_seed.sql).
	PermReligionOrgManage Permission = "religionorg.manage"
	PermSiteManage        Permission = "site.manage"
	PermScheduleManage    Permission = "schedule.manage"
	PermAssignmentGrant   Permission = "assignment.grant"
	PermAssignmentRevoke  Permission = "assignment.revoke"
	PermPersonCreate      Permission = "person.create"
	PermPersonUpdate      Permission = "person.update"
	PermPersonRead        Permission = "person.read"
	PermMembershipCreate  Permission = "membership.create"
	PermMembershipUpdate  Permission = "membership.update"
	PermMembershipRead    Permission = "membership.read"
	PermPositionCreate    Permission = "position.create"
	PermPositionUpdate    Permission = "position.update"
	PermPositionRead      Permission = "position.read"
	PermUnitRead          Permission = "unit.read"
	PermUnitLifecycle     Permission = "unit.lifecycle"
	PermUnitEdgesManage   Permission = "unit.edges.manage"
	PermReligionRead      Permission = "religion.read"
	PermLocationCreate    Permission = "location.create"
	PermRoleRead          Permission = "role.read"
	// PermModerationStanding is platform-moderator's own identity marker (M12.0), split out of
	// unit.lifecycle: that code was reused for this purpose only because go-oikumenea's permission
	// catalog was closed pre-port (D-PlatformModerator's M5 addendum) — a constraint that no longer
	// applies now that this catalog is this repo's own Go code (D-InProcessAuthz). Splitting it frees
	// unit.lifecycle for M12.1's real setUnitState/deleteUnit endpoints without instantly handing
	// every platform-moderator archive/suspend/delete power over every unit under root as a side
	// effect of two unrelated features sharing one code.
	PermModerationStanding Permission = "moderation.standing"

	// Instance-scope — satisfiable only on the instance-admin plane (PDP.Decide step 2), never via a
	// role assignment. Minimal set: granting/revoking the plane itself, and managing the custom-role
	// catalog those grants are made of (needed once M10.8's super-admin screens exist).
	PermInstanceAdminManage Permission = "instance.admin.manage"
	PermRoleCreate          Permission = "role.create"
	PermRoleUpdate          Permission = "role.update"
	PermRoleDelete          Permission = "role.delete"
)

// instanceScope is the closed set of instance-plane-only permissions. Everything in catalog but not
// here is unit-scoped by construction — there is no separate positive unit-scoped set.
var instanceScope = map[Permission]struct{}{
	PermInstanceAdminManage: {},
	PermRoleCreate:          {},
	PermRoleUpdate:          {},
	PermRoleDelete:          {},
}

// catalog is the full closed vocabulary.
var catalog = map[Permission]struct{}{
	PermReligionOrgManage: {}, PermSiteManage: {}, PermScheduleManage: {},
	PermAssignmentGrant: {}, PermAssignmentRevoke: {},
	PermPersonCreate: {}, PermPersonUpdate: {}, PermPersonRead: {},
	PermMembershipCreate: {}, PermMembershipUpdate: {}, PermMembershipRead: {},
	PermPositionCreate: {}, PermPositionUpdate: {}, PermPositionRead: {},
	PermUnitRead: {}, PermUnitLifecycle: {}, PermUnitEdgesManage: {},
	PermReligionRead: {}, PermLocationCreate: {}, PermRoleRead: {},
	PermModerationStanding:  {},
	PermInstanceAdminManage: {}, PermRoleCreate: {}, PermRoleUpdate: {}, PermRoleDelete: {},
}

// IsKnownPermission reports whether code is in the closed catalog.
func IsKnownPermission(code string) bool {
	_, ok := catalog[Permission(code)]
	return ok
}

// IsInstanceScope reports whether code is an instance-plane-only permission. The PDP satisfies these
// only via the instance-admin plane (authz_instance_admins), never via a role assignment.
func IsInstanceScope(code string) bool {
	_, ok := instanceScope[Permission(code)]
	return ok
}

// UnitScopedPermissionCodes returns every catalog code except the instance-scope ones (M11.9's
// listPermissionCatalog: an API key's allowlist can only ever be satisfied through the unit-scoped
// PDP path, since RequireInstanceAdmin hard-denies every API-key-authenticated subject regardless of
// allowlist contents — offering instance-scope codes in the creation picker would be misleading).
func UnitScopedPermissionCodes() []string {
	codes := make([]string, 0, len(catalog)-len(instanceScope))
	for p := range catalog {
		if _, ok := instanceScope[p]; !ok {
			codes = append(codes, string(p))
		}
	}
	slices.Sort(codes)
	return codes
}
