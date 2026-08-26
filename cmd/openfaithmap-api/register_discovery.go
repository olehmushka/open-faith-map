// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	gendiscovery "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/discovery"
	discoveryadapters "github.com/olehmushka/open-faith-map/internal/discovery/adapters"
	discoveryapplication "github.com/olehmushka/open-faith-map/internal/discovery/application"
	discoverytransport "github.com/olehmushka/open-faith-map/internal/discovery/transport"
	"github.com/olehmushka/open-faith-map/internal/platform/ratelimit"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

// registerDiscovery depends on deps.ContentAppSvc (set by registerContent) and
// deps.ReligionSvc/AuthzSvc (set by registerCore) — see registerOrder in main.go for the enforced
// ordering. RootUnitID is deps.CoreRootUnitID, the fixed structural RID from
// internal/platform/seed.
//
// M10.6: the anonymous public Search route gains the same rate limiter moderation's two anonymous
// endpoints already use (internal/platform/ratelimit, moved there from internal/moderation/transport
// this same milestone precisely so this call site could reuse it) — it had none before, unlike every
// other genuinely anonymous write/read endpoint in this API.
func registerDiscovery(ctx context.Context, info witchcraft.InitInfo, deps *Deps) error {
	discoveryStore := discoveryadapters.NewRepository(deps.Pool)
	discoveryAppSvc := discoveryapplication.NewService(discoveryStore, &contentSiteResolver{content: deps.ContentAppSvc}, deps.ReligionSvc, deps.AuthzSvc, discoveryapplication.Config{
		RootUnitID: deps.CoreRootUnitID,
	})
	discoveryTransportSvc := discoverytransport.NewService(discoveryAppSvc)
	discoveryPublicTransportSvc := discoverytransport.NewPublicService(discoveryAppSvc)

	if err := gendiscovery.RegisterRoutesDiscoveryService(info.Router, discoveryTransportSvc); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register discovery routes")
	}
	discoveryRateLimiter := ratelimit.NewLimiter("openfaithmap.discovery.rate_limit_rejections")
	if err := gendiscovery.RegisterRoutesDiscoveryPublicService(
		info.Router, discoveryPublicTransportSvc,
		wrouter.RouteMiddleware(discoveryRateLimiter.Middleware),
	); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register discovery public routes")
	}
	return nil
}
