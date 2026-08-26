// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the religion module's Postgres store, ported from
// ../go-oikumenea/internal/religion/adapters/{repository.go,discovery.go} (tenant_* -> directory_*,
// oikumenea -> openfaithmap schema, RLS/org-scoping dropped per D-InProcessAuthz/D-CorePortScope).
// sqlc-generated (docs/architecture/decisions.md's D-Stack) — queries live in queries/religion.sql,
// generated code in religionsql/. SearchSites is the one exception: its WHERE/ORDER BY shape is
// genuinely dynamic (see queries/religion.sql's own comment on why it isn't a sqlc query), so it
// stays hand-written against the same db.DBTX this package's Queries wraps.
package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
	"github.com/olehmushka/open-faith-map/internal/religion/adapters/religionsql"
	"github.com/olehmushka/open-faith-map/internal/religion/domain"
)

type Repository struct {
	conn db.DBTX
	q    *religionsql.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{conn: conn, q: religionsql.New(conn)}
}

func fromNullableText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func fromNullableInt4(i pgtype.Int4) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int32)
	return &v
}

// ---------------------------------------------------------------- taxa

func toTaxon(row religionsql.GetTaxonRow) domain.Taxon {
	return domain.Taxon{ID: row.ID, ParentID: fromNullableText(row.ParentID), RankID: row.RankID, RankCode: row.RankCode, Code: row.Code, Name: row.Name, SortOrder: fromNullableInt4(row.SortOrder)}
}

func (r *Repository) GetTaxon(ctx context.Context, id string) (domain.Taxon, error) {
	row, err := r.q.GetTaxon(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Taxon{}, domain.ErrTaxonNotFound
	}
	if err != nil {
		return domain.Taxon{}, err
	}
	return toTaxon(row), nil
}

// ListTaxa is M10.7's core.conjure.yml ListTaxa — a plain ILIKE search over code/name (mirrors
// ListSiteTypes'/ListOrgKinds' catalog-read shape), capped at limit (default/max 50). Replaces the
// pre-cutover admin app's religion.listTaxa call (lib/dictionaries.ts).
func (r *Repository) ListTaxa(ctx context.Context, query string, limit int) ([]domain.Taxon, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	rows, err := r.q.ListTaxa(ctx, religionsql.ListTaxaParams{Query: query, LimitCount: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Taxon, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Taxon{ID: row.ID, ParentID: fromNullableText(row.ParentID), RankID: row.RankID, RankCode: row.RankCode, Code: row.Code, Name: row.Name, SortOrder: fromNullableInt4(row.SortOrder)})
	}
	return out, nil
}

// ---------------------------------------------------------------- org profile + classifications

