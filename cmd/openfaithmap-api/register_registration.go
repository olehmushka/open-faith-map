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

// registerRegistration depends on deps.ReligionSvc/LocationSvc/MembershipSvc/DirectorySvc/AuthzSvc,
// all populated by registerCore (M10.6) before registerOrder (main.go) reaches this function.
func registerRegistration(ctx context.Context, info witchcraft.InitInfo, deps *Deps) error {
	store := regadapters.NewRepository(deps.Pool)
	appSvc := regapplication.NewService(store, deps.ReligionSvc, deps.LocationSvc, deps.MembershipSvc, deps.DirectorySvc, deps.AuthzSvc, regapplication.Config{
		RootUnitID:              deps.CoreRootUnitID,
		CongregationAdminRoleID: deps.CoreCongregationAdminRoleID,
	})
	transportSvc := regtransport.NewService(appSvc)

	if err := genregistration.RegisterRoutesRegistrationService(info.Router, transportSvc); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register registration routes")
	}
	return nil
}
