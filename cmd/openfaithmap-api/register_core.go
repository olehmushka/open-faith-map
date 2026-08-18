// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	"github.com/olehmushka/open-faith-map/internal/authz"
	authzadapters "github.com/olehmushka/open-faith-map/internal/authz/adapters"
	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	directoryadapters "github.com/olehmushka/open-faith-map/internal/directory/adapters"
	directoryapplication "github.com/olehmushka/open-faith-map/internal/directory/application"
	locationapplication "github.com/olehmushka/open-faith-map/internal/location/application"
	membershipapplication "github.com/olehmushka/open-faith-map/internal/membership/application"
	refdataapplication "github.com/olehmushka/open-faith-map/internal/refdata/application"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
)

// registerCore builds the M10.1-M10.5 in-process core modules and populates deps for every
// consumer module registered afterward (registerOrder in main.go runs this first, before any
// consumer). Registers no HTTP routes of its own — these modules get a Conjure surface at M10.7.
//
// internal/authz's PDP is wired over internal/directory's own store as its ClosurePort
// (IsAncestorOrSelf/IsAuthorityBearing) — the composition root is the ONLY place this assignment may
// happen (D-InProcessAuthz amendment #4): internal/directory must not import internal/authz, and
// internal/authz must not import internal/directory, so the concrete wiring lives here, not in
// either module.
func registerCore(ctx context.Context, info witchcraft.InitInfo, deps *Deps) error {
	directorySvc := directoryapplication.NewService(deps.Pool)
	closurePort := directoryadapters.NewStore(deps.Pool)

	pdp := authzdomain.NewPDP(closurePort)
	authzStore := authzadapters.NewStore(deps.Pool)
	authzSvc := authz.NewService(pdp, authzStore)

	religionSvc := religionapplication.NewService(deps.Pool, directorySvc)
	locationSvc := locationapplication.NewService(deps.Pool)
	membershipSvc := membershipapplication.NewService(deps.Pool)
	refdataSvc := refdataapplication.NewService(deps.Pool)

	deps.DirectorySvc = directorySvc
	deps.AuthzSvc = authzSvc
	deps.ReligionSvc = religionSvc
	deps.LocationSvc = locationSvc
	deps.MembershipSvc = membershipSvc
	deps.RefdataSvc = refdataSvc
	return nil
}
