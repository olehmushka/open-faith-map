// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the content module's Postgres store — sqlc-generated
// (docs/architecture/decisions.md's D-Stack) — queries live in queries/content.sql, generated code
// in contentsql/. Pool-bound (not db.DBTX): ReplaceBlocks is the one method needing its own
// multi-statement transaction (delete-then-insert), same as the hand-written store before it.
package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/content/adapters/contentsql"
	"github.com/olehmushka/open-faith-map/internal/content/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
)

type Repository struct {
	pool *pgxpool.Pool
	q    *contentsql.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: contentsql.New(pool)}
}

func nullableText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func fromNullableText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

// socialLinksFromJSON unmarshals content_sites.social_links — an empty/absent object (the column's
// own NOT NULL DEFAULT '{}') decodes to the zero SocialLinks value (every field nil), same
// degrade-gracefully shape attributesFromJSON already uses on religion's side.
func socialLinksFromJSON(b json.RawMessage) domain.SocialLinks {
	var s domain.SocialLinks
	if len(b) == 0 {
		return s
	}
	_ = json.Unmarshal(b, &s)
	return s
}

func toSite(id, congregationUnitRID, slug string, theme json.RawMessage, logoURL pgtype.Text, socialLinks json.RawMessage, createdAt, updatedAt time.Time) domain.Site {
	return domain.Site{
		ID: id, CongregationUnitRID: congregationUnitRID, Slug: slug, Theme: theme,
		LogoURL: fromNullableText(logoURL), SocialLinks: socialLinksFromJSON(socialLinks),
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

// InsertSite implements U5's resolution race-safely: INSERT, catch the unique-violation, translate
// — never check-then-insert (TOCTOU).
func (r *Repository) InsertSite(ctx context.Context, in domain.CreateSiteInput) (domain.Site, error) {
	row, err := r.q.InsertSite(ctx, contentsql.InsertSiteParams{CongregationUnitRid: in.CongregationUnitRID, Slug: in.Slug})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "content_sites_slug_idx" {
			return domain.Site{}, &domain.SlugTakenError{Slug: in.Slug, Scope: "site"}
		}
		return domain.Site{}, fmt.Errorf("content: site already exists for this congregation unit: %w", err)
	}
	if err != nil {
		return domain.Site{}, err
	}
	return toSite(row.ID, row.CongregationUnitRid, row.Slug, row.Theme, row.LogoUrl, row.SocialLinks, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) GetSiteByID(ctx context.Context, id string) (domain.Site, error) {
	row, err := r.q.GetSiteByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Site{}, domain.ErrSiteNotFound
	}
	if err != nil {
		return domain.Site{}, err
	}
	return toSite(row.ID, row.CongregationUnitRid, row.Slug, row.Theme, row.LogoUrl, row.SocialLinks, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) GetSiteByUnit(ctx context.Context, congregationUnitRID string) (domain.Site, error) {
	row, err := r.q.GetSiteByUnit(ctx, congregationUnitRID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Site{}, domain.ErrSiteNotFound
	}
	if err != nil {
		return domain.Site{}, err
	}
	return toSite(row.ID, row.CongregationUnitRid, row.Slug, row.Theme, row.LogoUrl, row.SocialLinks, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) GetSiteBySlug(ctx context.Context, slug string) (domain.Site, error) {
	row, err := r.q.GetSiteBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Site{}, domain.ErrSiteNotFound
	}
	if err != nil {
		return domain.Site{}, err
	}
	return toSite(row.ID, row.CongregationUnitRid, row.Slug, row.Theme, row.LogoUrl, row.SocialLinks, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) UpdateSiteTheme(ctx context.Context, id string, theme []byte) (domain.Site, error) {
	row, err := r.q.UpdateSiteTheme(ctx, contentsql.UpdateSiteThemeParams{ID: id, Theme: theme})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Site{}, domain.ErrSiteNotFound
	}
	if err != nil {
		return domain.Site{}, err
	}
	return toSite(row.ID, row.CongregationUnitRid, row.Slug, row.Theme, row.LogoUrl, row.SocialLinks, row.CreatedAt, row.UpdatedAt), nil
}

// UpdateSiteChrome overwrites logoUrl/socialLinks wholesale (M14.11) — the admin form always
// submits the complete shape, same "full replace, never a partial patch" convention
// UpdateSiteAttributesByID (religion, M13.2) already established.
func (r *Repository) UpdateSiteChrome(ctx context.Context, id string, logoURL *string, socialLinks domain.SocialLinks) (domain.Site, error) {
	b, err := json.Marshal(socialLinks)
	if err != nil {
		return domain.Site{}, err
	}
	row, err := r.q.UpdateSiteChrome(ctx, contentsql.UpdateSiteChromeParams{ID: id, LogoUrl: nullableText(logoURL), SocialLinks: b})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Site{}, domain.ErrSiteNotFound
	}
	if err != nil {
		return domain.Site{}, err
	}
	return toSite(row.ID, row.CongregationUnitRid, row.Slug, row.Theme, row.LogoUrl, row.SocialLinks, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) GetBlockTypeByCode(ctx context.Context, code string) (domain.BlockType, error) {
	row, err := r.q.GetBlockTypeByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.BlockType{}, domain.ErrBlockTypeNotFound
	}
	if err != nil {
		return domain.BlockType{}, err
	}
	return domain.BlockType{ID: row.ID, Code: row.Code, Name: row.Name, JSONSchema: row.JsonSchema, UISchema: row.UiSchema, Status: domain.BlockTypeStatus(row.Status), SortOrder: int(row.SortOrder)}, nil
}

// ListActiveBlockTypes is the public read (ContentPublicService) — active types only, ordered for
// display/authoring.
func (r *Repository) ListActiveBlockTypes(ctx context.Context) ([]domain.BlockType, error) {
	rows, err := r.q.ListActiveBlockTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("content: list block types: %w", err)
	}
	out := make([]domain.BlockType, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.BlockType{ID: row.ID, Code: row.Code, Name: row.Name, JSONSchema: row.JsonSchema, UISchema: row.UiSchema, Status: domain.BlockTypeStatus(row.Status), SortOrder: int(row.SortOrder)})
	}
	return out, nil
}

