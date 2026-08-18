// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	religionadapters "github.com/olehmushka/open-faith-map/internal/religion/adapters"
	religiondomain "github.com/olehmushka/open-faith-map/internal/religion/domain"
)

// JurisdictionSyncSummary is RunJurisdictionSync's result — an in-memory summary only, not persisted
// as its own run-history row the way congregationimport_runs is for connectors. The
// congregationimport_jurisdiction_units table IS this job's durable, resumable state; a separate run
// ledger was judged not worth the extra schema for a job with no operator-facing review queue (unlike
// RunConnector, whose runs a human browses in the admin UI).
type JurisdictionSyncSummary struct {
	SourceCode     string
	NodesFetched   int
	UnitsCreated   int
	UnitsSkipped   int
	UnitsFailed    int
	AliasesCreated int
}

// RunJurisdictionSync drives sourceCode's JurisdictionSource to completion and creates/resolves its
// jurisdiction-tier Units in go-oikumenea — D-CatholicJurisdictionSync's automated, unattended
// counterpart to RunConnector. Unlike RunConnector, the whole node set is fetched into memory before
// any write happens: a real hierarchy source is a few thousand nodes at most (6,655 worldwide for
// wikidata-catholic, live-verified), three orders of magnitude smaller than a congregation source, so
// there is no memory-pressure reason to stream batch-by-batch the way RunConnector must for a
// hundred-thousand-record source. Buffering in full is also what lets this job compute real
// topological order and the "referenced as someone's parent" org-kind upgrade (see
// upgradeGroupingOrgKinds) without a second pass over the database.
//
// requireOperator-gated at the trigger (this method's own top, M10.6 fix — see D-InProcessAuthz's
// amendment: pre-cutover this was a live gap, transport resolving only `whoami` with no
// authorization check at all, unlike every sibling write in this module). The WRITE itself still
// runs under authz.SystemContext, not the triggering caller's own subject — the one deliberate,
// narrowly-scoped exception in this module to the "writes always use the human operator's own
// subject" precedent (provision.go's ApproveCandidate, D-CongregationImport). This is the
// load-bearing design point D-CatholicJurisdictionSync exists to document: the exception is scoped
// to JURISDICTION-TIER units only (created here), never a congregation-level Unit (still
// exclusively created by ApproveCandidate, unchanged) — see D-CatholicJurisdictionSync for the full
// reasoning.
func (s *Service) RunJurisdictionSync(ctx context.Context, sourceCode string) (JurisdictionSyncSummary, error) {
	if err := s.requireOperator(ctx); err != nil {
		return JurisdictionSyncSummary{}, err
	}
	sysCtx := authz.SystemContext(ctx)

	source, ok := s.jurisdictionSources[sourceCode]
	if !ok {
		return JurisdictionSyncSummary{}, fmt.Errorf("%w: %q", domain.ErrJurisdictionSourceNotFound, sourceCode)
	}

	nodes, err := fetchAllNodes(sysCtx, source)
	if err != nil {
		return JurisdictionSyncSummary{}, err
	}
	summary := JurisdictionSyncSummary{SourceCode: sourceCode, NodesFetched: len(nodes)}
	if len(nodes) == 0 {
		return summary, nil
	}

	byExternalID := make(map[string]domain.JurisdictionNode, len(nodes))
	for _, n := range nodes {
		byExternalID[n.ExternalID] = n
	}
	upgradeGroupingOrgKinds(byExternalID)

	orgKindIDByCode, err := resolveOrgKindIDs(sysCtx, s.religion)
	if err != nil {
		return summary, fmt.Errorf("congregationimport: list org kinds: %w", err)
	}

	resolved := make(map[string]string, len(nodes)) // externalID -> created go-oikumenea unit RID
	remaining := make(map[string]domain.JurisdictionNode, len(nodes))
	for id, n := range byExternalID {
		remaining[id] = n
	}

	// Topological creation: repeat passes over the remaining set, processing any node whose parent is
	// either unset (goes directly under the configured anchor) or already resolved. A pass that
	// resolves nothing means every remaining node's parent chain is broken (points outside this
	// source's own node set, or a genuine cycle) — stop rather than loop forever; those nodes are
	// left PENDING/absent for the next sync run to retry once the real cause is fixed upstream.
	for len(remaining) > 0 {
		progressed := false
		for id, n := range remaining {
			parentUnitID, ready := s.resolveParentUnitID(sysCtx, sourceCode, n, resolved)
			if !ready {
				continue
			}
			unitID, created, skipped, failed := s.ensureJurisdictionUnit(sysCtx, s.religion, sourceCode, n, parentUnitID, orgKindIDByCode)
			if created {
				summary.UnitsCreated++
				if n2, err := s.upsertJurisdictionAliases(sysCtx, unitID, n); err == nil {
					summary.AliasesCreated += n2
				}
			} else if skipped {
				summary.UnitsSkipped++
			} else if failed {
				summary.UnitsFailed++
			}
			if unitID != "" {
				resolved[id] = unitID
			}
			delete(remaining, id)
			progressed = true
		}
		if !progressed {
			break
		}
	}

	return summary, nil
}

