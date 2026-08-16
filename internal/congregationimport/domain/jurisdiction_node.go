// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import "context"

// JurisdictionNode is one hierarchy-tree node from a JurisdictionSource — a prospective
// jurisdiction-tier go-oikumenea Unit (country/province/diocese/eparchy/deanery, never a
// congregation), before it's been created or resolved. Distinct from NormalizedCandidate, which is
// always congregation-level.
type JurisdictionNode struct {
	// ExternalID is the source's own stable natural key for this node (e.g. a Wikidata QID like
	// "Q40741") — the idempotency anchor alongside SourceCode, mirroring RawRecord.SourceRecordID's
	// role for congregation candidates.
	ExternalID string
	// ParentExternalID is the immediate parent's ExternalID within the same source/run, or nil for a
	// node with no modeled parent in this source (created directly under the sync's configured
	// anchor unit). Resolved against already-synced nodes before this one is created — see
	// JurisdictionSyncStore.
	ParentExternalID *string
	// Name is the node's primary display name, in whatever language the source considers canonical
	// (English, for Wikidata's rdfs:label default).
	Name string
	// AliasNames are additional known name variants (e.g. Wikidata's per-locale labels/altLabels) —
	// each becomes its own congregationimport_jurisdiction_aliases row, so a connector's free-text
	// JurisdictionHint in any of these languages/scripts can match. May be empty.
	AliasNames []string
	// CountryHint is the node's country, when known (e.g. resolved from Wikidata's wdt:P17) — carried
	// through for future country-scoped filtering; not currently consumed beyond that.
	CountryHint *string
	// SuggestedOrgKindID is this node's go-oikumenea religion_org_kinds code (e.g. "jurisdiction",
	// "diocese") — purely descriptive on the created Unit, never branched on (D-JurisdictionUnits).
	SuggestedOrgKindID string
}

// JurisdictionSource is the Strategy-pattern interface a hierarchy-tree source implements —
// deliberately separate from Connector (which is congregation-shaped and produces
// NormalizedCandidate, not tree nodes). Kept as small and symmetrical to Connector as the shape
// allows, so the same citation/robots.txt discipline (D-CongregationImport decision #4) applies
// uniformly.
type JurisdictionSource interface {
	// Code is the stable source_code stored on every congregationimport_jurisdiction_units row.
	Code() string
	// Citation mirrors Connector.Citation's own doc comment exactly.
	Citation() SourceCitation
	// Fetch returns one batch of JurisdictionNodes plus a cursor to resume from, or a nil cursor when
	// the source is exhausted. A node's parent, when set, must already have appeared in an earlier
	// batch of the SAME run (or a prior run) — sources are expected to emit nodes in a
	// parent-before-child order; JurisdictionSyncService does not reorder across batches.
	Fetch(ctx context.Context, cursor *string) (batch []JurisdictionNode, nextCursor *string, err error)
}
