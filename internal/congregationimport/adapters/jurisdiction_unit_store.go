// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

const jurisdictionUnitSelectColumns = `
	id, source_code, external_id, parent_external_id, name, org_kind_id, status, created_unit_id,
	failure_reason, created_at, updated_at`

// GetJurisdictionUnitByNaturalKey looks up an already-synced node by (sourceCode, externalID) — the
// idempotency check JurisdictionSyncService runs before ever calling createChildOrg. pgx.ErrNoRows
// surfaces to the caller unwrapped (unlike GetCandidate/GetRun) since "not yet synced" is the
// expected, common case here, not an error condition needing its own domain.Err sentinel.
func (s *Store) GetJurisdictionUnitByNaturalKey(ctx context.Context, sourceCode, externalID string) (domain.JurisdictionUnitRecord, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+jurisdictionUnitSelectColumns+` FROM openfaithmap.congregationimport_jurisdiction_units
		WHERE source_code = $1 AND external_id = $2`,
		sourceCode, externalID,
	)
	return scanJurisdictionUnit(row)
}

// CreatePendingJurisdictionUnit inserts a new PENDING row before the real go-oikumenea write is
// attempted — mirrors registration/provision.go's MarkProvisioning precedent: persist the durable
// fact ("we're about to try this node") before the step that can't be cheaply re-derived on a crash.
func (s *Store) CreatePendingJurisdictionUnit(ctx context.Context, sourceCode, externalID string, parentExternalID *string, name, orgKindID string) (domain.JurisdictionUnitRecord, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.congregationimport_jurisdiction_units
			(source_code, external_id, parent_external_id, name, org_kind_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+jurisdictionUnitSelectColumns,
		sourceCode, externalID, parentExternalID, name, orgKindID,
	)
	return scanJurisdictionUnit(row)
}

// MarkJurisdictionUnitCreated records the real go-oikumenea unit RID createChildOrg returned —
// the terminal success state, mirroring MarkProvisioned's shape.
func (s *Store) MarkJurisdictionUnitCreated(ctx context.Context, id, createdUnitID string) (domain.JurisdictionUnitRecord, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.congregationimport_jurisdiction_units
		SET status = 'CREATED', created_unit_id = $2, failure_reason = NULL
		WHERE id = $1
		RETURNING `+jurisdictionUnitSelectColumns,
		id, createdUnitID,
	)
	return scanJurisdictionUnit(row)
}

// MarkJurisdictionUnitFailed records a real createChildOrg failure for this node — a subsequent sync
// run re-attempts a FAILED row (unlike CREATED, which is always skipped), since the underlying cause
// (a transient go-oikumenea outage, a since-fixed parent) may no longer apply.
func (s *Store) MarkJurisdictionUnitFailed(ctx context.Context, id, reason string) (domain.JurisdictionUnitRecord, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE openfaithmap.congregationimport_jurisdiction_units
		SET status = 'FAILED', failure_reason = $2
		WHERE id = $1
		RETURNING `+jurisdictionUnitSelectColumns,
		id, reason,
	)
	return scanJurisdictionUnit(row)
}

func scanJurisdictionUnit(row rowScanner) (domain.JurisdictionUnitRecord, error) {
	var r domain.JurisdictionUnitRecord
	var status string
	if err := row.Scan(
		&r.ID, &r.SourceCode, &r.ExternalID, &r.ParentExternalID, &r.Name, &r.OrgKindID, &status,
		&r.CreatedUnitID, &r.FailureReason, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.JurisdictionUnitRecord{}, err
		}
		return domain.JurisdictionUnitRecord{}, fmt.Errorf("congregationimport: scan jurisdiction unit: %w", err)
	}
	r.Status = domain.JurisdictionUnitStatus(status)
	return r, nil
}
