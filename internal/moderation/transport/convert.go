// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	genmoderation "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/moderation"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
	"github.com/palantir/pkg/datetime"
)

func toAPIReport(r domain.Report) genmoderation.Report {
	return genmoderation.Report{
		Id:               r.ID,
		TargetKind:       genmoderation.New_TargetKind(genmoderation.TargetKind_Value(r.TargetKind)),
		TargetRef:        r.TargetRef,
		ReasonCode:       genmoderation.New_ReasonCode(genmoderation.ReasonCode_Value(r.ReasonCode)),
		Detail:           r.Detail,
		ReporterPersonId: r.ReporterPersonID,
		QueueScope:       genmoderation.New_QueueScope(genmoderation.QueueScope_Value(r.QueueScope)),
		Status:           genmoderation.New_ReportStatus(genmoderation.ReportStatus_Value(r.Status)),
		CreatedAt:        datetime.DateTime(r.CreatedAt),
		UpdatedAt:        datetime.DateTime(r.UpdatedAt),
	}
}

func toAPIReports(reports []domain.Report) []genmoderation.Report {
	out := make([]genmoderation.Report, 0, len(reports))
	for _, r := range reports {
		out = append(out, toAPIReport(r))
	}
	return out
}

func toAPIAction(a domain.Action) genmoderation.ModerationAction {
	return genmoderation.ModerationAction{
		Id:                 a.ID,
		ReportId:           a.ReportID,
		ActionKind:         genmoderation.New_ActionKind(genmoderation.ActionKind_Value(a.ActionKind)),
		TargetKind:         genmoderation.New_TargetKind(genmoderation.TargetKind_Value(a.TargetKind)),
		TargetRef:          a.TargetRef,
		ActorPersonId:      a.ActorPersonID,
		Reason:             a.Reason,
		ReversedByActionId: a.ReversedByActionID,
		CreatedAt:          datetime.DateTime(a.CreatedAt),
	}
}

func toAPIAppeal(a domain.Appeal) genmoderation.Appeal {
	return genmoderation.Appeal{
		Id:                        a.ID,
		ActionId:                  a.ActionID,
		CongregationAdminPersonId: a.CongregationAdminPersonID,
		Statement:                 a.Statement,
		AssignedModeratorPersonId: a.AssignedModeratorPersonID,
		Status:                    genmoderation.New_AppealStatus(genmoderation.AppealStatus_Value(a.Status)),
		CreatedAt:                 datetime.DateTime(a.CreatedAt),
		UpdatedAt:                 datetime.DateTime(a.UpdatedAt),
	}
}

func toAPIAppeals(appeals []domain.Appeal) []genmoderation.Appeal {
	out := make([]genmoderation.Appeal, 0, len(appeals))
	for _, a := range appeals {
		out = append(out, toAPIAppeal(a))
	}
	return out
}
