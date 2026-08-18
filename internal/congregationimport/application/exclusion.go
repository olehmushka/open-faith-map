// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"fmt"

	registrationdomain "github.com/olehmushka/open-faith-map/internal/registration/domain"
)

// checkExcluded mirrors moderation's own CheckExclusion/checkExcluded shape exactly — ctx is
// expected to already carry an authz.SystemContext marker (RunConnector's own doc comment), since a
// connector run has no caller token/subject to check reach against. Reuses registration's own
// domain.ExcludedTaxonCodes directly (import, not copy) — a forked exclusion list is a correctness
// hazard D-Exclusions can't tolerate, the same discipline moderation's own copy already established.
func (s *Service) checkExcluded(ctx context.Context, taxonID string) (excluded bool, excludedCode string, err error) {
	id := taxonID
	for i := 0; i < 10; i++ { // hard cap: never loop forever on an unexpected cycle
		taxon, err := s.religion.GetTaxon(ctx, id)
		if err != nil {
			return false, "", fmt.Errorf("congregationimport: get taxon %s: %w", taxonID, err)
		}
		if registrationdomain.ExcludedTaxonCodes[taxon.Code] {
			return true, taxon.Code, nil
		}
		if taxon.ParentID == nil {
			return false, "", nil
		}
		id = *taxon.ParentID
	}
	return false, "", fmt.Errorf("congregationimport: ancestor chain too deep for %s", taxonID)
}
