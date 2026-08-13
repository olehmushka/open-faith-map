// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

// GetCitation returns nil, nil if no citation row exists for connectorCode — callers (in
// particular adapters/connectors/html/base) treat that as "refuse to run," not an error.
func (s *Store) GetCitation(ctx context.Context, connectorCode string) (*domain.SourceCitation, error) {
	var c domain.SourceCitation
	err := s.pool.QueryRow(ctx, `
		SELECT robots_txt_url, robots_checked_at, terms_url, terms_checked_at, user_agent,
		       rate_limit_notes, citation_notes
		FROM openfaithmap.congregationimport_connector_citations WHERE connector_code = $1`,
		connectorCode,
	).Scan(&c.RobotsTxtURL, &c.RobotsCheckedAt, &c.TermsURL, &c.TermsCheckedAt, &c.UserAgent, &c.RateLimitNotes, &c.Notes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) UpsertCitation(ctx context.Context, connectorCode string, c domain.SourceCitation) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO openfaithmap.congregationimport_connector_citations
			(connector_code, robots_txt_url, robots_checked_at, terms_url, terms_checked_at,
			 user_agent, rate_limit_notes, citation_notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (connector_code) DO UPDATE SET
			robots_txt_url = EXCLUDED.robots_txt_url, robots_checked_at = EXCLUDED.robots_checked_at,
			terms_url = EXCLUDED.terms_url, terms_checked_at = EXCLUDED.terms_checked_at,
			user_agent = EXCLUDED.user_agent, rate_limit_notes = EXCLUDED.rate_limit_notes,
			citation_notes = EXCLUDED.citation_notes`,
		connectorCode, c.RobotsTxtURL, c.RobotsCheckedAt, c.TermsURL, c.TermsCheckedAt, c.UserAgent,
		c.RateLimitNotes, c.Notes,
	)
	return err
}