// ---- block-type catalog admin (M14.13, content.catalog.manage) ----

func blockTypeFromCatalogRow(id, code, name string, jsonSchema, uiSchema json.RawMessage, status string, sortOrder int32) domain.BlockType {
	return domain.BlockType{ID: id, Code: code, Name: name, JSONSchema: jsonSchema, UISchema: uiSchema, Status: domain.BlockTypeStatus(status), SortOrder: int(sortOrder)}
}

// ListAllBlockTypes is the admin catalog read (ContentService, catalog.manage) — every status,
// unlike ListActiveBlockTypes.
func (r *Repository) ListAllBlockTypes(ctx context.Context) ([]domain.BlockType, error) {
	rows, err := r.q.ListAllBlockTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("content: list all block types: %w", err)
	}
	out := make([]domain.BlockType, 0, len(rows))
	for _, row := range rows {
		out = append(out, blockTypeFromCatalogRow(row.ID, row.Code, row.Name, row.JsonSchema, row.UiSchema, row.Status, row.SortOrder))
	}
	return out, nil
}

// InsertBlockType implements the same race-safe insert-then-catch-unique-violation discipline as
// InsertSite — content_block_types_code_idx (migrations/0002_content.sql) is the existing unique
// index this collides against, no new index needed.
func (r *Repository) InsertBlockType(ctx context.Context, in domain.CreateBlockTypeInput) (domain.BlockType, error) {
	row, err := r.q.InsertBlockType(ctx, contentsql.InsertBlockTypeParams{
		Code: in.Code, Name: in.Name, JsonSchema: in.JSONSchema, UiSchema: in.UISchema, SortOrder: int32(in.SortOrder),
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "content_block_types_code_idx" {
		return domain.BlockType{}, &domain.BlockTypeCodeTakenError{Code: in.Code}
	}
	if err != nil {
		return domain.BlockType{}, fmt.Errorf("content: insert block type: %w", err)
	}
	return blockTypeFromCatalogRow(row.ID, row.Code, row.Name, row.JsonSchema, row.UiSchema, row.Status, row.SortOrder), nil
}

// UpdateBlockType only ever touches name/status/sortOrder — json_schema/ui_schema are locked after
// creation (owner decision, M14.13; see queries/content.sql's own UpdateBlockType comment).
func (r *Repository) UpdateBlockType(ctx context.Context, id string, in domain.UpdateBlockTypeInput) (domain.BlockType, error) {
	var status pgtype.Text
	if in.Status != nil {
		status = pgtype.Text{String: string(*in.Status), Valid: true}
	}
	var sortOrder pgtype.Int4
	if in.SortOrder != nil {
		sortOrder = pgtype.Int4{Int32: int32(*in.SortOrder), Valid: true}
	}
	row, err := r.q.UpdateBlockType(ctx, contentsql.UpdateBlockTypeParams{ID: id, Name: nullableText(in.Name), Status: status, SortOrder: sortOrder})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.BlockType{}, domain.ErrBlockTypeNotFound
	}
	if err != nil {
		return domain.BlockType{}, fmt.Errorf("content: update block type: %w", err)
	}
	return blockTypeFromCatalogRow(row.ID, row.Code, row.Name, row.JsonSchema, row.UiSchema, row.Status, row.SortOrder), nil
}

// ---- patterns (M14.13, D-SitePatterns) ----

// patternBlock is content_patterns.blocks' on-disk shape — the same {blockTypeCode,position,data}
// triple content_document_revisions.data already uses (application/revisionsnapshot.go's
// revisionSnapshotBlock), duplicated here rather than imported: adapters sits below application in
// this module's layering (adapters must never import application), and this is a small, self-
// contained shape, not a cross-cutting abstraction worth a shared home.
type patternBlock struct {
	BlockTypeCode string          `json:"blockTypeCode"`
	Position      int             `json:"position"`
	Data          json.RawMessage `json:"data"`
}

func marshalPatternBlocks(blocks []domain.BlockInput) (json.RawMessage, error) {
	out := make([]patternBlock, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, patternBlock{BlockTypeCode: b.BlockTypeCode, Position: b.Position, Data: b.Data})
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("content: marshal pattern blocks: %w", err)
	}
	return data, nil
}

