// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"strings"

	congregationimportadapters "github.com/olehmushka/open-faith-map/internal/congregationimport/adapters"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/adapters/connectors/arrnc"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/adapters/connectors/osm"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/adapters/connectors/uaedr"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/adapters/geocoders/nominatim"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/adapters/jurisdictionsources/wikidatacatholic"
	congregationimportapplication "github.com/olehmushka/open-faith-map/internal/congregationimport/application"
	congregationimportdomain "github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	congregationimporttransport "github.com/olehmushka/open-faith-map/internal/congregationimport/transport"
	gencongregationimport "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/congregationimport"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
)

// registerCongregationImport (D-CongregationImport, docs/modules/congregationimport.md): connectors
// are a fixed registry built here, not a plugin-discovery mechanism (application.Service's own doc
// comment). UAEDR_UO_FILE_PATH/UAEDR_SOURCE_URL are both optional and mutually exclusive (see
// uaedr.New) — the ЄДР connector is only registered when an operator has configured one of them;
// omitting both leaves the module running with zero connectors registered (RunConnector then returns
// ErrRunNotFound for any sourceCode), never a boot failure. UAEDR_SOURCE_URL streams directly from
// the remote export (no local disk landing spot needed — a cheap, memory-constrained VM deployment);
// the exact current data.gov.ua resource URL is an operator-supplied value, never hardcoded here.
func registerCongregationImport(ctx context.Context, info witchcraft.InitInfo, deps *Deps) error {
	var connectors []congregationimportdomain.Connector
	if uaedrFilePath, uaedrSourceURL := os.Getenv("UAEDR_UO_FILE_PATH"), os.Getenv("UAEDR_SOURCE_URL"); uaedrFilePath != "" || uaedrSourceURL != "" {
		uaedrConnector, err := uaedr.New(uaedrFilePath, uaedrSourceURL, nil)
		if err != nil {
			return werror.WrapWithContextParams(ctx, err, "construct uaedr connector")
		}
		connectors = append(connectors, uaedrConnector)
	}
	// ar-rnc (Argentina's Registro Nacional de Cultos) — same optional/never-a-boot-failure pattern
	// as uaedr above; ARRNC_FILE_PATH/ARRNC_SOURCE_URL are both optional and mutually exclusive
	// (see arrnc.New). Unlike uaedr, this source is small (3.6MB, not ~326MB) — no HTTP-streaming
	// mode exists or is needed, see arrnc's own package doc comment.
	if arrncFilePath, arrncSourceURL := os.Getenv("ARRNC_FILE_PATH"), os.Getenv("ARRNC_SOURCE_URL"); arrncFilePath != "" || arrncSourceURL != "" {
		arrncConnector, err := arrnc.New(arrncFilePath, arrncSourceURL, nil)
		if err != nil {
			return werror.WrapWithContextParams(ctx, err, "construct arrnc connector")
		}
		connectors = append(connectors, arrncConnector)
	}
	// osm (OpenStreetMap, Overpass API) — same optional/never-a-boot-failure pattern as uaedr/arrnc
	// above, gated on OSM_COUNTRY_CODES rather than a file/URL pair since OSM_OVERPASS_BASE_URL has a
	// sensible default (overpass.kumi.systems — see osm's own package doc comment for why this mirror
	// was chosen over the main OSM-Foundation-run one). OSM_COUNTRY_CODES is a comma-separated list
	// of ISO 3166-1 alpha-2 codes (e.g. "UY,PY,CO,CL"), deliberately not defaulted here — the
	// operator must opt in to a country list explicitly.
	if osmCountryCodesRaw := os.Getenv("OSM_COUNTRY_CODES"); osmCountryCodesRaw != "" {
		var osmCountryCodes []string
		for _, code := range strings.Split(osmCountryCodesRaw, ",") {
			if code = strings.TrimSpace(code); code != "" {
				osmCountryCodes = append(osmCountryCodes, code)
			}
		}
		osmConnector, err := osm.New(os.Getenv("OSM_OVERPASS_BASE_URL"), osmCountryCodes, nil)
		if err != nil {
			return werror.WrapWithContextParams(ctx, err, "construct osm connector")
		}
		connectors = append(connectors, osmConnector)
	}
	// Geocoders: a fixed registry, same shape/reasoning as the connectors slice above — Nominatim
	// is free and keyless, so it's always registered (no env-gate needed to enable it), but which
	// one actually SERVES SuggestCoordinates is still an env-driven choice
	// (CONGREGATIONIMPORT_GEOCODER, application.Config.ActiveGeocoderCode), so adding a second
	// provider (LocationIQ, Google) later — and switching to it — needs no code change here beyond
	// one more append.
	geocoders := []congregationimportdomain.Geocoder{nominatim.New(nil)}

	// wikidata-catholic (D-CatholicJurisdictionSync, docs/architecture/decisions.md): only
	// registered when an operator has configured the one-time, human-created anchor unit
	// (CATHOLIC_JURISDICTION_ANCHOR_UNIT_ID) — same never-a-boot-failure, opt-in pattern as
	// uaedr/arrnc/osm above. CATHOLIC_JURISDICTION_COUNTRY_QIDS optionally scopes the sync to
	// specific Wikidata country QIDs (e.g. "Q212" for Ukraine, the owner's own first
	// live-verification target) — comma-separated, empty means every country worldwide.
	var jurisdictionSources []congregationimportdomain.JurisdictionSource
	catholicAnchorUnitID := os.Getenv("CATHOLIC_JURISDICTION_ANCHOR_UNIT_ID")
	if catholicAnchorUnitID != "" {
		var countryQIDs []string
		for _, q := range strings.Split(os.Getenv("CATHOLIC_JURISDICTION_COUNTRY_QIDS"), ",") {
			if q = strings.TrimSpace(q); q != "" {
				countryQIDs = append(countryQIDs, q)
			}
		}
		wikidataSource, err := wikidatacatholic.New(os.Getenv("CATHOLIC_JURISDICTION_WIKIDATA_BASE_URL"), countryQIDs, nil)
		if err != nil {
			return werror.WrapWithContextParams(ctx, err, "construct wikidata-catholic jurisdiction source")
		}
		jurisdictionSources = append(jurisdictionSources, wikidataSource)
	}

	congregationimportStore := congregationimportadapters.NewRepository(deps.Pool)
	congregationimportAppSvc := congregationimportapplication.NewService(
		congregationimportStore, deps.ReligionSvc, deps.LocationSvc, deps.RefdataSvc, deps.AuthzSvc,
		congregationimportapplication.Config{
			RootUnitID:                       deps.CoreRootUnitID,
			ActiveGeocoderCode:               os.Getenv("CONGREGATIONIMPORT_GEOCODER"),
			CatholicJurisdictionAnchorUnitID: catholicAnchorUnitID,
		}, connectors, geocoders, jurisdictionSources)
	congregationimportTransportSvc := congregationimporttransport.NewService(congregationimportAppSvc)

	if err := gencongregationimport.RegisterRoutesCongregationImportService(info.Router, congregationimportTransportSvc); err != nil {
		return werror.WrapWithContextParams(ctx, err, "register congregationimport routes")
	}
	return nil
}
