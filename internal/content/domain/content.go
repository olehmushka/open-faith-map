// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package domain holds the content module's plain Go types — no Conjure, no SQL, no HTTP
// (docs/architecture/overview.md's transport → application → domain → adapters layering).
package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// DocumentKind/DocumentState/BlockTypeStatus values match their Conjure enum values exactly
// (api/content.conjure.yml) and the DB CHECK constraints (migrations/0004_content.sql) verbatim —
// same convention internal/registration/domain.Status already uses, so no case conversion is ever
// needed crossing the transport/domain/adapters boundaries.
type DocumentKind string

const (
	KindPage  DocumentKind = "PAGE"
	KindPost  DocumentKind = "POST"
	KindEvent DocumentKind = "EVENT"
)

type DocumentState string

const (
	StateDraft     DocumentState = "DRAFT"
	StatePublished DocumentState = "PUBLISHED"
	StateUnlisted  DocumentState = "UNLISTED"
)

type TransitionAction string

const (
	ActionPublish       TransitionAction = "PUBLISH"
	ActionUnlist        TransitionAction = "UNLIST"
	ActionRevertToDraft TransitionAction = "REVERT_TO_DRAFT"
)

type BlockTypeStatus string

const (
	BlockTypeActive  BlockTypeStatus = "ACTIVE"
	BlockTypeRetired BlockTypeStatus = "RETIRED"
)

