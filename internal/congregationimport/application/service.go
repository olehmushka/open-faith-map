// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application holds congregationimport's business logic: running a connector
// (service.go), the D-Exclusions check (exclusion.go), taxon-alias resolution (taxonmatch.go),
// naive geo/name dedup (dedup.go), and the operator review + resumable-provisioning write path
// (provision.go).
//
// M10.6 cutover: provisioning writes (ApproveCandidate) call internal/religion/internal/location
// in-process under the approving operator's own context-resolved subject — never the service
// principal, since "is this really a legitimate congregation" is a human judgment call
// (D-CongregationImport, unchanged by the cutover). The read-only paths that used to run under the
// service principal (D-Exclusions check, dedup search, country-name resolution, the jurisdiction
// sync's node fetch) now run under authz.SystemContext instead — one of D-InProcessAuthz amendment
// #5's five named system-context paths per read, since internal/religion/internal/refdata carry no
// authorization logic of their own to check a subject against anyway.
package application

import (
	"context"
	"fmt"

	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/adapters"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	locationapplication "github.com/olehmushka/open-faith-map/internal/location/application"
	refdataapplication "github.com/olehmushka/open-faith-map/internal/refdata/application"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
	"github.com/palantir/pkg/metrics"
)

type Config struct {
	// RootUnitID is the same shared root unit registration/content/discovery/moderation already use
	// (internal/platform/seed.Resolve's RootUnitID) — the target of the operator-scoped Require check, and
	// the default parent for provisioned units when a candidate has no jurisdiction chosen.
	RootUnitID string
	// ActiveGeocoderCode selects which registered Geocoder SuggestCoordinates uses (see geocoders
	// param on NewService) — an env-driven choice (cmd/openfaithmap-api/main.go), not a code change,
	// so swapping providers (or adding a second one to run alongside Nominatim) never touches this
	// module's interface or Conjure surface. Defaults to "nominatim" if empty.
	ActiveGeocoderCode string
	// CatholicJurisdictionAnchorUnitID is the pre-existing Unit RunJurisdictionSync creates every
	// top-level (no-parent) jurisdiction node under — created ONCE, out-of-band, by a human operator
	// through the admin "Create unit" modal (D-CatholicJurisdictionSync,
	// docs/architecture/decisions.md). RunJurisdictionSync never creates or touches anything above or
	// outside this unit's own subtree.
	CatholicJurisdictionAnchorUnitID string
}

type Service struct {
	store    *adapters.Repository
	religion *religionapplication.Service
	location *locationapplication.Service
	refdata  *refdataapplication.Service
	authzSvc *authz.Service
	cfg      Config
	// connectors is the fixed registry of available sources, keyed by Connector.Code() — a plain
	// map rather than a plugin-discovery mechanism, matching this repo's own bias against
	// infrastructure a real need hasn't justified yet (DS-OFM-2's precedent). Adding a source is
	// one line at construction (cmd/openfaithmap-api/main.go), not a schema or interface change.
	connectors map[string]domain.Connector
	// geocoder is the currently-active provider (Config.ActiveGeocoderCode), resolved once here —
	// nil if that code isn't registered, checked at call time in SuggestCoordinates, never a boot
	// failure (same "never a hard failure" discipline connectors already follow).
	geocoder domain.Geocoder
	// jurisdictionSources is RunJurisdictionSync's own fixed registry, same shape/reasoning as
	// connectors above — keyed by JurisdictionSource.Code().
	jurisdictionSources map[string]domain.JurisdictionSource
}

func NewService(
	store *adapters.Repository,
	religionSvc *religionapplication.Service,
	locationSvc *locationapplication.Service,
	refdataSvc *refdataapplication.Service,
	authzSvc *authz.Service,
	cfg Config,
	connectors []domain.Connector,
	geocoders []domain.Geocoder,
	jurisdictionSources []domain.JurisdictionSource,
) *Service {
	byCode := make(map[string]domain.Connector, len(connectors))
	for _, c := range connectors {
		byCode[c.Code()] = c
	}
	geocoderCode := cfg.ActiveGeocoderCode
	if geocoderCode == "" {
		geocoderCode = "nominatim"
	}
	var activeGeocoder domain.Geocoder
	for _, g := range geocoders {
		if g.Code() == geocoderCode {
			activeGeocoder = g
			break
		}
	}
	jsByCode := make(map[string]domain.JurisdictionSource, len(jurisdictionSources))
	for _, js := range jurisdictionSources {
		jsByCode[js.Code()] = js
	}
	return &Service{
		store: store, religion: religionSvc, location: locationSvc, refdata: refdataSvc, authzSvc: authzSvc,
		cfg: cfg, connectors: byCode, geocoder: activeGeocoder, jurisdictionSources: jsByCode,
	}
}

