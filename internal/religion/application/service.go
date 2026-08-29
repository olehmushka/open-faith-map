// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application is the religion module's orchestration layer: taxon lookups, per-unit org
// profile/classification management, the excludes_child_creation policy check, site management, and
// the closure-aware discovery search. Ported from
// ../go-oikumenea/internal/religion/application/{service.go,discovery.go}, trimmed of clergy/
// affiliation, taxon/classification/org-kind/policy-kind CRUD (static seeded catalogs — see
// migrations/0018_core_religion.sql), and audit recording (no audit log in this port).
package application

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/olehmushka/open-faith-map/internal/directory/domain"
	"github.com/olehmushka/open-faith-map/internal/religion/adapters"
	religiondomain "github.com/olehmushka/open-faith-map/internal/religion/domain"
)

// UnitCreator is the directory-module capability CreateChildOrg needs — reading a unit and creating
// a child unit under it with its canonical edge. religion depends on this interface, defined in
// terms of internal/directory's own domain types (a data dependency, not a call dependency:
// internal/directory never imports internal/religion, matching D-InProcessAuthz amendment #4's
// no-upward-import rule already established for authz<->directory). *directory/application.Service
// satisfies this structurally.
type UnitCreator interface {
	GetUnit(ctx context.Context, id string) (domain.Unit, error)
	CreateUnitWithEdge(ctx context.Context, u domain.Unit, parentID, graphCode string) (domain.Unit, error)
}

type Service struct {
	pool     *pgxpool.Pool
	units    UnitCreator
	authzSvc *authz.Service
}

func NewService(pool *pgxpool.Pool, units UnitCreator, authzSvc *authz.Service) *Service {
	return &Service{pool: pool, units: units, authzSvc: authzSvc}
}

