// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M10.5.5: the composition root's shared wiring state. Split out of what was one flat 473-line
// initServer function (docs/milestones.md's own M10.5.5 row) so each module's registration is a
// small, independently readable register<Module>(ctx, info, deps) function — done before M10.6's
// cutover, not after, so the composition root isn't being refactored at the same time it's the thing
// most likely to be wrong.
package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/authz"
	contentapplication "github.com/olehmushka/open-faith-map/internal/content/application"
	contentdomain "github.com/olehmushka/open-faith-map/internal/content/domain"
	directoryapplication "github.com/olehmushka/open-faith-map/internal/directory/application"
	locationapplication "github.com/olehmushka/open-faith-map/internal/location/application"
	membershipapplication "github.com/olehmushka/open-faith-map/internal/membership/application"
	moderationapplication "github.com/olehmushka/open-faith-map/internal/moderation/application"
	moderationdomain "github.com/olehmushka/open-faith-map/internal/moderation/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/config"
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
	refdataapplication "github.com/olehmushka/open-faith-map/internal/refdata/application"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
	vouchingapplication "github.com/olehmushka/open-faith-map/internal/vouching/application"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
)

// Deps is the shared state every register<Module> function reads from and, where a later module
// needs an earlier one's app service (discovery needs content's, vouching needs moderation's),
// writes into. Registration order therefore matters: content and moderation must run before
// discovery and vouching respectively — enforced by registerAll's own fixed order in main.go, not by
// anything in this type.
type Deps struct {
	Pool    *pgxpool.Pool
	Install config.Install

	// CoreRootUnitID/CoreCongregationAdminRoleID are fixed structural RIDs from
	// internal/platform/seed (migrations/0022_core_seed.sql) — every one of the six consumer
	// modules now reads these directly; the go-oikumenea-SDK-era OikumeneaBaseURL/RootUnitID/
	// CongregationAdminRoleID/ServicePrincipal fields this struct carried pre-M10.6 are gone
	// (D-SeedBootstrap: "three required environment variables disappear" once every module reading
	// them is cut over — true as of this milestone's close).
	CoreRootUnitID              string
	CoreCongregationAdminRoleID string

	// Populated by registerContent/registerModeration for registerDiscovery/registerVouching to
	// consume — see the cross-module adapter types below.
	ContentAppSvc    *contentapplication.Service
	ModerationAppSvc *moderationapplication.Service

	// Populated by registerCore — the M10.1-M10.5 in-process modules every one of the six consumer
	// modules now depends on directly.
	DirectorySvc  *directoryapplication.Service
	AuthzSvc      *authz.Service
	ReligionSvc   *religionapplication.Service
	LocationSvc   *locationapplication.Service
	MembershipSvc *membershipapplication.Service
	RefdataSvc    *refdataapplication.Service
}

func newDeps(pool *pgxpool.Pool, install config.Install) *Deps {
	return &Deps{
		Pool:                        pool,
		Install:                     install,
		CoreRootUnitID:              seed.RootUnitID,
		CoreCongregationAdminRoleID: seed.CongregationAdminRoleID,
	}
}

// registerFunc is the shape every module's register<Module> function has.
type registerFunc func(ctx context.Context, info witchcraft.InitInfo, deps *Deps) error

// contentSiteResolver adapts contentapplication.Service's public read onto discovery's own
// ContentResolver interface — an interface-call cross-module dependency (conventions.md), even
// though the underlying discovery_site_cache.content_site_id column is a real in-schema FK
// (DS-OFM-13, docs/modules/discovery.md).
type contentSiteResolver struct {
	content *contentapplication.Service
}

func (r *contentSiteResolver) GetSiteByUnit(ctx context.Context, congregationUnitRID string) (string, bool, error) {
	site, err := r.content.GetSite(ctx, congregationUnitRID)
	if err == contentdomain.ErrSiteNotFound {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return site.ID, true, nil
}

// moderationVouchReporter adapts moderationapplication.Service's FileReport onto vouching's own
// ModerationReporter interface — the same in-process interface-call cross-module dependency shape
// as contentSiteResolver above. Translates vouching's own GuarantorRevokedEvent vocabulary into
// moderation's FileReportInput here, at the composition root, so internal/vouching's application
// package never imports internal/moderation's domain or application packages directly (M6's own
// decision, docs/modules/vouching.md: revocation queues moderator review, it never invalidates
// anything automatically). Uses moderation's OTHER reason code with a descriptive detail string
// rather than a new GUARANTOR_REVOKED enum value, to avoid touching M5's already-migrated CHECK
// constraint and generated SDKs for this one caller.
type moderationVouchReporter struct {
	moderation *moderationapplication.Service
}

func (r *moderationVouchReporter) ReportGuarantorRevoked(ctx context.Context, event vouchingapplication.GuarantorRevokedEvent) error {
	detail := fmt.Sprintf("guarantor_revoked: guarantor=%s claimant=%s congregation=%s reason=%s",
		event.GuarantorPersonRID, event.ClaimantPersonRID, event.CongregationUnitID, event.RevokedReason)
	_, err := r.moderation.FileReport(ctx, moderationdomain.FileReportInput{
		TargetKind: moderationdomain.TargetVouchingEdge,
		TargetRef:  event.VouchID,
		ReasonCode: moderationdomain.ReasonOther,
		Detail:     &detail,
	})
	return err
}
