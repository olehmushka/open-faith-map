// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	genregistration "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/registration"
	regadapters "github.com/olehmushka/open-faith-map/internal/registration/adapters"
	regapplication "github.com/olehmushka/open-faith-map/internal/registration/application"
	regtransport "github.com/olehmushka/open-faith-map/internal/registration/transport"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
)

func registerRegistration(ctx context.Context, info witchcraft.InitInfo, deps *Deps) error {
	store := regadapters.NewStore(deps.Pool)
	appSvc := regapplication.NewService(store, regapplication.Config{
		OikumeneaBaseURL:            deps.OikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: deps.OikumeneaInsecureSkipVerify,
		RootUnitID:                  deps.RootUnitID,
		CongregationAdminRoleID:     deps.CongregationAdminRoleID,
	})
	transportSvc := regtransport.NewService(appSvc, regtransport.Config{
		OikumeneaBaseURL:            deps.OikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: deps.OikumeneaInsecureSkipVerify,
	})

	if err := genregistration.RegisterRoutesRegistrationService(info.Router, transportSvc); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register registration routes")
	}
	return nil
}
