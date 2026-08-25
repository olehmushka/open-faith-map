// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the content module's Postgres store — sqlc-generated
// (docs/architecture/decisions.md's D-Stack) — queries live in queries/content.sql, generated code
// in contentsql/. Pool-bound (not db.DBTX): ReplaceBlocks is the one method needing its own
// multi-statement transaction (delete-then-insert), same as the hand-written store before it.
package adapters

import (
	"context"
	"errors"
	"fmt"

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
	return domain.Site{ID: row.ID, CongregationUnitRID: row.CongregationUnitRid, Slug: row.Slug, Theme: row.Theme, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (r *Repository) GetSiteByID(ctx context.Context, id string) (domain.Site, error) {
	row, err := r.q.GetSiteByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Site{}, domain.ErrSiteNotFound
	}
	if err != nil {
		return domain.Site{}, err
	}
	return domain.Site{ID: row.ID, CongregationUnitRID: row.CongregationUnitRid, Slug: row.Slug, Theme: row.Theme, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (r *Repository) GetSiteByUnit(ctx context.Context, congregationUnitRID string) (domain.Site, error) {
	row, err := r.q.GetSiteByUnit(ctx, congregationUnitRID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Site{}, domain.ErrSiteNotFound
	}
	if err != nil {
		return domain.Site{}, err
	}
	return domain.Site{ID: row.ID, CongregationUnitRID: row.CongregationUnitRid, Slug: row.Slug, Theme: row.Theme, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (r *Repository) UpdateSiteTheme(ctx context.Context, id string, theme []byte) (domain.Site, error) {
	row, err := r.q.UpdateSiteTheme(ctx, contentsql.UpdateSiteThemeParams{ID: id, Theme: theme})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Site{}, domain.ErrSiteNotFound
	}
	if err != nil {
		return domain.Site{}, err
	}
	return domain.Site{ID: row.ID, CongregationUnitRID: row.CongregationUnitRid, Slug: row.Slug, Theme: row.Theme, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (r *Repository) GetBlockTypeByCode(ctx context.Context, code string) (domain.BlockType, error) {
	row, err := r.q.GetBlockTypeByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.BlockType{}, domain.ErrBlockTypeNotFound
	}
	if err != nil {
		return domain.BlockType{}, err
	}
	return domain.BlockType{ID: row.ID, Code: row.Code, Name: row.Name, JSONSchema: row.JsonSchema, Status: domain.BlockTypeStatus(row.Status), SortOrder: int(row.SortOrder)}, nil
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
		out = append(out, domain.BlockType{ID: row.ID, Code: row.Code, Name: row.Name, JSONSchema: row.JsonSchema, Status: domain.BlockTypeStatus(row.Status), SortOrder: int(row.SortOrder)})
	}
	return out, nil
}

func (r *Repository) ListBlocks(ctx context.Context, documentID string) ([]domain.Block, error) {
	rows, err := r.q.ListBlocks(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("content: list blocks: %w", err)
	}
	out := make([]domain.Block, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Block{ID: row.ID, DocumentID: row.DocumentID, BlockTypeID: row.BlockTypeID, BlockTypeCode: row.BlockTypeCode, Position: int(row.Position), Data: row.Data, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return out, nil
}

// ReplaceBlocks is a transactional delete-then-insert — application.Service has already validated
// every block's data against its type's json_schema and rejected duplicate positions before this
// is ever called, so this method trusts its input completely.
func (r *Repository) ReplaceBlocks(ctx context.Context, documentID string, blocks []domain.BlockInput) ([]domain.Block, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("content: replace blocks: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txq := contentsql.New(tx)
	if err := txq.DeleteBlocksForDocument(ctx, documentID); err != nil {
		return nil, fmt.Errorf("content: replace blocks: delete: %w", err)
	}
	for _, b := range blocks {
		if err := txq.InsertBlockByTypeCode(ctx, contentsql.InsertBlockByTypeCodeParams{
			DocumentID: documentID, Position: int32(b.Position), Data: b.Data, BlockTypeCode: b.BlockTypeCode,
		}); err != nil {
			return nil, fmt.Errorf("content: replace blocks: insert position %d: %w", b.Position, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("content: replace blocks: commit: %w", err)
	}
	return r.ListBlocks(ctx, documentID)
}

// InsertDocument implements U5's resolution race-safely: INSERT, catch the unique-violation,
// translate. A nil in.TranslationGroupID starts a new group (gen_random_uuid()); a non-nil value
// joins an existing one as another locale variant.
func (r *Repository) InsertDocument(ctx context.Context, siteID string, in domain.CreateDocumentInput) (domain.Document, error) {
	row, err := r.q.InsertDocument(ctx, contentsql.InsertDocumentParams{
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
		return domain.Document{}, err
	}
	return domain.Document{
		ID: row.ID, SiteID: row.SiteID, Kind: domain.DocumentKind(row.Kind), TranslationGroupID: row.TranslationGroupID,
		Locale: row.Locale, ParentDocumentID: fromNullableText(row.ParentDocumentID), Slug: row.Slug,
		State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt),
		EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
		EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
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
		State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt),
		EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
		EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
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
		State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt),
		EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
		EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

// UpdateDocumentState applies a transition already validated by application.Service — sets
// published_at on the first-ever transition into published, never overwriting it on a later one.
func (r *Repository) UpdateDocumentState(ctx context.Context, id string, next domain.DocumentState, firstPublish bool) (domain.Document, error) {
	row, err := r.q.UpdateDocumentState(ctx, contentsql.UpdateDocumentStateParams{ID: id, State: string(next), FirstPublish: firstPublish})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	if err != nil {
		return domain.Document{}, err
	}
	return domain.Document{
		ID: row.ID, SiteID: row.SiteID, Kind: domain.DocumentKind(row.Kind), TranslationGroupID: row.TranslationGroupID,
		Locale: row.Locale, ParentDocumentID: fromNullableText(row.ParentDocumentID), Slug: row.Slug,
		State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt),
		EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
		EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
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
			State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt),
			EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
			EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
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
			State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt),
			EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
			EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}
