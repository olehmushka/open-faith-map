// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"testing"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

func strPtrJS(s string) *string { return &s }

// TestUpgradeGroupingOrgKinds is a regression test for the topology-derived org-kind heuristic:
// a node referenced as another node's parent within the SAME fetched set becomes "jurisdiction"
// tier; every other node (including one referenced by nothing, and one whose parent lies OUTSIDE
// this set entirely) stays the source's own default ("diocese").
func TestUpgradeGroupingOrgKinds(t *testing.T) {
	byExternalID := map[string]domain.JurisdictionNode{
		"Q1": {ExternalID: "Q1", ParentExternalID: nil, SuggestedOrgKindID: "diocese"},              // province, groups Q2/Q3
		"Q2": {ExternalID: "Q2", ParentExternalID: strPtrJS("Q1"), SuggestedOrgKindID: "diocese"},   // leaf diocese
		"Q3": {ExternalID: "Q3", ParentExternalID: strPtrJS("Q1"), SuggestedOrgKindID: "diocese"},   // leaf diocese
		"Q4": {ExternalID: "Q4", ParentExternalID: strPtrJS("Q999"), SuggestedOrgKindID: "diocese"}, // parent outside this set
	}

	upgradeGroupingOrgKinds(byExternalID)

	wantKind := map[string]string{
		"Q1": "jurisdiction", // referenced as Q2 and Q3's parent
		"Q2": "diocese",
		"Q3": "diocese",
		"Q4": "diocese", // its own parent is unresolved-in-set, but Q4 itself is referenced by nobody
	}
	for id, want := range wantKind {
		if got := byExternalID[id].SuggestedOrgKindID; got != want {
			t.Errorf("byExternalID[%q].SuggestedOrgKindID = %q, want %q", id, got, want)
		}
	}
}

func TestJurisdictionSlugCodeIsStableAcrossCalls(t *testing.T) {
	a := jurisdictionSlugCode("Roman Catholic Archdiocese of Lviv", "Q1103533")
	b := jurisdictionSlugCode("Roman Catholic Archdiocese of Lviv", "Q1103533")
	if a != b {
		t.Errorf("jurisdictionSlugCode must be deterministic (retry-safe): got %q then %q", a, b)
	}
}

func TestJurisdictionSlugCodeDistinguishesDifferentNodes(t *testing.T) {
	a := jurisdictionSlugCode("Diocese of Example", "Q1")
	b := jurisdictionSlugCode("Diocese of Example", "Q2")
	if a == b {
		t.Errorf("jurisdictionSlugCode(%q, Q1) and (%q, Q2) must differ, both got %q", "Diocese of Example", "Diocese of Example", a)
	}
}

func TestJurisdictionSlugCodeHandlesEmptyName(t *testing.T) {
	got := jurisdictionSlugCode("", "Q212")
	if got != "jurisdiction-q212" {
		t.Errorf("jurisdictionSlugCode(\"\", %q) = %q, want %q", "Q212", got, "jurisdiction-q212")
	}
}
