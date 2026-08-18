// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport implements the generated registration.RegistrationService (Conjure server
// interface): translates Conjure structs <-> domain types, resolves the caller's own person RID
// from the request context (M10.6: the identity middleware already resolved and verified it —
// server.WithMiddleware(authenticator.Handle) is attached; there is no separate whoami round-trip
// any more), and maps domain errors to the module's typed Conjure errors.
package transport

import (
	"context"

	"github.com/olehmushka/open-faith-map/internal/authz"
	genregistration "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/registration"
	"github.com/olehmushka/open-faith-map/internal/registration/application"
	"github.com/olehmushka/open-faith-map/internal/registration/domain"
	"github.com/palantir/pkg/bearertoken"
	"github.com/palantir/pkg/datetime"
)

type Service struct {
	appService *application.Service
}

func NewService(appService *application.Service) *Service {
	return &Service{appService: appService}
}

var _ genregistration.RegistrationService = (*Service)(nil)

// personID resolves the caller's own person RID from the request context, populated by
// internal/identity's authenticator middleware — never trusted from a client-supplied value. Its
// only failure mode here is defensive: server.WithMiddleware(authenticator.Handle) already refuses
// any request with no valid subject before a handler ever runs, so a missing subject at this point
// means the middleware isn't actually wired, not a normal per-request condition.
func personID(ctx context.Context) (string, error) {
	subject, ok := authz.SubjectFromContext(ctx)
	if !ok || subject.PersonID == "" {
		return "", domain.ErrNotFound
	}
	return subject.PersonID, nil
}

func (s *Service) SubmitRequest(ctx context.Context, authHeader bearertoken.Token, requestArg genregistration.SubmitRegistrationRequest) (genregistration.RegistrationRequest, error) {
	pid, err := personID(ctx)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapErr(err, "", "")
	}
	req, err := s.appService.Submit(ctx, pid, toDomainSubmit(requestArg))
	if err != nil {
		return genregistration.RegistrationRequest{}, mapErr(err, "", "")
	}
	return toAPI(req), nil
}

func (s *Service) ListRequests(ctx context.Context, authHeader bearertoken.Token, statusArg *string, pageSizeArg *int, pageTokenArg *string) (genregistration.RegistrationRequestPage, error) {
	pid, err := personID(ctx)
	if err != nil {
		return genregistration.RegistrationRequestPage{}, mapErr(err, "", "")
	}
	var status *domain.Status
	if statusArg != nil {
		st := domain.Status(*statusArg)
		status = &st
	}
	reqs, err := s.appService.List(ctx, pid, status)
	if err != nil {
		return genregistration.RegistrationRequestPage{}, mapErr(err, "", "")
	}
	out := make([]genregistration.RegistrationRequest, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, toAPI(r))
	}
	return genregistration.RegistrationRequestPage{Requests: out}, nil
}

func (s *Service) GetRequest(ctx context.Context, authHeader bearertoken.Token, requestIdArg string) (genregistration.RegistrationRequest, error) {
	pid, err := personID(ctx)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapErr(err, "", "")
	}
	req, err := s.appService.Get(ctx, pid, requestIdArg)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapErr(err, requestIdArg, "")
	}
	return toAPI(req), nil
}

func (s *Service) ApproveRequest(ctx context.Context, authHeader bearertoken.Token, requestIdArg string, requestArg genregistration.ApproveRegistrationRequest) (genregistration.RegistrationRequest, error) {
	pid, err := personID(ctx)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapErr(err, "", "")
	}
	req, err := s.appService.Approve(ctx, pid, requestIdArg, requestArg.UnitCode, requestArg.JurisdictionUnitId)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapErr(err, requestIdArg, string(domain.StatusPending))
	}
	return toAPI(req), nil
}

func (s *Service) RejectRequest(ctx context.Context, authHeader bearertoken.Token, requestIdArg string, requestArg genregistration.RejectRegistrationRequest) (genregistration.RegistrationRequest, error) {
	pid, err := personID(ctx)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapErr(err, "", "")
	}
	req, err := s.appService.Reject(ctx, pid, requestIdArg, requestArg.Reason)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapErr(err, requestIdArg, string(domain.StatusPending))
	}
	return toAPI(req), nil
}