func (r *Repository) GetOrgProfileRow(ctx context.Context, unitID string) (domain.OrgProfile, error) {
	row, err := r.q.GetOrgProfileRow(ctx, unitID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OrgProfile{}, domain.ErrProfileNotFound
	}
	if err != nil {
		return domain.OrgProfile{}, err
	}
	p := domain.OrgProfile{UnitID: row.UnitID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if row.OrgKindID.Valid {
		p.OrgKindID = row.OrgKindID.String
	}
	if row.ShortCode.Valid {
		p.ShortCode = row.ShortCode.String
	}
	return p, nil
}

func (r *Repository) UpsertOrgProfile(ctx context.Context, unitID string, orgKindID, shortCode *string) (domain.OrgProfile, error) {
	var orgKindArg, shortCodeArg pgtype.Text
	if orgKindID != nil {
		orgKindArg = pgtype.Text{String: *orgKindID, Valid: true}
	}
	if shortCode != nil {
		shortCodeArg = pgtype.Text{String: *shortCode, Valid: true}
	}
	if err := r.q.UpsertOrgProfile(ctx, religionsql.UpsertOrgProfileParams{UnitID: unitID, OrgKindID: orgKindArg, ShortCode: shortCodeArg}); err != nil {
		return domain.OrgProfile{}, err
	}
	return r.GetOrgProfileRow(ctx, unitID)
}

func (r *Repository) ListOrgClassifications(ctx context.Context, unitID string) ([]domain.OrgClassification, error) {
	rows, err := r.q.ListOrgClassifications(ctx, unitID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.OrgClassification, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.OrgClassification{ID: row.ID, UnitID: row.UnitID, TaxonID: row.TaxonID, TaxonCode: row.TaxonCode, TaxonName: row.TaxonName, IsPrimary: row.IsPrimary, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func (r *Repository) ClearPrimaryClassification(ctx context.Context, unitID string) error {
	return r.q.ClearPrimaryClassification(ctx, unitID)
}

func (r *Repository) AddOrgClassification(ctx context.Context, unitID, taxonID string, isPrimary bool) (domain.OrgClassification, error) {
	id, err := r.q.InsertOrgClassification(ctx, religionsql.InsertOrgClassificationParams{UnitID: unitID, TaxonID: taxonID, IsPrimary: isPrimary})
	if err != nil {
		return domain.OrgClassification{}, err
	}
	rows, err := r.ListOrgClassifications(ctx, unitID)
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

func (r *Repository) HasActivePolicy(ctx context.Context, unitID, policyKindCode string) (bool, error) {
	n, err := r.q.HasActivePolicy(ctx, religionsql.HasActivePolicyParams{UnitID: unitID, PolicyKindCode: policyKindCode})
	return n > 0, err
}

// ---------------------------------------------------------------- site types + sites

type SiteType struct {
	ID   string
	Code string
	Name string
}

func (r *Repository) ListSiteTypes(ctx context.Context) ([]SiteType, error) {
	rows, err := r.q.ListSiteTypes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SiteType, 0, len(rows))
	for _, row := range rows {
		out = append(out, SiteType{ID: row.ID, Code: row.Code, Name: row.Name})
	}
	return out, nil
}

// OrgKind is a generic-or-per-religion classification of a religious-body unit (religion_org_kinds)
// — e.g. "diocese"/"jurisdiction" for the Catholic hierarchy sync (congregationimport's own
// resolveOrgKindIDs).
type OrgKind struct {
	ID   string
	Code string
	Name string
}

func (r *Repository) ListOrgKinds(ctx context.Context) ([]OrgKind, error) {
	rows, err := r.q.ListOrgKinds(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]OrgKind, 0, len(rows))
	for _, row := range rows {
		out = append(out, OrgKind{ID: row.ID, Code: row.Code, Name: row.Name})
	}
	return out, nil
}

// attributesFromJSON unmarshals a religion_sites.attributes column into its Go shape; an empty
// document degrades to the zero-value SiteAttributes (every criterion unset) rather than erroring —
// matches the column's own `NOT NULL DEFAULT '{}'`.
func attributesFromJSON(raw json.RawMessage) domain.SiteAttributes {
	var a domain.SiteAttributes
	if len(raw) == 0 {
		return a
	}
	_ = json.Unmarshal(raw, &a)
	return a
}

// attributesContainmentFilter builds the JSONB document SearchSites containment-matches
// religion_sites.attributes against (M13.1's Accessibility/OnlineOnly filter, GIN index
// religion_sites_attributes_gin) — e.g. {"accessibility":{"stepFreeEntrance":true},
// "onlineStream":true}. ok is false (no filter to apply) when neither accessibility nor onlineOnly
// was requested.
func attributesContainmentFilter(accessibility []string, onlineOnly bool) (doc string, ok bool) {
	if len(accessibility) == 0 && !onlineOnly {
		return "", false
	}
	filter := map[string]any{}
	if len(accessibility) > 0 {
		acc := make(map[string]bool, len(accessibility))
		for _, key := range accessibility {
			acc[key] = true
		}
		filter["accessibility"] = acc
	}
	if onlineOnly {
		filter["onlineStream"] = true
	}
	b, err := json.Marshal(filter)
	if err != nil {
		return "", false
	}
	return string(b), true
}

func toSite(id, orgUnitID, locationID, siteTypeID, siteTypeCode, siteTypeName, visibility, publicPrecision string, isPrimary bool, attributes json.RawMessage, latitude, longitude float64) domain.Site {
	return domain.Site{
		ID: id, OrgUnitID: orgUnitID, LocationID: locationID, SiteTypeID: siteTypeID, SiteTypeCode: siteTypeCode,
		SiteTypeName: siteTypeName, Visibility: visibility, PublicPrecision: publicPrecision, IsPrimary: isPrimary,
		Attributes: attributesFromJSON(attributes), Latitude: latitude, Longitude: longitude,
	}
}

func (r *Repository) ListSitesByUnit(ctx context.Context, unitID string) ([]domain.Site, error) {
	rows, err := r.q.ListSitesByUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Site, 0, len(rows))
	for _, row := range rows {
		out = append(out, toSite(row.ID, row.OrgUnitID, row.LocationID, row.SiteTypeID, row.SiteTypeCode, row.SiteTypeName, row.Visibility, row.PublicPrecision, row.IsPrimary, row.Attributes, row.Latitude, row.Longitude))
	}
	return out, nil
}

type CreateSiteInput struct {
	OrgUnitID  string
	LocationID string
	SiteTypeID string
	IsPrimary  bool
}

func (r *Repository) InsertSite(ctx context.Context, in CreateSiteInput) (domain.Site, error) {
	id, err := r.q.InsertSite(ctx, religionsql.InsertSiteParams{OrgUnitID: in.OrgUnitID, LocationID: in.LocationID, SiteTypeID: in.SiteTypeID, IsPrimary: in.IsPrimary})
	if err != nil {
		return domain.Site{}, err
	}
	row, err := r.q.GetSiteRow(ctx, id)
	if err != nil {
		return domain.Site{}, err
	}
	return toSite(row.ID, row.OrgUnitID, row.LocationID, row.SiteTypeID, row.SiteTypeCode, row.SiteTypeName, row.Visibility, row.PublicPrecision, row.IsPrimary, row.Attributes, row.Latitude, row.Longitude), nil
}

// ---------------------------------------------------------------- discovery search

// siteCols/siteFrom back SearchSites only (M13.0 extended them with the public-projection
// enrichment: congregation name, address components, primary tradition tag, and aggregated
// service-schedule language/day — GetSiteRow/ListSitesByUnit stay on their own, unenriched sqlc
// query text in queries/religion.sql, since those authenticated-owner paths already know their own
// unit's name/address and have no discovery-card use for it).
const siteCols = `s.id, s.org_unit_id, s.location_id, s.site_type_id, st.code, st.name,
	s.visibility, s.public_precision, s.is_primary, s.attributes,
	ST_Y(l.geom::geometry)::double precision, ST_X(l.geom::geometry)::double precision,
	u.name, COALESCE(l.locality,''), COALESCE(l.admin_area_1,''), COALESCE(l.admin_area_2,''),
	COALESCE(l.street,''), COALESCE(l.house_number,''), COALESCE(l.postal_code,''),
	prim.taxon_id, prim.taxon_code, prim.taxon_name,
	COALESCE(svc.languages, '{}'), COALESCE(svc.days, '{}')`

const siteFrom = `FROM openfaithmap.religion_sites s
	JOIN openfaithmap.religion_site_types st ON st.id = s.site_type_id
	JOIN openfaithmap.location_locations l ON l.id = s.location_id
	JOIN openfaithmap.directory_units u ON u.id = s.org_unit_id
	LEFT JOIN LATERAL (
		SELECT oc.taxon_id, t.code AS taxon_code, t.name AS taxon_name
		FROM openfaithmap.religion_org_classifications oc
		JOIN openfaithmap.religion_taxa t ON t.id = oc.taxon_id
		WHERE oc.unit_id = s.org_unit_id AND oc.deleted_at IS NULL
		ORDER BY oc.is_primary DESC, t.code
		LIMIT 1
	) prim ON true
	LEFT JOIN LATERAL (
		SELECT array_agg(DISTINCT sch.language) FILTER (WHERE sch.language IS NOT NULL) AS languages,
			array_agg(DISTINCT sch.day_of_week) FILTER (WHERE sch.day_of_week IS NOT NULL) AS days
		FROM openfaithmap.religion_service_schedules sch
		WHERE sch.site_id = s.id AND sch.deleted_at IS NULL
	) svc ON true`

func scanSite(row pgx.Row) (domain.Site, error) {
	var s domain.Site
	var attributes json.RawMessage
	var days []int16
	err := row.Scan(&s.ID, &s.OrgUnitID, &s.LocationID, &s.SiteTypeID, &s.SiteTypeCode, &s.SiteTypeName,
		&s.Visibility, &s.PublicPrecision, &s.IsPrimary, &attributes, &s.Latitude, &s.Longitude,
		&s.Name, &s.Locality, &s.AdminArea1, &s.AdminArea2, &s.Street, &s.HouseNumber, &s.PostalCode,
		&s.TraditionTaxonID, &s.TraditionTaxonCode, &s.TraditionTaxonName,
		&s.ServiceLanguages, &days)
	if err != nil {
		return domain.Site{}, err
	}
	s.Attributes = attributesFromJSON(attributes)
	s.ServiceDays = make([]int, len(days))
	for i, d := range days {
		s.ServiceDays[i] = int(d)
	}
	return s, nil
}

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
//
// Kept hand-written (not sqlc) against the same db.DBTX this package's Queries wraps: the WHERE/
// ORDER BY shape genuinely varies by which optional filters are present (radius XOR bbox XOR
// neither), which isn't a fixed set of "narg IS NULL OR" branches sqlc's static-query model fits.
func (r *Repository) SearchSites(ctx context.Context, q domain.DiscoveryQuery) ([]domain.Site, error) {
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
	case q.UnitID != nil:
		// A unit-scoped lookup (M13.0's single-site detail-page fetch) has no spatial ordering to
		// fall back on — prefer the unit's primary site, matching ListSitesByUnit's own tiebreak.
		orderBy = "s.is_primary DESC, s.id"
	}

	if q.UnitID != nil {
		conds = append(conds, "s.org_unit_id = "+add(*q.UnitID))
	}

	if q.Religion != "" {
		// M13.1 fix: q.Religion is a taxon CODE (api/discovery.conjure.yml's own `tradition` docs),
		// not an id — join through religion_taxa.code (unique among active rows,
		// religion_taxa_code_active) instead of binding the raw string straight to
		// religion_taxa_closure.ancestor_id, a uuid column, which previously matched nothing.
		conds = append(conds, `s.org_unit_id IN (
			SELECT oc.unit_id FROM openfaithmap.religion_org_classifications oc
			JOIN openfaithmap.religion_taxa_closure tc ON tc.descendant_id = oc.taxon_id
			JOIN openfaithmap.religion_taxa rt ON rt.id = tc.ancestor_id
			WHERE rt.code = `+add(q.Religion)+` AND rt.deleted_at IS NULL AND oc.deleted_at IS NULL)`)
	}

	if q.Query != "" {
		like := add("%" + strings.ToLower(q.Query) + "%")
		conds = append(conds, `(EXISTS (SELECT 1 FROM openfaithmap.religion_aliases a
				WHERE a.unit_id = s.org_unit_id AND a.deleted_at IS NULL AND lower(a.alias_text) LIKE `+like+`)
			OR EXISTS (SELECT 1 FROM openfaithmap.directory_units u
				WHERE u.id = s.org_unit_id AND (lower(u.code) LIKE `+like+` OR lower(u.name) LIKE `+like+`)))`)
	}

	if q.Language != nil {
		conds = append(conds, `EXISTS (SELECT 1 FROM openfaithmap.religion_service_schedules sch
				WHERE sch.site_id = s.id AND sch.deleted_at IS NULL AND sch.language = `+add(*q.Language)+`)`)
	}
	if q.DayOfWeek != nil {
		conds = append(conds, `EXISTS (SELECT 1 FROM openfaithmap.religion_service_schedules sch
				WHERE sch.site_id = s.id AND sch.deleted_at IS NULL AND sch.day_of_week = `+add(*q.DayOfWeek)+`)`)
	}

	if filter, ok := attributesContainmentFilter(q.Accessibility, q.OnlineOnly); ok {
		conds = append(conds, "s.attributes @> "+add(filter)+"::jsonb")
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	sql := `SELECT ` + siteCols + ` ` + siteFrom + `
		WHERE ` + strings.Join(conds, " AND ") + `
		ORDER BY ` + orderBy + `
		LIMIT ` + add(limit)
	rows, err := r.conn.Query(ctx, sql, args...)
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

// searchableSitePredicate is SearchSites' own base WHERE conditions (deleted/public/non-hidden),
// duplicated here rather than shared as a helper: SearchSites builds its conds slice dynamically
// alongside optional filters, while SearchFacets' two queries are fixed-shape and each only need
// this one clause once.
const searchableSitePredicate = `s.deleted_at IS NULL AND s.visibility = 'public' AND s.public_precision <> 'hidden'`

// SearchFacets returns every distinct tradition taxon / service-schedule language actually present
// among public, non-hidden sites (M13.1) — the same visibility predicate SearchSites itself applies,
// so a hidden or private site's tradition/language never leaks into the picker UI's options.
func (r *Repository) SearchFacets(ctx context.Context) (domain.Facets, error) {
	traditionRows, err := r.conn.Query(ctx, `
		SELECT DISTINCT t.id, t.code, t.name
		FROM openfaithmap.religion_sites s
		JOIN openfaithmap.religion_org_classifications oc ON oc.unit_id = s.org_unit_id AND oc.deleted_at IS NULL
		JOIN openfaithmap.religion_taxa t ON t.id = oc.taxon_id AND t.deleted_at IS NULL
		WHERE `+searchableSitePredicate+`
		ORDER BY t.name`)
	if err != nil {
		return domain.Facets{}, err
	}
	var out domain.Facets
	for traditionRows.Next() {
		var f domain.TraditionFacet
		if err := traditionRows.Scan(&f.TaxonID, &f.TaxonCode, &f.TaxonName); err != nil {
			traditionRows.Close()
			return domain.Facets{}, err
		}
		out.Traditions = append(out.Traditions, f)
	}
	traditionRows.Close()
	if err := traditionRows.Err(); err != nil {
		return domain.Facets{}, err
	}

	languageRows, err := r.conn.Query(ctx, `
		SELECT DISTINCT sch.language
		FROM openfaithmap.religion_sites s
		JOIN openfaithmap.religion_service_schedules sch ON sch.site_id = s.id AND sch.deleted_at IS NULL
		WHERE `+searchableSitePredicate+`
		ORDER BY sch.language`)
	if err != nil {
		return domain.Facets{}, err
	}
	defer languageRows.Close()
	for languageRows.Next() {
		var lang string
		if err := languageRows.Scan(&lang); err != nil {
			return domain.Facets{}, err
		}
		out.Languages = append(out.Languages, lang)
	}
	return out, languageRows.Err()
}
