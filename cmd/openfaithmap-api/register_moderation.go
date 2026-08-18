// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	genmoderation "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/moderation"
	moderationadapters "github.com/olehmushka/open-faith-map/internal/moderation/adapters"
	moderationapplication "github.com/olehmushka/open-faith-map/internal/moderation/application"
	moderationtransport "github.com/olehmushka/open-faith-map/internal/moderation/transport"
	"github.com/olehmushka/open-faith-map/internal/platform/ratelimit"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

// registerModeration populates deps.ModerationAppSvc — registerVouching, run after this one in
// registerOrder (main.go), reads it to build its own moderationVouchReporter.
//
// M10.6: depends on deps.ReligionSvc/AuthzSvc (populated by registerCore, which runs before every
// consumer module) instead of the go-oikumenea SDK config; RootUnitID is deps.CoreRootUnitID, the
// fixed structural RID from internal/platform/seed.
func registerModeration(ctx context.Context, info witchcraft.InitInfo, deps *Deps) error {
	moderationStore := moderationadapters.NewStore(deps.Pool)
	moderationAppSvc := moderationapplication.NewService(moderationStore, deps.ReligionSvc, deps.AuthzSvc, moderationapplication.Config{
		RootUnitID: deps.CoreRootUnitID,
	})
	moderationTransportSvc := moderationtransport.NewService(moderationAppSvc)
	moderationPublicTransportSvc := moderationtransport.NewPublicService(moderationAppSvc)

	if err := genmoderation.RegisterRoutesModerationService(info.Router, moderationTransportSvc); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register moderation routes")
	}
	// M7: an in-process, per-(client IP, endpoint) rate limiter wired onto exactly this call — the
	// only two genuinely anonymous write endpoints in the whole API (D-Hardening). Every other
	// RegisterRoutes* call in this file, including ModerationService above, is untouched.
	// internal/platform/ratelimit since M10.6, so registerDiscovery can wrap its own anonymous
	// Search route with the same limiter type.
	moderationRateLimiter := ratelimit.NewLimiter("openfaithmap.moderation.rate_limit_rejections")
	if err := genmoderation.RegisterRoutesModerationPublicService(
		info.Router, moderationPublicTransportSvc,
		wrouter.RouteMiddleware(moderationRateLimiter.Middleware),
	); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register moderation public routes")
	}

	deps.ModerationAppSvc = moderationAppSvc
	return nil
}