// fetchAllNodes drains source.Fetch to completion, mirroring RunConnector's own cursor loop shape
// but collecting into a slice instead of writing each batch immediately — see RunJurisdictionSync's
// own doc comment for why buffering is the right call here specifically.
func fetchAllNodes(ctx context.Context, source domain.JurisdictionSource) ([]domain.JurisdictionNode, error) {
	var all []domain.JurisdictionNode
	var cursor *string
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		batch, next, err := source.Fetch(ctx, cursor)
		if err != nil {
			return nil, fmt.Errorf("congregationimport: fetch jurisdiction nodes: %w", err)
		}
		all = append(all, batch...)
		cursor = next
		if cursor == nil {
			return all, nil
		}
	}
}

// upgradeGroupingOrgKinds re-tags any node that another node in this SAME fetched set points to as
// its parent from the source's own default ("diocese") to "jurisdiction" — a topology-derived
// heuristic instead of a hand-maintained Wikidata-type→org-kind table (which would need to enumerate
// every ecclesiastical-circumscription type Wikidata models — diocese, archdiocese, eparchy,
// apostolic vicariate, ecclesiastical province, ... — with no authoritative single source for the
// mapping). A node that groups other nodes under it structurally IS the "jurisdiction" tier
// go-oikumenea's own religion_org_kinds seeds regardless of its canonical-law title; a leaf
// (everything else) stays "diocese" tier, the safer default.
func upgradeGroupingOrgKinds(byExternalID map[string]domain.JurisdictionNode) {
	isParent := make(map[string]bool, len(byExternalID))
	for _, n := range byExternalID {
		if n.ParentExternalID != nil {
			isParent[*n.ParentExternalID] = true
		}
	}
	for id, n := range byExternalID {
		if isParent[id] {
			n.SuggestedOrgKindID = "jurisdiction"
			byExternalID[id] = n
		}
	}
}

// resolveParentUnitID returns the go-oikumenea unit RID n's Unit should be created under, and
// whether that parent is ready (already resolved this run, or already CREATED on a prior run, or n
// has no parent at all — the configured anchor). Not ready means n's parent hasn't been processed
// yet THIS pass; RunJurisdictionSync's own loop retries it on the next pass.
func (s *Service) resolveParentUnitID(ctx context.Context, sourceCode string, n domain.JurisdictionNode, resolvedThisRun map[string]string) (parentUnitID string, ready bool) {
	if n.ParentExternalID == nil {
		return s.cfg.CatholicJurisdictionAnchorUnitID, true
	}
	if id, ok := resolvedThisRun[*n.ParentExternalID]; ok {
		return id, true
	}
	existing, err := s.store.GetJurisdictionUnitByNaturalKey(ctx, sourceCode, *n.ParentExternalID)
	if err == nil && existing.Status == domain.JurisdictionUnitStatusCreated && existing.CreatedUnitID != nil {
		return *existing.CreatedUnitID, true
	}
	return "", false
}

// ensureJurisdictionUnit checks the natural-key idempotency anchor before ever calling
// createChildOrg — an already-CREATED row is skipped outright (unitID, created=false, skipped=true);
// a FAILED or absent row is (re)attempted. orgKindIDByCode resolves n.SuggestedOrgKindID (a stable
// code like "diocese") to the real go-oikumenea OrgKind RID CreateChildOrgRequest.OrgKindId actually
// expects — found live (not by review): the field name reads like a code but is a real RID, the same
// list-then-match-by-Code pattern provision.go's churchSiteTypeID already established for site
// types. An unresolvable code is a config/deploy error (a future org-kind this map doesn't yet cover,
// or a genuine go-oikumenea catalog gap), not a per-node failure — it fails this node loudly via
// MarkJurisdictionUnitFailed rather than falling back to an unrelated kind.
func (s *Service) ensureJurisdictionUnit(ctx context.Context, c serviceClient, sourceCode string, n domain.JurisdictionNode, parentUnitID string, orgKindIDByCode map[string]string) (unitID string, created, skipped, failed bool) {
	existing, err := s.store.GetJurisdictionUnitByNaturalKey(ctx, sourceCode, n.ExternalID)
	hasExisting := err == nil
	if hasExisting && existing.Status == domain.JurisdictionUnitStatusCreated && existing.CreatedUnitID != nil {
		return *existing.CreatedUnitID, false, true, false
	}

	recordID := ""
	if hasExisting {
		recordID = existing.ID
	} else {
		rec, cerr := s.store.CreatePendingJurisdictionUnit(ctx, sourceCode, n.ExternalID, n.ParentExternalID, n.Name, n.SuggestedOrgKindID)
		if cerr != nil {
			return "", false, false, true
		}
		recordID = rec.ID
	}

	orgKindID, ok := orgKindIDByCode[n.SuggestedOrgKindID]
	if !ok {
		_, _ = s.store.MarkJurisdictionUnitFailed(ctx, recordID, fmt.Sprintf("no religion_org_kinds row found for code %q", n.SuggestedOrgKindID))
		return "", false, false, true
	}
	profile, cerr := c.CreateChildOrg(ctx, parentUnitID, jurisdictionSlugCode(n.Name, n.ExternalID), n.Name, &orgKindID, nil)
	if cerr != nil {
		_, _ = s.store.MarkJurisdictionUnitFailed(ctx, recordID, cerr.Error())
		return "", false, false, true
	}
	if _, merr := s.store.MarkJurisdictionUnitCreated(ctx, recordID, profile.UnitID); merr != nil {
		return "", false, false, true
	}
	return profile.UnitID, true, false, false
}

