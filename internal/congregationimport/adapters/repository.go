// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the congregationimport module's Postgres store. sqlc-generated
// (docs/architecture/decisions.md's D-Stack) — queries live in queries/congregationimport.sql,
// generated code in congregationimportsql/.
package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/adapters/congregationimportsql"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
)

type Repository struct {
	q *congregationimportsql.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{q: congregationimportsql.New(conn)}
}

func nullableText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func nullableTextPtr(s *string) pgtype.Text {
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

func nullableFloat8(f *float64) pgtype.Float8 {
	if f == nil {
		return pgtype.Float8{}
	}
	return pgtype.Float8{Float64: *f, Valid: true}
}

func fromNullableFloat8(f pgtype.Float8) *float64 {
	if !f.Valid {
		return nil
	}
	v := f.Float64
	return &v
}

// ---------------------------------------------------------------- candidates

func toCandidate(row congregationimportsql.OpenfaithmapCongregationimportCandidate) domain.Candidate {
	return domain.Candidate{
		ID: row.ID, ImportRunID: fromNullableText(row.ImportRunID), SourceCode: row.SourceCode, SourceRecordID: row.SourceRecordID,
		Name: row.Name, TaxonHint: fromNullableText(row.TaxonHint), TaxonID: fromNullableText(row.TaxonID),
		JurisdictionHint: fromNullableText(row.JurisdictionHint), SuggestedJurisdictionUnitID: fromNullableText(row.SuggestedJurisdictionUnitID),
		CountryID: fromNullableText(row.CountryID), AdminArea1: fromNullableText(row.AdminArea1), Locality: fromNullableText(row.Locality),
		Street: fromNullableText(row.Street), HouseNumber: fromNullableText(row.HouseNumber), PostalCode: fromNullableText(row.PostalCode),
		Latitude: fromNullableFloat8(row.Latitude), Longitude: fromNullableFloat8(row.Longitude), GeocodePrecision: fromNullableText(row.GeocodePrecision),
		RawPayload: row.RawPayload, Status: domain.Status(row.Status),
		PossibleDuplicateCandidateID: fromNullableText(row.PossibleDuplicateOfCandidateID), PossibleDuplicateUnitID: fromNullableText(row.PossibleDuplicateOfUnitID),
		RejectionReason: fromNullableText(row.RejectionReason), ReviewedByPersonRID: fromNullableText(row.ReviewedByPersonRid),
		ReviewedAt: db.NullableTime(row.ReviewedAt), CreatedUnitID: fromNullableText(row.CreatedUnitID), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

// UpsertCandidate inserts a new candidate or, on a (source_code, source_record_id) conflict,
// updates the scraped fields of an existing STAGED/NEEDS_*/POSSIBLE_DUPLICATE one — never touches a
// row already past operator review, so a re-scrape can never silently undo a decision. Returns
// (candidate, created bool, error).
func (r *Repository) UpsertCandidate(ctx context.Context, importRunID *string, sourceCode, sourceRecordID string, in domain.NormalizedCandidate, rawPayload []byte, status domain.Status) (domain.Candidate, bool, error) {
	row, err := r.q.UpsertCandidate(ctx, congregationimportsql.UpsertCandidateParams{
		ImportRunID: nullableTextPtr(importRunID), SourceCode: sourceCode, SourceRecordID: sourceRecordID, Name: in.Name,
		TaxonHint: nullableTextPtr(in.TaxonHint), JurisdictionHint: nullableTextPtr(in.JurisdictionHint),
		AdminArea1: nullableTextPtr(in.AdminArea1), Locality: nullableTextPtr(in.Locality), Street: nullableTextPtr(in.Street),
		HouseNumber: nullableTextPtr(in.HouseNumber), PostalCode: nullableTextPtr(in.PostalCode),
		Latitude: nullableFloat8(in.Latitude), Longitude: nullableFloat8(in.Longitude),
		RawPayload: json.RawMessage(rawPayload), Status: string(status),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// The WHERE clause on the DO UPDATE arm didn't match (row exists but already past review) —
		// fetch it as-is so the caller still gets a real candidate back, just unmodified.
		existing, getErr := r.getCandidateBySource(ctx, sourceCode, sourceRecordID)
		return existing, false, getErr
	}
	if err != nil {
		return domain.Candidate{}, false, err
	}
	c := toCandidate(congregationimportsql.OpenfaithmapCongregationimportCandidate{
		ID: row.ID, ImportRunID: row.ImportRunID, SourceCode: row.SourceCode, SourceRecordID: row.SourceRecordID, Name: row.Name,
		TaxonHint: row.TaxonHint, TaxonID: row.TaxonID, JurisdictionHint: row.JurisdictionHint, SuggestedJurisdictionUnitID: row.SuggestedJurisdictionUnitID,
		CountryID: row.CountryID, AdminArea1: row.AdminArea1, Locality: row.Locality, Street: row.Street, HouseNumber: row.HouseNumber,
		PostalCode: row.PostalCode, Latitude: row.Latitude, Longitude: row.Longitude, GeocodePrecision: row.GeocodePrecision,
		RawPayload: row.RawPayload, Status: row.Status, PossibleDuplicateOfCandidateID: row.PossibleDuplicateOfCandidateID,
		PossibleDuplicateOfUnitID: row.PossibleDuplicateOfUnitID, RejectionReason: row.RejectionReason, ReviewedByPersonRid: row.ReviewedByPersonRid,
		ReviewedAt: row.ReviewedAt, CreatedUnitID: row.CreatedUnitID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	})
	return c, row.Inserted, nil
}

func (r *Repository) getCandidateBySource(ctx context.Context, sourceCode, sourceRecordID string) (domain.Candidate, error) {
	row, err := r.q.GetCandidateBySource(ctx, congregationimportsql.GetCandidateBySourceParams{SourceCode: sourceCode, SourceRecordID: sourceRecordID})
	if err != nil {
		return domain.Candidate{}, err
	}
	return toCandidate(row), nil
}

func (r *Repository) GetCandidate(ctx context.Context, id string) (domain.Candidate, error) {
	row, err := r.q.GetCandidate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrCandidateNotFound
	}
	if err != nil {
		return domain.Candidate{}, err
	}
	return toCandidate(row), nil
}

// ListCandidates uses real keyset pagination — the caller passes after, the decoded cursor of the
// last row of the previous page, and this queries pageSize+1 rows ordered (created_at DESC, id
// DESC) so the caller can tell whether a next page exists without a second round trip.
func (r *Repository) ListCandidates(ctx context.Context, status *domain.Status, sourceCode *string, pageSize int, after *domain.PageCursor) ([]domain.Candidate, error) {
	params := congregationimportsql.ListCandidatesParams{SourceCode: nullableTextPtr(sourceCode), PageSize: int32(pageSize)}
	if status != nil {
		params.Status = pgtype.Text{String: string(*status), Valid: true}
	}
	if after != nil {
		params.AfterCreatedAt = db.NullableTimeArg(&after.CreatedAt)
		params.AfterID = pgtype.Text{String: after.ID, Valid: true}
	}
	rows, err := r.q.ListCandidates(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Candidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, toCandidate(row))
	}
	return out, nil
}

// SetTaxonMatch records an automated or alias-resolved taxon match, without changing status.
func (r *Repository) SetTaxonMatch(ctx context.Context, id, taxonID string) (domain.Candidate, error) {
	row, err := r.q.SetTaxonMatch(ctx, congregationimportsql.SetTaxonMatchParams{ID: id, TaxonID: nullableText(taxonID)})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrCandidateNotFound
	}
	if err != nil {
		return domain.Candidate{}, err
	}
	return toCandidate(row), nil
}

// SetJurisdictionMatch records an alias-matched jurisdiction suggestion, without changing status.
func (r *Repository) SetJurisdictionMatch(ctx context.Context, id, suggestedJurisdictionUnitID string) (domain.Candidate, error) {
	row, err := r.q.SetJurisdictionMatch(ctx, congregationimportsql.SetJurisdictionMatchParams{ID: id, SuggestedJurisdictionUnitID: nullableText(suggestedJurisdictionUnitID)})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrCandidateNotFound
	}
	if err != nil {
		return domain.Candidate{}, err
	}
	return toCandidate(row), nil
}