// Site is the root aggregate for one congregation's web presence (docs/modules/content.md).
type Site struct {
	ID                  string
	CongregationUnitRID string
	Slug                string
	Theme               json.RawMessage
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type CreateSiteInput struct {
	CongregationUnitRID string
	Slug                string
}

// Document is a Page, Post, or Event.
type Document struct {
	ID                   string
	SiteID               string
	Kind                 DocumentKind
	TranslationGroupID   string
	Locale               string
	ParentDocumentID     *string
	Slug                 string
	State                DocumentState
	PublishedAt          *time.Time
	EventStartsAt        *time.Time
	EventEndsAt          *time.Time
	EventRecurrenceRRule *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	// DraftRevisionID/PublishedRevisionID (M14.6, D-ContentRevisions) point into
	// content_document_revisions. DraftRevisionID is set once, at document creation, and never
	// changes — GetBlocks/PutBlocks always read/write that one row in place. PublishedRevisionID
	// is nil until the document's first publish, then repointed at a fresh immutable checkpoint
	// row on every subsequent publish.
	DraftRevisionID     *string
	PublishedRevisionID *string
}

// DocumentRevision is one row of a document's revision history (M14.6). Two roles share this
// shape, distinguished only by which of Document's two pointers references a given row: the one
// row a document's DraftRevisionID points at is mutated in place by every save; every other row is
// an immutable checkpoint created at publish time. Data is a full ordered blocks snapshot — the
// same shape Service.PutBlocks already validates — never a second copy of individual block rows.
type DocumentRevision struct {
	ID             string
	DocumentID     string
	RevisionNo     int
	Data           json.RawMessage
	AuthorPersonID *string
	CreatedAt      time.Time
	Label          *string
}

type CreateDocumentInput struct {
	Kind                 DocumentKind
	TranslationGroupID   *string
	Locale               string
	ParentDocumentID     *string
	Slug                 string
	EventStartsAt        *time.Time
	EventEndsAt          *time.Time
	EventRecurrenceRRule *string
}

type UpdateDocumentInput struct {
	Slug             *string
	ParentDocumentID *string
	ClearParent      bool
}

// Block is a typed, schema-validated content unit within a document.
type Block struct {
	ID            string
	DocumentID    string
	BlockTypeID   string
	BlockTypeCode string
	Position      int
	Data          json.RawMessage
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type BlockInput struct {
	BlockTypeCode string
	Position      int
	Data          json.RawMessage
}

// BlockType is a catalog row naming a valid block shape (id/code/json_schema/status/sort_order).
type BlockType struct {
	ID         string
	Code       string
	Name       string
	JSONSchema json.RawMessage
	UISchema   json.RawMessage
	Status     BlockTypeStatus
	SortOrder  int
}

// NavItem is one entry in a site's hand-built nav menu (M14.10) — independent of
// Document.ParentDocumentID (M14.0 replaced the page-tree-derived-nav assumption with a curated
// menu). Exactly one of TargetDocumentID/TargetURL is ever set.
type NavItem struct {
	ID               string
	SiteID           string
	Label            string
	TargetDocumentID *string
	TargetURL        *string
	SortOrder        int
}

type NavItemInput struct {
	Label            string
	TargetDocumentID *string
	TargetURL        *string
	SortOrder        int
}

// PublicNavItem is a NavItem as read by openfaithmap-web: Href is already resolved (an internal
// target's href walks its real ancestor chain into a hierarchical path server-side, so the caller
// never re-derives that). Items whose target document is missing or DRAFT are omitted upstream,
// never represented here as a broken link.
type PublicNavItem struct {
	Label    string
	Href     string
	External bool
}

var (
	ErrSiteNotFound       = errors.New("content site not found")
	ErrDocumentNotFound   = errors.New("content document not found")
	ErrForbidden          = errors.New("caller does not hold content.manage on this site's congregation unit")
	ErrEventMissingStart  = errors.New("kind=EVENT requires eventStartsAt to be set")
	ErrParentTooDeep      = errors.New("parent document chain exceeds 3 levels")
	ErrInvalidTransition  = errors.New("invalid document state transition")
	ErrBlockTypeNotFound  = errors.New("block type not found or retired")
	ErrBlockDataInvalid   = errors.New("block data failed json schema validation")
	ErrBlockUrlNotAllowed = errors.New("block field failed URL scheme/embed host allowlist")
	ErrRevisionNotFound   = errors.New("content document revision not found")
	// ErrPreviewTokenInvalid covers a missing, malformed, expired, or wrong-site preview token alike
	// (M14.7) — deliberately one sentinel for every case, so a caller probing the preview endpoints
	// learns nothing about which check failed.
	ErrPreviewTokenInvalid = errors.New("preview token missing, invalid, expired, or scoped to a different site")
)

// SlugTakenError carries U5's resolution: an admin-chosen slug, probed for uniqueness at write
// time, with a real typed error on collision — no silent suffixing. Scope is "site" or "document".
type SlugTakenError struct {
	Slug  string
	Scope string
}

func (e *SlugTakenError) Error() string {
	return fmt.Sprintf("slug %q already taken (scope: %s)", e.Slug, e.Scope)
}

// SlugReservedError carries D-TenantSubdomains' reserved-subdomain blocklist rejection (M14.9):
// content_sites.slug is a hostname component as of this milestone, so a fixed set of names can
// never be claimed by any congregation, checked server-side (internal/content/application/
// slugvalidation.go) — never only in the admin form's client-side format pattern.
type SlugReservedError struct {
	Slug string
}

func (e *SlugReservedError) Error() string {
	return fmt.Sprintf("slug %q is reserved and cannot be claimed", e.Slug)
}

// BlockDataInvalidError carries the block type/position a json-schema validation failure was found
// at — never the raw validator message, which could echo arbitrary submitted content into a
// safe-arg (see internal/content/transport/errors.go).
type BlockDataInvalidError struct {
	BlockTypeCode string
	Position      int
	Field         string
}

func (e *BlockDataInvalidError) Error() string {
	return fmt.Sprintf("block at position %d (type %q) failed schema validation", e.Position, e.BlockTypeCode)
}

func (e *BlockDataInvalidError) Unwrap() error { return ErrBlockDataInvalid }

// DuplicateBlockPositionError: two blocks in one putBlocks call shared a position — rejected up
// front, no re-derivation (matches the UI's plain numbered-position-input approach).
type DuplicateBlockPositionError struct {
	Position int
}

func (e *DuplicateBlockPositionError) Error() string {
	return fmt.Sprintf("duplicate block position %d", e.Position)
}

// BlockUrlNotAllowedError carries the field a URL-bearing block value failed D-PublicSiteCSP's
// scheme/host allowlist at — never the offending value itself (same safe-arg discipline as
// BlockDataInvalidError).
type BlockUrlNotAllowedError struct {
	BlockTypeCode string
	Position      int
	Field         string
}

func (e *BlockUrlNotAllowedError) Error() string {
	return fmt.Sprintf("block at position %d (type %q): field %q not allowed by URL/embed allowlist", e.Position, e.BlockTypeCode, e.Field)
}

func (e *BlockUrlNotAllowedError) Unwrap() error { return ErrBlockUrlNotAllowed }

// NavTargetInvalidError: a nav item's TargetDocumentID didn't resolve to a PAGE document belonging
// to this same site.
type NavTargetInvalidError struct {
	TargetDocumentID string
}

func (e *NavTargetInvalidError) Error() string {
	return fmt.Sprintf("nav item target document %q is not a PAGE in this site", e.TargetDocumentID)
}

// NavTargetAmbiguousError: a nav item had neither or both of TargetDocumentID/TargetURL set —
// exactly one is required.
type NavTargetAmbiguousError struct {
	SortOrder int
}

func (e *NavTargetAmbiguousError) Error() string {
	return fmt.Sprintf("nav item at sortOrder %d must set exactly one of targetDocumentId/targetUrl", e.SortOrder)
}

// DuplicateNavItemSortOrderError: two nav items in one putNavItems call shared a sortOrder —
// rejected up front, mirroring DuplicateBlockPositionError's own discipline.
type DuplicateNavItemSortOrderError struct {
	SortOrder int
}

func (e *DuplicateNavItemSortOrderError) Error() string {
	return fmt.Sprintf("duplicate nav item sortOrder %d", e.SortOrder)
}
