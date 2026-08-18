// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport implements the generated congregationimport.CongregationImportService
// (Conjure server interface): translates Conjure structs <-> domain types and maps domain errors
// to the module's typed Conjure errors.
//
// M10.6: the caller's identity no longer arrives via a per-request whoami round-trip — it's resolved
// from context (populated by internal/identity's authenticator middleware) via personID below, the
// same pattern internal/registration/transport already uses.
package transport

import (
	"context"

	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/application"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	gencongregationimport "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/congregationimport"
	"github.com/palantir/pkg/bearertoken"
)

const defaultPageSize = 50

// maxPageSize (mirroring moderation's own M7 hardening pass, docs/modules/hardening.md) is a
// provisional ceiling — a 4x margin over defaultPageSize, generous for an operator paging a
// backlog by hand in the admin console (the only caller today).
const maxPageSize = 200

type Service struct {
	appService *application.Service
}

func NewService(appService *application.Service) *Service {
	return &Service{appService: appService}
}

var _ gencongregationimport.CongregationImportService = (*Service)(nil)

// personID resolves the caller's own person RID from the request context, populated by
// internal/identity's authenticator middleware — never trusted from a client-supplied value. Its
// only failure mode here is defensive, matching internal/registration/transport's own personID
// helper: the middleware already refuses any request with no valid subject before a handler runs.
func personID(ctx context.Context) (string, error) {
	subject, ok := authz.SubjectFromContext(ctx)
	if !ok || subject.PersonID == "" {
		return "", domain.ErrForbidden
	}
	return subject.PersonID, nil
}

func (s *Service) RunConnector(ctx context.Context, authHeader bearertoken.Token, requestArg gencongregationimport.RunConnectorRequest) (gencongregationimport.ImportRun, error) {
	pid, err := personID(ctx)
	if err != nil {
		return gencongregationimport.ImportRun{}, err
	}
	// RunConnector itself has no operator gate at the application layer (it makes no write under
	// the caller's own subject — only the read-only D-Exclusions/dedup checks under
	// authz.SystemContext), but requiring a resolvable caller identity (personID above) still means
	// an anonymous/invalid token can never trigger one.
	var parameters map[string]string
	if requestArg.Parameters != nil {
		parameters = *requestArg.Parameters
	}
	run, err := s.appService.RunConnector(ctx, requestArg.SourceCode, pid, parameters)
	if err != nil {
		return gencongregationimport.ImportRun{}, mapRunErr(err, requestArg.SourceCode)
	}
	return toAPIRun(run), nil
}

func (s *Service) ListRuns(ctx context.Context, authHeader bearertoken.Token, sourceCodeArg *string, pageSizeArg *int, pageTokenArg *string) (gencongregationimport.RunPage, error) {
	if _, err := personID(ctx); err != nil {
		return gencongregationimport.RunPage{}, err
	}
	var after *domain.PageCursor
	if pageTokenArg != nil {
		c, err := decodeCursor(*pageTokenArg)
		if err != nil {
			return gencongregationimport.RunPage{}, gencongregationimport.NewInvalidPageToken()
		}
		after = &c
	}
	pageSize := pageSizeOrDefault(pageSizeArg)
	runs, err := s.appService.ListRuns(ctx, sourceCodeArg, pageSize, after)
	if err != nil {
		return gencongregationimport.RunPage{}, mapErr(err, "", "")
	}
	var nextToken *string
	if len(runs) > pageSize {
		last := runs[pageSize-1]
		t := encodeCursor(domain.PageCursor{CreatedAt: last.StartedAt, ID: last.ID})
		nextToken = &t
		runs = runs[:pageSize]
	}
	out := make([]gencongregationimport.ImportRun, 0, len(runs))
	for _, r := range runs {
		out = append(out, toAPIRun(r))
	}
	return gencongregationimport.RunPage{Runs: out, NextPageToken: nextToken}, nil
}

func (s *Service) GetRun(ctx context.Context, authHeader bearertoken.Token, runIdArg string) (gencongregationimport.ImportRun, error) {
	if _, err := personID(ctx); err != nil {
		return gencongregationimport.ImportRun{}, err
	}
	run, err := s.appService.GetRun(ctx, runIdArg)
	if err != nil {
		return gencongregationimport.ImportRun{}, mapErr(err, "", "")
	}
	return toAPIRun(run), nil
}

