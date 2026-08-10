// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/olehmushka/open-faith-map/internal/content/domain"
)

const documentColumns = `
	id, site_id, kind, translation_group_id, locale, parent_document_id, slug, state, published_at,
	event_starts_at, event_ends_at, event_recurrence_rrule, created_at, updated_at`

// InsertDocument implements U5's resolution race-safely: INSERT, catch the unique-violation,
// translate. A nil in.TranslationGroupID starts a new group (gen_random_uuid()); a non-nil value
// joins an existing one as another locale variant.
func (s *Store) InsertDocument(ctx context.Context, siteID string, in domain.CreateDocumentInput) (domain.Document, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.content_documents
			(site_id, kind, translation_group_id, locale, parent_document_id, slug)
		VALUES ($1, $2, COALESCE($3::uuid, gen_random_uuid()), $4, $5, $6)
		RETURNING `+documentColumns,
		siteID, string(in.Kind), in.TranslationGroupID, in.Locale, in.ParentDocumentID, in.Slug,
	)
	doc, err := scanDocument(row)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "content_documents_slug_idx" {
		return domain.Document{}, &domain.SlugTakenError{Slug: in.Slug, Scope: "document"}
	}
	return doc, err
}

func (s *Store) GetDocument(ctx context.Context, id string) (domain.Document, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+documentColumns+` FROM openfaithmap.content_documents WHERE id = $1 AND deleted_at IS NULL`, id)
	doc, err := scanDocument(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	return doc, err
}

// UpdateDocument updates slug and/or parent — nil fields in `in` leave the column unchanged;
// in.ClearParent explicitly nulls parent_document_id.
func (s *Store) UpdateDocument(ctx context.Context, id string, in domain.UpdateDocumentInput) (domain.Document, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.content_documents
		SET slug = COALESCE($2, slug),
		    parent_document_id = CASE WHEN $3 THEN NULL WHEN $4::uuid IS NOT NULL THEN $4::uuid ELSE parent_document_id END
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+documentColumns,
		id, in.Slug, in.ClearParent, in.ParentDocumentID,
	)
	doc, err := scanDocument(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "content_documents_slug_idx" && in.Slug != nil {
		return domain.Document{}, &domain.SlugTakenError{Slug: *in.Slug, Scope: "document"}
	}
	return doc, err
}

// UpdateDocumentState applies a transition already validated by application.Service — sets
// published_at on the first-ever transition into published, never overwriting it on a later one.
func (s *Store) UpdateDocumentState(ctx context.Context, id string, next domain.DocumentState, firstPublish bool) (domain.Document, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.content_documents
		SET state = $2, published_at = CASE WHEN $3 THEN now() ELSE published_at END
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+documentColumns,
		id, string(next), firstPublish,
	)
	doc, err := scanDocument(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	return doc, err
}

// ListDocuments is the admin read — every state; kind/locale/state filters are optional.
func (s *Store) ListDocuments(ctx context.Context, siteID string, kind, locale, state *string) ([]domain.Document, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+documentColumns+` FROM openfaithmap.content_documents
		WHERE site_id = $1 AND deleted_at IS NULL
		  AND ($2::text IS NULL OR kind = $2)
		  AND ($3::text IS NULL OR locale = $3)
		  AND ($4::text IS NULL OR state = $4)
		ORDER BY created_at DESC`,
		siteID, kind, locale, state,
	)
	if err != nil {
		return nil, fmt.Errorf("content: list documents: %w", err)
	}
	defer rows.Close()
	return scanDocuments(rows)
}

// ListPublicDocuments always filters to published/unlisted — never discloses drafts.
func (s *Store) ListPublicDocuments(ctx context.Context, siteID string, kind, locale *string) ([]domain.Document, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+documentColumns+` FROM openfaithmap.content_documents
		WHERE site_id = $1 AND deleted_at IS NULL AND state IN ('PUBLISHED', 'UNLISTED')
		  AND ($2::text IS NULL OR kind = $2)
		  AND ($3::text IS NULL OR locale = $3)
		ORDER BY created_at DESC`,
		siteID, kind, locale,
	)
	if err != nil {
		return nil, fmt.Errorf("content: list public documents: %w", err)
	}
	defer rows.Close()
	return scanDocuments(rows)
}

func scanDocuments(rows pgx.Rows) ([]domain.Document, error) {
	var out []domain.Document
	for rows.Next() {
		doc, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, doc)
	}
	return out, rows.Err()
}

func scanDocument(row rowScanner) (domain.Document, error) {
	var d domain.Document
	var kind, state string
	if err := row.Scan(
		&d.ID, &d.SiteID, &kind, &d.TranslationGroupID, &d.Locale, &d.ParentDocumentID, &d.Slug,
		&state, &d.PublishedAt, &d.EventStartsAt, &d.EventEndsAt, &d.EventRecurrenceRRule,
		&d.CreatedAt, &d.UpdatedAt,
	); err != nil {
		return domain.Document{}, err
	}
	d.Kind = domain.DocumentKind(kind)
	d.State = domain.DocumentState(state)
	return d, nil
}
