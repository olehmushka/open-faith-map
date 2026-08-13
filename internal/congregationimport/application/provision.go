// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	oikumenea "github.com/olehmushka/go-oikumenea/clients/go"
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/authorization"
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/location"
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/religion"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	"github.com/palantir/pkg/metrics"
)

// operatorPermission mirrors registration/content/discovery/moderation's own identical constant —
// this repo's deliberate convention is each module holding its own copy of the target-scoped
// capability check, not importing another module's application package.
const operatorPermission = "religionorg.manage"

// requireOperator asks go-oikumenea's real PDP whether the caller holds operatorPermission on
// Config.RootUnitID specifically — the same target-scoped pattern every other module's own
// IsOperator/requireOperator/requireManage already uses.
func (s *Service) requireOperator(ctx context.Context, token, callerPersonID string) error {
	c, err := s.userClient(token)
	if err != nil {
		return err
	}
	rootUnitID := s.cfg.RootUnitID
	resp, err := c.Authorization.Authorize(ctx, authorization.AuthorizeRequest{
		SubjectPersonId: callerPersonID,
		Action:          operatorPermission,
		UnitId:          &rootUnitID,
	})
	if err != nil {
		if authorization.IsPermissionDenied(err) {
			return domain.ErrForbidden
		}
		return err
	}
	if !resp.Allow {
		return domain.ErrForbidden
	}
	return nil
}

func (s *Service) EditCandidate(ctx context.Context, token, callerPersonID, id string, in domain.EditInput) (domain.Candidate, error) {
	if err := s.requireOperator(ctx, token, callerPersonID); err != nil {
		return domain.Candidate{}, err
	}
	return s.store.Edit(ctx, id, in)
}

func (s *Service) RejectCandidate(ctx context.Context, token, callerPersonID, id, reason string) (domain.Candidate, error) {
	if err := s.requireOperator(ctx, token, callerPersonID); err != nil {
		return domain.Candidate{}, err
	}
	return s.store.Reject(ctx, id, callerPersonID, reason)
}

// ---- alias management (production-hardening pass: previously SQL-only, see
// docs/modules/congregationimport.md) ----

func (s *Service) ListTaxonAliases(ctx context.Context, token, callerPersonID string, sourceCode *string) ([]domain.TaxonAlias, error) {
	if err := s.requireOperator(ctx, token, callerPersonID); err != nil {
		return nil, err
	}
	if sourceCode == nil {
		return s.store.ListAllTaxonAliases(ctx)
	}
	return s.store.ListAliasesForMatching(ctx, *sourceCode)
}

func (s *Service) CreateTaxonAlias(ctx context.Context, token, callerPersonID string, sourceCode *string, aliasText, taxonID string) (domain.TaxonAlias, error) {
	if err := s.requireOperator(ctx, token, callerPersonID); err != nil {
		return domain.TaxonAlias{}, err
	}
	return s.store.CreateTaxonAlias(ctx, sourceCode, normalizeAlias(aliasText), taxonID, callerPersonID)
}

func (s *Service) ListJurisdictionAliases(ctx context.Context, token, callerPersonID string, sourceCode *string) ([]domain.JurisdictionAlias, error) {
	if err := s.requireOperator(ctx, token, callerPersonID); err != nil {
		return nil, err
	}
	if sourceCode == nil {
		return s.store.ListAllJurisdictionAliases(ctx)
	}
	return s.store.ListJurisdictionAliasesForMatching(ctx, *sourceCode)
}

func (s *Service) CreateJurisdictionAlias(ctx context.Context, token, callerPersonID string, sourceCode *string, aliasText, jurisdictionUnitID string) (domain.JurisdictionAlias, error) {
	if err := s.requireOperator(ctx, token, callerPersonID); err != nil {
		return domain.JurisdictionAlias{}, err
	}
	return s.store.CreateJurisdictionAlias(ctx, sourceCode, normalizeAlias(aliasText), jurisdictionUnitID, callerPersonID)
}

