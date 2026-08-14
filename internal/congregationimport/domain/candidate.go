// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package domain holds the congregationimport module's plain Go types — no Conjure, no SQL, no
// HTTP (docs/architecture/conventions.md's transport → application → domain → adapters layering).
package domain

import (
	"encoding/json"
	"errors"
	"time"
)

// Status values match migrations/0010_congregationimport.sql's CHECK constraint verbatim.
type Status string

const (
	StatusStaged            Status = "STAGED"
	StatusNeedsTaxonReview  Status = "NEEDS_TAXON_REVIEW"
	StatusNeedsGeocode      Status = "NEEDS_GEOCODE"
	StatusPossibleDuplicate Status = "POSSIBLE_DUPLICATE"
	StatusApproved          Status = "APPROVED"
	StatusProvisioning      Status = "PROVISIONING"
	StatusProvisioned       Status = "PROVISIONED"
	StatusRejected          Status = "REJECTED"
	StatusRejectedExcluded  Status = "REJECTED_EXCLUDED"
)

var (
	ErrCandidateNotFound = errors.New("congregationimport: candidate not found")
	ErrRunNotFound       = errors.New("congregationimport: run not found")
	// ErrRunParametersNotSupported is returned when the caller supplies a non-empty parameters map
	// for a connector that doesn't implement ConnectorConfigurable — fail loudly rather than
	// silently ignoring parameters the operator explicitly typed in.
	ErrRunParametersNotSupported = errors.New("congregationimport: this connector does not accept run parameters")
	ErrForbidden                 = errors.New("congregationimport: forbidden")
	ErrNotEditable               = errors.New("congregationimport: candidate is not in an editable status")
	ErrNotApprovable             = errors.New("congregationimport: candidate is not in an approvable status")
	ErrInvalidPageToken          = errors.New("congregationimport: pageToken is malformed or tampered")
	ErrAliasConflict             = errors.New("congregationimport: an alias with this (sourceCode, aliasText) already exists")
)

// PageCursor is the decoded shape of an opaque pageToken (mirrors moderation's M7 pagination
// fix, docs/modules/hardening.md) — encodes the (createdAt, id) of the last row in the previous
// page for keyset pagination. The wire encoding (base64/JSON) is a transport-only concern; domain
// only knows these two fields.
type PageCursor struct {
	CreatedAt time.Time
	ID        string
}

// RawRecord is one connector's fetch result for a single upstream record, before normalization.
// SourceRecordID is the source's own natural key — the idempotency anchor
// (congregationimport_candidates_source_key). RawPayload is kept verbatim for audit/reprocessing.
type RawRecord struct {
	SourceRecordID string
	RawPayload     json.RawMessage
	FetchedAt      time.Time
}

// NormalizedCandidate is a Connector's Normalize output — the common shape every source maps onto,
// regardless of how heterogeneous the source's own fields are (mirrors hermenea's own
// declared-source → canonical-shape pattern, docs/modules/import.md).
type NormalizedCandidate struct {
	Name      string
	TaxonHint *string
	// JurisdictionHint is a free-text hint naming the parish's superior jurisdiction (a diocese,
	// eparchy, synod, ...), for connectors whose source data carries one. Only meaningful for
	// denominations with a real institutional hierarchy (Catholic, Orthodox, Lutheran,
	// Anglican/Episcopal) — nil for everything else, by design (see matchJurisdiction).
	JurisdictionHint *string
	CountryHint      *string
	AdminArea1       *string
	Locality         *string
	Street           *string
	HouseNumber      *string
	PostalCode       *string
	Latitude         *float64
	Longitude        *float64
}

// Candidate is a staged row — congregationimport_candidates.
type Candidate struct {
	ID               string
	ImportRunID      *string
	SourceCode       string
	SourceRecordID   string
	Name             string
	TaxonHint        *string
	TaxonID          *string
	JurisdictionHint *string
	// SuggestedJurisdictionUnitID is an alias-matched suggestion only — never applied automatically.
	// D-JurisdictionUnits: jurisdiction is operator-assigned at approval time, never inferred.
	SuggestedJurisdictionUnitID  *string
	CountryID                    *string
	AdminArea1                   *string
	Locality                     *string
	Street                       *string
	HouseNumber                  *string
	PostalCode                   *string
	Latitude                     *float64
	Longitude                    *float64
	GeocodePrecision             *string
	RawPayload                   json.RawMessage
	Status                       Status
	PossibleDuplicateCandidateID *string
	PossibleDuplicateUnitID      *string
	RejectionReason              *string
	ReviewedByPersonRID          *string
	ReviewedAt                   *time.Time
	CreatedUnitID                *string
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
}

// EditInput carries an operator's corrections to a staged candidate before approval — scraped data
// is noisy by nature (docs/modules/congregationimport.md).
type EditInput struct {
	Name        *string
	TaxonID     *string
	CountryID   *string
	AdminArea1  *string
	Locality    *string
	Street      *string
	HouseNumber *string
	PostalCode  *string
	Latitude    *float64
	Longitude   *float64
}
