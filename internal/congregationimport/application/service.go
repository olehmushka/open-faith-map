// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application holds congregationimport's business logic: running a connector
// (service.go), the D-Exclusions check under the service principal (exclusion.go), taxon-alias
// resolution (taxonmatch.go), naive geo/name dedup (dedup.go), and the operator review +
// resumable-provisioning write path (provision.go) — always with the CALLER's own forwarded token
// for any go-oikumenea write (D-Facade), never the service principal, since "is this really a
// legitimate congregation" is a human judgment call.
package application

import (
	"context"

	oikumenea "github.com/olehmushka/go-oikumenea/clients/go"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/adapters"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	"github.com/olehmushka/open-faith-map/internal/coreintegration"
	"github.com/palantir/pkg/metrics"
)

type Config struct {
	OikumeneaBaseURL            string
	OikumeneaInsecureSkipVerify bool
	// RootUnitID is the same shared root unit registration/content/discovery/moderation already use
	// (scripts/bootstrap-registration-org) — the target of the operator-scoped Authorize check, and
	// the default parent for provisioned units when a candidate has no jurisdiction chosen.
	RootUnitID string
	// ServicePrincipal configures the server's own go-oikumenea call for the D-Exclusions check and
	// dedup search — read-only uses only; provisioning writes always use the operator's own token.
	ServicePrincipal coreintegration.Config
}

type Service struct {
	store *adapters.Store
	cfg   Config
	// connectors is the fixed registry of available sources, keyed by Connector.Code() — a plain
	// map rather than a plugin-discovery mechanism, matching this repo's own bias against
	// infrastructure a real need hasn't justified yet (DS-OFM-2's precedent). Adding a source is
	// one line at construction (cmd/openfaithmap-api/main.go), not a schema or interface change.
	connectors map[string]domain.Connector
}

func NewService(store *adapters.Store, cfg Config, connectors []domain.Connector) *Service {
	byCode := make(map[string]domain.Connector, len(connectors))
	for _, c := range connectors {
		byCode[c.Code()] = c
	}
	return &Service{store: store, cfg: cfg, connectors: byCode}
}

func (s *Service) userClient(token string) (*oikumenea.Client, error) {
	return coreintegration.NewUserClient(s.cfg.OikumeneaBaseURL, token, s.cfg.OikumeneaInsecureSkipVerify)
}

func (s *Service) serviceClient(ctx context.Context) (*oikumenea.Client, error) {
	return coreintegration.NewServiceClient(ctx, s.cfg.ServicePrincipal)
}

// RunConnector drives sourceCode's connector to completion: Fetch/Normalize one batch at a time,
// write each batch to the staging table immediately (never buffering the whole run in memory — a
// real source can be hundreds of thousands of records), then loop until the connector reports
// exhaustion (a nil nextCursor) or ctx is cancelled. Every batch's D-Exclusions/taxon-match/dedup
// pass runs before that batch is written, so a crash mid-run leaves only fully-processed candidates
// behind, never a half-checked one.
func (s *Service) RunConnector(ctx context.Context, sourceCode, triggeredByPersonRID string) (domain.Run, error) {
	connector, ok := s.connectors[sourceCode]
	if !ok {
		return domain.Run{}, domain.ErrRunNotFound
	}
	// Some connectors (an HTTP-streaming source) hold run-scoped resources across Fetch calls that
	// only their own success path releases — without this, an error/ctx-cancellation exit below
	// would leak the connector's lock/stream for the rest of the process's life. Best-effort: this
	// run's own outcome is already decided by the time Close runs, never overridden by it.
	if closer, ok := connector.(domain.ConnectorCloser); ok {
		defer func() { _ = closer.Close() }()
	}

	run, err := s.store.CreateRun(ctx, sourceCode, triggeredByPersonRID, nil)
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

	svc, err := s.serviceClient(ctx)
	if err != nil {
		return failRun(nil, 0, 0, 0, 0, err)
	}

	var cursor *string
	var fetched, created, updated, autoRejected int
	for {
		if err := ctx.Err(); err != nil {
			return failRun(cursor, fetched, created, updated, autoRejected, err)
		}

		batch, next, fetchErr := connector.Fetch(ctx, cursor)
		fetched += len(batch)

		for _, raw := range batch {
			_, isNew, autoExcluded, procErr := s.processRawRecord(ctx, svc, connector, run.ID, raw)
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
// record — the per-record body of RunConnector's streaming loop.
func (s *Service) processRawRecord(ctx context.Context, svc *oikumenea.Client, connector domain.Connector, runID string, raw domain.RawRecord) (candidate domain.Candidate, isNew, taxonRejected bool, err error) {
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

	excluded, excludedCode, err := s.checkExcluded(ctx, svc, taxonID)
	if err != nil {
		return domain.Candidate{}, false, false, err
	}
	if excluded {
		c, err = s.store.RejectExcluded(ctx, c.ID, "D-Exclusions: taxon ancestor matched excluded code "+excludedCode)
		return c, isNew, true, err
	}

	dupCandidateID, dupUnitID, isDup, err := s.findPossibleDuplicate(ctx, svc, c)
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

func (s *Service) ListCandidates(ctx context.Context, status *domain.Status, pageSize int, after *domain.PageCursor) ([]domain.Candidate, error) {
	return s.store.ListCandidates(ctx, status, pageSize+1, after)
}