// isApprovable is an allowlist, not a denylist — a real bug, caught live (not by review): the
// original denylist only excluded REJECTED/REJECTED_EXCLUDED, so calling ApproveCandidate a second
// time on an already-PROVISIONED candidate fell through to ensureUnit, whose own resume-check only
// short-circuits on PROVISIONING (not PROVISIONED), and called createChildOrg a second time — a
// real duplicate unit, confirmed created in go-oikumenea before this fix. Only PROVISIONING (a
// genuine crash-resume) and the pre-approval statuses may reach ensureUnit at all. Pure — no I/O —
// split out from ApproveCandidate so it's directly unit-testable (a regression test for the exact
// bug above) without a live store/go-oikumenea client.
func isApprovable(status domain.Status) bool {
	switch status {
	case domain.StatusStaged, domain.StatusNeedsTaxonReview, domain.StatusNeedsGeocode,
		domain.StatusPossibleDuplicate, domain.StatusApproved, domain.StatusProvisioning:
		return true
	default:
		return false
	}
}

// ApproveCandidate performs the real go-oikumenea write, under the APPROVING OPERATOR'S OWN
// forwarded token — never the service principal (createChildOrg's real gate,
// religionorg.manage/assignment.grant, is a human-held permission; "is this really a legitimate
// congregation" is exactly the judgment call this review step exists for — D-CongregationImport).
// Resumable, mirroring registration.Approve's exact ensureUnit/ensureSite shape (M2.3's
// crash-resume pattern) — minus the position/fill/grantAssignment steps: there is no submitter to
// grant congregation-admin to. On success, writes the congregationimport_congregation_status
// overlay row recording the approving operator as the verifier.
func (s *Service) ApproveCandidate(ctx context.Context, token, callerPersonID, id string, jurisdictionUnitID *string) (domain.Candidate, error) {
	if err := s.requireOperator(ctx, token, callerPersonID); err != nil {
		return domain.Candidate{}, err
	}
	cand, err := s.store.GetCandidate(ctx, id)
	if err != nil {
		return domain.Candidate{}, err
	}
	if !isApprovable(cand.Status) {
		return domain.Candidate{}, domain.ErrNotApprovable
	}
	if cand.TaxonID == nil {
		return domain.Candidate{}, fmt.Errorf("%w: no taxon resolved — edit the candidate first", domain.ErrNotApprovable)
	}
	if cand.Latitude == nil || cand.Longitude == nil || cand.CountryID == nil {
		return domain.Candidate{}, fmt.Errorf("%w: missing coordinates or country — edit the candidate first", domain.ErrNotApprovable)
	}

	c, err := s.userClient(token)
	if err != nil {
		return domain.Candidate{}, err
	}

	unitID, err := s.ensureUnit(ctx, c, callerPersonID, cand, jurisdictionUnitID)
	if err != nil {
		return domain.Candidate{}, err
	}
	if err := s.ensureSite(ctx, c, unitID, cand); err != nil {
		return domain.Candidate{}, err
	}

	if _, err := s.store.CreateCongregationStatus(ctx, unitID, cand.SourceCode, &cand.ID, callerPersonID); err != nil {
		return domain.Candidate{}, fmt.Errorf("congregationimport: write congregation_status: %w", err)
	}
	provisioned, err := s.store.MarkProvisioned(ctx, cand.ID)
	if err != nil {
		return domain.Candidate{}, err
	}
	metrics.FromContext(ctx).Counter("openfaithmap.congregationimport.candidates_provisioned").Inc(1)
	return provisioned, nil
}

