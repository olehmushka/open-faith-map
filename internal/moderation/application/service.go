// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application holds the moderation module's business logic: the platform-moderator
// target-scoped gate (authorize.go), report/action/appeal workflows (service.go), and the
// standalone D-Exclusions dry-run (exclusion_check.go) — always deciding write authority against
// go-oikumenea's real PDP with the CALLER's own forwarded token (D-Facade), except where the caller
// is genuinely anonymous and has none to forward (the two ModerationPublicService endpoints), which
// use the server's own service-principal token instead.
package application

import (
	"context"
	"time"

	oikumenea "github.com/olehmushka/go-oikumenea/clients/go"
	"github.com/olehmushka/open-faith-map/internal/coreintegration"
	"github.com/olehmushka/open-faith-map/internal/moderation/adapters"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
)

type Config struct {
	OikumeneaBaseURL            string
	OikumeneaInsecureSkipVerify bool
	// RootUnitID is the same shared root unit registration/content/discovery already use
	// (scripts/bootstrap-registration-org) — the target of the platform-moderator-scoped Authorize
	// check.
	RootUnitID string
	// ServicePrincipal configures the server's own go-oikumenea call for CheckExclusion — never a
	// forwarded caller token, since POST /exclusion-check has no caller token to forward (the caller
	// is anonymous).
	ServicePrincipal coreintegration.Config
}

type Service struct {
	store *adapters.Store
	cfg   Config
}

func NewService(store *adapters.Store, cfg Config) *Service {
	return &Service{store: store, cfg: cfg}
}

func (s *Service) userClient(token string) (*oikumenea.Client, error) {
	return coreintegration.NewUserClient(s.cfg.OikumeneaBaseURL, token, s.cfg.OikumeneaInsecureSkipVerify)
}

func (s *Service) serviceClient(ctx context.Context) (*oikumenea.Client, error) {
	return coreintegration.NewServiceClient(ctx, s.cfg.ServicePrincipal)
}

// ---- reports ----

// FileReport answers ModerationPublicService.fileReport — genuinely anonymous, no caller identity
// asked or trusted (reporterPersonID always unset, matching Conjure's own doc comment on the
// endpoint).
//
// queueScope is always domain.ScopePlatform for now. moderation.md's own "Open seams" section
// already flags the CONGREGATION/JURISDICTION ancestor-chain walk needed to classify a report by
// its target's jurisdiction as "not yet wired" — and since M5's only moderator role
// (platform-moderator, D-PlatformModerator) is granted subtree on the ROOT unit, not scoped to any
// individual jurisdiction, there is no moderator authority boundary yet for a narrower scope to
// mean anything. Building that walk now would be dead code with nothing to filter for. The column
// and the GET /reports?scope= filter both exist and work mechanically; every report just lands in
// PLATFORM until a future milestone gives jurisdiction-scoped moderator authority a reason to exist.
func (s *Service) FileReport(ctx context.Context, in domain.FileReportInput) (domain.Report, error) {
	if err := domain.ValidateReasonCode(in.ReasonCode); err != nil {
		return domain.Report{}, err
	}
	return s.store.InsertReport(ctx, in, domain.ScopePlatform)
}

func (s *Service) ListReports(ctx context.Context, token, callerPersonID string, scope *domain.QueueScope, status *domain.ReportStatus, pageSize int) ([]domain.Report, error) {
	if err := s.requireModerate(ctx, token, callerPersonID); err != nil {
		return nil, err
	}
	return s.store.ListReports(ctx, scope, status, pageSize)
}

// ---- actions ----

// TakeActionOnReport answers ModerationService.takeActionOnReport: the target is derived from the
// report itself (transport passes reportID, application loads it), never re-specified by the
// caller. The moderation_actions row is written before any local effect is applied
// (D-Moderation's Correction replacement invariant) — this module ships no real go-oikumenea-side
// or content-side effect yet (moderation.md doesn't specify one in enough detail to build blind);
// the row IS the recorded decision. Marks the report ACTIONED.
func (s *Service) TakeActionOnReport(ctx context.Context, token, callerPersonID, reportID string, kind domain.ActionKind, reason string) (domain.Action, error) {
	if err := s.requireModerate(ctx, token, callerPersonID); err != nil {
		return domain.Action{}, err
	}
	report, err := s.store.GetReportByID(ctx, reportID)
	if err != nil {
		return domain.Action{}, err
	}
	action, err := s.store.InsertAction(ctx, domain.TakeActionInput{
		ReportID:      &report.ID,
		ActionKind:    kind,
		TargetKind:    report.TargetKind,
		TargetRef:     report.TargetRef,
		ActorPersonID: callerPersonID,
		Reason:        reason,
	})
	if err != nil {
		return domain.Action{}, err
	}
	if _, err := s.store.MarkReportStatus(ctx, report.ID, domain.ReportActioned); err != nil {
		return domain.Action{}, err
	}
	return action, nil
}

