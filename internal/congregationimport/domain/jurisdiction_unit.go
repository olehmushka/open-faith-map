// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import "time"

// JurisdictionUnitStatus matches migrations/0013_congregationimport_jurisdiction_units.sql's CHECK
// constraint verbatim.
type JurisdictionUnitStatus string

const (
	JurisdictionUnitStatusPending JurisdictionUnitStatus = "PENDING"
	JurisdictionUnitStatusCreated JurisdictionUnitStatus = "CREATED"
	JurisdictionUnitStatusFailed  JurisdictionUnitStatus = "FAILED"
)

// JurisdictionUnitRecord is one row of congregationimport_jurisdiction_units — the idempotency
// anchor and resumable state machine for D-CatholicJurisdictionSync's automated, unattended
// jurisdiction-tier Unit creation (docs/architecture/decisions.md). (SourceCode, ExternalID) is the
// natural key: a re-run of the same JurisdictionSource recognizes an already-CREATED node and skips
// it, rather than calling createChildOrg a second time.
type JurisdictionUnitRecord struct {
	ID               string
	SourceCode       string
	ExternalID       string
	ParentExternalID *string
	Name             string
	OrgKindID        string
	Status           JurisdictionUnitStatus
	CreatedUnitID    *string
	FailureReason    *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
