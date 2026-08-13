// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"math"

	oikumenea "github.com/olehmushka/go-oikumenea/clients/go"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

// duplicateRadiusMeters is the fixed geo-radius dedup uses — deliberately simple for v1 (see
// docs/modules/congregationimport.md's open seams; a pg_trgm fuzzy-name upgrade is real future
// work, not built here).
const duplicateRadiusMeters = 250.0

// findPossibleDuplicate checks c against already-provisioned go-oikumenea sites near its own
// coordinates, via the service principal's SearchSites bbox call — the exact call discovery's own
// RefreshRegion already makes. Geo-proximity only (go-oikumenea's DiscoverySite carries no name
// field to compare against — checked directly against the Conjure struct, not assumed) — a real
// gap for precision, deliberately accepted for v1: this never auto-merges or auto-rejects, only
// flags for the operator's own judgment, so a same-building-different-congregation false positive
// costs one extra human decision, not a wrong automated one. A candidate with no coordinates yet
// (e.g. every ЄДР row — that source has no address field at all, see the uaedr connector's own doc
// comment) has nothing to compare against and is never flagged here; NEEDS_GEOCODE already covers
// that case.
func (s *Service) findPossibleDuplicate(ctx context.Context, c *oikumenea.Client, cand domain.Candidate) (dupCandidateID, dupUnitID *string, isDup bool, err error) {
	if cand.Latitude == nil || cand.Longitude == nil {
		return nil, nil, false, nil
	}

	const deltaDeg = 0.01 // ~1.1km at the equator — generous bbox, haversineMeters does the real cut
	minLat, maxLat := *cand.Latitude-deltaDeg, *cand.Latitude+deltaDeg
	minLng, maxLng := *cand.Longitude-deltaDeg, *cand.Longitude+deltaDeg
	page, err := c.Religion.SearchSites(ctx,
		nil, nil, nil, &minLat, &minLng, &maxLat, &maxLng, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return nil, nil, false, err
	}
	for _, site := range page.Sites {
		if site.Latitude == nil || site.Longitude == nil {
			continue
		}
		if haversineMeters(*cand.Latitude, *cand.Longitude, *site.Latitude, *site.Longitude) > duplicateRadiusMeters {
			continue
		}
		unitID := site.OrgUnitId
		return nil, &unitID, true, nil
	}
	return nil, nil, false, nil
}

// haversineMeters — deliberate duplication of discovery's own client-side, non-PostGIS distance
// calc (D-Facade: OpenFaithMap owns no location index of its own), not imported, matching this
// repo's established "each module holds its own copy" convention for small shared shapes.
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6371000.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
