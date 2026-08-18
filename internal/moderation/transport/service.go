// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport implements the generated moderation.ModerationService and
// moderation.ModerationPublicService (Conjure server interfaces): translates Conjure structs <->
// domain types and maps domain errors to this module's typed Conjure errors.
//
// M10.6: the caller's identity no longer arrives via a per-request whoami round-trip — it's resolved
// from context (populated by internal/identity's authenticator middleware) via personID below, the
// same pattern internal/registration/transport already uses.
package transport

import (
	"context"

	"github.com/olehmushka/open-faith-map/internal/authz"
	genmoderation "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/moderation"
	"github.com/olehmushka/open-faith-map/internal/moderation/application"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
	"github.com/palantir/pkg/bearertoken"
)

// defaultPageSize matches registration/content's own unspecified-pageSize fallback.
const defaultPageSize = 50

// maxPageSize (M7, docs/modules/hardening.md) is a provisional ceiling — a 4x margin over
// defaultPageSize, generous for a moderator paging a backlog by hand in the admin console (the only
// real caller today; no bulk-export use case). Not data-tuned, same as the rate-limit thresholds.
const maxPageSize = 200

type Service struct {
	appService *application.Service
}

func NewService(appService *application.Service) *Service {
	return &Service{appService: appService}
}

var _ genmoderation.ModerationService = (*Service)(nil)

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
	if *p > maxPageSize {
		return maxPageSize
	}
	return *p
}

func (s *Service) ListReports(ctx context.Context, authHeader bearertoken.Token, scopeArg *genmoderation.QueueScope, statusArg *genmoderation.ReportStatus, pageSizeArg *int, pageTokenArg *string) (genmoderation.ReportPage, error) {
	if _, err := personID(ctx); err != nil {
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
	var after *domain.PageCursor
	if pageTokenArg != nil {
		c, err := decodeCursor(*pageTokenArg)
		if err != nil {
			return genmoderation.ReportPage{}, genmoderation.NewInvalidPageToken()
		}
		after = &c
	}
	pageSize := pageSizeOrDefault(pageSizeArg)
	reports, err := s.appService.ListReports(ctx, scope, status, pageSize, after)
	if err != nil {
		return genmoderation.ReportPage{}, mapErr(err, errCtx{})
	}
	var nextToken *string
	if len(reports) > pageSize {
		last := reports[pageSize-1]
		t := encodeCursor(domain.PageCursor{CreatedAt: last.CreatedAt, ID: last.ID})
		nextToken = &t
		reports = reports[:pageSize]
	}
	return genmoderation.ReportPage{Reports: toAPIReports(reports), NextPageToken: nextToken}, nil
}

func (s *Service) TakeActionOnReport(ctx context.Context, authHeader bearertoken.Token, reportIdArg string, requestArg genmoderation.TakeActionOnReportRequest) (genmoderation.ModerationAction, error) {
	pid, err := personID(ctx)
	if err != nil {
		return genmoderation.ModerationAction{}, err
	}
	action, err := s.appService.TakeActionOnReport(ctx, pid, reportIdArg, domain.ActionKind(requestArg.ActionKind.Value()), requestArg.Reason)
	if err != nil {
		return genmoderation.ModerationAction{}, mapErr(err, errCtx{ReportID: reportIdArg})
	}
	return toAPIAction(action), nil
}

func (s *Service) TakeAction(ctx context.Context, authHeader bearertoken.Token, requestArg genmoderation.TakeActionRequest) (genmoderation.ModerationAction, error) {
	pid, err := personID(ctx)
	if err != nil {
		return genmoderation.ModerationAction{}, err
	}
	action, err := s.appService.TakeAction(ctx, pid,
		domain.ActionKind(requestArg.ActionKind.Value()), domain.TargetKind(requestArg.TargetKind.Value()), requestArg.TargetRef, requestArg.Reason)
	if err != nil {
		return genmoderation.ModerationAction{}, mapErr(err, errCtx{})
	}
	return toAPIAction(action), nil
}

func (s *Service) ReverseAction(ctx context.Context, authHeader bearertoken.Token, actionIdArg string, requestArg genmoderation.ReverseActionRequest) (genmoderation.ModerationAction, error) {
	pid, err := personID(ctx)
	if err != nil {
		return genmoderation.ModerationAction{}, err
	}
	action, err := s.appService.ReverseAction(ctx, pid, actionIdArg, requestArg.Reason)
	if err != nil {
		return genmoderation.ModerationAction{}, mapErr(err, errCtx{ActionID: actionIdArg})
	}
	return toAPIAction(action), nil
}

func (s *Service) FileAppeal(ctx context.Context, authHeader bearertoken.Token, actionIdArg string, requestArg genmoderation.FileAppealRequest) (genmoderation.Appeal, error) {
	pid, err := personID(ctx)
	if err != nil {
		return genmoderation.Appeal{}, err
	}
	appeal, err := s.appService.FileAppeal(ctx, pid, actionIdArg, requestArg.Statement)
	if err != nil {
		return genmoderation.Appeal{}, mapErr(err, errCtx{ActionID: actionIdArg})
	}
	return toAPIAppeal(appeal), nil
}

func (s *Service) ListAppeals(ctx context.Context, authHeader bearertoken.Token, statusArg *genmoderation.AppealStatus, pageSizeArg *int, pageTokenArg *string) (genmoderation.AppealPage, error) {
	if _, err := personID(ctx); err != nil {
		return genmoderation.AppealPage{}, err
	}
	var status *domain.AppealStatus
	if statusArg != nil {
		v := domain.AppealStatus(statusArg.Value())
		status = &v
	}
	var after *domain.PageCursor
	if pageTokenArg != nil {
		c, err := decodeCursor(*pageTokenArg)
		if err != nil {
			return genmoderation.AppealPage{}, genmoderation.NewInvalidPageToken()
		}
		after = &c
	}
	pageSize := pageSizeOrDefault(pageSizeArg)
	appeals, err := s.appService.ListAppeals(ctx, status, pageSize, after)
	if err != nil {
		return genmoderation.AppealPage{}, mapErr(err, errCtx{})
	}
	var nextToken *string
	if len(appeals) > pageSize {
		last := appeals[pageSize-1]
		t := encodeCursor(domain.PageCursor{CreatedAt: last.CreatedAt, ID: last.ID})
		nextToken = &t
		appeals = appeals[:pageSize]
	}
	return genmoderation.AppealPage{Appeals: toAPIAppeals(appeals), NextPageToken: nextToken}, nil
}

func (s *Service) DecideAppeal(ctx context.Context, authHeader bearertoken.Token, appealIdArg string, requestArg genmoderation.DecideAppealRequest) (genmoderation.Appeal, error) {
	pid, err := personID(ctx)
	if err != nil {
		return genmoderation.Appeal{}, err
	}
	appeal, err := s.appService.DecideAppeal(ctx, pid, appealIdArg, domain.AppealDecision(requestArg.Decision.Value()))
	if err != nil {
		return genmoderation.Appeal{}, mapErr(err, errCtx{AppealID: appealIdArg})
	}
	return toAPIAppeal(appeal), nil
}
