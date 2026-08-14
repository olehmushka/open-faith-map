// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport implements the generated congregationimport.CongregationImportService
// (Conjure server interface): translates Conjure structs <-> domain types, resolves the caller's
// own person RID from their forwarded token (never a client-supplied id), and maps domain errors
// to the module's typed Conjure errors.
package transport

import (
	"context"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/application"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	gencongregationimport "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/congregationimport"
	"github.com/olehmushka/open-faith-map/internal/coreintegration"
	"github.com/palantir/pkg/bearertoken"
)

const defaultPageSize = 50

// maxPageSize (mirroring moderation's own M7 hardening pass, docs/modules/hardening.md) is a
// provisional ceiling — a 4x margin over defaultPageSize, generous for an operator paging a
// backlog by hand in the admin console (the only caller today).
const maxPageSize = 200

type Service struct {
	oikumeneaBaseURL  string
	oikumeneaInsecure bool
	appService        *application.Service
}

type Config struct {
	OikumeneaBaseURL            string
	OikumeneaInsecureSkipVerify bool
}

func NewService(appService *application.Service, cfg Config) *Service {
	return &Service{appService: appService, oikumeneaBaseURL: cfg.OikumeneaBaseURL, oikumeneaInsecure: cfg.OikumeneaInsecureSkipVerify}
}

var _ gencongregationimport.CongregationImportService = (*Service)(nil)

// whoami resolves the caller's own go-oikumenea person RID from their forwarded token — never
// trusts a client-supplied id, same as every other module's transport layer.
func (s *Service) whoami(ctx context.Context, token bearertoken.Token) (string, error) {
	c, err := coreintegration.NewUserClient(s.oikumeneaBaseURL, string(token), s.oikumeneaInsecure)
	if err != nil {
		return "", err
	}
	who, err := c.IdentityFederation.Whoami(ctx)
	if err != nil {
		return "", err
	}
	return who.PersonId, nil
}

func (s *Service) RunConnector(ctx context.Context, authHeader bearertoken.Token, requestArg gencongregationimport.RunConnectorRequest) (gencongregationimport.ImportRun, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencongregationimport.ImportRun{}, mapUpstreamErr(err)
	}
	// RunConnector itself has no operator gate at the application layer (it makes no go-oikumenea
	// WRITE — only the read-only D-Exclusions/dedup checks under the service principal), but
	// requiring a resolvable caller identity (whoami above) still means an anonymous/invalid token
	// can never trigger one.
	run, err := s.appService.RunConnector(ctx, requestArg.SourceCode, personID)
	if err != nil {
		return gencongregationimport.ImportRun{}, mapErr(err, "", "")
	}
	return toAPIRun(run), nil
}