// SetCountryMatch records a connector-resolved country directly on CountryID, never overwriting an
// operator's own already-set value (COALESCE).
func (r *Repository) SetCountryMatch(ctx context.Context, id, countryID string) (domain.Candidate, error) {
	row, err := r.q.SetCountryMatch(ctx, congregationimportsql.SetCountryMatchParams{ID: id, CountryID: nullableText(countryID)})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrCandidateNotFound
	}
	if err != nil {
		return domain.Candidate{}, err
	}
	return toCandidate(row), nil
}

// SetStatus is the plain status transition used for the automated STAGED/NEEDS_TAXON_REVIEW/
// NEEDS_GEOCODE/POSSIBLE_DUPLICATE moves that don't carry extra fields.
func (r *Repository) SetStatus(ctx context.Context, id string, status domain.Status, dupCandidateID, dupUnitID *string) (domain.Candidate, error) {
	row, err := r.q.SetCandidateStatus(ctx, congregationimportsql.SetCandidateStatusParams{
		ID: id, Status: string(status), PossibleDuplicateOfCandidateID: nullableTextPtr(dupCandidateID), PossibleDuplicateOfUnitID: nullableTextPtr(dupUnitID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrCandidateNotFound
	}
	if err != nil {
		return domain.Candidate{}, err
	}
	return toCandidate(row), nil
}

// RejectExcluded is the automatic, pre-review D-Exclusions rejection — never reachable through the
// operator-facing RejectCandidate path, distinguished by its own status value (REJECTED_EXCLUDED).
func (r *Repository) RejectExcluded(ctx context.Context, id, reason string) (domain.Candidate, error) {
	row, err := r.q.RejectExcludedCandidate(ctx, congregationimportsql.RejectExcludedCandidateParams{ID: id, RejectionReason: nullableText(reason)})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrCandidateNotFound
	}
	if err != nil {
		return domain.Candidate{}, err
	}
	return toCandidate(row), nil
}

// Edit applies an operator's corrections to a candidate still in an editable (pre-approval) status.
// Only non-nil fields in in are applied (COALESCE against the existing value).
func (r *Repository) Edit(ctx context.Context, id string, in domain.EditInput) (domain.Candidate, error) {
	row, err := r.q.EditCandidate(ctx, congregationimportsql.EditCandidateParams{
		ID: id, Name: nullableTextPtr(in.Name), TaxonID: nullableTextPtr(in.TaxonID), CountryID: nullableTextPtr(in.CountryID),
		AdminArea1: nullableTextPtr(in.AdminArea1), Locality: nullableTextPtr(in.Locality), Street: nullableTextPtr(in.Street),
		HouseNumber: nullableTextPtr(in.HouseNumber), PostalCode: nullableTextPtr(in.PostalCode),
		Latitude: nullableFloat8(in.Latitude), Longitude: nullableFloat8(in.Longitude),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrNotEditable
	}
	if err != nil {
		return domain.Candidate{}, err
	}
	return toCandidate(row), nil
}

// Reject is the operator-decided rejection — distinct from the automatic RejectExcluded.
func (r *Repository) Reject(ctx context.Context, id, reviewedByPersonRID, reason string) (domain.Candidate, error) {
	row, err := r.q.RejectCandidate(ctx, congregationimportsql.RejectCandidateParams{
		ID: id, RejectionReason: nullableText(reason), ReviewedByPersonRid: nullableText(reviewedByPersonRID), ReviewedAt: db.NullableTimeArg(nowPtr()),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrNotEditable
	}
	if err != nil {
		return domain.Candidate{}, err
	}
	return toCandidate(row), nil
}

// MarkProvisioning transitions an APPROVED (or already-PROVISIONING, on a resumed retry) candidate
// to PROVISIONING, persisting createdUnitID as soon as the first, unrecoverable write succeeds.
func (r *Repository) MarkProvisioning(ctx context.Context, id, reviewedByPersonRID, createdUnitID string) (domain.Candidate, error) {
	row, err := r.q.MarkCandidateProvisioning(ctx, congregationimportsql.MarkCandidateProvisioningParams{
		ID: id, ReviewedByPersonRid: nullableText(reviewedByPersonRID), ReviewedAt: db.NullableTimeArg(nowPtr()), CreatedUnitID: nullableText(createdUnitID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrNotApprovable
	}
	if err != nil {
		return domain.Candidate{}, err
	}
	return toCandidate(row), nil
}

// MarkProvisioned completes provisioning — the terminal, successful state.
func (r *Repository) MarkProvisioned(ctx context.Context, id string) (domain.Candidate, error) {
	row, err := r.q.MarkCandidateProvisioned(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Candidate{}, domain.ErrNotApprovable
	}
	if err != nil {
		return domain.Candidate{}, err
	}
	return toCandidate(row), nil
}

func nowPtr() *time.Time { t := time.Now(); return &t }

// ---------------------------------------------------------------- congregation status

// CreateCongregationStatus writes the verified overlay row at provisioning time.
func (r *Repository) CreateCongregationStatus(ctx context.Context, congregationUnitRID, sourceCode string, importCandidateID *string, verifiedByPersonRID string) (domain.CongregationStatus, error) {
	row, err := r.q.CreateCongregationStatus(ctx, congregationimportsql.CreateCongregationStatusParams{
		CongregationUnitRid: congregationUnitRID, SourceCode: sourceCode, ImportCandidateID: nullableTextPtr(importCandidateID), VerifiedByPersonRid: verifiedByPersonRID,
	})
	if err != nil {
		return domain.CongregationStatus{}, err
	}
	return domain.CongregationStatus{
		CongregationUnitRID: row.CongregationUnitRid, SourceCode: row.SourceCode, ImportCandidateID: fromNullableText(row.ImportCandidateID),
		VerifiedByPersonRID: row.VerifiedByPersonRid, VerifiedAt: row.VerifiedAt, ClaimedByPersonRID: fromNullableText(row.ClaimedByPersonRid),
		ClaimedAt: db.NullableTime(row.ClaimedAt), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

// ---------------------------------------------------------------- connector citations

// GetCitation returns nil, nil if no citation row exists for connectorCode.
func (r *Repository) GetCitation(ctx context.Context, connectorCode string) (*domain.SourceCitation, error) {
	row, err := r.q.GetCitation(ctx, connectorCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.SourceCitation{
		RobotsTxtURL: fromNullableText(row.RobotsTxtUrl), RobotsCheckedAt: db.NullableTime(row.RobotsCheckedAt),
		TermsURL: fromNullableText(row.TermsUrl), TermsCheckedAt: db.NullableTime(row.TermsCheckedAt),
		UserAgent: row.UserAgent, RateLimitNotes: fromNullableText(row.RateLimitNotes), Notes: row.CitationNotes,
	}, nil
}

func (r *Repository) UpsertCitation(ctx context.Context, connectorCode string, c domain.SourceCitation) error {
	return r.q.UpsertCitation(ctx, congregationimportsql.UpsertCitationParams{
		ConnectorCode: connectorCode, RobotsTxtUrl: nullableTextPtr(c.RobotsTxtURL), RobotsCheckedAt: db.NullableTimeArg(c.RobotsCheckedAt),
		TermsUrl: nullableTextPtr(c.TermsURL), TermsCheckedAt: db.NullableTimeArg(c.TermsCheckedAt),
		UserAgent: c.UserAgent, RateLimitNotes: nullableTextPtr(c.RateLimitNotes), CitationNotes: c.Notes,
	})
}

// ---------------------------------------------------------------- jurisdiction aliases

func toJurisdictionAlias(row congregationimportsql.OpenfaithmapCongregationimportJurisdictionAlias) domain.JurisdictionAlias {
	return domain.JurisdictionAlias{
		ID: row.ID, SourceCode: fromNullableText(row.SourceCode), AliasText: row.AliasText, JurisdictionUnitID: row.JurisdictionUnitID,
		CreatedByPersonRID: row.CreatedByPersonRid, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

// CreateJurisdictionAlias implements the unique-alias constraint race-safely: INSERT, catch the
// unique-violation, translate — mirrors CreateTaxonAlias exactly.
func (r *Repository) CreateJurisdictionAlias(ctx context.Context, sourceCode *string, aliasText, jurisdictionUnitID, createdByPersonRID string) (domain.JurisdictionAlias, error) {
	row, err := r.q.CreateJurisdictionAlias(ctx, congregationimportsql.CreateJurisdictionAliasParams{
		SourceCode: nullableTextPtr(sourceCode), AliasText: aliasText, JurisdictionUnitID: jurisdictionUnitID, CreatedByPersonRid: createdByPersonRID,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.JurisdictionAlias{}, domain.ErrAliasConflict
		}
		return domain.JurisdictionAlias{}, err
	}
	return toJurisdictionAlias(row), nil
}

// ListJurisdictionAliasesForMatching mirrors ListAliasesForMatching exactly: small,
// operator-maintained, loaded in full so the caller can substring-match against a candidate's
// free-text jurisdiction hint.
func (r *Repository) ListJurisdictionAliasesForMatching(ctx context.Context, sourceCode string) ([]domain.JurisdictionAlias, error) {
	rows, err := r.q.ListJurisdictionAliasesForMatching(ctx, nullableText(sourceCode))
	if err != nil {
		return nil, err
	}
	out := make([]domain.JurisdictionAlias, 0, len(rows))
	for _, row := range rows {
		out = append(out, toJurisdictionAlias(row))
	}
	return out, nil
}

// ListAllJurisdictionAliases returns every alias, across every source, for the alias-management UI.
func (r *Repository) ListAllJurisdictionAliases(ctx context.Context) ([]domain.JurisdictionAlias, error) {
	rows, err := r.q.ListAllJurisdictionAliases(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.JurisdictionAlias, 0, len(rows))
	for _, row := range rows {
		out = append(out, toJurisdictionAlias(row))
	}
	return out, nil
}

// ---------------------------------------------------------------- taxon aliases

func toTaxonAlias(row congregationimportsql.OpenfaithmapCongregationimportTaxonAlias) domain.TaxonAlias {
	return domain.TaxonAlias{
		ID: row.ID, SourceCode: fromNullableText(row.SourceCode), AliasText: row.AliasText, TaxonID: row.TaxonID,
		CreatedByPersonRID: row.CreatedByPersonRid, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

// CreateTaxonAlias implements the unique-alias constraint race-safely: INSERT, catch the
// unique-violation, translate — never check-then-insert (TOCTOU).
func (r *Repository) CreateTaxonAlias(ctx context.Context, sourceCode *string, aliasText, taxonID, createdByPersonRID string) (domain.TaxonAlias, error) {
	row, err := r.q.CreateTaxonAlias(ctx, congregationimportsql.CreateTaxonAliasParams{
		SourceCode: nullableTextPtr(sourceCode), AliasText: aliasText, TaxonID: taxonID, CreatedByPersonRid: createdByPersonRID,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.TaxonAlias{}, domain.ErrAliasConflict
		}
		return domain.TaxonAlias{}, err
	}
	return toTaxonAlias(row), nil
}

// ListAliasesForMatching returns every alias applicable to sourceCode (source-scoped ones first,
// then global ones) — small enough to load in full and let the caller do substring/keyword
// matching against a candidate's free-text name.
func (r *Repository) ListAliasesForMatching(ctx context.Context, sourceCode string) ([]domain.TaxonAlias, error) {
	rows, err := r.q.ListAliasesForMatching(ctx, nullableText(sourceCode))
	if err != nil {
		return nil, err
	}
	out := make([]domain.TaxonAlias, 0, len(rows))
	for _, row := range rows {
		out = append(out, toTaxonAlias(row))
	}
	return out, nil
}

// ListAllTaxonAliases returns every alias, across every source, for the alias-management UI.
func (r *Repository) ListAllTaxonAliases(ctx context.Context) ([]domain.TaxonAlias, error) {
	rows, err := r.q.ListAllTaxonAliases(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TaxonAlias, 0, len(rows))
	for _, row := range rows {
		out = append(out, toTaxonAlias(row))
	}
	return out, nil
}

// ---------------------------------------------------------------- runs

func (r *Repository) rowToRun(id, sourceCode, status, triggeredByPersonRID string, parameters []byte, cursorAtStart, cursorAtEnd pgtype.Text, recordsFetched, created, updated, autoRejected int32, errMsg pgtype.Text, startedAt time.Time, finishedAt pgtype.Timestamptz) (domain.Run, error) {
	params, err := unmarshalParameters(parameters)
	if err != nil {
		return domain.Run{}, err
	}
	return domain.Run{
		ID: id, SourceCode: sourceCode, Status: domain.RunStatus(status), TriggeredByPersonRID: triggeredByPersonRID, Parameters: params,
		CursorAtStart: fromNullableText(cursorAtStart), CursorAtEnd: fromNullableText(cursorAtEnd),
		RecordsFetched: int(recordsFetched), CandidatesCreated: int(created), CandidatesUpdated: int(updated), CandidatesAutoRejected: int(autoRejected),
		Error: fromNullableText(errMsg), StartedAt: startedAt, FinishedAt: db.NullableTime(finishedAt),
	}, nil
}

func unmarshalParameters(b []byte) (map[string]string, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func marshalParameters(m map[string]string) ([]byte, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

// CreateRun persists a new run row, including whatever parameters the caller actually supplied to
// RunConnector (nil when none were) — an empty map is stored as SQL NULL, not an empty JSON object.
func (r *Repository) CreateRun(ctx context.Context, sourceCode, triggeredByPersonRID string, parameters map[string]string, cursorAtStart *string) (domain.Run, error) {
	paramBytes, err := marshalParameters(parameters)
	if err != nil {
		return domain.Run{}, err
	}
	row, err := r.q.CreateRun(ctx, congregationimportsql.CreateRunParams{
		SourceCode: sourceCode, TriggeredByPersonRid: triggeredByPersonRID, Parameters: paramBytes, CursorAtStart: nullableTextPtr(cursorAtStart),
	})
	if err != nil {
		return domain.Run{}, err
	}
	return r.rowToRun(row.ID, row.SourceCode, row.Status, row.TriggeredByPersonRid, row.Parameters, row.CursorAtStart, row.CursorAtEnd,
		row.RecordsFetched, row.CandidatesCreated, row.CandidatesUpdated, row.CandidatesAutoRejected, row.Error, row.StartedAt, row.FinishedAt)
}

func (r *Repository) GetRun(ctx context.Context, id string) (domain.Run, error) {
	row, err := r.q.GetRun(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Run{}, domain.ErrRunNotFound
	}
	if err != nil {
		return domain.Run{}, err
	}
	return r.rowToRun(row.ID, row.SourceCode, row.Status, row.TriggeredByPersonRid, row.Parameters, row.CursorAtStart, row.CursorAtEnd,
		row.RecordsFetched, row.CandidatesCreated, row.CandidatesUpdated, row.CandidatesAutoRejected, row.Error, row.StartedAt, row.FinishedAt)
}

// ListRuns uses real keyset pagination, ordered (started_at DESC, id DESC).
func (r *Repository) ListRuns(ctx context.Context, sourceCode *string, pageSize int, after *domain.PageCursor) ([]domain.Run, error) {
	params := congregationimportsql.ListRunsParams{SourceCode: nullableTextPtr(sourceCode), PageSize: int32(pageSize)}
	if after != nil {
		params.AfterStartedAt = db.NullableTimeArg(&after.CreatedAt)
		params.AfterID = pgtype.Text{String: after.ID, Valid: true}
	}
	rows, err := r.q.ListRuns(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Run, 0, len(rows))
	for _, row := range rows {
		run, err := r.rowToRun(row.ID, row.SourceCode, row.Status, row.TriggeredByPersonRid, row.Parameters, row.CursorAtStart, row.CursorAtEnd,
			row.RecordsFetched, row.CandidatesCreated, row.CandidatesUpdated, row.CandidatesAutoRejected, row.Error, row.StartedAt, row.FinishedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, nil
}

// FinishRun records the terminal state of a run — counts, the end cursor, and either SUCCEEDED or
// FAILED (with errMsg set only in the latter case).
func (r *Repository) FinishRun(ctx context.Context, id string, status domain.RunStatus, cursorAtEnd *string, recordsFetched, created, updated, autoRejected int, errMsg *string) (domain.Run, error) {
	row, err := r.q.FinishRun(ctx, congregationimportsql.FinishRunParams{
		ID: id, Status: string(status), CursorAtEnd: nullableTextPtr(cursorAtEnd), RecordsFetched: int32(recordsFetched),
		CandidatesCreated: int32(created), CandidatesUpdated: int32(updated), CandidatesAutoRejected: int32(autoRejected),
		Error: nullableTextPtr(errMsg), FinishedAt: db.NullableTimeArg(nowPtr()),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Run{}, domain.ErrRunNotFound
	}
	if err != nil {
		return domain.Run{}, err
	}
	return r.rowToRun(row.ID, row.SourceCode, row.Status, row.TriggeredByPersonRid, row.Parameters, row.CursorAtStart, row.CursorAtEnd,
		row.RecordsFetched, row.CandidatesCreated, row.CandidatesUpdated, row.CandidatesAutoRejected, row.Error, row.StartedAt, row.FinishedAt)
}

// ---------------------------------------------------------------- jurisdiction units

func toJurisdictionUnit(row congregationimportsql.OpenfaithmapCongregationimportJurisdictionUnit) domain.JurisdictionUnitRecord {
	return domain.JurisdictionUnitRecord{
		ID: row.ID, SourceCode: row.SourceCode, ExternalID: row.ExternalID, ParentExternalID: fromNullableText(row.ParentExternalID),
		Name: row.Name, OrgKindID: row.OrgKindID, Status: domain.JurisdictionUnitStatus(row.Status),
		CreatedUnitID: fromNullableText(row.CreatedUnitID), FailureReason: fromNullableText(row.FailureReason), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

// GetJurisdictionUnitByNaturalKey looks up an already-synced node by (sourceCode, externalID) — the
// idempotency check JurisdictionSyncService runs before ever calling createChildOrg. pgx.ErrNoRows
// surfaces to the caller unwrapped since "not yet synced" is the expected, common case here.
func (r *Repository) GetJurisdictionUnitByNaturalKey(ctx context.Context, sourceCode, externalID string) (domain.JurisdictionUnitRecord, error) {
	row, err := r.q.GetJurisdictionUnitByNaturalKey(ctx, congregationimportsql.GetJurisdictionUnitByNaturalKeyParams{SourceCode: sourceCode, ExternalID: externalID})
	if err != nil {
		return domain.JurisdictionUnitRecord{}, err
	}
	return toJurisdictionUnit(row), nil
}

// CreatePendingJurisdictionUnit inserts a new PENDING row before the real write is attempted —
// persist the durable fact before the step that can't be cheaply re-derived on a crash.
func (r *Repository) CreatePendingJurisdictionUnit(ctx context.Context, sourceCode, externalID string, parentExternalID *string, name, orgKindID string) (domain.JurisdictionUnitRecord, error) {
	row, err := r.q.CreatePendingJurisdictionUnit(ctx, congregationimportsql.CreatePendingJurisdictionUnitParams{
		SourceCode: sourceCode, ExternalID: externalID, ParentExternalID: nullableTextPtr(parentExternalID), Name: name, OrgKindID: orgKindID,
	})
	if err != nil {
		return domain.JurisdictionUnitRecord{}, err
	}
	return toJurisdictionUnit(row), nil
}

// MarkJurisdictionUnitCreated records the real unit RID a successful create returned — the
// terminal success state.
func (r *Repository) MarkJurisdictionUnitCreated(ctx context.Context, id, createdUnitID string) (domain.JurisdictionUnitRecord, error) {
	row, err := r.q.MarkJurisdictionUnitCreated(ctx, congregationimportsql.MarkJurisdictionUnitCreatedParams{ID: id, CreatedUnitID: nullableText(createdUnitID)})
	if err != nil {
		return domain.JurisdictionUnitRecord{}, err
	}
	return toJurisdictionUnit(row), nil
}

// MarkJurisdictionUnitFailed records a real creation failure for this node — a subsequent sync run
// re-attempts a FAILED row.
func (r *Repository) MarkJurisdictionUnitFailed(ctx context.Context, id, reason string) (domain.JurisdictionUnitRecord, error) {
	row, err := r.q.MarkJurisdictionUnitFailed(ctx, congregationimportsql.MarkJurisdictionUnitFailedParams{ID: id, FailureReason: nullableText(reason)})
	if err != nil {
		return domain.JurisdictionUnitRecord{}, err
	}
	return toJurisdictionUnit(row), nil
}
