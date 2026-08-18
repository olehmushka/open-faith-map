// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	genmoderation "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/moderation"
	moderationadapters "github.com/olehmushka/open-faith-map/internal/moderation/adapters"
	moderationapplication "github.com/olehmushka/open-faith-map/internal/moderation/application"
	moderationtransport "github.com/olehmushka/open-faith-map/internal/moderation/transport"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

// registerModeration populates deps.ModerationAppSvc — registerVouching, run after this one in
// registerOrder (main.go), reads it to build its own moderationVouchReporter. M5: platform-
// moderator's own root-unit-scoped Authorize check (application/authorize.go) reuses RootUnitID;
// CheckExclusion reuses the same service-principal credentials discovery's cache refresh already
// wires — the caller of POST /exclusion-check is anonymous, same reason.
func registerModeration(ctx context.Context, info witchcraft.InitInfo, deps *Deps) error {
	moderationStore := moderationadapters.NewStore(deps.Pool)
	moderationAppSvc := moderationapplication.NewService(moderationStore, moderationapplication.Config{
		OikumeneaBaseURL:            deps.OikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: deps.OikumeneaInsecureSkipVerify,
		RootUnitID:                  deps.RootUnitID,
		ServicePrincipal:            deps.ServicePrincipal,
	})
	moderationTransportSvc := moderationtransport.NewService(moderationAppSvc, moderationtransport.Config{
		OikumeneaBaseURL:            deps.OikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: deps.OikumeneaInsecureSkipVerify,
	})
	moderationPublicTransportSvc := moderationtransport.NewPublicService(moderationAppSvc)

	if err := genmoderation.RegisterRoutesModerationService(info.Router, moderationTransportSvc); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register moderation routes")
	}
	// M7: an in-process, per-(client IP, endpoint) rate limiter wired onto exactly this call — the
	// only two genuinely anonymous write endpoints in the whole API (D-Hardening). Every other
	// RegisterRoutes* call in this file, including ModerationService above, is untouched.
	moderationRateLimiter := moderationtransport.NewRateLimiter()
	if err := genmoderation.RegisterRoutesModerationPublicService(
		info.Router, moderationPublicTransportSvc,
		wrouter.RouteMiddleware(moderationRateLimiter.Middleware),
	); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register moderation public routes")
	}

	deps.ModerationAppSvc = moderationAppSvc
	return nil
}
