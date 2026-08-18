// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"time"
)

// ErrAssignmentNotFound / ErrInstanceAdminGrantNotFound: the target of a revoke wasn't an active
// grant — a redundant revoke (already revoked, or never granted) is a real error here, unlike
// InsertRoleAssignment's own insert-side idempotency (a repeat grant identical to an existing one is
// success). Revoking twice has no natural "already done" reading worth swallowing silently.
var (
	ErrAssignmentNotFound         = errors.New("authz: role assignment not found or already revoked")
	ErrInstanceAdminGrantNotFound = errors.New("authz: instance-admin grant not found or already revoked")
)

// Role is authz_roles — the grantable role catalog, read by M10.7's super-admin role-grant screen.
type Role struct {
	ID          string
	Code        string
	Name        string
	Description string
	IsBase      bool
}

// RoleAssignment is one active row of authz_role_assignments, as read back for a unit's role-grants
// screen (M10.7) — distinct from ActiveGrant (the PDP's per-decision input, which carries the role's
// resolved permission set rather than display fields).
type RoleAssignment struct {
	ID           string
	PersonID     string
	PersonName   string
	RoleID       string
	RoleCode     string
	TargetUnitID string
	Scope        Scope
	GrantedAt    time.Time
}

// InstanceAdminGrant is one active row of authz_instance_admins, as read back for the super-admin
// instance-admins screen (M10.7).
type InstanceAdminGrant struct {
	ID         string
	PersonID   string
	PersonName string
	GrantedAt  time.Time
}
