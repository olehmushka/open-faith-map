// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the religion module's Postgres store, ported from
// ../go-oikumenea/internal/religion/adapters/{repository.go,discovery.go} (tenant_* -> directory_*,
// oikumenea -> openfaithmap schema, RLS/org-scoping dropped per D-InProcessAuthz/D-CorePortScope).
package adapters

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/olehmushka/open-faith-map/internal/religion/domain"
)

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx — see internal/directory/adapters.Querier
// for why a Store needs both binding modes.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Store struct {
	q Querier
}

func NewStore(q Querier) *Store {
	return &Store{q: q}
}

// ---------------------------------------------------------------- taxa

func (s *Store) GetTaxon(ctx context.Context, id string) (domain.Taxon, error) {
	var t domain.Taxon
	var parentID *string
	var sortOrder *int
	err := s.q.QueryRow(ctx, `
		SELECT t.id, t.parent_id, t.rank_id, rk.code, t.code, t.name, t.sort_order
		FROM openfaithmap.religion_taxa t
		JOIN openfaithmap.religion_taxon_ranks rk ON rk.id = t.rank_id
		WHERE t.id = $1 AND t.deleted_at IS NULL`, id,
	).Scan(&t.ID, &parentID, &t.RankID, &t.RankCode, &t.Code, &t.Name, &sortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Taxon{}, domain.ErrTaxonNotFound
	}
	if err != nil {
		return domain.Taxon{}, err
	}
	t.ParentID, t.SortOrder = parentID, sortOrder
	return t, nil
}

// ---------------------------------------------------------------- org profile + classifications

func (s *Store) GetOrgProfileRow(ctx context.Context, unitID string) (domain.OrgProfile, error) {
	var p domain.OrgProfile
	var kindID, shortCode *string
	err := s.q.QueryRow(ctx, `
		SELECT unit_id, org_kind_id, short_code, created_at, updated_at
		FROM openfaithmap.religion_org_profiles WHERE unit_id = $1 AND deleted_at IS NULL`, unitID,
	).Scan(&p.UnitID, &kindID, &shortCode, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OrgProfile{}, domain.ErrProfileNotFound
	}
	if err != nil {
		return domain.OrgProfile{}, err
	}
	if kindID != nil {
		p.OrgKindID = *kindID
	}
	if shortCode != nil {
		p.ShortCode = *shortCode
	}
	return p, nil
}

func (s *Store) UpsertOrgProfile(ctx context.Context, unitID string, orgKindID, shortCode *string) (domain.OrgProfile, error) {
	_, err := s.q.Exec(ctx, `
		INSERT INTO openfaithmap.religion_org_profiles (unit_id, org_kind_id, short_code)
		VALUES ($1, $2, $3)
		ON CONFLICT (unit_id) DO UPDATE SET
			org_kind_id = EXCLUDED.org_kind_id, short_code = EXCLUDED.short_code, deleted_at = NULL`,
		unitID, orgKindID, shortCode)
	if err != nil {
		return domain.OrgProfile{}, err
	}
	return s.GetOrgProfileRow(ctx, unitID)
}