func (s *Service) ListRuns(ctx context.Context, authHeader bearertoken.Token, sourceCodeArg *string, pageSizeArg *int, pageTokenArg *string) (gencongregationimport.RunPage, error) {
	if _, err := s.whoami(ctx, authHeader); err != nil {
		return gencongregationimport.RunPage{}, mapUpstreamErr(err)
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
	if _, err := s.whoami(ctx, authHeader); err != nil {
		return gencongregationimport.ImportRun{}, mapUpstreamErr(err)
	}
	run, err := s.appService.GetRun(ctx, runIdArg)
	if err != nil {
		return gencongregationimport.ImportRun{}, mapErr(err, "", "")
	}
	return toAPIRun(run), nil
}

func (s *Service) ListCandidates(ctx context.Context, authHeader bearertoken.Token, statusArg *string, pageSizeArg *int, pageTokenArg *string) (gencongregationimport.CandidatePage, error) {
	if _, err := s.whoami(ctx, authHeader); err != nil {
		return gencongregationimport.CandidatePage{}, mapUpstreamErr(err)
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
	cands, err := s.appService.ListCandidates(ctx, status, pageSize, after)
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
	if _, err := s.whoami(ctx, authHeader); err != nil {
		return gencongregationimport.Candidate{}, mapUpstreamErr(err)
	}
	c, err := s.appService.GetCandidate(ctx, candidateIdArg)
	if err != nil {
		return gencongregationimport.Candidate{}, mapErr(err, candidateIdArg, "")
	}
	return toAPICandidate(c), nil
}

func (s *Service) EditCandidate(ctx context.Context, authHeader bearertoken.Token, candidateIdArg string, requestArg gencongregationimport.EditCandidateRequest) (gencongregationimport.Candidate, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencongregationimport.Candidate{}, mapUpstreamErr(err)
	}
	c, err := s.appService.EditCandidate(ctx, string(authHeader), personID, candidateIdArg, toDomainEdit(requestArg))
	if err != nil {
		return gencongregationimport.Candidate{}, mapErr(err, candidateIdArg, "")
	}
	return toAPICandidate(c), nil
}

func (s *Service) ApproveCandidate(ctx context.Context, authHeader bearertoken.Token, candidateIdArg string, requestArg gencongregationimport.ApproveCandidateRequest) (gencongregationimport.Candidate, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencongregationimport.Candidate{}, mapUpstreamErr(err)
	}
	c, err := s.appService.ApproveCandidate(ctx, string(authHeader), personID, candidateIdArg, requestArg.JurisdictionUnitId)
	if err != nil {
		return gencongregationimport.Candidate{}, mapErr(err, candidateIdArg, "")
	}
	return toAPICandidate(c), nil
}

func (s *Service) RejectCandidate(ctx context.Context, authHeader bearertoken.Token, candidateIdArg string, requestArg gencongregationimport.RejectCandidateRequest) (gencongregationimport.Candidate, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencongregationimport.Candidate{}, mapUpstreamErr(err)
	}
	c, err := s.appService.RejectCandidate(ctx, string(authHeader), personID, candidateIdArg, requestArg.Reason)
	if err != nil {
		return gencongregationimport.Candidate{}, mapErr(err, candidateIdArg, "")
	}
	return toAPICandidate(c), nil
}

func (s *Service) ListTaxonAliases(ctx context.Context, authHeader bearertoken.Token, sourceCodeArg *string) (gencongregationimport.TaxonAliasList, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencongregationimport.TaxonAliasList{}, mapUpstreamErr(err)
	}
	aliases, err := s.appService.ListTaxonAliases(ctx, string(authHeader), personID, sourceCodeArg)
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
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencongregationimport.TaxonAlias{}, mapUpstreamErr(err)
	}
	a, err := s.appService.CreateTaxonAlias(ctx, string(authHeader), personID, requestArg.SourceCode, requestArg.AliasText, requestArg.TaxonId)
	if err != nil {
		return gencongregationimport.TaxonAlias{}, mapAliasErr(err, requestArg.AliasText)
	}
	return toAPITaxonAlias(a), nil
}

func (s *Service) ListJurisdictionAliases(ctx context.Context, authHeader bearertoken.Token, sourceCodeArg *string) (gencongregationimport.JurisdictionAliasList, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencongregationimport.JurisdictionAliasList{}, mapUpstreamErr(err)
	}
	aliases, err := s.appService.ListJurisdictionAliases(ctx, string(authHeader), personID, sourceCodeArg)
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
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencongregationimport.JurisdictionAlias{}, mapUpstreamErr(err)
	}
	a, err := s.appService.CreateJurisdictionAlias(ctx, string(authHeader), personID, requestArg.SourceCode, requestArg.AliasText, requestArg.JurisdictionUnitId)
	if err != nil {
		return gencongregationimport.JurisdictionAlias{}, mapAliasErr(err, requestArg.AliasText)
	}
	return toAPIJurisdictionAlias(a), nil
}

func (s *Service) SuggestCoordinates(ctx context.Context, authHeader bearertoken.Token, candidateIdArg string) (gencongregationimport.SuggestCoordinatesResponse, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencongregationimport.SuggestCoordinatesResponse{}, mapUpstreamErr(err)
	}
	result, err := s.appService.SuggestCoordinates(ctx, string(authHeader), personID, candidateIdArg)
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