// TakeAction answers ModerationService.takeAction: a proactive action with no prior report (e.g.
// enforcing D-Exclusions directly against an already-registered congregation).
func (s *Service) TakeAction(ctx context.Context, token, callerPersonID string, kind domain.ActionKind, targetKind domain.TargetKind, targetRef, reason string) (domain.Action, error) {
	if err := s.requireModerate(ctx, token, callerPersonID); err != nil {
		return domain.Action{}, err
	}
	return s.store.InsertAction(ctx, domain.TakeActionInput{
		ActionKind:    kind,
		TargetKind:    targetKind,
		TargetRef:     targetRef,
		ActorPersonID: callerPersonID,
		Reason:        reason,
	})
}

// ReverseAction answers ModerationService.reverseAction: writes a new, append-only REVERSE row
// pointing backward at the original (domain.Action's doc comment) — the original row is never
// edited. Rejects with domain.ErrActionNotReversible if the grace window has passed or a reversal
// already exists (the store's unique index on reverses_action_id is the real guarantee; this check
// gives a clean typed error instead of a raw constraint-violation).
func (s *Service) ReverseAction(ctx context.Context, token, callerPersonID, actionID, reason string) (domain.Action, error) {
	if err := s.requireModerate(ctx, token, callerPersonID); err != nil {
		return domain.Action{}, err
	}
	original, err := s.store.GetActionByID(ctx, actionID)
	if err != nil {
		return domain.Action{}, err
	}
	if err := domain.CanReverse(original, time.Now()); err != nil {
		return domain.Action{}, err
	}
	return s.store.InsertAction(ctx, domain.TakeActionInput{
		ActionKind:       domain.ActionReverse,
		TargetKind:       original.TargetKind,
		TargetRef:        original.TargetRef,
		ActorPersonID:    callerPersonID,
		Reason:           reason,
		ReversesActionID: &original.ID,
	})
}

// ---- appeals ----

// FileAppeal answers ModerationService.fileAppeal: the caller must be the affected congregation's
// admin, verified live via requireCongregationAdmin against the action's own target unit — never a
// platform-moderator check (moderation.md: filing is the affected admin's act, not a moderator's).
//
// Scope cut: only CONGREGATION-kind actions can be appealed in this PR — for those, TargetRef IS
// the go-oikumenea unit RID directly. SITE/DOCUMENT actions would need resolving through the content
// module's own site->congregation-unit mapping first (a real cross-module lookup, same shape as
// discovery's ContentResolver interface), which this PR doesn't wire up; appealing those returns
// domain.ErrForbidden rather than silently succeeding or guessing a unit.
func (s *Service) FileAppeal(ctx context.Context, token, callerPersonID, actionID, statement string) (domain.Appeal, error) {
	action, err := s.store.GetActionByID(ctx, actionID)
	if err != nil {
		return domain.Appeal{}, err
	}
	if action.TargetKind != domain.TargetCongregation {
		return domain.Appeal{}, domain.ErrForbidden
	}
	if err := s.requireCongregationAdmin(ctx, token, callerPersonID, action.TargetRef); err != nil {
		return domain.Appeal{}, err
	}
	return s.store.InsertAppeal(ctx, action.ID, callerPersonID, statement)
}

func (s *Service) ListAppeals(ctx context.Context, token, callerPersonID string, status *domain.AppealStatus, pageSize int) ([]domain.Appeal, error) {
	if err := s.requireModerate(ctx, token, callerPersonID); err != nil {
		return nil, err
	}
	return s.store.ListAppeals(ctx, status, pageSize)
}

// DecideAppeal answers ModerationService.decideAppeal. Rejects with domain.ErrAppealActorConflict if
// the caller is the original action's own actor — enforced here at write time, never left to
// moderator discipline (moderation.md's invariant).
func (s *Service) DecideAppeal(ctx context.Context, token, callerPersonID, appealID string, decision domain.AppealDecision) (domain.Appeal, error) {
	if err := s.requireModerate(ctx, token, callerPersonID); err != nil {
		return domain.Appeal{}, err
	}
	appeal, err := s.store.GetAppealByID(ctx, appealID)
	if err != nil {
		return domain.Appeal{}, err
	}
	action, err := s.store.GetActionByID(ctx, appeal.ActionID)
	if err != nil {
		return domain.Appeal{}, err
	}
	if err := domain.CanDecideAppeal(action.ActorPersonID, callerPersonID); err != nil {
		return domain.Appeal{}, err
	}
	status := domain.AppealOverturned
	if decision == domain.DecisionUphold {
		status = domain.AppealUpheld
	}
	return s.store.DecideAppeal(ctx, appealID, callerPersonID, status)
}
