// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package domain holds the moderation module's plain Go types — no Conjure, no SQL, no HTTP
// (docs/architecture/overview.md's transport → application → domain → adapters layering).
package domain

import (
	"errors"
	"time"
)

// TargetKind/ReasonCode/QueueScope/ReportStatus/ActionKind/AppealStatus values match their Conjure
// enum values exactly (api/moderation.conjure.yml) and the DB CHECK constraints
// (migrations/0007_moderation.sql) verbatim — same convention every other module's domain package
// already follows, so no case conversion is ever needed crossing the transport/domain/adapters
// boundaries.
type TargetKind string

const (
	TargetSite         TargetKind = "SITE"
	TargetDocument     TargetKind = "DOCUMENT"
	TargetCongregation TargetKind = "CONGREGATION"
	TargetVouchingEdge TargetKind = "VOUCHING_EDGE"
)

// ReasonCode is deliberately missing a DOCTRINAL_CONCERN value — see ErrDoctrinalReasonNotAllowed.
type ReasonCode string

const (
	ReasonSpam                 ReasonCode = "SPAM"
	ReasonIncorrectInformation ReasonCode = "INCORRECT_INFORMATION"
	ReasonInappropriateContent ReasonCode = "INAPPROPRIATE_CONTENT"
	ReasonDuplicate            ReasonCode = "DUPLICATE"
	ReasonOther                ReasonCode = "OTHER"
)

type QueueScope string

const (
	ScopePlatform     QueueScope = "PLATFORM"
	ScopeCongregation QueueScope = "CONGREGATION"
	ScopeJurisdiction QueueScope = "JURISDICTION"
)

type ReportStatus string

const (
	ReportOpen      ReportStatus = "OPEN"
	ReportActioned  ReportStatus = "ACTIONED"
	ReportDismissed ReportStatus = "DISMISSED"
)

type ActionKind string

const (
	ActionHide        ActionKind = "HIDE"
	ActionSuspend     ActionKind = "SUSPEND"
	ActionArchive     ActionKind = "ARCHIVE"
	ActionWarnAdmin   ActionKind = "WARN_ADMIN"
	ActionRevokeVouch ActionKind = "REVOKE_VOUCH"
	ActionReverse     ActionKind = "REVERSE"
)

type AppealStatus string

const (
	AppealOpen       AppealStatus = "OPEN"
	AppealUpheld     AppealStatus = "UPHELD"
	AppealOverturned AppealStatus = "OVERTURNED"
)

type AppealDecision string

const (
	DecisionUphold   AppealDecision = "UPHELD"
	DecisionOverturn AppealDecision = "OVERTURNED"
)

// ReversalGracePeriod bounds how long after an action was taken it may still be reversed —
// moderation.md's "every action is reversible within its grace window" invariant, made concrete.
const ReversalGracePeriod = 72 * time.Hour

// Report is a flag raised by anyone (including an anonymous visitor) against a site, a piece of
// content, a congregation's claimed identity, or a vouching edge.
type Report struct {
	ID               string
	TargetKind       TargetKind
	TargetRef        string
	ReasonCode       ReasonCode
	Detail           *string
	ReporterPersonID *string
	QueueScope       QueueScope
	Status           ReportStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type FileReportInput struct {
	TargetKind TargetKind
	TargetRef  string
	ReasonCode ReasonCode
	Detail     *string
}

// Action is an immutable record of a moderator decision — append-only (reject_mutation()-guarded);
// a reversal is itself a new Action row, never an edit of the original.
//
// ReversesActionID is a real stored column, set only on a REVERSE row, pointing backward at what it
// reverses. ReversedByActionID is the opposite direction (does some later row reverse THIS one?) —
// deliberately NOT a stored column, since reject_mutation() forbids ever writing it onto an existing
// row after the fact. The store populates it at read time by looking for a REVERSE row whose
// ReversesActionID equals this row's ID (adapters.ActionStore.reversedBy).
type Action struct {
	ID                 string
	ReportID           *string
	ActionKind         ActionKind
	TargetKind         TargetKind
	TargetRef          string
	ActorPersonID      string
	Reason             string
	ReversesActionID   *string
	ReversedByActionID *string
	CreatedAt          time.Time
}

type TakeActionInput struct {
	ReportID      *string
	ActionKind    ActionKind
	TargetKind    TargetKind
	TargetRef     string
	ActorPersonID string
	Reason        string
	// ReversesActionID is set only when ActionKind = ActionReverse.
	ReversesActionID *string
}

// Appeal is a congregation admin's structured challenge to an action affecting them, routed to a
// different moderator than the one who took the original action.
type Appeal struct {
	ID                        string
	ActionID                  string
	CongregationAdminPersonID string
	Statement                 string
	AssignedModeratorPersonID *string
	Status                    AppealStatus
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

var (
	ErrReportNotFound            = errors.New("moderation report not found")
	ErrActionNotFound            = errors.New("moderation action not found")
	ErrAppealNotFound            = errors.New("moderation appeal not found")
	ErrForbidden                 = errors.New("caller does not hold platform-moderator's grant on the root unit")
	ErrDoctrinalReasonNotAllowed = errors.New("doctrinal disputes are not adjudicated: file under OTHER with free text, see D-Exclusions")
	ErrActionNotReversible       = errors.New("action's reversal grace window has passed, or it was already reversed")
	ErrAppealActorConflict       = errors.New("an appeal may not be decided by the moderator who took the original action")
	ErrTaxonNotFound             = errors.New("taxon not found")
	ErrInvalidPageToken          = errors.New("pageToken is malformed or tampered")
)

// PageCursor is the decoded shape of an opaque pageToken (M7, docs/modules/hardening.md) —
// encodes the (createdAt, id) of the last row in the previous page for keyset pagination.
// The wire encoding (base64/JSON) is a transport-only concern; domain only knows these two fields.
type PageCursor struct {
	CreatedAt time.Time
	ID        string
}
