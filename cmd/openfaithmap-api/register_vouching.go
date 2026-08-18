// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	genvouching "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/vouching"
	vouchingadapters "github.com/olehmushka/open-faith-map/internal/vouching/adapters"
	vouchingapplication "github.com/olehmushka/open-faith-map/internal/vouching/application"
	vouchingtransport "github.com/olehmushka/open-faith-map/internal/vouching/transport"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
)

// registerVouching depends on deps.ModerationAppSvc, set by registerModeration — see registerOrder
// in main.go for the enforced ordering. M6: vouching has no genuinely-anonymous endpoint (unlike
// content/discovery/moderation), so it gets a single authenticated service, and no ServicePrincipal
// config at all. Its moderation.read/moderation.act gates reuse the same RootUnitID as moderation's
// own requireModerate; RevokeGuarantor's moderation-report fan-out is wired through
// moderationVouchReporter (deps.go), an in-process call into the moderationAppSvc registerModeration
// already constructed.
func registerVouching(ctx context.Context, info witchcraft.InitInfo, deps *Deps) error {
	vouchingStore := vouchingadapters.NewStore(deps.Pool)
	vouchingAppSvc := vouchingapplication.NewService(vouchingStore, &moderationVouchReporter{moderation: deps.ModerationAppSvc}, vouchingapplication.Config{
		OikumeneaBaseURL:            deps.OikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: deps.OikumeneaInsecureSkipVerify,
		RootUnitID:                  deps.RootUnitID,
	})
	vouchingTransportSvc := vouchingtransport.NewService(vouchingAppSvc, vouchingtransport.Config{
		OikumeneaBaseURL:            deps.OikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: deps.OikumeneaInsecureSkipVerify,
	})

	if err := genvouching.RegisterRoutesVouchingService(info.Router, vouchingTransportSvc); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register vouching routes")
	}
	return nil
}