// RunConnector drives sourceCode's connector to completion: Fetch/Normalize one batch at a time,
// write each batch to the staging table immediately (never buffering the whole run in memory — a
// real source can be hundreds of thousands of records), then loop until the connector reports
// exhaustion (a nil nextCursor) or ctx is cancelled. Every batch's D-Exclusions/taxon-match/dedup
// pass runs before that batch is written, so a crash mid-run leaves only fully-processed candidates
// behind, never a half-checked one.
func (s *Service) RunConnector(ctx context.Context, sourceCode, triggeredByPersonRID string, parameters map[string]string) (domain.Run, error) {
	base, ok := s.connectors[sourceCode]
	if !ok {
		return domain.Run{}, domain.ErrRunNotFound
	}
	// Always run against a FRESH, run-scoped connector value — never the long-lived registry
	// instance directly. Real bug fixed by this (found live 2026-08-14): a connector holding
	// in-memory state across Fetch calls (arrnc/osm's loadOnce-cached rows) would otherwise silently
	// replay a PRIOR run's data forever on every subsequent call against the same registered
	// instance, never re-querying the real source. See domain.Connector.Clone's own doc comment.
	var connector domain.Connector
	if len(parameters) > 0 {
		configurable, ok := base.(domain.ConnectorConfigurable)
		if !ok {
			return domain.Run{}, domain.ErrRunParametersNotSupported
		}
		configured, cerr := configurable.WithParameters(parameters)
		if cerr != nil {
			return domain.Run{}, fmt.Errorf("congregationimport: apply run parameters: %w", cerr)
		}
		connector = configured
	} else {
		connector = base.Clone()
	}
	// Some connectors (an HTTP-streaming source) hold run-scoped resources across Fetch calls that
	// only their own success path releases — without this, an error/ctx-cancellation exit below
	// would leak the connector's lock/stream for the rest of the process's life. Best-effort: this
	// run's own outcome is already decided by the time Close runs, never overridden by it.
	if closer, ok := connector.(domain.ConnectorCloser); ok {
		defer func() { _ = closer.Close() }()
	}

	run, err := s.store.CreateRun(ctx, sourceCode, triggeredByPersonRID, parameters, nil)
	if err != nil {
		return domain.Run{}, err
	}

	// failRun centralizes every FAILED-return path below so the connector_run_failures counter
	// (production-hardening pass, mirroring D-Hardening's metrics pattern) is incremented exactly
	// once per real failure, not duplicated at each call site.
	failRun := func(cursor *string, fetched, created, updated, autoRejected int, err error) (domain.Run, error) {
		metrics.FromContext(ctx).Counter("openfaithmap.congregationimport.connector_run_failures").Inc(1)
		errMsg := err.Error()
		return s.store.FinishRun(ctx, run.ID, domain.RunStatusFailed, cursor, fetched, created, updated, autoRejected, &errMsg)
	}

	// sysCtx marks every per-record read below (D-Exclusions check, country match, dedup search) as
	// running under no human subject — the connector run has no caller token to check reach against,
	// and internal/religion/internal/refdata carry no authorization logic of their own to gate on
	// anyway (D-InProcessAuthz amendment #5's "RunConnector import loop (read)" entry).
	sysCtx := authz.SystemContext(ctx)

	var cursor *string
	var fetched, created, updated, autoRejected int
	for {
		if err := ctx.Err(); err != nil {
			return failRun(cursor, fetched, created, updated, autoRejected, err)
		}

		batch, next, fetchErr := connector.Fetch(ctx, cursor)
		fetched += len(batch)

		for _, raw := range batch {
			_, isNew, autoExcluded, procErr := s.processRawRecord(sysCtx, connector, run.ID, raw)
			if procErr != nil {
				return failRun(cursor, fetched, created, updated, autoRejected, procErr)
			}
			if autoExcluded {
				autoRejected++
				metrics.FromContext(ctx).Counter("openfaithmap.congregationimport.candidates_auto_rejected").Inc(1)
			} else if isNew {
				created++
				metrics.FromContext(ctx).Counter("openfaithmap.congregationimport.candidates_staged").Inc(1)
			} else {
				updated++
			}
		}

		cursor = next
		if fetchErr != nil {
			return failRun(cursor, fetched, created, updated, autoRejected, fetchErr)
		}
		if cursor == nil {
			break // connector reports exhaustion
		}
	}

	return s.store.FinishRun(ctx, run.ID, domain.RunStatusSucceeded, cursor, fetched, created, updated, autoRejected, nil)
}

