// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport implements the generated vouching.VouchingService (Conjure server interface):
// translates Conjure structs <-> domain types, resolves the caller's own person RID from their
// forwarded token (never a client-supplied id — always asked of go-oikumenea itself via whoami),
// and maps domain errors to this module's typed Conjure errors.
package transport

import (
	"context"
	"errors"

	genvouching "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/vouching"
	"github.com/olehmushka/open-faith-map/internal/coreintegration"
	"github.com/olehmushka/open-faith-map/internal/vouching/application"
	"github.com/olehmushka/open-faith-map/internal/vouching/domain"
	"github.com/palantir/pkg/bearertoken"
)

// defaultPageSize matches every other module's own unspecified-pageSize fallback.
const defaultPageSize = 50

type Config struct {
	OikumeneaBaseURL            string
	OikumeneaInsecureSkipVerify bool
}

type Service struct {
	oikumeneaBaseURL  string
	oikumeneaInsecure bool
	appService        *application.Service
}

func NewService(appService *application.Service, cfg Config) *Service {
	return &Service{appService: appService, oikumeneaBaseURL: cfg.OikumeneaBaseURL, oikumeneaInsecure: cfg.OikumeneaInsecureSkipVerify}
}

var _ genvouching.VouchingService = (*Service)(nil)

// whoami resolves the caller's own go-oikumenea person RID from their forwarded token — never
// trusts a client-supplied id (same pattern every other module's transport already uses).
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

func pageSizeOrDefault(p *int) int {
	if p == nil || *p <= 0 {
		return defaultPageSize
	}
	return *p
}

func (s *Service) CreateVouch(ctx context.Context, authHeader bearertoken.Token, requestArg genvouching.CreateVouchRequest) (genvouching.Vouch, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genvouching.Vouch{}, err
	}
	vouch, err := s.appService.CreateVouch(ctx, string(authHeader), personID, domain.CreateVouchInput{
		ClaimantPersonRID:           requestArg.ClaimantPersonId,
		CongregationUnitID:          requestArg.CongregationUnitId,
		GuarantorCongregationUnitID: requestArg.GuarantorCongregationUnitId,
		Statement:                   requestArg.Statement,
	})
	if err != nil {
		return genvouching.Vouch{}, mapErr(err, errCtx{GuarantorPersonID: personID})
	}
	return toAPIVouch(vouch), nil
}

func (s *Service) ListVouches(ctx context.Context, authHeader bearertoken.Token, claimantArg, congregationArg *string, pageSizeArg *int, pageTokenArg *string) (genvouching.VouchPage, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genvouching.VouchPage{}, err
	}
	vouches, err := s.appService.ListVouches(ctx, string(authHeader), personID, claimantArg, congregationArg, pageSizeOrDefault(pageSizeArg))
	if err != nil {
		return genvouching.VouchPage{}, mapErr(err, errCtx{})
	}
	return genvouching.VouchPage{Vouches: toAPIVouches(vouches)}, nil
}

func (s *Service) RevokeGuarantor(ctx context.Context, authHeader bearertoken.Token, personRidArg string, requestArg genvouching.RevokeGuarantorRequest) (genvouching.GuarantorStatusRecord, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genvouching.GuarantorStatusRecord{}, err
	}
	status, err := s.appService.RevokeGuarantor(ctx, string(authHeader), personID, personRidArg, requestArg.Reason)
	// ErrGuarantorRevokeFanoutIncomplete wraps a status that was already committed successfully —
	// the caller (moderator) gets the real, current status back, not an error; the incomplete
	// fan-out is not this endpoint's contract to report (moderation.md's own reports queue is the
	// place to notice a missing report, not this response).
	if err != nil && !errors.Is(err, domain.ErrGuarantorRevokeFanoutIncomplete) {
		return genvouching.GuarantorStatusRecord{}, mapErr(err, errCtx{GuarantorPersonID: personRidArg})
	}
	return toAPIGuarantorStatus(status), nil
}

func (s *Service) GetGuarantorStatus(ctx context.Context, authHeader bearertoken.Token, personRidArg string) (genvouching.GuarantorStatusRecord, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genvouching.GuarantorStatusRecord{}, err
	}
	status, err := s.appService.GetGuarantorStatus(ctx, string(authHeader), personID, personRidArg)
	if err != nil {
		return genvouching.GuarantorStatusRecord{}, mapErr(err, errCtx{GuarantorPersonID: personRidArg})
	}
	return toAPIGuarantorStatus(status), nil
}
