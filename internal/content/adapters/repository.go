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
		State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt),
		EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
		EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		DraftRevisionID: &revision.ID, PublishedRevisionID: nil,
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
		DraftRevisionID: fromNullableText(row.DraftRevisionID), PublishedRevisionID: fromNullableText(row.PublishedRevisionID),
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
		DraftRevisionID: fromNullableText(row.DraftRevisionID), PublishedRevisionID: fromNullableText(row.PublishedRevisionID),
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
			State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt),
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
			State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt),
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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Document{}, fmt.Errorf("content: publish document: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	txq := contentsql.New(tx)

	draft, err := txq.GetRevision(ctx, draftRevisionID)
	if err != nil {
		return domain.Document{}, fmt.Errorf("content: publish document: get draft: %w", err)
	}
	nextNo, err := txq.NextRevisionNo(ctx, documentID)
	if err != nil {
		return domain.Document{}, fmt.Errorf("content: publish document: next revision no: %w", err)
	}
	checkpoint, err := txq.InsertRevision(ctx, contentsql.InsertRevisionParams{
		DocumentID: documentID, RevisionNo: nextNo, Data: draft.Data, AuthorPersonID: nullableText(authorPersonID),
	})
	if err != nil {
		return domain.Document{}, fmt.Errorf("content: publish document: insert checkpoint: %w", err)
	}
	if err := txq.SetPublishedRevision(ctx, contentsql.SetPublishedRevisionParams{ID: documentID, PublishedRevisionID: nullableText(&checkpoint.ID)}); err != nil {
		return domain.Document{}, fmt.Errorf("content: publish document: set published revision: %w", err)
	}
	row, err := txq.UpdateDocumentState(ctx, contentsql.UpdateDocumentStateParams{ID: documentID, State: string(domain.StatePublished), FirstPublish: firstPublish})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	if err != nil {
		return domain.Document{}, fmt.Errorf("content: publish document: update state: %w", err)
	}
	if err := txq.PruneCheckpointRevisions(ctx, contentsql.PruneCheckpointRevisionsParams{
		DocumentID: documentID, KeepDraftID: draftRevisionID, KeepPublishedID: checkpoint.ID, KeepCount: revisionKeepCount,
	}); err != nil {
		return domain.Document{}, fmt.Errorf("content: publish document: prune: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Document{}, fmt.Errorf("content: publish document: commit: %w", err)
	}

	return domain.Document{
		ID: row.ID, SiteID: row.SiteID, Kind: domain.DocumentKind(row.Kind), TranslationGroupID: row.TranslationGroupID,
		Locale: row.Locale, ParentDocumentID: fromNullableText(row.ParentDocumentID), Slug: row.Slug,
		State: domain.DocumentState(row.State), PublishedAt: db.NullableTime(row.PublishedAt),
		EventStartsAt: db.NullableTime(row.EventStartsAt), EventEndsAt: db.NullableTime(row.EventEndsAt),
		EventRecurrenceRRule: fromNullableText(row.EventRecurrenceRrule), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		DraftRevisionID: fromNullableText(row.DraftRevisionID), PublishedRevisionID: fromNullableText(row.PublishedRevisionID),
	}, nil
}
