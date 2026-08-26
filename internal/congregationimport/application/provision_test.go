// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"testing"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

// TestIsApprovable is a regression test for a real, live-caught bug (docs/milestones-2026-08-07-2026-08-26.md's M8
// detail): the original precondition was a denylist that only excluded
// REJECTED/REJECTED_EXCLUDED, so re-approving an already-PROVISIONED candidate fell through and
// created a real duplicate go-oikumenea unit. isApprovable must be an allowlist — every status not
// explicitly named here must return false, not just the two that were wrongly permitted before.
func TestIsApprovable(t *testing.T) {
	approvable := map[domain.Status]bool{
		domain.StatusStaged:            true,
		domain.StatusNeedsTaxonReview:  true,
		domain.StatusNeedsGeocode:      true,
		domain.StatusPossibleDuplicate: true,
		domain.StatusApproved:          true,
		domain.StatusProvisioning:      true, // genuine crash-resume
		domain.StatusProvisioned:       false,
		domain.StatusRejected:          false,
		domain.StatusRejectedExcluded:  false,
	}
	for status, want := range approvable {
		t.Run(string(status), func(t *testing.T) {
			if got := isApprovable(status); got != want {
				t.Errorf("isApprovable(%s) = %v, want %v", status, got, want)
			}
		})
	}
}

func TestIsApprovableRejectsUnknownStatus(t *testing.T) {
	if isApprovable(domain.Status("SOMETHING_NEW")) {
		t.Error("isApprovable must default-deny any status it doesn't explicitly recognize")
	}
}
