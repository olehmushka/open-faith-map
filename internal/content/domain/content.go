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

var (
	ErrSiteNotFound       = errors.New("content site not found")
	ErrDocumentNotFound   = errors.New("content document not found")
	ErrForbidden          = errors.New("caller does not hold religionorg.manage on this site's congregation unit")
	ErrEventMissingStart  = errors.New("kind=EVENT requires eventStartsAt to be set")
	ErrParentTooDeep      = errors.New("parent document chain exceeds 3 levels")
	ErrInvalidTransition  = errors.New("invalid document state transition")
	ErrBlockTypeNotFound  = errors.New("block type not found or retired")
	ErrBlockDataInvalid   = errors.New("block data failed json schema validation")
	ErrBlockUrlNotAllowed = errors.New("block field failed URL scheme/embed host allowlist")
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
