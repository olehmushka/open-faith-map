// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"time"
)

// CongregationStatus is the verified/claimed overlay — congregationimport_congregation_status.
// Same shape as vouching_guarantor_status: a mutable projection keyed by an immutable
// go-oikumenea entity (the provisioned Unit). Proposal, not a settled design — D-CongregationImport
// flags the exact state machine as open for revision. ClaimedByPersonRID/ClaimedAt stay nil until a
// future claim flow exists (vouching.md already names this gap).
type CongregationStatus struct {
	CongregationUnitRID string
	SourceCode          string
	ImportCandidateID   *string
	VerifiedByPersonRID string
	VerifiedAt          time.Time
	ClaimedByPersonRID  *string
	ClaimedAt           *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
