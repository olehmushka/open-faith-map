// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application holds the vouching module's business logic: the target-scoped
// congregation-standing and platform-moderator gates (authorize.go), and the vouch/guarantor
// workflows (this file).
//
// M10.6: write authority is decided by internal/authz.Require against the request's context-resolved
// subject, not a per-call go-oikumenea client built from the caller's forwarded token. vouching has
// no genuinely-anonymous endpoint, unlike moderation/content/discovery, so every path goes through
// this same gate.
package application

import (
	"context"
	"errors"

	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/olehmushka/open-faith-map/internal/vouching/adapters"
	"github.com/olehmushka/open-faith-map/internal/vouching/domain"
)

type Config struct {
	// RootUnitID is the same shared root unit registration/content/discovery/moderation already
	// use (internal/platform/seed.RootUnitID) — the target of the platform-moderator-scoped
	// Require check.
	RootUnitID string
}

// GuarantorRevokedEvent is vouching's own vocabulary for a fan-out notification, translated into
// moderation's FileReportInput only by the ModerationReporter implementation at the composition
// root (cmd/openfaithmap-api/main.go) — this package never imports moderation's domain or
// application packages directly.
type GuarantorRevokedEvent struct {
	VouchID            string
	GuarantorPersonRID string
	ClaimantPersonRID  string
	CongregationUnitID string
	RevokedReason      string
}

// ModerationReporter is how RevokeGuarantor queues moderator review of a revoked guarantor's past
// vouches (vouching.md's invariant: revocation never silently invalidates them). Implemented by an
// adapter over moderation's own application.Service, wired at the composition root — an in-process
// interface call, same shape as discovery's ContentResolver (the only existing cross-module
// precedent in this repo), not an HTTP round-trip to itself.
type ModerationReporter interface {
	ReportGuarantorRevoked(ctx context.Context, event GuarantorRevokedEvent) error
}

type Service struct {
	store      *adapters.Repository
	moderation ModerationReporter
	authzSvc   *authz.Service
	cfg        Config
}

func NewService(store *adapters.Repository, moderation ModerationReporter, authzSvc *authz.Service, cfg Config) *Service {
	return &Service{store: store, moderation: moderation, authzSvc: authzSvc, cfg: cfg}
}

// CreateVouch answers VouchingService.createVouch: the caller (guarantor) must hold
// religionorg.manage on in.GuarantorCongregationUnitID — a unit of their own, deliberately
// independent of in.CongregationUnitID (the claim) — and must not currently hold REVOKED status,
// checked live against the store on every call (core-integration.md's no-shadow-authorization-state
// invariant: never cached).
func (s *Service) CreateVouch(ctx context.Context, callerPersonID string, in domain.CreateVouchInput) (domain.Vouch, error) {
	if err := s.requireCongregationStanding(ctx, in.GuarantorCongregationUnitID); err != nil {
		return domain.Vouch{}, err
	}
	status, err := s.store.GetGuarantorStatus(ctx, callerPersonID)
	if err != nil {
		return domain.Vouch{}, err
	}
	if err := domain.CanVouch(status); err != nil {
		return domain.Vouch{}, err
	}
	in.GuarantorPersonRID = callerPersonID
	return s.store.InsertVouch(ctx, in)
}

func (s *Service) ListVouches(ctx context.Context, claimant, congregation *string, pageSize int) ([]domain.Vouch, error) {
	if err := s.requireModerate(ctx); err != nil {
		return nil, err
	}
	return s.store.ListVouches(ctx, claimant, congregation, pageSize)
}

func (s *Service) GetGuarantorStatus(ctx context.Context, targetPersonRID string) (domain.GuarantorStatus, error) {
	if err := s.requireModerate(ctx); err != nil {
		return domain.GuarantorStatus{}, err
	}
	return s.store.GetGuarantorStatus(ctx, targetPersonRID)
}

// RevokeGuarantor answers VouchingService.revokeGuarantor. The revoked-status write is the
// load-bearing state change — "cannot vouch while revoked" must take effect immediately — and is
// committed before the fan-out begins. The fan-out itself is best-effort and non-transactional
// (matching moderation's own established multi-write style, e.g. TakeActionOnReport): every prior
// vouch by this guarantor gets one moderation report queued for review; a failure partway through
// does not roll back the revocation, and is surfaced by wrapping
// domain.ErrGuarantorRevokeFanoutIncomplete around the already-committed status.
func (s *Service) RevokeGuarantor(ctx context.Context, callerPersonID, targetPersonRID, reason string) (domain.GuarantorStatus, error) {
	if err := s.requireModerate(ctx); err != nil {
		return domain.GuarantorStatus{}, err
	}
	status, err := s.store.UpsertRevoked(ctx, targetPersonRID, reason, callerPersonID)
	if err != nil {
		return domain.GuarantorStatus{}, err
	}

	vouches, err := s.store.ListVouchesByGuarantor(ctx, targetPersonRID)
	if err != nil {
		return status, errors.Join(domain.ErrGuarantorRevokeFanoutIncomplete, err)
	}
	var fanoutErr error
	for _, v := range vouches {
		if err := s.moderation.ReportGuarantorRevoked(ctx, GuarantorRevokedEvent{
			VouchID:            v.ID,
			GuarantorPersonRID: v.GuarantorPersonRID,
			ClaimantPersonRID:  v.ClaimantPersonRID,
			CongregationUnitID: v.CongregationUnitID,
			RevokedReason:      reason,
		}); err != nil {
			fanoutErr = errors.Join(fanoutErr, err)
		}
	}
	if fanoutErr != nil {
		return status, errors.Join(domain.ErrGuarantorRevokeFanoutIncomplete, fanoutErr)
	}
	return status, nil
}
