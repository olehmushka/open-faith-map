// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport implements the generated registration.RegistrationService (Conjure server
// interface): translates Conjure structs <-> domain types, resolves the caller's own person RID
// from their forwarded token (never a client-supplied id — always asked of go-oikumenea itself via
// whoami), and maps domain errors to the module's typed Conjure errors.
package transport

import (
	"context"

	genregistration "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/registration"
	"github.com/olehmushka/open-faith-map/internal/coreintegration"
	"github.com/olehmushka/open-faith-map/internal/registration/application"
	"github.com/olehmushka/open-faith-map/internal/registration/domain"
	"github.com/palantir/pkg/bearertoken"
	"github.com/palantir/pkg/datetime"
)

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

var _ genregistration.RegistrationService = (*Service)(nil)

// whoami resolves the caller's own go-oikumenea person RID from their forwarded token — never
// trusts a client-supplied id.
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

func (s *Service) SubmitRequest(ctx context.Context, authHeader bearertoken.Token, requestArg genregistration.SubmitRegistrationRequest) (genregistration.RegistrationRequest, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapUpstreamErr(err)
	}
	req, err := s.appService.Submit(ctx, string(authHeader), personID, toDomainSubmit(requestArg))
	if err != nil {
		return genregistration.RegistrationRequest{}, mapErr(err, "", "")
	}
	return toAPI(req), nil
}

func (s *Service) ListRequests(ctx context.Context, authHeader bearertoken.Token, statusArg *string, pageSizeArg *int, pageTokenArg *string) (genregistration.RegistrationRequestPage, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genregistration.RegistrationRequestPage{}, mapUpstreamErr(err)
	}
	var status *domain.Status
	if statusArg != nil {
		st := domain.Status(*statusArg)
		status = &st
	}
	reqs, err := s.appService.List(ctx, string(authHeader), personID, status)
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
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapUpstreamErr(err)
	}
	req, err := s.appService.Get(ctx, string(authHeader), personID, requestIdArg)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapErr(err, requestIdArg, "")
	}
	return toAPI(req), nil
}

func (s *Service) ApproveRequest(ctx context.Context, authHeader bearertoken.Token, requestIdArg string, requestArg genregistration.ApproveRegistrationRequest) (genregistration.RegistrationRequest, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapUpstreamErr(err)
	}
	req, err := s.appService.Approve(ctx, string(authHeader), personID, requestIdArg, requestArg.UnitCode)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapErr(err, requestIdArg, string(domain.StatusPending))
	}
	return toAPI(req), nil
}

func (s *Service) RejectRequest(ctx context.Context, authHeader bearertoken.Token, requestIdArg string, requestArg genregistration.RejectRegistrationRequest) (genregistration.RegistrationRequest, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapUpstreamErr(err)
	}
	req, err := s.appService.Reject(ctx, personID, requestIdArg, requestArg.Reason)
	if err != nil {
		return genregistration.RegistrationRequest{}, mapErr(err, requestIdArg, string(domain.StatusPending))
	}
	return toAPI(req), nil
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
		CreatedAt:           datetime.DateTime(r.CreatedAt),
		UpdatedAt:           datetime.DateTime(r.UpdatedAt),
	}
}