func (s *Service) ListCandidates(ctx context.Context, authHeader bearertoken.Token, statusArg, sourceCodeArg *string, pageSizeArg *int, pageTokenArg *string) (gencongregationimport.CandidatePage, error) {
	if _, err := personID(ctx); err != nil {
		return gencongregationimport.CandidatePage{}, err
	}
	var status *domain.Status
	if statusArg != nil {
		st := domain.Status(*statusArg)
		status = &st
	}
	var after *domain.PageCursor
	if pageTokenArg != nil {
		c, err := decodeCursor(*pageTokenArg)
		if err != nil {
			return gencongregationimport.CandidatePage{}, gencongregationimport.NewInvalidPageToken()
		}
		after = &c
	}
	pageSize := pageSizeOrDefault(pageSizeArg)
	cands, err := s.appService.ListCandidates(ctx, status, sourceCodeArg, pageSize, after)
	if err != nil {
		return gencongregationimport.CandidatePage{}, mapErr(err, "", "")
	}
	var nextToken *string
	if len(cands) > pageSize {
		last := cands[pageSize-1]
		t := encodeCursor(domain.PageCursor{CreatedAt: last.CreatedAt, ID: last.ID})
		nextToken = &t
		cands = cands[:pageSize]
	}
	out := make([]gencongregationimport.Candidate, 0, len(cands))
	for _, c := range cands {
		out = append(out, toAPICandidate(c))
	}
	return gencongregationimport.CandidatePage{Candidates: out, NextPageToken: nextToken}, nil
}

func (s *Service) GetCandidate(ctx context.Context, authHeader bearertoken.Token, candidateIdArg string) (gencongregationimport.Candidate, error) {
	if _, err := personID(ctx); err != nil {
		return gencongregationimport.Candidate{}, err
	}
	c, err := s.appService.GetCandidate(ctx, candidateIdArg)
	if err != nil {
		return gencongregationimport.Candidate{}, mapErr(err, candidateIdArg, "")
	}
	return toAPICandidate(c), nil
}

func (s *Service) EditCandidate(ctx context.Context, authHeader bearertoken.Token, candidateIdArg string, requestArg gencongregationimport.EditCandidateRequest) (gencongregationimport.Candidate, error) {
	if _, err := personID(ctx); err != nil {
		return gencongregationimport.Candidate{}, err
	}
	c, err := s.appService.EditCandidate(ctx, candidateIdArg, toDomainEdit(requestArg))
	if err != nil {
		return gencongregationimport.Candidate{}, mapErr(err, candidateIdArg, "")
	}
	return toAPICandidate(c), nil
}

func (s *Service) ApproveCandidate(ctx context.Context, authHeader bearertoken.Token, candidateIdArg string, requestArg gencongregationimport.ApproveCandidateRequest) (gencongregationimport.Candidate, error) {
	pid, err := personID(ctx)
	if err != nil {
		return gencongregationimport.Candidate{}, err
	}
	c, err := s.appService.ApproveCandidate(ctx, pid, candidateIdArg, requestArg.JurisdictionUnitId)
	if err != nil {
		return gencongregationimport.Candidate{}, mapErr(err, candidateIdArg, "")
	}
	return toAPICandidate(c), nil
}

func (s *Service) RejectCandidate(ctx context.Context, authHeader bearertoken.Token, candidateIdArg string, requestArg gencongregationimport.RejectCandidateRequest) (gencongregationimport.Candidate, error) {
	pid, err := personID(ctx)
	if err != nil {
		return gencongregationimport.Candidate{}, err
	}
	c, err := s.appService.RejectCandidate(ctx, pid, candidateIdArg, requestArg.Reason)
	if err != nil {
		return gencongregationimport.Candidate{}, mapErr(err, candidateIdArg, "")
	}
	return toAPICandidate(c), nil
}

// RunJurisdictionSync triggers sourceCode's JurisdictionSource — D-CatholicJurisdictionSync's
// automated jurisdiction-tier Unit creation (docs/architecture/decisions.md). M10.6: the operator
// gate that used to be missing entirely (a live gap on main — see D-InProcessAuthz's amendment) now
// lives inside application.Service.RunJurisdictionSync itself (requireOperator, its own top), not
// here — this handler only needs a resolvable subject so an anonymous/invalid token can never even
// reach the application layer.
func (s *Service) RunJurisdictionSync(ctx context.Context, authHeader bearertoken.Token, requestArg gencongregationimport.RunJurisdictionSyncRequest) (gencongregationimport.JurisdictionSyncResult, error) {
	if _, err := personID(ctx); err != nil {
		return gencongregationimport.JurisdictionSyncResult{}, err
	}
	summary, err := s.appService.RunJurisdictionSync(ctx, requestArg.SourceCode)
	if err != nil {
		return gencongregationimport.JurisdictionSyncResult{}, mapJurisdictionSyncErr(err, requestArg.SourceCode)
	}
	return gencongregationimport.JurisdictionSyncResult{
		SourceCode:     summary.SourceCode,
		NodesFetched:   summary.NodesFetched,
		UnitsCreated:   summary.UnitsCreated,
		UnitsSkipped:   summary.UnitsSkipped,
		UnitsFailed:    summary.UnitsFailed,
		AliasesCreated: summary.AliasesCreated,
	}, nil
}

