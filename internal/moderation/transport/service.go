// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"context"

	genmoderation "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/moderation"
	"github.com/olehmushka/open-faith-map/internal/coreintegration"
	"github.com/olehmushka/open-faith-map/internal/moderation/application"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
	"github.com/palantir/pkg/bearertoken"
)

// defaultPageSize matches registration/content's own unspecified-pageSize fallback.
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

var _ genmoderation.ModerationService = (*Service)(nil)

// whoami resolves the caller's own go-oikumenea person RID from their forwarded token — never
// trusts a client-supplied id (same pattern content/registration transport already use).
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

func (s *Service) ListReports(ctx context.Context, authHeader bearertoken.Token, scopeArg *genmoderation.QueueScope, statusArg *genmoderation.ReportStatus, pageSizeArg *int, pageTokenArg *string) (genmoderation.ReportPage, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genmoderation.ReportPage{}, err
	}
	var scope *domain.QueueScope
	if scopeArg != nil {
		v := domain.QueueScope(scopeArg.Value())
		scope = &v
	}
	var status *domain.ReportStatus
	if statusArg != nil {
		v := domain.ReportStatus(statusArg.Value())
		status = &v
	}
	reports, err := s.appService.ListReports(ctx, string(authHeader), personID, scope, status, pageSizeOrDefault(pageSizeArg))
	if err != nil {
		return genmoderation.ReportPage{}, mapErr(err, errCtx{})
	}
	return genmoderation.ReportPage{Reports: toAPIReports(reports)}, nil
}

func (s *Service) TakeActionOnReport(ctx context.Context, authHeader bearertoken.Token, reportIdArg string, requestArg genmoderation.TakeActionOnReportRequest) (genmoderation.ModerationAction, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genmoderation.ModerationAction{}, err
	}
	action, err := s.appService.TakeActionOnReport(ctx, string(authHeader), personID, reportIdArg, domain.ActionKind(requestArg.ActionKind.Value()), requestArg.Reason)
	if err != nil {
		return genmoderation.ModerationAction{}, mapErr(err, errCtx{ReportID: reportIdArg})
	}
	return toAPIAction(action), nil
}

func (s *Service) TakeAction(ctx context.Context, authHeader bearertoken.Token, requestArg genmoderation.TakeActionRequest) (genmoderation.ModerationAction, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genmoderation.ModerationAction{}, err
	}
	action, err := s.appService.TakeAction(ctx, string(authHeader), personID,
		domain.ActionKind(requestArg.ActionKind.Value()), domain.TargetKind(requestArg.TargetKind.Value()), requestArg.TargetRef, requestArg.Reason)
	if err != nil {
		return genmoderation.ModerationAction{}, mapErr(err, errCtx{})
	}
	return toAPIAction(action), nil
}

func (s *Service) ReverseAction(ctx context.Context, authHeader bearertoken.Token, actionIdArg string, requestArg genmoderation.ReverseActionRequest) (genmoderation.ModerationAction, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genmoderation.ModerationAction{}, err
	}
	action, err := s.appService.ReverseAction(ctx, string(authHeader), personID, actionIdArg, requestArg.Reason)
	if err != nil {
		return genmoderation.ModerationAction{}, mapErr(err, errCtx{ActionID: actionIdArg})
	}
	return toAPIAction(action), nil
}

func (s *Service) FileAppeal(ctx context.Context, authHeader bearertoken.Token, actionIdArg string, requestArg genmoderation.FileAppealRequest) (genmoderation.Appeal, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genmoderation.Appeal{}, err
	}
	appeal, err := s.appService.FileAppeal(ctx, string(authHeader), personID, actionIdArg, requestArg.Statement)
	if err != nil {
		return genmoderation.Appeal{}, mapErr(err, errCtx{ActionID: actionIdArg})
	}
	return toAPIAppeal(appeal), nil
}

func (s *Service) ListAppeals(ctx context.Context, authHeader bearertoken.Token, statusArg *genmoderation.AppealStatus, pageSizeArg *int, pageTokenArg *string) (genmoderation.AppealPage, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genmoderation.AppealPage{}, err
	}
	var status *domain.AppealStatus
	if statusArg != nil {
		v := domain.AppealStatus(statusArg.Value())
		status = &v
	}
	appeals, err := s.appService.ListAppeals(ctx, string(authHeader), personID, status, pageSizeOrDefault(pageSizeArg))
	if err != nil {
		return genmoderation.AppealPage{}, mapErr(err, errCtx{})
	}
	return genmoderation.AppealPage{Appeals: toAPIAppeals(appeals)}, nil
}

func (s *Service) DecideAppeal(ctx context.Context, authHeader bearertoken.Token, appealIdArg string, requestArg genmoderation.DecideAppealRequest) (genmoderation.Appeal, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genmoderation.Appeal{}, err
	}
	appeal, err := s.appService.DecideAppeal(ctx, string(authHeader), personID, appealIdArg, domain.AppealDecision(requestArg.Decision.Value()))
	if err != nil {
		return genmoderation.Appeal{}, mapErr(err, errCtx{AppealID: appealIdArg})
	}
	return toAPIAppeal(appeal), nil
}
