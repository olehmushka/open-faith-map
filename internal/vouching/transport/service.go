// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport implements the generated vouching.VouchingService (Conjure server interface):
// translates Conjure structs <-> domain types and maps domain errors to this module's typed Conjure
// errors.
//
// M10.6: the caller's identity no longer arrives via a per-request whoami round-trip — it's resolved
// from context (populated by internal/identity's authenticator middleware) via personID below, the
// same pattern internal/registration/transport already uses.
package transport

import (
	"context"
	"errors"

	"github.com/olehmushka/open-faith-map/internal/authz"
	genvouching "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/vouching"
	"github.com/olehmushka/open-faith-map/internal/vouching/application"
	"github.com/olehmushka/open-faith-map/internal/vouching/domain"
	"github.com/palantir/pkg/bearertoken"
)

// defaultPageSize matches every other module's own unspecified-pageSize fallback.
const defaultPageSize = 50

type Service struct {
	appService *application.Service
}

func NewService(appService *application.Service) *Service {
	return &Service{appService: appService}
}

var _ genvouching.VouchingService = (*Service)(nil)

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

func pageSizeOrDefault(p *int) int {
	if p == nil || *p <= 0 {
		return defaultPageSize
	}
	return *p
}

func (s *Service) CreateVouch(ctx context.Context, authHeader bearertoken.Token, requestArg genvouching.CreateVouchRequest) (genvouching.Vouch, error) {
	pid, err := personID(ctx)
	if err != nil {
		return genvouching.Vouch{}, err
	}
	vouch, err := s.appService.CreateVouch(ctx, pid, domain.CreateVouchInput{
		ClaimantPersonRID:           requestArg.ClaimantPersonId,
		CongregationUnitID:          requestArg.CongregationUnitId,
		GuarantorCongregationUnitID: requestArg.GuarantorCongregationUnitId,
		Statement:                   requestArg.Statement,
	})
	if err != nil {
		return genvouching.Vouch{}, mapErr(err, errCtx{GuarantorPersonID: pid})
	}
	return toAPIVouch(vouch), nil
}

func (s *Service) ListVouches(ctx context.Context, authHeader bearertoken.Token, claimantArg, congregationArg *string, pageSizeArg *int, pageTokenArg *string) (genvouching.VouchPage, error) {
	if _, err := personID(ctx); err != nil {
		return genvouching.VouchPage{}, err
	}
	vouches, err := s.appService.ListVouches(ctx, claimantArg, congregationArg, pageSizeOrDefault(pageSizeArg))
	if err != nil {
		return genvouching.VouchPage{}, mapErr(err, errCtx{})
	}
	return genvouching.VouchPage{Vouches: toAPIVouches(vouches)}, nil
}

func (s *Service) RevokeGuarantor(ctx context.Context, authHeader bearertoken.Token, personRidArg string, requestArg genvouching.RevokeGuarantorRequest) (genvouching.GuarantorStatusRecord, error) {
	pid, err := personID(ctx)
	if err != nil {
		return genvouching.GuarantorStatusRecord{}, err
	}
	status, err := s.appService.RevokeGuarantor(ctx, pid, personRidArg, requestArg.Reason)
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
	if _, err := personID(ctx); err != nil {
		return genvouching.GuarantorStatusRecord{}, err
	}
	status, err := s.appService.GetGuarantorStatus(ctx, personRidArg)
	if err != nil {
		return genvouching.GuarantorStatusRecord{}, mapErr(err, errCtx{GuarantorPersonID: personRidArg})
	}
	return toAPIGuarantorStatus(status), nil
}