// ensureUnit reuses cand's persisted CreatedUnitID on a resumed PROVISIONING candidate rather than
// calling createChildOrg a second time — registration.ensureUnit's exact pattern.
func (s *Service) ensureUnit(ctx context.Context, c *oikumenea.Client, callerPersonID string, cand domain.Candidate, jurisdictionUnitID *string) (string, error) {
	if cand.Status == domain.StatusProvisioning && cand.CreatedUnitID != nil {
		return *cand.CreatedUnitID, nil
	}

	parentUnitID := s.cfg.RootUnitID
	if jurisdictionUnitID != nil && *jurisdictionUnitID != "" {
		parentUnitID = *jurisdictionUnitID
	}
	profile, err := c.Religion.CreateChildOrg(ctx, parentUnitID, religion.CreateChildOrgRequest{
		Code:           slugCode(cand.Name),
		Name:           cand.Name,
		PrimaryTaxonId: cand.TaxonID,
	})
	if err != nil {
		return "", fmt.Errorf("createChildOrg: %w", err)
	}
	if _, err := s.store.MarkProvisioning(ctx, cand.ID, callerPersonID, profile.UnitId); err != nil {
		return "", fmt.Errorf("markProvisioning: %w", err)
	}
	return profile.UnitId, nil
}

// ensureSite mirrors registration.ensureSite exactly: check-then-create, since createSite has no
// natural duplicate-conflict to rely on for a resumed retry.
func (s *Service) ensureSite(ctx context.Context, c *oikumenea.Client, unitID string, cand domain.Candidate) error {
	sites, err := c.Religion.ListUnitSites(ctx, unitID)
	if err != nil {
		return fmt.Errorf("listUnitSites: %w", err)
	}
	for _, site := range sites.Sites {
		if site.IsPrimary {
			return nil
		}
	}

	siteTypeID, err := churchSiteTypeID(ctx, c)
	if err != nil {
		return err
	}
	loc, err := c.Location.CreateLocation(ctx, location.LocationWrite{
		Coordinate: &location.CoordinateInput{
			Format:    "latlon",
			Latitude:  cand.Latitude,
			Longitude: cand.Longitude,
		},
		CountryId:   *cand.CountryID,
		AdminArea1:  cand.AdminArea1,
		Locality:    cand.Locality,
		Street:      cand.Street,
		HouseNumber: cand.HouseNumber,
		PostalCode:  cand.PostalCode,
	})
	if err != nil {
		return fmt.Errorf("createLocation: %w", err)
	}

	isPrimary := true
	if _, err := c.Religion.CreateSite(ctx, unitID, religion.CreateSiteRequest{
		LocationId: loc.Id,
		SiteTypeId: siteTypeID,
		IsPrimary:  &isPrimary,
	}); err != nil {
		return fmt.Errorf("createSite: %w", err)
	}
	return nil
}

// churchSiteTypeID and slugCode are deliberate, minimal duplicates of registration's own
// identically-named unexported helpers — this repo's established convention (unexported symbols
// aren't importable across packages, and every module already hand-duplicates small shared shapes
// rather than promoting them to a common package).
func churchSiteTypeID(ctx context.Context, c *oikumenea.Client) (string, error) {
	types, err := c.Religion.ListSiteTypes(ctx)
	if err != nil {
		return "", fmt.Errorf("listSiteTypes: %w", err)
	}
	for _, t := range types.SiteTypes {
		if t.Code == "church" {
			return t.Id, nil
		}
	}
	if len(types.SiteTypes) > 0 {
		return types.SiteTypes[0].Id, nil
	}
	return "", fmt.Errorf("no religion site types configured on this instance")
}

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugCode(name string) string {
	slug := strings.Trim(slugNonAlnum.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if slug == "" {
		slug = "congregation"
	}
	if len(slug) > 40 {
		slug = slug[:40]
	}
	const suffixChars = "abcdefghijklmnopqrstuvwxyz0123456789"
	suffix := make([]byte, 4)
	for i := range suffix {
		suffix[i] = suffixChars[rand.Intn(len(suffixChars))]
	}
	return slug + "-" + string(suffix)
}
