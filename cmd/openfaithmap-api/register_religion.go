// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	genreligion "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/religion"
	religiontransport "github.com/olehmushka/open-faith-map/internal/religion/transport"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
)

// registerReligion wires ReligionService (M13.2, religion's first transport layer) — needs only
// deps.ReligionSvc, populated by registerCore, which runs before this one in registerOrder
// (main.go).
func registerReligion(ctx context.Context, info witchcraft.InitInfo, deps *Deps) error {
	religionTransportSvc := religiontransport.NewService(deps.ReligionSvc)
	if err := genreligion.RegisterRoutesReligionService(info.Router, religionTransportSvc); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register religion routes")
	}
	return nil
}
