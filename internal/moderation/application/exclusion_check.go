// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"fmt"

	oikumenea "github.com/olehmushka/go-oikumenea/clients/go"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
	registrationdomain "github.com/olehmushka/open-faith-map/internal/registration/domain"
)

// CheckExclusion answers ModerationPublicService.checkExclusion — a standalone dry-run of the same
// D-Exclusions taxon-ancestor walk internal/registration's own Submit runs inline at registration
// time (registration's copy is untouched by this PR; consolidating the two is deferred, see
// docs/modules/moderation.md's open seams). Reuses registration's own domain.ExcludedTaxonCodes list
// directly (import, not copy) — a forked exclusion list is a correctness hazard D-Exclusions can't
// tolerate.
//
// Runs under the server's OWN service-principal token (coreintegration.NewServiceClient), never a
// forwarded caller token — the caller here is anonymous and has none. This only works because M2.5
// verified the service principal holds religion.read and go-oikumenea's own fix
// (RequireServiceOrPerson, oikumenea:0.0.2) made that grant actually reachable by a machine subject;
// scripts/bootstrap-service-principal already grants it, live-confirmed, not theoretical.
func (s *Service) CheckExclusion(ctx context.Context, taxonID string) (excluded bool, excludedCode string, err error) {
	c, err := s.serviceClient(ctx)
	if err != nil {
		return false, "", err
	}
	return checkExcluded(ctx, c, taxonID)
}

// checkExcluded mirrors internal/registration/application's checkNotExcluded ancestor-walk shape
// exactly, inverted to report the match rather than erroring on it (this endpoint is a dry-run, not
// a gate) — same shallow taxonomy, same hard cap against an unexpected cycle.
func checkExcluded(ctx context.Context, c *oikumenea.Client, taxonID string) (bool, string, error) {
	id := taxonID
	for i := 0; i < 10; i++ {
		taxon, err := c.Religion.GetTaxon(ctx, id)
		if err != nil {
			return false, "", fmt.Errorf("%w: %s: %v", domain.ErrTaxonNotFound, taxonID, err)
		}
		if registrationdomain.ExcludedTaxonCodes[taxon.Code] {
			return true, taxon.Code, nil
		}
		if taxon.ParentId == nil {
			return false, "", nil
		}
		id = *taxon.ParentId
	}
	return false, "", fmt.Errorf("%w: ancestor chain too deep for %s", domain.ErrTaxonNotFound, taxonID)
}
