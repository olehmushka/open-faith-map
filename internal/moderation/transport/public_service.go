// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport implements the generated moderation.ModerationService and
// moderation.ModerationPublicService (Conjure server interfaces): translates Conjure structs <->
// domain types, resolves the caller's own person RID from their forwarded token (never a
// client-supplied id — always asked of go-oikumenea itself via whoami), and maps domain errors to
// this module's typed Conjure errors.
package transport

import (
	"context"

	genmoderation "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/moderation"
	"github.com/olehmushka/open-faith-map/internal/moderation/application"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
	"github.com/palantir/pkg/metrics"
)

// PublicService implements the generated ModerationPublicService — genuinely anonymous, no
// bearertoken.Token parameter anywhere (confirmed by the generated interface: this service declares
// no default-auth in api/moderation.conjure.yml, so conjure-go emits no auth arg at all).
type PublicService struct {
	appService *application.Service
}

func NewPublicService(appService *application.Service) *PublicService {
	return &PublicService{appService: appService}
}

var _ genmoderation.ModerationPublicService = (*PublicService)(nil)

func (s *PublicService) FileReport(ctx context.Context, requestArg genmoderation.FileReportRequest) (genmoderation.Report, error) {
	report, err := s.appService.FileReport(ctx, domain.FileReportInput{
		TargetKind: domain.TargetKind(requestArg.TargetKind.Value()),
		TargetRef:  requestArg.TargetRef,
		// .String(), not .Value(): .Value() collapses ANY unrecognized enum string to "UNKNOWN"
		// before domain.ValidateReasonCode ever sees it, which would silently defeat the specific
		// DOCTRINAL_CONCERN check it exists to make (found live: a DOCTRINAL_CONCERN report fell
		// through to a raw DB CHECK-constraint 500 instead of Moderation:... until this fix).
		// .String() preserves the raw wire value; a genuinely unknown string still fails the same
		// DB CHECK constraint either way, matching every other module's enum-handling convention.
		ReasonCode: domain.ReasonCode(requestArg.ReasonCode.String()),
		Detail:     requestArg.Detail,
	})
	if err != nil {
		return genmoderation.Report{}, mapErr(err, errCtx{})
	}
	// Counted here, transport, not application.Service.FileReport — that method is also called
	// internally by vouching's revocation fan-out, bypassing this genuinely-anonymous-public
	// endpoint entirely (M7, docs/modules/hardening.md: reports_filed means "a public report was
	// filed via this endpoint," not "the domain operation ran").
	metrics.FromContext(ctx).Counter("openfaithmap.moderation.reports_filed").Inc(1)
	return toAPIReport(report), nil
}

func (s *PublicService) CheckExclusion(ctx context.Context, requestArg genmoderation.ExclusionCheckRequest) (genmoderation.ExclusionCheckResult, error) {
	excluded, code, err := s.appService.CheckExclusion(ctx, requestArg.TaxonId)
	if err != nil {
		return genmoderation.ExclusionCheckResult{}, mapErr(err, errCtx{TaxonID: requestArg.TaxonId})
	}
	metrics.FromContext(ctx).Counter("openfaithmap.moderation.exclusion_checks_run").Inc(1)
	result := genmoderation.ExclusionCheckResult{Excluded: excluded}
	if excluded {
		result.ExcludedTaxonCode = &code
	}
	return result, nil
}
