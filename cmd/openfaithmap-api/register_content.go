// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	gencontent "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/content"
	contentadapters "github.com/olehmushka/open-faith-map/internal/content/adapters"
	contentapplication "github.com/olehmushka/open-faith-map/internal/content/application"
	contenttransport "github.com/olehmushka/open-faith-map/internal/content/transport"
	"github.com/olehmushka/open-faith-map/internal/platform/ratelimit"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

// registerContent populates deps.ContentAppSvc — registerDiscovery, run after this one in
// registerOrder (main.go), reads it to build its own contentSiteResolver.
//
// M10.6: depends on deps.AuthzSvc (populated by registerCore, which runs before every consumer
// module) instead of the go-oikumenea SDK config.
//
// M14.11: also depends on deps.ReligionSvc (populated by registerCore too) — content's own
// GetSiteChrome composes a site-chrome header/footer from religion's live data, the same
// direct-interface-call cross-module shape registerDiscovery already uses one line below.
//
// M14.13: Config.RootUnitID is deps.CoreRootUnitID, the same fixed structural RID
// register_moderation.go already passes moderationapplication.Config — content.catalog.manage's
// requireCatalogManage checks platform-moderator standing against this same root unit.
func registerContent(ctx context.Context, info witchcraft.InitInfo, deps *Deps) error {
	contentStore := contentadapters.NewRepository(deps.Pool)
	contentAppSvc := contentapplication.NewService(contentStore, deps.AuthzSvc, deps.ReligionSvc, deps.Install.ContentPreviewHMACKey, contentapplication.Config{
		RootUnitID: deps.CoreRootUnitID,
	})
	contentTransportSvc := contenttransport.NewService(contentAppSvc)
	contentPublicTransportSvc := contenttransport.NewPublicService(contentAppSvc)

	if err := gencontent.RegisterRoutesContentService(info.Router, contentTransportSvc); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register content routes")
	}
	// M14.16, D-InAppInbox: submitContactForm is this service's first genuinely anonymous *write*
	// (every other ContentPublicService endpoint is a read) — rate-limited exactly like
	// moderation's two anonymous writes (register_moderation.go), same internal/platform/ratelimit
	// type. Named consequence, not an oversight: middleware here applies to this whole
	// RegisterRoutes call, so every one of ContentPublicService's existing anonymous reads also
	// gets its own per-(IP, endpoint) bucket for free — the identical shared-bucket tradeoff
	// registerDiscovery's own public-read limiter already made.
	contentRateLimiter := ratelimit.NewLimiter("openfaithmap.content.public.rate_limit_rejections")
	if err := gencontent.RegisterRoutesContentPublicService(
		info.Router, contentPublicTransportSvc,
		wrouter.RouteMiddleware(contentRateLimiter.Middleware),
	); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register content public routes")
	}

	deps.ContentAppSvc = contentAppSvc
	return nil
}