func (s *Service) ReparentRequest(ctx context.Context, authHeader bearertoken.Token, requestIdArg string, requestArg genregistration.ReparentRegistrationRequest) (genregistration.ReparentingJob, error) {
	pid, err := personID(ctx)
	if err != nil {
		return genregistration.ReparentingJob{}, mapErr(err, "", "")
	}
	job, err := s.appService.Reparent(ctx, pid, requestIdArg, requestArg.NewParentUnitId)
	if err != nil {
		return genregistration.ReparentingJob{}, mapErr(err, requestIdArg, string(domain.StatusApproved))
	}
	return toAPIReparentJob(job), nil
}

func (s *Service) GetReparentStatus(ctx context.Context, authHeader bearertoken.Token, requestIdArg string) (*genregistration.ReparentingJob, error) {
	if _, err := personID(ctx); err != nil {
		return nil, mapErr(err, "", "")
	}
	job, err := s.appService.GetReparentStatus(ctx, requestIdArg)
	if err != nil {
		return nil, mapErr(err, requestIdArg, "")
	}
	if job == nil {
		return nil, nil
	}
	out := toAPIReparentJob(*job)
	return &out, nil
}

func toAPIReparentJob(j domain.ReparentingJob) genregistration.ReparentingJob {
	return genregistration.ReparentingJob{
		Id:                    j.ID,
		RegistrationRequestId: j.RegistrationRequestID,
		CongregationUnitId:    j.CongregationUnitID,
		OldParentUnitId:       j.OldParentUnitID,
		NewParentUnitId:       j.NewParentUnitID,
		Status:                genregistration.New_ReparentStatus(genregistration.ReparentStatus_Value(j.Status)),
		PerformedByPersonId:   j.PerformedByPersonID,
		Error:                 j.Error,
		CreatedAt:             datetime.DateTime(j.CreatedAt),
		UpdatedAt:             datetime.DateTime(j.UpdatedAt),
	}
}

func toDomainSubmit(r genregistration.SubmitRegistrationRequest) domain.SubmitInput {
	return domain.SubmitInput{
		TaxonID:          r.TaxonId,
		CongregationName: r.CongregationName,
		CountryID:        r.CountryId,
		AdminArea1:       r.AdminArea1,
		Locality:         r.Locality,
		Street:           r.Street,
		HouseNumber:      r.HouseNumber,
		PostalCode:       r.PostalCode,
		Coordinate: domain.Coordinate{
			Latitude:  r.Coordinate.Latitude,
			Longitude: r.Coordinate.Longitude,
		},
	}
}

func toAPI(r domain.Request) genregistration.RegistrationRequest {
	var decidedAt *datetime.DateTime
	if r.DecidedAt != nil {
		dt := datetime.DateTime(*r.DecidedAt)
		decidedAt = &dt
	}
	return genregistration.RegistrationRequest{
		Id:                  r.ID,
		SubmittedByPersonId: r.SubmittedByPersonID,
		TaxonId:             r.TaxonID,
		CongregationName:    r.CongregationName,
		CountryId:           r.CountryID,
		AdminArea1:          r.AdminArea1,
		Locality:            r.Locality,
		Street:              r.Street,
		HouseNumber:         r.HouseNumber,
		PostalCode:          r.PostalCode,
		Coordinate:          genregistration.Coordinate{Latitude: r.Coordinate.Latitude, Longitude: r.Coordinate.Longitude},
		Status:              genregistration.New_RegistrationStatus(genregistration.RegistrationStatus_Value(r.Status)),
		RejectionReason:     r.RejectionReason,
		DecidedByPersonId:   r.DecidedByPersonID,
		DecidedAt:           decidedAt,
		CreatedUnitId:       r.CreatedUnitID,
		JurisdictionUnitId:  r.JurisdictionUnitID,
		CreatedAt:           datetime.DateTime(r.CreatedAt),
		UpdatedAt:           datetime.DateTime(r.UpdatedAt),
	}
}