func (s *Service) inTx(ctx context.Context, fn func(store *adapters.Repository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(adapters.NewRepository(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GetTaxon reads a single taxon outside any transaction (read-only, pool-bound).
func (s *Service) GetTaxon(ctx context.Context, id string) (religiondomain.Taxon, error) {
	return adapters.NewRepository(s.pool).GetTaxon(ctx, id)
}

// ListTaxa searches the seeded taxa catalog by code/name — M10.7's core.conjure.yml surface.
func (s *Service) ListTaxa(ctx context.Context, query string, limit int) ([]religiondomain.Taxon, error) {
	return adapters.NewRepository(s.pool).ListTaxa(ctx, query, limit)
}

// GetOrgProfile reads unitID's profile plus its classifications.
func (s *Service) GetOrgProfile(ctx context.Context, unitID string) (religiondomain.OrgProfile, error) {
	store := adapters.NewRepository(s.pool)
	p, err := store.GetOrgProfileRow(ctx, unitID)
	if err != nil {
		return religiondomain.OrgProfile{}, err
	}
	p.Classifications, err = store.ListOrgClassifications(ctx, unitID)
	return p, err
}

// SetOrgProfile upserts unitID's org-kind/short-code and returns the refreshed profile.
func (s *Service) SetOrgProfile(ctx context.Context, unitID string, orgKindID, shortCode *string) (religiondomain.OrgProfile, error) {
	var out religiondomain.OrgProfile
	err := s.inTx(ctx, func(store *adapters.Repository) error {
		p, err := store.UpsertOrgProfile(ctx, unitID, orgKindID, shortCode)
		if err != nil {
			return err
		}
		p.Classifications, err = store.ListOrgClassifications(ctx, unitID)
		out = p
		return err
	})
	return out, err
}

// AddOrgClassification tags unitID with taxonID, clearing any existing primary first if isPrimary.
func (s *Service) AddOrgClassification(ctx context.Context, unitID, taxonID string, isPrimary bool) (religiondomain.OrgClassification, error) {
	var out religiondomain.OrgClassification
	err := s.inTx(ctx, func(store *adapters.Repository) error {
		if isPrimary {
			if err := store.ClearPrimaryClassification(ctx, unitID); err != nil {
				return err
			}
		}
		c, err := store.AddOrgClassification(ctx, unitID, taxonID, isPrimary)
		out = c
		return err
	})
	return out, err
}

// CreateChildOrg builds a child religious-body unit under parentUnitID in the canonical graph — a
// directory unit + its canonical parent->child edge (internal/directory.CreateUnitWithEdge, atomic)
// + the child's org profile + an optional primary classification — rejecting it if the parent
// carries an active excludes_child_creation policy. Ported from
// ../go-oikumenea/internal/religion/application/service.go:559-606; the profile/classification
// writes that follow the atomic unit+edge creation still run in their own transactions, matching
// upstream's own non-atomicity there (D-Hexagonal cross-module mutation).
func (s *Service) CreateChildOrg(ctx context.Context, parentUnitID, code, name string, orgKindID, primaryTaxonID *string) (religiondomain.OrgProfile, error) {
	excluded, err := adapters.NewRepository(s.pool).HasActivePolicy(ctx, parentUnitID, religiondomain.PolicyExcludesChildCreation)
	if err != nil {
		return religiondomain.OrgProfile{}, err
	}
	if excluded {
		return religiondomain.OrgProfile{}, religiondomain.ErrChildCreationExcluded
	}
	if _, err := s.units.GetUnit(ctx, parentUnitID); err != nil {
		return religiondomain.OrgProfile{}, err
	}
	child, err := s.units.CreateUnitWithEdge(ctx, domain.Unit{Code: code, Name: name}, parentUnitID, domain.CanonicalGraphCode)
	if err != nil {
		return religiondomain.OrgProfile{}, err
	}
	if _, err := s.SetOrgProfile(ctx, child.ID, orgKindID, nil); err != nil {
		return religiondomain.OrgProfile{}, err
	}
	if primaryTaxonID != nil && *primaryTaxonID != "" {
		if _, err := s.AddOrgClassification(ctx, child.ID, *primaryTaxonID, true); err != nil {
			return religiondomain.OrgProfile{}, err
		}
	}
	return s.GetOrgProfile(ctx, child.ID)
}

// ---------------------------------------------------------------- sites

// ListSiteTypes returns the seeded religion_site_types catalog.
func (s *Service) ListSiteTypes(ctx context.Context) ([]adapters.SiteType, error) {
	return adapters.NewRepository(s.pool).ListSiteTypes(ctx)
}

// ListOrgKinds returns the seeded religion_org_kinds catalog — added at M10.6 for
// congregationimport's jurisdiction sync (resolveOrgKindIDs), which needs to resolve a stable code
// like "diocese"/"jurisdiction" to its real RID before calling CreateChildOrg.
func (s *Service) ListOrgKinds(ctx context.Context) ([]adapters.OrgKind, error) {
	return adapters.NewRepository(s.pool).ListOrgKinds(ctx)
}

// ListSitesByUnit returns unitID's sites with their EXACT coordinate — callers exposing this to an
// anonymous caller must run each result through religiondomain.Coarsen first; every current caller
// (registration/congregationimport's own-unit site management) is an authenticated owner, not the
// public search arm SearchSites serves.
func (s *Service) ListSitesByUnit(ctx context.Context, unitID string) ([]religiondomain.Site, error) {
	return adapters.NewRepository(s.pool).ListSitesByUnit(ctx, unitID)
}

// GetPrimarySiteByUnit answers M14.11's content-module cross-module read for a site-chrome header/
// footer: the congregation's name and full address components (uncoarsened — the caller applies
// religiondomain.CoarsenAddress itself using the site's own PublicPrecision). Ungated like
// ListSitesByUnit: this method exists specifically for an anonymous public caller (content's
// GetSiteChrome), unlike GetSiteByUnit's owner-only site.manage gate. Returns found=false if the
// unit has no religion site at all.
func (s *Service) GetPrimarySiteByUnit(ctx context.Context, unitID string) (religiondomain.Site, bool, error) {
	return adapters.NewRepository(s.pool).GetPrimarySiteByUnit(ctx, unitID)
}

// ListServiceSchedulesByUnit answers M14.11's site-chrome footer: unitID's primary site's real
// service schedule rows (day/time/language/mode), not just SearchSites'/SearchFacets' aggregated
// facets. Ungated — public-safe, same trust level ServiceDays already gets in DiscoverySite.
// Returns an empty slice (not an error) if the unit has no religion site.
func (s *Service) ListServiceSchedulesByUnit(ctx context.Context, unitID string) ([]religiondomain.ServiceSchedule, error) {
	site, found, err := s.GetPrimarySiteByUnit(ctx, unitID)
	if err != nil || !found {
		return nil, err
	}
	return adapters.NewRepository(s.pool).ListServiceSchedulesBySite(ctx, site.ID)
}

// CreateSite attaches a site to unitID at locationID.
func (s *Service) CreateSite(ctx context.Context, in adapters.CreateSiteInput) (religiondomain.Site, error) {
	return adapters.NewRepository(s.pool).InsertSite(ctx, in)
}

// GetSiteByUnit answers ReligionService.getSite (M13.2) — the owner's own private view of their
// unit's primary site, exact/uncoarsened (unlike discovery's public-precision-filtered
// DiscoverySite). site.manage-gated, target-scoped to unitID. Resolves via ListSitesByUnit's own
// is_primary DESC, id ordering — the same "prefer primary" convention SearchSites(UnitID) already
// establishes — returning ErrSiteNotFound if the unit has no site at all (site creation stays
// registration's/congregationimport's own job, never this method's).
func (s *Service) GetSiteByUnit(ctx context.Context, unitID string) (religiondomain.Site, error) {
	if err := s.requireManage(ctx, unitID); err != nil {
		return religiondomain.Site{}, err
	}
	sites, err := s.ListSitesByUnit(ctx, unitID)
	if err != nil {
		return religiondomain.Site{}, err
	}
	if len(sites) == 0 {
		return religiondomain.Site{}, religiondomain.ErrSiteNotFound
	}
	return sites[0], nil
}

// UpdateSiteAttributes overwrites unitID's primary site's attributes wholesale (M13.2) — the admin
// form always submits the complete SiteAttributes shape, never a partial patch. site.manage-gated,
// target-scoped to unitID; returns ErrSiteNotFound if the unit has no site yet.
func (s *Service) UpdateSiteAttributes(ctx context.Context, unitID string, attrs religiondomain.SiteAttributes) (religiondomain.Site, error) {
	if err := s.requireManage(ctx, unitID); err != nil {
		return religiondomain.Site{}, err
	}
	sites, err := s.ListSitesByUnit(ctx, unitID)
	if err != nil {
		return religiondomain.Site{}, err
	}
	if len(sites) == 0 {
		return religiondomain.Site{}, religiondomain.ErrSiteNotFound
	}
	site := sites[0]
	if err := adapters.NewRepository(s.pool).UpdateSiteAttributesByID(ctx, site.ID, attrs); err != nil {
		return religiondomain.Site{}, err
	}
	site.Attributes = attrs
	return site, nil
}

// SearchSites runs the public discovery search and coarsens each hit's coordinate and address text
// per its own publish precision (religiondomain.Coarsen/CoarsenAddress, D-DiscoveryAddressPrecision)
// — the adapter-level snappedGeom fix keeps a `hidden` site out of the result set (and every other
// site's predicate off the exact geometry) in the first place; this coarsens the RETURNED coordinate
// and address on top of that. Name and Attributes pass through unfiltered — neither is gated by
// precision (see DiscoverySite's own doc comment for why).
func (s *Service) SearchSites(ctx context.Context, q religiondomain.DiscoveryQuery) ([]religiondomain.DiscoverySite, error) {
	sites, err := adapters.NewRepository(s.pool).SearchSites(ctx, q)
	if err != nil {
		return nil, err
	}
	out := make([]religiondomain.DiscoverySite, 0, len(sites))
	for _, site := range sites {
		hit := religiondomain.DiscoverySite{
			ID: site.ID, OrgUnitID: site.OrgUnitID, SiteTypeID: site.SiteTypeID,
			SiteTypeCode: site.SiteTypeCode, SiteTypeName: site.SiteTypeName,
			PublicPrecision: site.PublicPrecision, IsPrimary: site.IsPrimary,
			Name: site.Name, Attributes: site.Attributes,
			TraditionTaxonID: site.TraditionTaxonID, TraditionTaxonCode: site.TraditionTaxonCode,
			TraditionTaxonName: site.TraditionTaxonName,
			ServiceLanguages:   site.ServiceLanguages, ServiceDays: site.ServiceDays,
		}
		if lat, lng, ok := religiondomain.Coarsen(site.Latitude, site.Longitude, site.PublicPrecision); ok {
			hit.Latitude, hit.Longitude = &lat, &lng
		}
		if line, ok := religiondomain.CoarsenAddress(site.Locality, site.AdminArea1, site.AdminArea2, site.Street, site.HouseNumber, site.PostalCode, site.PublicPrecision); ok {
			hit.Address = &line
		}
		out = append(out, hit)
	}
	return out, nil
}

// SearchSitesExact is SearchSites' internal counterpart: the same position-oracle-safe predicate
// (hidden sites excluded, others matched/ordered on snapped geometry — that fix stays), but returns
// each hit's real, uncoarsened coordinate rather than DiscoverySite's public-safe projection.
// Reserved for trusted background callers with a real precision need the coarsened public API can't
// support (congregationimport's dedup — a 250m distance check) — never wire this to an HTTP route.
// Panics if ctx is not authz.SystemContext-marked (D-InProcessAuthz amendment #5's system-context
// convention, enforced here for real rather than left to caller discipline, since what this method
// exposes is exactly what the public search arm exists to withhold).
func (s *Service) SearchSitesExact(ctx context.Context, q religiondomain.DiscoveryQuery) ([]religiondomain.Site, error) {
	authz.MustBeSystemContext(ctx)
	return adapters.NewRepository(s.pool).SearchSites(ctx, q)
}

// SearchFacets returns every distinct tradition taxon / service-schedule language actually present
// among public, non-hidden sites (M13.1) — backs the discovery picker UI so it never offers a
// filter value that would zero out every result.
func (s *Service) SearchFacets(ctx context.Context) (religiondomain.Facets, error) {
	return adapters.NewRepository(s.pool).SearchFacets(ctx)
}
