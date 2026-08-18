// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	gendiscovery "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/discovery"
	discoveryadapters "github.com/olehmushka/open-faith-map/internal/discovery/adapters"
	discoveryapplication "github.com/olehmushka/open-faith-map/internal/discovery/application"
	discoverytransport "github.com/olehmushka/open-faith-map/internal/discovery/transport"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
)

// registerDiscovery depends on deps.ContentAppSvc, set by registerContent — see registerOrder in
// main.go for the enforced ordering. M4: the service principal's own credentials — first production
// use of coreintegration.NewServiceClient (M1 built it, only an integration test called it until
// now). GOOGLE_APPLICATION_CREDENTIALS is the standard convention already used by
// scripts/bootstrap-service-principal (.env.example); audience matches that script's own
// registration ("openfaithmap-api", also what client_integration_test.go proves against).
func registerDiscovery(ctx context.Context, info witchcraft.InitInfo, deps *Deps) error {
	discoveryStore := discoveryadapters.NewStore(deps.Pool)
	discoveryAppSvc := discoveryapplication.NewService(discoveryStore, &contentSiteResolver{content: deps.ContentAppSvc}, discoveryapplication.Config{
		OikumeneaBaseURL:            deps.OikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: deps.OikumeneaInsecureSkipVerify,
		RootUnitID:                  deps.RootUnitID,
		ServicePrincipal:            deps.ServicePrincipal,
	})
	discoveryTransportSvc := discoverytransport.NewService(discoveryAppSvc, discoverytransport.Config{
		OikumeneaBaseURL:            deps.OikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: deps.OikumeneaInsecureSkipVerify,
	})
	discoveryPublicTransportSvc := discoverytransport.NewPublicService(discoveryAppSvc)

	if err := gendiscovery.RegisterRoutesDiscoveryService(info.Router, discoveryTransportSvc); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register discovery routes")
	}
	if err := gendiscovery.RegisterRoutesDiscoveryPublicService(info.Router, discoveryPublicTransportSvc); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register discovery public routes")
	}
	return nil
}
