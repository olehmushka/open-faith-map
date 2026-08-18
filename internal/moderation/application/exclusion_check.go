// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"fmt"

	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
	registrationdomain "github.com/olehmushka/open-faith-map/internal/registration/domain"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
)

// CheckExclusion answers ModerationPublicService.checkExclusion — a standalone dry-run of the same
// D-Exclusions taxon-ancestor walk internal/registration's own Submit runs inline at registration
// time (registration's copy is untouched by this PR; consolidating the two is deferred, see
// docs/modules/moderation.md's open seams). Reuses registration's own domain.ExcludedTaxonCodes list
// directly (import, not copy) — a forked exclusion list is a correctness hazard D-Exclusions can't
// tolerate.
//
// M10.6: runs under authz.SystemContext(ctx) — one of D-InProcessAuthz amendment #5's five named
// system-context call sites. The caller here is anonymous and has no subject to check against;
// internal/religion carries no authorization logic of its own to trip on this, but the wrapping is
// the named, greppable marker the amendment requires so "this path has no human subject" is never
// implicit.
func (s *Service) CheckExclusion(ctx context.Context, taxonID string) (excluded bool, excludedCode string, err error) {
	return checkExcluded(authz.SystemContext(ctx), s.religion, taxonID)
}

// checkExcluded mirrors internal/registration/application's checkNotExcluded ancestor-walk shape
// exactly, inverted to report the match rather than erroring on it (this endpoint is a dry-run, not
// a gate) — same shallow taxonomy, same hard cap against an unexpected cycle.
func checkExcluded(ctx context.Context, religion *religionapplication.Service, taxonID string) (bool, string, error) {
	id := taxonID
	for i := 0; i < 10; i++ {
		taxon, err := religion.GetTaxon(ctx, id)
		if err != nil {
			return false, "", fmt.Errorf("%w: %s: %v", domain.ErrTaxonNotFound, taxonID, err)
		}
		if registrationdomain.ExcludedTaxonCodes[taxon.Code] {
			return true, taxon.Code, nil
		}
		if taxon.ParentID == nil {
			return false, "", nil
		}
		id = *taxon.ParentID
	}
	return false, "", fmt.Errorf("%w: ancestor chain too deep for %s", domain.ErrTaxonNotFound, taxonID)
}