func unmarshalPatternBlocks(data json.RawMessage) ([]domain.BlockInput, error) {
	var raw []patternBlock
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("content: unmarshal pattern blocks: %w", err)
	}
	out := make([]domain.BlockInput, 0, len(raw))
	for _, b := range raw {
		out = append(out, domain.BlockInput{BlockTypeCode: b.BlockTypeCode, Position: b.Position, Data: b.Data})
	}
	return out, nil
}

func patternFromRow(id, name, description string, blocksJSON json.RawMessage, sortOrder int32, createdAt, updatedAt time.Time) (domain.Pattern, error) {
	blocks, err := unmarshalPatternBlocks(blocksJSON)
	if err != nil {
		return domain.Pattern{}, err
	}
	return domain.Pattern{ID: id, Name: name, Description: description, Blocks: blocks, SortOrder: int(sortOrder), CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func (r *Repository) InsertPattern(ctx context.Context, in domain.CreatePatternInput) (domain.Pattern, error) {
	blocksJSON, err := marshalPatternBlocks(in.Blocks)
	if err != nil {
		return domain.Pattern{}, err
	}
	row, err := r.q.InsertPattern(ctx, contentsql.InsertPatternParams{
		Name: in.Name, Description: in.Description, Blocks: blocksJSON, SortOrder: int32(in.SortOrder),
	})
	if err != nil {
		return domain.Pattern{}, fmt.Errorf("content: insert pattern: %w", err)
	}
	return patternFromRow(row.ID, row.Name, row.Description, row.Blocks, row.SortOrder, row.CreatedAt, row.UpdatedAt)
}

func (r *Repository) UpdatePattern(ctx context.Context, id string, in domain.UpdatePatternInput) (domain.Pattern, error) {
	var sortOrder pgtype.Int4
	if in.SortOrder != nil {
		sortOrder = pgtype.Int4{Int32: int32(*in.SortOrder), Valid: true}
	}
	var blocksJSON []byte
	if in.Blocks != nil {
		marshaled, err := marshalPatternBlocks(in.Blocks)
		if err != nil {
			return domain.Pattern{}, err
		}
		blocksJSON = marshaled
	}
	row, err := r.q.UpdatePattern(ctx, contentsql.UpdatePatternParams{
		ID: id, Name: nullableText(in.Name), Description: nullableText(in.Description), Blocks: blocksJSON, SortOrder: sortOrder,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Pattern{}, &domain.PatternNotFoundError{PatternID: id}
	}
	if err != nil {
		return domain.Pattern{}, fmt.Errorf("content: update pattern: %w", err)
	}
	return patternFromRow(row.ID, row.Name, row.Description, row.Blocks, row.SortOrder, row.CreatedAt, row.UpdatedAt)
}

// DeletePattern soft-deletes — no natural unique key exists on this table (unlike
// content_block_types.code), so "not found" is decided by rows-affected, not a unique-violation
// catch.
func (r *Repository) DeletePattern(ctx context.Context, id string) error {
	rows, err := r.q.DeletePattern(ctx, id)
	if err != nil {
		return fmt.Errorf("content: delete pattern: %w", err)
	}
	if rows == 0 {
		return &domain.PatternNotFoundError{PatternID: id}
	}
	return nil
}

// ListPatterns is the public read (ContentPublicService, M14.13) — not sensitive data, same
// no-auth reasoning ListActiveBlockTypes already uses.
func (r *Repository) ListPatterns(ctx context.Context) ([]domain.Pattern, error) {
	rows, err := r.q.ListPatterns(ctx)
	if err != nil {
		return nil, fmt.Errorf("content: list patterns: %w", err)
	}
	out := make([]domain.Pattern, 0, len(rows))
	for _, row := range rows {
		pattern, err := patternFromRow(row.ID, row.Name, row.Description, row.Blocks, row.SortOrder, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, pattern)
	}
	return out, nil
}

// InsertFormSubmission is the anonymous write (M14.16, D-InAppInbox) — called only after the
// application layer has already decided this submission isn't a silently-discarded honeypot/
// too-fast hit, so every row that reaches here is real.
func (r *Repository) InsertFormSubmission(ctx context.Context, in domain.SubmitContactFormInput) (domain.FormSubmission, error) {
	row, err := r.q.InsertFormSubmission(ctx, contentsql.InsertFormSubmissionParams{
		SiteID: in.SiteID, Name: nullableText(in.Name), Email: nullableText(in.Email), Message: in.Message,
	})
	if err != nil {
		return domain.FormSubmission{}, fmt.Errorf("content: insert form submission: %w", err)
	}
	return formSubmissionFromRow(row), nil
}

// ListFormSubmissions backs the admin Messages screen — content.manage-gated by the caller.
func (r *Repository) ListFormSubmissions(ctx context.Context, siteID string) ([]domain.FormSubmission, error) {
	rows, err := r.q.ListFormSubmissionsBySite(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("content: list form submissions: %w", err)
	}
	out := make([]domain.FormSubmission, 0, len(rows))
	for _, row := range rows {
		out = append(out, formSubmissionFromRow(row))
	}
	return out, nil
}

func formSubmissionFromRow(row contentsql.OpenfaithmapContentFormSubmission) domain.FormSubmission {
	return domain.FormSubmission{
		ID:        row.ID,
		SiteID:    row.SiteID,
		Name:      fromNullableText(row.Name),
		Email:     fromNullableText(row.Email),
		Message:   row.Message,
		CreatedAt: row.CreatedAt,
	}
}

// InsertDocument implements U5's resolution race-safely: INSERT, catch the unique-violation,
// translate. A nil in.TranslationGroupID starts a new group (gen_random_uuid()); a non-nil value
// joins an existing one as another locale variant.
//
// M14.6: also creates the document's one-and-only draft revision (an empty blocks snapshot) in the
// same transaction and points draft_revision_id at it — content_document_revisions.document_id
// can't exist before the document row does, so this can't be a single INSERT; see
// migrations/0025_content_revisions.sql's own note on why draft_revision_id isn't a DB-level
// NOT NULL (the two tables reference each other).
func (r *Repository) InsertDocument(ctx context.Context, siteID string, in domain.CreateDocumentInput) (domain.Document, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Document{}, fmt.Errorf("content: insert document: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txq := contentsql.New(tx)
	row, err := txq.InsertDocument(ctx, contentsql.InsertDocumentParams{
		SiteID:               siteID,
		Kind:                 string(in.Kind),
		TranslationGroupID:   nullableText(in.TranslationGroupID),
		Locale:               in.Locale,
		ParentDocumentID:     nullableText(in.ParentDocumentID),
		Slug:                 in.Slug,
		EventStartsAt:        db.NullableTimeArg(in.EventStartsAt),
		EventEndsAt:          db.NullableTimeArg(in.EventEndsAt),
		EventRecurrenceRrule: nullableText(in.EventRecurrenceRRule),
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "content_documents_slug_idx" {
		return domain.Document{}, &domain.SlugTakenError{Slug: in.Slug, Scope: "document"}
	}
	if err != nil {
		return domain.Document{}, fmt.Errorf("content: insert document: %w", err)
	}

	revision, err := txq.InsertRevision(ctx, contentsql.InsertRevisionParams{
		DocumentID: row.ID, RevisionNo: 1, Data: json.RawMessage(`[]`),
	})
	if err != nil {
		return domain.Document{}, fmt.Errorf("content: insert document: initial revision: %w", err)
	}
	if err := txq.SetDraftRevision(ctx, contentsql.SetDraftRevisionParams{ID: row.ID, DraftRevisionID: nullableText(&revision.ID)}); err != nil {
		return domain.Document{}, fmt.Errorf("content: insert document: set draft revision: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Document{}, fmt.Errorf("content: insert document: commit: %w", err)
	}

	return domain.Document{
		ID: row.ID, SiteID: row.SiteID, Kind: domain.DocumentKind(row.Kind), TranslationGroupID: row.TranslationGroupID,
		Locale: row.Locale, ParentDocumentID: fromNullableText(row.ParentDocumentID), Slug: row.Slug,
		State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt), PublishAt: db.NullableTime(row.PublishAt),
		EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
		EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		DraftRevisionID: &revision.ID, PublishedRevisionID: nil,
	}, nil
}

// GetDocumentBySlug resolves a document by its natural key (site+kind+locale+slug), matching the
// unique index content_documents_slug_idx — used by M14.10's page-route resolver.
func (r *Repository) GetDocumentBySlug(ctx context.Context, siteID string, kind domain.DocumentKind, locale, slug string) (domain.Document, error) {
	row, err := r.q.GetDocumentBySlug(ctx, contentsql.GetDocumentBySlugParams{SiteID: siteID, Kind: string(kind), Locale: locale, Slug: slug})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	if err != nil {
		return domain.Document{}, err
	}
	return domain.Document{
		ID: row.ID, SiteID: row.SiteID, Kind: domain.DocumentKind(row.Kind), TranslationGroupID: row.TranslationGroupID,
		Locale: row.Locale, ParentDocumentID: fromNullableText(row.ParentDocumentID), Slug: row.Slug,
		State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt), PublishAt: db.NullableTime(row.PublishAt),
		EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
		EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		DraftRevisionID: fromNullableText(row.DraftRevisionID), PublishedRevisionID: fromNullableText(row.PublishedRevisionID),
	}, nil
}

func (r *Repository) GetDocument(ctx context.Context, id string) (domain.Document, error) {
	row, err := r.q.GetDocument(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	if err != nil {
		return domain.Document{}, err
	}
	return domain.Document{
		ID: row.ID, SiteID: row.SiteID, Kind: domain.DocumentKind(row.Kind), TranslationGroupID: row.TranslationGroupID,
		Locale: row.Locale, ParentDocumentID: fromNullableText(row.ParentDocumentID), Slug: row.Slug,
		State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt), PublishAt: db.NullableTime(row.PublishAt),
		EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
		EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		DraftRevisionID: fromNullableText(row.DraftRevisionID), PublishedRevisionID: fromNullableText(row.PublishedRevisionID),
	}, nil
}

// ---- nav items (M14.10) ----

// ReplaceNavItems is a full replace (delete-then-insert in one transaction) — a nav menu is a
// small, hand-curated list edited as a batch (application.Service.PutNavItems), and mirrors
// InsertDocument's own transaction shape rather than PutBlocks' (which since M14.6 is a single-row
// JSON update — the wrong precedent for a genuinely relational table like this one).
func (r *Repository) ReplaceNavItems(ctx context.Context, siteID string, items []domain.NavItemInput) ([]domain.NavItem, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("content: replace nav items: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	txq := contentsql.New(tx)

	if err := txq.DeleteNavItems(ctx, siteID); err != nil {
		return nil, fmt.Errorf("content: replace nav items: delete: %w", err)
	}

	out := make([]domain.NavItem, 0, len(items))
	for _, item := range items {
		row, err := txq.InsertNavItem(ctx, contentsql.InsertNavItemParams{
			SiteID: siteID, Label: item.Label,
			TargetDocumentID: nullableText(item.TargetDocumentID), TargetUrl: nullableText(item.TargetURL),
			SortOrder: int32(item.SortOrder),
		})
		if err != nil {
			return nil, fmt.Errorf("content: replace nav items: insert: %w", err)
		}
		out = append(out, domain.NavItem{
			ID: row.ID, SiteID: row.SiteID, Label: row.Label,
			TargetDocumentID: fromNullableText(row.TargetDocumentID), TargetURL: fromNullableText(row.TargetUrl),
			SortOrder: int(row.SortOrder),
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("content: replace nav items: commit: %w", err)
	}
	return out, nil
}

// ListNavItems is used by both the admin read and the public read (application.Service.ListNavItems
// / ListPublicNavItems) — the table itself has no draft/published distinction, only what each
// caller does with a target document's own state.
func (r *Repository) ListNavItems(ctx context.Context, siteID string) ([]domain.NavItem, error) {
	rows, err := r.q.ListNavItems(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("content: list nav items: %w", err)
	}
	out := make([]domain.NavItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.NavItem{
			ID: row.ID, SiteID: row.SiteID, Label: row.Label,
			TargetDocumentID: fromNullableText(row.TargetDocumentID), TargetURL: fromNullableText(row.TargetUrl),
			SortOrder: int(row.SortOrder),
		})
	}
	return out, nil
}

// UpdateDocument updates slug and/or parent — nil fields in `in` leave the column unchanged;
// in.ClearParent explicitly nulls parent_document_id.
func (r *Repository) UpdateDocument(ctx context.Context, id string, in domain.UpdateDocumentInput) (domain.Document, error) {
	row, err := r.q.UpdateDocument(ctx, contentsql.UpdateDocumentParams{
		ID: id, Slug: nullableText(in.Slug), ClearParent: in.ClearParent, ParentDocumentID: nullableText(in.ParentDocumentID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "content_documents_slug_idx" && in.Slug != nil {
		return domain.Document{}, &domain.SlugTakenError{Slug: *in.Slug, Scope: "document"}
	}
	if err != nil {
		return domain.Document{}, err
	}
	return domain.Document{
		ID: row.ID, SiteID: row.SiteID, Kind: domain.DocumentKind(row.Kind), TranslationGroupID: row.TranslationGroupID,
		Locale: row.Locale, ParentDocumentID: fromNullableText(row.ParentDocumentID), Slug: row.Slug,
		State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt), PublishAt: db.NullableTime(row.PublishAt),
		EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
		EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		DraftRevisionID: fromNullableText(row.DraftRevisionID), PublishedRevisionID: fromNullableText(row.PublishedRevisionID),
	}, nil
}

// UpdateDocumentState applies a transition already validated by application.Service — sets
// published_at on the first-ever transition into published, never overwriting it on a later one.
// publishAt is nil for every transition here (SCHEDULE goes through ScheduleDocument instead), so
// this always clears a stale schedule date when a document leaves SCHEDULED by any other path.
func (r *Repository) UpdateDocumentState(ctx context.Context, id string, next domain.DocumentState, firstPublish bool, publishAt *time.Time) (domain.Document, error) {
	row, err := r.q.UpdateDocumentState(ctx, contentsql.UpdateDocumentStateParams{
		ID: id, State: string(next), FirstPublish: firstPublish, PublishAt: db.NullableTimeArg(publishAt),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	if err != nil {
		return domain.Document{}, err
	}
	return domain.Document{
		ID: row.ID, SiteID: row.SiteID, Kind: domain.DocumentKind(row.Kind), TranslationGroupID: row.TranslationGroupID,
		Locale: row.Locale, ParentDocumentID: fromNullableText(row.ParentDocumentID), Slug: row.Slug,
		State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt), PublishAt: db.NullableTime(row.PublishAt),
		EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
		EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		DraftRevisionID: fromNullableText(row.DraftRevisionID), PublishedRevisionID: fromNullableText(row.PublishedRevisionID),
	}, nil
}

// ListDocuments is the admin read — every state; kind/locale/state filters are optional.
func (r *Repository) ListDocuments(ctx context.Context, siteID string, kind, locale, state *string) ([]domain.Document, error) {
	rows, err := r.q.ListDocuments(ctx, contentsql.ListDocumentsParams{
		SiteID: siteID, Kind: nullableText(kind), Locale: nullableText(locale), State: nullableText(state),
	})
	if err != nil {
		return nil, fmt.Errorf("content: list documents: %w", err)
	}
	out := make([]domain.Document, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Document{
			ID: row.ID, SiteID: row.SiteID, Kind: domain.DocumentKind(row.Kind), TranslationGroupID: row.TranslationGroupID,
			Locale: row.Locale, ParentDocumentID: fromNullableText(row.ParentDocumentID), Slug: row.Slug,
			State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt), PublishAt: db.NullableTime(row.PublishAt),
			EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
			EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
			DraftRevisionID: fromNullableText(row.DraftRevisionID), PublishedRevisionID: fromNullableText(row.PublishedRevisionID),
		})
	}
	return out, nil
}

// ListDocumentsByTranslationGroup (M14.14) is the one lookup shared by CreateDocument's
// cross-site/duplicate-locale guard and GetPublicDocumentByPath's sibling-translations resolution —
// every document sharing one translation_group_id, any state, any site (see the query's own
// comment for why site_id is never a filter here).
func (r *Repository) ListDocumentsByTranslationGroup(ctx context.Context, translationGroupID string) ([]domain.Document, error) {
	rows, err := r.q.ListDocumentsByTranslationGroup(ctx, translationGroupID)
	if err != nil {
		return nil, fmt.Errorf("content: list documents by translation group: %w", err)
	}
	out := make([]domain.Document, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Document{
			ID: row.ID, SiteID: row.SiteID, Kind: domain.DocumentKind(row.Kind), TranslationGroupID: row.TranslationGroupID,
			Locale: row.Locale, ParentDocumentID: fromNullableText(row.ParentDocumentID), Slug: row.Slug,
			State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt), PublishAt: db.NullableTime(row.PublishAt),
			EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
			EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
			DraftRevisionID: fromNullableText(row.DraftRevisionID), PublishedRevisionID: fromNullableText(row.PublishedRevisionID),
		})
	}
	return out, nil
}

// ListPublicDocuments always filters to published/unlisted — never discloses drafts.
func (r *Repository) ListPublicDocuments(ctx context.Context, siteID string, kind, locale *string) ([]domain.Document, error) {
	rows, err := r.q.ListPublicDocuments(ctx, contentsql.ListPublicDocumentsParams{
		SiteID: siteID, Kind: nullableText(kind), Locale: nullableText(locale),
	})
	if err != nil {
		return nil, fmt.Errorf("content: list public documents: %w", err)
	}
	out := make([]domain.Document, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Document{
			ID: row.ID, SiteID: row.SiteID, Kind: domain.DocumentKind(row.Kind), TranslationGroupID: row.TranslationGroupID,
			Locale: row.Locale, ParentDocumentID: fromNullableText(row.ParentDocumentID), Slug: row.Slug,
			State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt), PublishAt: db.NullableTime(row.PublishAt),
			EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
			EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
			DraftRevisionID: fromNullableText(row.DraftRevisionID), PublishedRevisionID: fromNullableText(row.PublishedRevisionID),
		})
	}
	return out, nil
}

// ---- revisions (M14.6) ----

// revisionKeepCount is the owner-decided retention cap (2026-08-28): the 50 most recent checkpoint
// revisions per document. A config change, not new code, once a paid storage tier removes the need
// for it.
const revisionKeepCount = 50

func revisionFromRow(row contentsql.OpenfaithmapContentDocumentRevision) domain.DocumentRevision {
	return domain.DocumentRevision{
		ID: row.ID, DocumentID: row.DocumentID, RevisionNo: int(row.RevisionNo), Data: row.Data,
		AuthorPersonID: fromNullableText(row.AuthorPersonID), CreatedAt: row.CreatedAt, Label: fromNullableText(row.Label),
	}
}

func (r *Repository) GetRevision(ctx context.Context, id string) (domain.DocumentRevision, error) {
	row, err := r.q.GetRevision(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DocumentRevision{}, domain.ErrRevisionNotFound
	}
	if err != nil {
		return domain.DocumentRevision{}, fmt.Errorf("content: get revision: %w", err)
	}
	return revisionFromRow(row), nil
}

// SaveDraftRevisionData overwrites the draft revision's blocks snapshot in place — the store side
// of every autosave/manual save and of RestoreRevision (which "restores into the draft" by calling
// this with an older checkpoint's data, per the owner's decision that restore never auto-publishes).
func (r *Repository) SaveDraftRevisionData(ctx context.Context, draftRevisionID string, data json.RawMessage) (domain.DocumentRevision, error) {
	row, err := r.q.UpdateRevisionData(ctx, contentsql.UpdateRevisionDataParams{ID: draftRevisionID, Data: data})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DocumentRevision{}, domain.ErrRevisionNotFound
	}
	if err != nil {
		return domain.DocumentRevision{}, fmt.Errorf("content: save draft revision: %w", err)
	}
	return revisionFromRow(row), nil
}

// ListCheckpointRevisions is the history list — every revision except the one row draftRevisionID
// points at (an in-progress draft isn't a "past" revision to restore into itself), newest first.
func (r *Repository) ListCheckpointRevisions(ctx context.Context, documentID, draftRevisionID string) ([]domain.DocumentRevision, error) {
	rows, err := r.q.ListCheckpointRevisions(ctx, contentsql.ListCheckpointRevisionsParams{DocumentID: documentID, ExcludeID: draftRevisionID})
	if err != nil {
		return nil, fmt.Errorf("content: list checkpoint revisions: %w", err)
	}
	out := make([]domain.DocumentRevision, 0, len(rows))
	for _, row := range rows {
		out = append(out, revisionFromRow(row))
	}
	return out, nil
}

// PublishDocument is the M14.6 extension of the old pure-state-flip UpdateDocumentState, used only
// for transitions into PUBLISHED: snapshots the current draft into a new immutable checkpoint
// revision, repoints published_revision_id at it, flips document state, and prunes checkpoints
// beyond revisionKeepCount — all in one transaction, so a caller can never observe a state flip
// without its matching revision pointer (or vice versa). Other transitions (revert-to-draft,
// unlist) never touch revisions and keep using the plain UpdateDocumentState below.
func (r *Repository) PublishDocument(ctx context.Context, documentID, draftRevisionID string, authorPersonID *string, firstPublish bool) (domain.Document, error) {
	return r.snapshotAndPromote(ctx, documentID, draftRevisionID, authorPersonID, domain.StatePublished, firstPublish, nil)
}

// ScheduleDocument (M14.15, D-PublishOnRead) is SCHEDULE's store-side counterpart to
// PublishDocument, reusing the exact same snapshot-and-promote transaction: nothing runs later to
// do this snapshot when publishAt actually arrives, so it must happen now, at schedule time.
// firstPublish is always false here — a document isn't considered ever-published until an actual
// PUBLISHED transition happens (see Document.PublishAt's own doc comment).
func (r *Repository) ScheduleDocument(ctx context.Context, documentID, draftRevisionID string, authorPersonID *string, publishAt time.Time) (domain.Document, error) {
	return r.snapshotAndPromote(ctx, documentID, draftRevisionID, authorPersonID, domain.StateScheduled, false, &publishAt)
}

// snapshotAndPromote is PublishDocument's and ScheduleDocument's shared transaction body: snapshot
// the draft into a new immutable checkpoint revision, repoint published_revision_id at it, flip
// document state (and publish_at), and prune old checkpoints — all atomically, so a caller can
// never observe a state flip without its matching revision pointer.
func (r *Repository) snapshotAndPromote(ctx context.Context, documentID, draftRevisionID string, authorPersonID *string, state domain.DocumentState, firstPublish bool, publishAt *time.Time) (domain.Document, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Document{}, fmt.Errorf("content: snapshot and promote: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	txq := contentsql.New(tx)

	draft, err := txq.GetRevision(ctx, draftRevisionID)
	if err != nil {
		return domain.Document{}, fmt.Errorf("content: snapshot and promote: get draft: %w", err)
	}
	nextNo, err := txq.NextRevisionNo(ctx, documentID)
	if err != nil {
		return domain.Document{}, fmt.Errorf("content: snapshot and promote: next revision no: %w", err)
	}
	checkpoint, err := txq.InsertRevision(ctx, contentsql.InsertRevisionParams{
		DocumentID: documentID, RevisionNo: nextNo, Data: draft.Data, AuthorPersonID: nullableText(authorPersonID),
	})
	if err != nil {
		return domain.Document{}, fmt.Errorf("content: snapshot and promote: insert checkpoint: %w", err)
	}
	if err := txq.SetPublishedRevision(ctx, contentsql.SetPublishedRevisionParams{ID: documentID, PublishedRevisionID: nullableText(&checkpoint.ID)}); err != nil {
		return domain.Document{}, fmt.Errorf("content: snapshot and promote: set published revision: %w", err)
	}
	row, err := txq.UpdateDocumentState(ctx, contentsql.UpdateDocumentStateParams{
		ID: documentID, State: string(state), FirstPublish: firstPublish, PublishAt: db.NullableTimeArg(publishAt),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	if err != nil {
		return domain.Document{}, fmt.Errorf("content: snapshot and promote: update state: %w", err)
	}
	if err := txq.PruneCheckpointRevisions(ctx, contentsql.PruneCheckpointRevisionsParams{
		DocumentID: documentID, KeepDraftID: draftRevisionID, KeepPublishedID: checkpoint.ID, KeepCount: revisionKeepCount,
	}); err != nil {
		return domain.Document{}, fmt.Errorf("content: snapshot and promote: prune: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Document{}, fmt.Errorf("content: snapshot and promote: commit: %w", err)
	}

	return domain.Document{
		ID: row.ID, SiteID: row.SiteID, Kind: domain.DocumentKind(row.Kind), TranslationGroupID: row.TranslationGroupID,
		Locale: row.Locale, ParentDocumentID: fromNullableText(row.ParentDocumentID), Slug: row.Slug,
		State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt), PublishAt: db.NullableTime(row.PublishAt),
		EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
		EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		DraftRevisionID: fromNullableText(row.DraftRevisionID), PublishedRevisionID: fromNullableText(row.PublishedRevisionID),
	}, nil
}