// upsertJurisdictionAliases writes one congregationimport_jurisdiction_aliases row per distinct name
// (Name + AliasNames) pointing at jurisdictionUnitID — the just-created/resolved go-oikumenea unit
// RID for n, never n's own external ID. Global (sourceCode = nil) so every existing and future
// connector's matchJurisdiction substring logic benefits immediately — see this file's own package
// doc reference to D-CatholicJurisdictionSync. An ErrAliasConflict (the alias text already exists,
// either from a prior sync run or a coincidental collision with another diocese's label) is treated
// as success, not a failure — first-writer-wins, same as every other idempotent-on-conflict write in
// this module.
func (s *Service) upsertJurisdictionAliases(ctx context.Context, jurisdictionUnitID string, n domain.JurisdictionNode) (int, error) {
	names := append([]string{n.Name}, n.AliasNames...)
	created := 0
	for _, name := range names {
		normalized := normalizeAlias(name)
		if normalized == "" {
			continue
		}
		_, err := s.store.CreateJurisdictionAlias(ctx, nil, normalized, jurisdictionUnitID, jurisdictionSyncSystemPersonRID)
		if err == nil {
			created++
			continue
		}
		if errors.Is(err, domain.ErrAliasConflict) {
			continue
		}
		return created, err
	}
	return created, nil
}

// jurisdictionSyncSystemPersonRID records WHO created an auto-synced alias — there is no real person
// behind this write (it runs under the service principal, RunJurisdictionSync's own doc comment), so
// a fixed, greppable sentinel value is used rather than an empty string or a fabricated person RID.
const jurisdictionSyncSystemPersonRID = "system:wikidata-catholic-jurisdiction-sync"

var jurisdictionSlugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// jurisdictionSlugCode builds a STABLE (not random-suffixed, unlike provision.go's slugCode) code
// from a node's name and external ID — stable because re-running this job must derive the identical
// code for the identical node, so a retry after a transient createChildOrg failure doesn't collide
// with (or orphan) a code from the failed attempt.
func jurisdictionSlugCode(name, externalID string) string {
	slug := strings.Trim(jurisdictionSlugNonAlnum.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if slug == "" {
		slug = "jurisdiction"
	}
	if len(slug) > 40 {
		slug = slug[:40]
	}
	return slug + "-" + strings.ToLower(externalID)
}

// serviceClient is the minimal surface RunJurisdictionSync needs from internal/religion's app
// service — narrowed so ensureJurisdictionUnit is unit-testable against a fake without constructing
// a real one. *religionapplication.Service satisfies this structurally.
type serviceClient interface {
	CreateChildOrg(ctx context.Context, parentUnitID, code, name string, orgKindID, primaryTaxonID *string) (religiondomain.OrgProfile, error)
	ListOrgKinds(ctx context.Context) ([]religionadapters.OrgKind, error)
}

// resolveOrgKindIDs lists the real religion_org_kinds catalog once per sync run and returns a
// code->RID map — CreateChildOrg's orgKindID param is a real RID, not the stable code string
// JurisdictionNode.SuggestedOrgKindID carries, the same list-then-match-by-Code pattern
// provision.go's churchSiteTypeID already established for site types.
func resolveOrgKindIDs(ctx context.Context, c serviceClient) (map[string]string, error) {
	kinds, err := c.ListOrgKinds(ctx)
	if err != nil {
		return nil, err
	}
	byCode := make(map[string]string, len(kinds))
	for _, k := range kinds {
		byCode[k.Code] = k.ID
	}
	return byCode, nil
}
