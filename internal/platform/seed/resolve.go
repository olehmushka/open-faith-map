// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package seed

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	authzadapters "github.com/olehmushka/open-faith-map/internal/authz/adapters"
	directoryadapters "github.com/olehmushka/open-faith-map/internal/directory/adapters"
)

// The stable codes migrations/0015_core_seed.sql assigns the root unit and the three base roles —
// directory_units and authz_roles both carry a unique-while-active code index, so these are real
// natural keys, not something invented for this package.
const (
	RootUnitCode                 = "root"
	RegistrationOperatorRoleCode = "registration-operator"
	CongregationAdminRoleCode    = "congregation-admin"
	PlatformModeratorRoleCode    = "platform-moderator"
)

// IDs is the boot-time resolution of the fixed seed rows, by code.
type IDs struct {
	RootUnitID                 string
	RegistrationOperatorRoleID string
	CongregationAdminRoleID    string
	PlatformModeratorRoleID    string
}

// Resolve looks up the deterministic seed rows by their stable code — the composition root's
// (cmd/openfaithmap-api) replacement for reading the RootUnitID/RegistrationOperatorRoleID/
// CongregationAdminRoleID/PlatformModeratorRoleID constants directly, so production wiring depends
// on the database's own natural key instead of a hardcoded UUID copied from the seed migration.
func Resolve(ctx context.Context, pool *pgxpool.Pool) (IDs, error) {
	dirRepo := directoryadapters.NewRepository(pool)
	unit, err := dirRepo.GetUnitByCode(ctx, RootUnitCode)
	if err != nil {
		return IDs{}, err
	}

	authzRepo := authzadapters.NewRepository(pool)
	opRole, err := authzRepo.GetRoleByCode(ctx, RegistrationOperatorRoleCode)
	if err != nil {
		return IDs{}, err
	}
	adminRole, err := authzRepo.GetRoleByCode(ctx, CongregationAdminRoleCode)
	if err != nil {
		return IDs{}, err
	}
	modRole, err := authzRepo.GetRoleByCode(ctx, PlatformModeratorRoleCode)
	if err != nil {
		return IDs{}, err
	}

	return IDs{
		RootUnitID:                 unit.ID,
		RegistrationOperatorRoleID: opRole.ID,
		CongregationAdminRoleID:    adminRole.ID,
		PlatformModeratorRoleID:    modRole.ID,
	}, nil
}
