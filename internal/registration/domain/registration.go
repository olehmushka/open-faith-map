// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package domain holds the registration module's plain Go types — no Conjure, no SQL, no HTTP
// (docs/architecture/overview.md's transport → application → domain → adapters layering).
package domain

import (
	"errors"
	"time"
)

type Status string

const (
	StatusPending Status = "PENDING"
	// StatusProvisioning marks a request that has passed the point of no return —
	// createChildOrg has produced a real go-oikumenea unit — but hasn't finished the remaining
	// approval writes yet. Approve resumes from here on retry rather than re-creating the org.
	StatusProvisioning Status = "PROVISIONING"
	StatusApproved     Status = "APPROVED"
	StatusRejected     Status = "REJECTED"
)

// Coordinate is WGS84 latitude/longitude.
type Coordinate struct {
	Latitude  float64
	Longitude float64
}

// Request is a congregation-registration request, pending a registration operator's decision.
type Request struct {
	ID                  string
	SubmittedByPersonID string
	TaxonID             string
	CongregationName    string
	CountryID           string
	AdminArea1          *string
	Locality            *string
	Street              *string
	HouseNumber         *string
	PostalCode          *string
	Coordinate          Coordinate
	Status              Status
	RejectionReason     *string
	DecidedByPersonID   *string
	DecidedAt           *time.Time
	CreatedUnitID       *string
	// JurisdictionUnitID is the operator's choice at approval time (M4.1, D-JurisdictionUnits) —
	// unset falls back to the configured root unit. A historical fact, not a live mirror: if the
	// congregation is later re-parented, this is NOT updated — the most recent VERIFIED
	// ReparentingJob's NewParentUnitID is the current source of truth for where it actually is.
	JurisdictionUnitID *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ReparentStatus is the state machine for moving an APPROVED request's congregation unit onto a
// different parent unit — see ReparentingJob.
type ReparentStatus string

const (
	ReparentPending        ReparentStatus = "PENDING"
	ReparentNewEdgeAdded   ReparentStatus = "NEW_EDGE_ADDED"
	ReparentOldEdgeRemoved ReparentStatus = "OLD_EDGE_REMOVED"
	ReparentVerified       ReparentStatus = "VERIFIED"
	ReparentFailed         ReparentStatus = "FAILED"
)

// ReparentingJob tracks one re-parenting attempt for an APPROVED request's congregation unit
// (M4.1). addEdge+removeEdge on go-oikumenea's canonical graph is two non-transactional calls, not
// one atomic move — this persists which one durably landed so a retry resumes rather than repeats.
type ReparentingJob struct {
	ID                    string
	RegistrationRequestID string
	CongregationUnitID    string
	OldParentUnitID       string
	NewParentUnitID       string
	Status                ReparentStatus
	PerformedByPersonID   string
	Error                 *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// SubmitInput is the caller-supplied shape for a new request (everything Request derives itself —
// id/status/timestamps — is excluded).
type SubmitInput struct {
	SubmittedByPersonID string
	TaxonID             string
	CongregationName    string
	CountryID           string
	AdminArea1          *string
	Locality            *string
	Street              *string
	HouseNumber         *string
	PostalCode          *string
	Coordinate          Coordinate
}

var (
	ErrNotFound      = errors.New("registration request not found")
	ErrNotPending    = errors.New("registration request is not pending")
	ErrExcluded      = errors.New("taxon is on the named exclusion list")
	ErrTaxonNotFound = errors.New("taxon not found")
	ErrNotApproved   = errors.New("registration request is not approved")
)

// ExcludedTaxonCodes is D-Exclusions' named, permanent denomination exclusion list
// (architecture/decisions.md), by go-oikumenea religion_taxa.code — verified against the actual
// seed data (go-oikumenea's deploy/religion-presets/gen-presets.py). Reopening this list means
// editing architecture/decisions.md's D-Exclusions block first (the same governance weight as the
// original FaithMap ADR-0001), then this constant to match.
var ExcludedTaxonCodes = map[string]bool{
	"russian_orthodox_church": true, // ROC — political exclusion
	"jehovahs_witnesses":      true, // doctrinal exclusion (non-Trinitarian)
	"lds_church":              true, // Mormons — doctrinal exclusion (non-Nicene Trinitarian)
}