func (s *Store) ListOrgClassifications(ctx context.Context, unitID string) ([]domain.OrgClassification, error) {
	rows, err := s.q.Query(ctx, `
		SELECT oc.id, oc.unit_id, oc.taxon_id, t.code, t.name, oc.is_primary, oc.created_at
		FROM openfaithmap.religion_org_classifications oc
		JOIN openfaithmap.religion_taxa t ON t.id = oc.taxon_id
		WHERE oc.unit_id = $1 AND oc.deleted_at IS NULL
		ORDER BY oc.is_primary DESC, t.code`, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.OrgClassification
	for rows.Next() {
		var c domain.OrgClassification
		if err := rows.Scan(&c.ID, &c.UnitID, &c.TaxonID, &c.TaxonCode, &c.TaxonName, &c.IsPrimary, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) ClearPrimaryClassification(ctx context.Context, unitID string) error {
	_, err := s.q.Exec(ctx, `
		UPDATE openfaithmap.religion_org_classifications
		SET is_primary = false WHERE unit_id = $1 AND is_primary AND deleted_at IS NULL`, unitID)
	return err
}

func (s *Store) AddOrgClassification(ctx context.Context, unitID, taxonID string, isPrimary bool) (domain.OrgClassification, error) {
	var id string
	err := s.q.QueryRow(ctx, `
		INSERT INTO openfaithmap.religion_org_classifications (unit_id, taxon_id, is_primary)
		VALUES ($1, $2, $3) RETURNING id`, unitID, taxonID, isPrimary,
	).Scan(&id)
	if err != nil {
		return domain.OrgClassification{}, err
	}
	rows, err := s.ListOrgClassifications(ctx, unitID)
	if err != nil {
		return domain.OrgClassification{}, err
	}
	for _, c := range rows {
		if c.ID == id {
			return c, nil
		}
	}
	return domain.OrgClassification{}, errors.New("religion: classification inserted but not found on read-back")
}

// ---------------------------------------------------------------- policies

func (s *Store) HasActivePolicy(ctx context.Context, unitID, policyKindCode string) (bool, error) {
	var n int
	err := s.q.QueryRow(ctx, `
		SELECT count(*) FROM openfaithmap.religion_org_policies p
		JOIN openfaithmap.religion_policy_kinds k ON k.id = p.policy_kind_id
		WHERE p.unit_id = $1 AND k.code = $2 AND p.deleted_at IS NULL`, unitID, policyKindCode,
	).Scan(&n)
	return n > 0, err
}

// ---------------------------------------------------------------- site types + sites

type SiteType struct {
	ID   string
	Code string
	Name string
}

func (s *Store) ListSiteTypes(ctx context.Context) ([]SiteType, error) {
	rows, err := s.q.Query(ctx, `
		SELECT id, code, name FROM openfaithmap.religion_site_types
		WHERE deleted_at IS NULL ORDER BY sort_order NULLS LAST, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SiteType
	for rows.Next() {
		var t SiteType
		if err := rows.Scan(&t.ID, &t.Code, &t.Name); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

const siteCols = `s.id, s.org_unit_id, s.location_id, s.site_type_id, st.code, st.name,
	s.visibility, s.public_precision, s.is_primary,
	ST_Y(l.geom::geometry)::double precision, ST_X(l.geom::geometry)::double precision`

const siteFrom = `FROM openfaithmap.religion_sites s
	JOIN openfaithmap.religion_site_types st ON st.id = s.site_type_id
	JOIN openfaithmap.location_locations l ON l.id = s.location_id`

func scanSite(row pgx.Row) (domain.Site, error) {
	var s domain.Site
	err := row.Scan(&s.ID, &s.OrgUnitID, &s.LocationID, &s.SiteTypeID, &s.SiteTypeCode, &s.SiteTypeName,
		&s.Visibility, &s.PublicPrecision, &s.IsPrimary, &s.Latitude, &s.Longitude)
	return s, err
}

func (s *Store) ListSitesByUnit(ctx context.Context, unitID string) ([]domain.Site, error) {
	rows, err := s.q.Query(ctx, `SELECT `+siteCols+` `+siteFrom+`
		WHERE s.org_unit_id = $1 AND s.deleted_at IS NULL
		ORDER BY s.is_primary DESC, s.id`, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Site
	for rows.Next() {
		site, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, site)
	}
	return out, rows.Err()
}

type CreateSiteInput struct {
	OrgUnitID  string
	LocationID string
	SiteTypeID string
	IsPrimary  bool
}

func (s *Store) InsertSite(ctx context.Context, in CreateSiteInput) (domain.Site, error) {
	var id string
	err := s.q.QueryRow(ctx, `
		INSERT INTO openfaithmap.religion_sites (org_unit_id, location_id, site_type_id, is_primary)
		VALUES ($1, $2, $3, $4) RETURNING id`, in.OrgUnitID, in.LocationID, in.SiteTypeID, in.IsPrimary,
	).Scan(&id)
	if err != nil {
		return domain.Site{}, err
	}
	row := s.q.QueryRow(ctx, `SELECT `+siteCols+` `+siteFrom+` WHERE s.id = $1`, id)
	return scanSite(row)
}

// ---------------------------------------------------------------- discovery search

// snappedGeom is the position-oracle fix (docs/architecture/decisions.md's D-InProcessAuthz
// amendment #6 / D-CorePortScope amendment): the predicate and ORDER BY run against a geometry
// snapped to the site's OWN public_precision, never the exact stored geometry — coarsening only the
// *returned* coordinate (as upstream does) leaves result-set membership as a boolean oracle on a
// `hidden` site's true position. `hidden` sites are additionally excluded from the WHERE clause
// entirely (below), so this CASE's ELSE arm is unreachable in practice; it returns the exact point
// only for `exact`, matching Coarsen's own precision table.
const snappedGeom = `(CASE s.public_precision
	WHEN 'street' THEN ST_SetSRID(ST_MakePoint(
		round(ST_X(l.geom::geometry)::numeric, 4)::float8,
		round(ST_Y(l.geom::geometry)::numeric, 4)::float8), 4326)::geography
	WHEN 'neighborhood' THEN ST_SetSRID(ST_MakePoint(
		round(ST_X(l.geom::geometry)::numeric, 3)::float8,
		round(ST_Y(l.geom::geometry)::numeric, 3)::float8), 4326)::geography
	WHEN 'city' THEN ST_SetSRID(ST_MakePoint(
		round(ST_X(l.geom::geometry)::numeric, 2)::float8,
		round(ST_Y(l.geom::geometry)::numeric, 2)::float8), 4326)::geography
	ELSE l.geom
END)`

// SearchSites runs the closure-aware PostGIS discovery search over PUBLIC, non-hidden sites. Ported
// from ../go-oikumenea/internal/religion/adapters/discovery.go:347-411 with one deliberate behaviour
// change: `public_precision = 'hidden'` sites are excluded outright (upstream relied on app-side
// Coarsen alone, which protects the returned coordinate but not the predicate), and every other
// non-exact site is filtered/ordered on snappedGeom above, never the exact geometry.
func (s *Store) SearchSites(ctx context.Context, q domain.DiscoveryQuery) ([]domain.Site, error) {
	conds := []string{"s.deleted_at IS NULL", "s.visibility = 'public'", "s.public_precision <> 'hidden'"}
	var args []any
	add := func(a any) string { args = append(args, a); return "$" + strconv.Itoa(len(args)) }

	orderBy := "s.id"
	switch {
	case q.Lat != nil && q.Lng != nil && q.RadiusM != nil:
		pt := "ST_SetSRID(ST_MakePoint(" + add(*q.Lng) + "::double precision," + add(*q.Lat) + "::double precision),4326)::geography"
		conds = append(conds, "ST_DWithin("+snappedGeom+", "+pt+", "+add(*q.RadiusM)+"::double precision)")
		orderBy = snappedGeom + " <-> " + pt + ", s.id"
	case q.MinLat != nil && q.MinLng != nil && q.MaxLat != nil && q.MaxLng != nil:
		env := "ST_MakeEnvelope(" + add(*q.MinLng) + "::double precision," + add(*q.MinLat) + "::double precision," +
			add(*q.MaxLng) + "::double precision," + add(*q.MaxLat) + "::double precision,4326)::geography"
		conds = append(conds, "ST_Intersects("+snappedGeom+", "+env+")")
	}

	if q.Religion != "" {
		conds = append(conds, `s.org_unit_id IN (
			SELECT oc.unit_id FROM openfaithmap.religion_org_classifications oc
			JOIN openfaithmap.religion_taxa_closure tc ON tc.descendant_id = oc.taxon_id
			WHERE tc.ancestor_id = `+add(q.Religion)+` AND oc.deleted_at IS NULL)`)
	}

	if q.Query != "" {
		like := add("%" + strings.ToLower(q.Query) + "%")
		conds = append(conds, `(EXISTS (SELECT 1 FROM openfaithmap.religion_aliases a
				WHERE a.unit_id = s.org_unit_id AND a.deleted_at IS NULL AND lower(a.alias_text) LIKE `+like+`)
			OR EXISTS (SELECT 1 FROM openfaithmap.directory_units u
				WHERE u.id = s.org_unit_id AND (lower(u.code) LIKE `+like+` OR lower(u.name) LIKE `+like+`)))`)
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	sql := `SELECT ` + siteCols + ` ` + siteFrom + `
		WHERE ` + strings.Join(conds, " AND ") + `
		ORDER BY ` + orderBy + `
		LIMIT ` + add(limit)
	rows, err := s.q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Site
	for rows.Next() {
		site, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, site)
	}
	return out, rows.Err()
}