func (s *Service) ListTaxonAliases(ctx context.Context, authHeader bearertoken.Token, sourceCodeArg *string) (gencongregationimport.TaxonAliasList, error) {
	if _, err := personID(ctx); err != nil {
		return gencongregationimport.TaxonAliasList{}, err
	}
	aliases, err := s.appService.ListTaxonAliases(ctx, sourceCodeArg)
	if err != nil {
		return gencongregationimport.TaxonAliasList{}, mapErr(err, "", "")
	}
	out := make([]gencongregationimport.TaxonAlias, 0, len(aliases))
	for _, a := range aliases {
		out = append(out, toAPITaxonAlias(a))
	}
	return gencongregationimport.TaxonAliasList{Aliases: out}, nil
}

func (s *Service) CreateTaxonAlias(ctx context.Context, authHeader bearertoken.Token, requestArg gencongregationimport.CreateTaxonAliasRequest) (gencongregationimport.TaxonAlias, error) {
	pid, err := personID(ctx)
	if err != nil {
		return gencongregationimport.TaxonAlias{}, err
	}
	a, err := s.appService.CreateTaxonAlias(ctx, pid, requestArg.SourceCode, requestArg.AliasText, requestArg.TaxonId)
	if err != nil {
		return gencongregationimport.TaxonAlias{}, mapAliasErr(err, requestArg.AliasText)
	}
	return toAPITaxonAlias(a), nil
}

func (s *Service) ListJurisdictionAliases(ctx context.Context, authHeader bearertoken.Token, sourceCodeArg *string) (gencongregationimport.JurisdictionAliasList, error) {
	if _, err := personID(ctx); err != nil {
		return gencongregationimport.JurisdictionAliasList{}, err
	}
	aliases, err := s.appService.ListJurisdictionAliases(ctx, sourceCodeArg)
	if err != nil {
		return gencongregationimport.JurisdictionAliasList{}, mapErr(err, "", "")
	}
	out := make([]gencongregationimport.JurisdictionAlias, 0, len(aliases))
	for _, a := range aliases {
		out = append(out, toAPIJurisdictionAlias(a))
	}
	return gencongregationimport.JurisdictionAliasList{Aliases: out}, nil
}

func (s *Service) CreateJurisdictionAlias(ctx context.Context, authHeader bearertoken.Token, requestArg gencongregationimport.CreateJurisdictionAliasRequest) (gencongregationimport.JurisdictionAlias, error) {
	pid, err := personID(ctx)
	if err != nil {
		return gencongregationimport.JurisdictionAlias{}, err
	}
	a, err := s.appService.CreateJurisdictionAlias(ctx, pid, requestArg.SourceCode, requestArg.AliasText, requestArg.JurisdictionUnitId)
	if err != nil {
		return gencongregationimport.JurisdictionAlias{}, mapAliasErr(err, requestArg.AliasText)
	}
	return toAPIJurisdictionAlias(a), nil
}

func (s *Service) SuggestCoordinates(ctx context.Context, authHeader bearertoken.Token, candidateIdArg string) (gencongregationimport.SuggestCoordinatesResponse, error) {
	if _, err := personID(ctx); err != nil {
		return gencongregationimport.SuggestCoordinatesResponse{}, err
	}
	result, err := s.appService.SuggestCoordinates(ctx, candidateIdArg)
	if err != nil {
		return gencongregationimport.SuggestCoordinatesResponse{}, mapErr(err, candidateIdArg, "")
	}
	return gencongregationimport.SuggestCoordinatesResponse{
		Latitude:    result.Latitude,
		Longitude:   result.Longitude,
		Precision:   result.Precision,
		DisplayName: result.DisplayName,
		Provider:    result.Provider,
	}, nil
}

func pageSizeOrDefault(p *int) int {
	if p == nil || *p <= 0 {
		return defaultPageSize
	}
	if *p > maxPageSize {
		return maxPageSize
	}
	return *p
}