// processRawRecord normalizes, taxon-matches, D-Exclusions-checks, dedup-checks, and stages one raw
// record — the per-record body of RunConnector's streaming loop. ctx is always
// authz.SystemContext-marked (RunConnector's own doc comment) — every internal/religion/internal/refdata
// read below relies on that, not on a per-call client.
func (s *Service) processRawRecord(ctx context.Context, connector domain.Connector, runID string, raw domain.RawRecord) (candidate domain.Candidate, isNew, taxonRejected bool, err error) {
	norm, err := connector.Normalize(raw)
	if err != nil {
		return domain.Candidate{}, false, false, err
	}

	status := domain.StatusStaged
	if norm.Latitude == nil || norm.Longitude == nil {
		status = domain.StatusNeedsGeocode
	}

	c, isNew, err := s.store.UpsertCandidate(ctx, &runID, connector.Code(), raw.SourceRecordID, norm, raw.RawPayload, status)
	if err != nil {
		return domain.Candidate{}, false, false, err
	}
	// A row already past review (see UpsertCandidate's own WHERE clause) is returned unmodified —
	// nothing further to do this run.
	if c.Status == domain.StatusApproved || c.Status == domain.StatusProvisioning ||
		c.Status == domain.StatusProvisioned || c.Status == domain.StatusRejected ||
		c.Status == domain.StatusRejectedExcluded {
		return c, false, false, nil
	}

	// Advisory only, and deliberately independent of the taxon-match outcome below: a candidate can
	// have a resolvable jurisdiction hint with an unresolved taxon or vice versa (e.g. an
	// independent-polity name and a still-unaliased denomination keyword). Never gates status, never
	// applied without the operator's own explicit choice at approval time — D-JurisdictionUnits,
	// jurisdictionmatch.go's own doc comment. Run before the taxon-match early return below (a real
	// bug, caught live: taxon's own !matched branch returns early, which silently skipped this
	// entirely for exactly the NEEDS_TAXON_REVIEW candidates most likely to carry a jurisdiction hint
	// worth surfacing).
	if jurisdictionUnitID, jMatched, jErr := s.matchJurisdiction(ctx, connector.Code(), norm.JurisdictionHint); jErr != nil {
		return domain.Candidate{}, false, false, jErr
	} else if jMatched {
		c, err = s.store.SetJurisdictionMatch(ctx, c.ID, jurisdictionUnitID)
		if err != nil {
			return domain.Candidate{}, false, false, err
		}
	}

	// Same advisory, never-overwrite-the-operator shape as the jurisdiction match above — but unlike
	// a jurisdiction hint, CountryHint (when a connector sets one, e.g. arrnc's Argentina-only
	// "Argentina") is a deterministic fact, not a fuzzy guess, so it's written straight to CountryID
	// rather than a separate "suggested" column. Found live: this hint used to be computed and then
	// silently dropped (countrymatch.go's own doc comment), leaving SuggestCoordinates with no
	// country to query Nominatim with on ~29.6k already-ingested candidates.
	//
	// A matchCountry ERROR is deliberately swallowed rather than propagated, unlike the jurisdiction
	// match above — go-oikumenea's GeoService.ListCountries used to be RequireAnywhere-gated, which
	// structurally denied every machine (service-principal) subject regardless of its grants (the
	// same class of gap scripts/bootstrap-service-principal's own comment documents for religion.read;
	// filed as go-oikumenea#37, fixed there as of image 0.0.5, RequireServiceOrPerson). Kept
	// non-fatal anyway even after that fix landed: a transient go-oikumenea outage or a stale image
	// still shouldn't take the whole connector run down over an advisory field, matching
	// resolveCountryName's own already-established "never blocks" precedent (application/geocode.go)
	// for this exact same ListCountries call.
	if countryID, cMatched, cErr := s.matchCountry(ctx, norm.CountryHint); cErr == nil && cMatched {
		c, err = s.store.SetCountryMatch(ctx, c.ID, countryID)
		if err != nil {
			return domain.Candidate{}, false, false, err
		}
	}

	taxonID, matched, err := s.matchTaxon(ctx, connector.Code(), norm.TaxonHint)
	if err != nil {
		return domain.Candidate{}, false, false, err
	}
	if !matched {
		// D-Scope pre-filter (Christian-only): a taxon hint that resolved to NOTHING at all is the
		// one case this catches — a record that already resolved a real taxon (Christian and
		// legitimately matched, or an excluded denomination caught by checkExcluded below) is never
		// touched here, so this can't ever override checkExcluded's more specific rejection reason
		// with the generic one. isLikelyChristian is a positive keyword match, not a blacklist — see
		// its own doc comment for why.
		if !isLikelyChristian(norm.Name) {
			c, err = s.store.RejectExcluded(ctx, c.ID, "D-Scope: name did not match a Christian-identifying keyword")
			return c, isNew, true, err
		}
		c, err = s.store.SetStatus(ctx, c.ID, domain.StatusNeedsTaxonReview, nil, nil)
		return c, isNew, false, err
	}
	c, err = s.store.SetTaxonMatch(ctx, c.ID, taxonID)
	if err != nil {
		return domain.Candidate{}, false, false, err
	}

	excluded, excludedCode, err := s.checkExcluded(ctx, taxonID)
	if err != nil {
		return domain.Candidate{}, false, false, err
	}
	if excluded {
		c, err = s.store.RejectExcluded(ctx, c.ID, "D-Exclusions: taxon ancestor matched excluded code "+excludedCode)
		return c, isNew, true, err
	}

	dupCandidateID, dupUnitID, isDup, err := s.findPossibleDuplicate(ctx, c)
	if err != nil {
		return domain.Candidate{}, false, false, err
	}
	if isDup {
		c, err = s.store.SetStatus(ctx, c.ID, domain.StatusPossibleDuplicate, dupCandidateID, dupUnitID)
		return c, isNew, false, err
	}

	if c.Status == domain.StatusNeedsTaxonReview {
		// taxon just resolved on this pass (e.g. an alias was added since the last run) — clear back
		// to a reviewable status.
		targetStatus := domain.StatusStaged
		if c.Latitude == nil || c.Longitude == nil {
			targetStatus = domain.StatusNeedsGeocode
		}
		c, err = s.store.SetStatus(ctx, c.ID, targetStatus, nil, nil)
	}
	return c, isNew, false, err
}

// ---- runs/candidates reads ----

func (s *Service) GetRun(ctx context.Context, id string) (domain.Run, error) {
	return s.store.GetRun(ctx, id)
}

// ListRuns/ListCandidates request pageSize+1 rows from the store, mirroring moderation's own M7
// pagination fix (internal/moderation/application/service.go's identical ListReports) — the
// transport layer trims the extra row and builds nextPageToken from it.
func (s *Service) ListRuns(ctx context.Context, sourceCode *string, pageSize int, after *domain.PageCursor) ([]domain.Run, error) {
	return s.store.ListRuns(ctx, sourceCode, pageSize+1, after)
}

func (s *Service) GetCandidate(ctx context.Context, id string) (domain.Candidate, error) {
	return s.store.GetCandidate(ctx, id)
}

func (s *Service) ListCandidates(ctx context.Context, status *domain.Status, sourceCode *string, pageSize int, after *domain.PageCursor) ([]domain.Candidate, error) {
	return s.store.ListCandidates(ctx, status, sourceCode, pageSize+1, after)
}
