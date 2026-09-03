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
	// StateScheduled (M14.15, D-PublishOnRead) is a promise to publish at PublishAt with no
	// scheduler ever acting on it — see Document.EffectiveState.
	StateScheduled DocumentState = "SCHEDULED"
)

type TransitionAction string

const (
	ActionPublish       TransitionAction = "PUBLISH"
	ActionUnlist        TransitionAction = "UNLIST"
	ActionRevertToDraft TransitionAction = "REVERT_TO_DRAFT"
	// ActionSchedule (M14.15) moves DRAFT/UNLISTED to SCHEDULED; requires a future PublishAt.
	ActionSchedule TransitionAction = "SCHEDULE"
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
	LogoURL             *string
	SocialLinks         SocialLinks
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// SocialLinks (M14.11) is content_sites.social_links' Go shape — a small, named field set rather
// than a free-form map, so the site-chrome renderer can show a known icon per field deterministically.
type SocialLinks struct {
	Facebook  *string `json:"facebook,omitempty"`
	Instagram *string `json:"instagram,omitempty"`
	YouTube   *string `json:"youtube,omitempty"`
	Twitter   *string `json:"twitter,omitempty"`
	Website   *string `json:"website,omitempty"`
}

// SiteChrome (M14.11) is a site's header/footer data, composed at read time — congregationName/
// address/schedules come live from religion_sites/religion_service_schedules (never copied into
// content's own tables, docs/modules/content.md's own invariant); logoUrl/socialLinks are
// content_sites' own persisted columns. Address is nil if the unit has no religion site, or if its
// PublicPrecision coarsens it away entirely (religiondomain.CoarsenAddress's own "hidden" case).
type SiteChrome struct {
	CongregationName string
	Address          *string
	LogoURL          *string
	SocialLinks      SocialLinks
	Schedules        []ServiceSchedule
}

// ServiceSchedule (M14.11) mirrors religion's own domain.ServiceSchedule shape — content's
// application layer composes GetSiteChrome from religion's live data without content importing
// religion's domain package into its own public API surface.
type ServiceSchedule struct {
	DayOfWeek   *int
	RRule       *string
	StartTime   *string
	EndTime     *string
	Timezone    string
	Language    *string
	Mode        string
	MeetingURL  *string
	Description *string
}

type CreateSiteInput struct {
	CongregationUnitRID string
	Slug                string
}

// Document is a Page, Post, or Event.
type Document struct {
	ID                 string
	SiteID             string
	Kind               DocumentKind
	TranslationGroupID string
	Locale             string
	ParentDocumentID   *string
	Slug               string
	State              DocumentState
	PublishedAt        *time.Time
	// PublishAt (M14.15, D-PublishOnRead) is set only while State is StateScheduled and cleared by
	// every other transition — see EffectiveState.
	PublishAt            *time.Time
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

// EffectiveState (M14.15, D-PublishOnRead) is State as the public read predicate actually
// evaluates it: PUBLISHED once a SCHEDULED document's PublishAt has passed, State unchanged
// otherwise. Nothing ever flips the stored State column itself — no timer, no goroutine — so this
// is the only state a public-visibility check or the admin UI may ever compare against; comparing
// State directly anywhere on the public read path re-hides a document that visitors can already
// see.
func (d Document) EffectiveState(now time.Time) DocumentState {
	if d.State == StateScheduled && d.PublishAt != nil && !d.PublishAt.After(now) {
		return StatePublished
	}
	return d.State
}

// IsPubliclyVisible is the read predicate every public-facing lookup gates on: a real PUBLISHED
// document, an UNLISTED one (reachable by direct link, excluded from listings — unchanged by
// M14.15), or a SCHEDULED one whose PublishAt has passed. Deliberately not "EffectiveState !=
// StateDraft" — a SCHEDULED document that isn't due yet must stay hidden, and EffectiveState only
// ever promotes SCHEDULED to PUBLISHED, never demotes anything to StateDraft, so a plain
// not-equal-Draft check would wrongly expose it early.
func (d Document) IsPubliclyVisible(now time.Time) bool {
	return d.EffectiveState(now) == StatePublished || d.State == StateUnlisted
}

// DocumentTranslation (M14.14, DS-OFM-7) is one PUBLISHED sibling in a document's translation
// group, already resolved to a ready-to-render href — mirrors gencontent.DocumentTranslation, kept
// as a separate domain type so application/service.go never imports the transport-layer package.
type DocumentTranslation struct {
	Locale string
	Href   string
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

// Pattern is a pre-built, church-specific starting layout (M14.13, D-SitePatterns). Blocks is the
// same []BlockInput shape a document's own block list uses everywhere else in this package — never
// an opaque json.RawMessage blob, matching the rest of this domain's "the list structure is a real
// Go slice; only a single block's own Data payload is opaque" convention. Inserting a pattern is
// unsynced: the caller copies Blocks into a document's own block list client-side (or
// application-side) and this row is never referenced again by that document — there is no ongoing
// link to detach.
type Pattern struct {
	ID          string
	Name        string
	Description string
	Blocks      []BlockInput
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreateBlockTypeInput seeds a new catalog row — content.catalog.manage-gated
// (requireCatalogManage), platform-moderator only.
type CreateBlockTypeInput struct {
	Code       string
	Name       string
	JSONSchema json.RawMessage
	UISchema   json.RawMessage
	SortOrder  int
}

// UpdateBlockTypeInput deliberately has no JSONSchema/UISchema field (owner decision, M14.13): a
// block type's schema is locked after creation, so a runtime catalog edit can never silently break
// already-saved blocks of that type or the admin form that was built from its old schema. A
// moderator wanting a different shape retires the old type (Status) and creates a new one.
type UpdateBlockTypeInput struct {
	Name      *string
	Status    *BlockTypeStatus
	SortOrder *int
}

// FormSubmission is an anonymous contact-form entry (M14.16, D-InAppInbox), read through
// openfaithmap-admin's Messages screen. Insert-only — never updated, never soft-deleted.
type FormSubmission struct {
	ID        string
	SiteID    string
	Name      *string
	Email     *string
	Message   string
	CreatedAt time.Time
}

// SubmitContactFormInput carries the two anti-spam signals alongside the visitor's own fields.
// Honeypot non-empty or FormRenderedAt too close to now() means "silently discard, still succeed"
// — SubmitContactForm never surfaces either case as an error.
type SubmitContactFormInput struct {
	SiteID         string
	Name           *string
	Email          *string
	Message        string
	Honeypot       string
	FormRenderedAt time.Time
}

type CreatePatternInput struct {
	Name        string
	Description string
	Blocks      []BlockInput
	SortOrder   int
}

// UpdatePatternInput.Blocks being nil means "leave unchanged" (optional<list<BlockInput>> in
// Conjure) — a caller wanting to genuinely empty a pattern's blocks passes a non-nil empty slice.
type UpdatePatternInput struct {
	Name        *string
	Description *string
	Blocks      []BlockInput
	SortOrder   *int
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
	ErrSiteNotFound     = errors.New("content site not found")
	ErrDocumentNotFound = errors.New("content document not found")
	// ErrForbidden covers both of this module's authorities: content.manage (per congregation unit)
	// and content.catalog.manage (platform-wide, M14.13, platform-moderator only) — deliberately one
	// sentinel for both, since neither ever needs to tell a caller which check failed.
	ErrForbidden          = errors.New("caller does not hold the required authority for this action")
	ErrPatternNotFound    = errors.New("content pattern not found")
	ErrBlockTypeCodeTaken = errors.New("block type code already taken")
	ErrEventMissingStart  = errors.New("kind=EVENT requires eventStartsAt to be set")
	ErrParentTooDeep      = errors.New("parent document chain exceeds 3 levels")
	ErrInvalidTransition  = errors.New("invalid document state transition")
	ErrBlockTypeNotFound  = errors.New("block type not found or retired")
	ErrBlockDataInvalid   = errors.New("block data failed json schema validation")
	ErrBlockUrlNotAllowed = errors.New("block field failed URL scheme/embed host allowlist")
	ErrRevisionNotFound   = errors.New("content document revision not found")
	// ErrScheduleMissingPublishAt/ErrSchedulePublishAtNotFuture back M14.15's SCHEDULE action
	// (D-PublishOnRead) — a schedule with no future publishAt is rejected rather than silently
	// treated as an immediate publish, which is what ActionPublish is for.
	ErrScheduleMissingPublishAt   = errors.New("action=SCHEDULE requires publishAt to be set")
	ErrSchedulePublishAtNotFuture = errors.New("action=SCHEDULE requires publishAt to be in the future")
	// ErrThemeInvalid/ErrThemeContrastFailed back M14.12's theme write-time gate (D-CuratedTheme).
	ErrThemeInvalid        = errors.New("theme value outside the curated vocabulary")
	ErrThemeContrastFailed = errors.New("theme accent/mode pair fails WCAG AA contrast")
	// ErrPreviewTokenInvalid covers a missing, malformed, expired, or wrong-site preview token alike
	// (M14.7) — deliberately one sentinel for every case, so a caller probing the preview endpoints
	// learns nothing about which check failed.
	ErrPreviewTokenInvalid = errors.New("preview token missing, invalid, expired, or scoped to a different site")
	// ErrFormSubmissionInvalid backs M14.16's one validation case (an empty message) — honeypot and
	// too-fast submissions are never errors at all (D-InAppInbox: "an error teaches the bot"), so
	// SubmitContactForm returns success and simply skips the insert for those.
	ErrFormSubmissionInvalid = errors.New("contact form submission invalid")
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

// ThemeInvalidError carries the theme field that failed the curated-vocabulary check (M14.12,
// D-CuratedTheme) — never the raw submitted value, which could be an arbitrary hex/font string an
// attacker chose (same safe-arg discipline as BlockDataInvalidError).
type ThemeInvalidError struct {
	Field string
}

func (e *ThemeInvalidError) Error() string {
	return fmt.Sprintf("theme field %q is not one of the curated values", e.Field)
}

func (e *ThemeInvalidError) Unwrap() error { return ErrThemeInvalid }

// ThemeContrastFailedError carries the accent/mode pair a WCAG AA contrast check rejected at write
// time (M14.12, D-CuratedTheme) — both are curated token names, never a raw hex value, so they're
// safe to surface directly.
type ThemeContrastFailedError struct {
	Accent string
	Mode   string
}

func (e *ThemeContrastFailedError) Error() string {
	return fmt.Sprintf("accent %q fails WCAG AA contrast in %q mode", e.Accent, e.Mode)
}

func (e *ThemeContrastFailedError) Unwrap() error { return ErrThemeContrastFailed }

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

// PatternNotFoundError carries the patternId a catalog lookup couldn't resolve (M14.13) —
// updatePattern/deletePattern against a missing or already-deleted row.
type PatternNotFoundError struct {
	PatternID string
}

func (e *PatternNotFoundError) Error() string {
	return fmt.Sprintf("pattern %q not found", e.PatternID)
}

func (e *PatternNotFoundError) Unwrap() error { return ErrPatternNotFound }

// BlockTypeCodeTakenError: createBlockType's code collided with an existing (non-deleted) row —
// content_block_types_code_idx, race-safe (insert-then-catch-unique-violation), same discipline as
// SlugTakenError.
type BlockTypeCodeTakenError struct {
	Code string
}

func (e *BlockTypeCodeTakenError) Error() string {
	return fmt.Sprintf("block type code %q already taken", e.Code)
}

// TranslationLocaleTakenError (M14.14): CreateDocument named a translationGroupId that already has
// a document at the requested locale. There's no DB constraint for this — content.md's own
// invariant is that a translation group's documents share nothing but the group id — so the guard
// is app-level (application.Service.CreateDocument).
type TranslationLocaleTakenError struct {
	TranslationGroupID string
	Locale             string
}

func (e *TranslationLocaleTakenError) Error() string {
	return fmt.Sprintf("translation group %q already has a document at locale %q", e.TranslationGroupID, e.Locale)
}

// TranslationGroupNotFoundError (M14.14): a translationGroupId was given that doesn't belong to the
// requesting site — either it doesn't exist at all, or every document holding it belongs to some
// other site's content_documents rows.
type TranslationGroupNotFoundError struct {
	TranslationGroupID string
}

func (e *TranslationGroupNotFoundError) Error() string {
	return fmt.Sprintf("translation group %q not found in this site", e.TranslationGroupID)
}

func (e *BlockTypeCodeTakenError) Unwrap() error { return ErrBlockTypeCodeTaken }

// DuplicateNavItemSortOrderError: two nav items in one putNavItems call shared a sortOrder —
// rejected up front, mirroring DuplicateBlockPositionError's own discipline.
type DuplicateNavItemSortOrderError struct {
	SortOrder int
}

func (e *DuplicateNavItemSortOrderError) Error() string {
	return fmt.Sprintf("duplicate nav item sortOrder %d", e.SortOrder)
}

// FormSubmissionInvalidError (M14.16): submitContactForm's message was empty/whitespace-only —
// the one validation failure this endpoint ever reports (honeypot/timing failures are silent).
type FormSubmissionInvalidError struct {
	Field string
}

func (e *FormSubmissionInvalidError) Error() string {
	return fmt.Sprintf("contact form field %q invalid", e.Field)
}

func (e *FormSubmissionInvalidError) Unwrap() error { return ErrFormSubmissionInvalid }
