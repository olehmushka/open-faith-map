// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	locationdomain "github.com/olehmushka/open-faith-map/internal/location/domain"
	religionadapters "github.com/olehmushka/open-faith-map/internal/religion/adapters"
	"github.com/palantir/pkg/metrics"
)

// operatorPermission mirrors registration/content/discovery/moderation/vouching's own identical
// constant — this repo's deliberate convention is each module holding its own copy of the
// target-scoped capability check, not importing another module's application package.
const operatorPermission = authzdomain.PermReligionOrgManage

// requireOperator asks internal/authz's PDP whether the request's subject (from ctx) holds
// operatorPermission on Config.RootUnitID specifically — the same target-scoped pattern every other
// module's own IsOperator/requireOperator/requireManage already uses.
func (s *Service) requireOperator(ctx context.Context) error {
	if err := s.authzSvc.Require(ctx, operatorPermission, s.cfg.RootUnitID); err != nil {
		if errors.Is(err, authzdomain.ErrPermissionDenied) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}

func (s *Service) EditCandidate(ctx context.Context, id string, in domain.EditInput) (domain.Candidate, error) {
	if err := s.requireOperator(ctx); err != nil {
		return domain.Candidate{}, err
	}
	return s.store.Edit(ctx, id, in)
}

func (s *Service) RejectCandidate(ctx context.Context, callerPersonID, id, reason string) (domain.Candidate, error) {
	if err := s.requireOperator(ctx); err != nil {
		return domain.Candidate{}, err
	}
	return s.store.Reject(ctx, id, callerPersonID, reason)
}

// ---- alias management (production-hardening pass: previously SQL-only, see
// docs/modules/congregationimport.md) ----

func (s *Service) ListTaxonAliases(ctx context.Context, sourceCode *string) ([]domain.TaxonAlias, error) {
	if err := s.requireOperator(ctx); err != nil {
		return nil, err
	}
	if sourceCode == nil {
		return s.store.ListAllTaxonAliases(ctx)
	}
	return s.store.ListAliasesForMatching(ctx, *sourceCode)
}

func (s *Service) CreateTaxonAlias(ctx context.Context, callerPersonID string, sourceCode *string, aliasText, taxonID string) (domain.TaxonAlias, error) {
	if err := s.requireOperator(ctx); err != nil {
		return domain.TaxonAlias{}, err
	}
	return s.store.CreateTaxonAlias(ctx, sourceCode, normalizeAlias(aliasText), taxonID, callerPersonID)
}

func (s *Service) ListJurisdictionAliases(ctx context.Context, sourceCode *string) ([]domain.JurisdictionAlias, error) {
	if err := s.requireOperator(ctx); err != nil {
		return nil, err
	}
	if sourceCode == nil {
		return s.store.ListAllJurisdictionAliases(ctx)
	}
	return s.store.ListJurisdictionAliasesForMatching(ctx, *sourceCode)
}

func (s *Service) CreateJurisdictionAlias(ctx context.Context, callerPersonID string, sourceCode *string, aliasText, jurisdictionUnitID string) (domain.JurisdictionAlias, error) {
	if err := s.requireOperator(ctx); err != nil {
		return domain.JurisdictionAlias{}, err
	}
	return s.store.CreateJurisdictionAlias(ctx, sourceCode, normalizeAlias(aliasText), jurisdictionUnitID, callerPersonID)
}

// isApprovable is an allowlist, not a denylist — a real bug, caught live (not by review): the
// original denylist only excluded REJECTED/REJECTED_EXCLUDED, so calling ApproveCandidate a second
// time on an already-PROVISIONED candidate fell through to ensureUnit, whose own resume-check only
// short-circuits on PROVISIONING (not PROVISIONED), and called createChildOrg a second time — a
// real duplicate unit, confirmed created before this fix. Only PROVISIONING (a genuine crash-resume)
// and the pre-approval statuses may reach ensureUnit at all. Pure — no I/O — split out from
// ApproveCandidate so it's directly unit-testable (a regression test for the exact bug above)
// without a live store/religion client.
func isApprovable(status domain.Status) bool {
	switch status {
	case domain.StatusStaged, domain.StatusNeedsTaxonReview, domain.StatusNeedsGeocode,
		domain.StatusPossibleDuplicate, domain.StatusApproved, domain.StatusProvisioning:
		return true
	default:
		return false
	}
}

// ApproveCandidate performs the real writes, under the APPROVING OPERATOR'S OWN context-resolved
// subject — internal/religion/internal/location carry no authorization logic of their own
// (D-InProcessAuthz: that's internal/authz's exclusive job), so requireOperator above is what makes
// "is this really a legitimate congregation" the human judgment call D-CongregationImport requires,
// not an implicit one. Resumable, mirroring registration.Approve's exact ensureUnit/ensureSite shape
// (M2.3's crash-resume pattern) — minus the position/fill/grantAssignment steps: there is no
// submitter to grant congregation-admin to. On success, writes the
// congregationimport_congregation_status overlay row recording the approving operator as the
// verifier.
func (s *Service) ApproveCandidate(ctx context.Context, callerPersonID, id string, jurisdictionUnitID *string) (domain.Candidate, error) {
	if err := s.requireOperator(ctx); err != nil {
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

	unitID, err := s.ensureUnit(ctx, callerPersonID, cand, jurisdictionUnitID)
	if err != nil {
		return domain.Candidate{}, err
	}
	if err := s.ensureSite(ctx, unitID, cand); err != nil {
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
// calling CreateChildOrg a second time — registration.ensureUnit's exact pattern.
func (s *Service) ensureUnit(ctx context.Context, callerPersonID string, cand domain.Candidate, jurisdictionUnitID *string) (string, error) {
	if cand.Status == domain.StatusProvisioning && cand.CreatedUnitID != nil {
		return *cand.CreatedUnitID, nil
	}

	parentUnitID := s.cfg.RootUnitID
	if jurisdictionUnitID != nil && *jurisdictionUnitID != "" {
		parentUnitID = *jurisdictionUnitID
	}
	profile, err := s.religion.CreateChildOrg(ctx, parentUnitID, slugCode(cand.Name), cand.Name, nil, cand.TaxonID)
	if err != nil {
		return "", fmt.Errorf("createChildOrg: %w", err)
	}
	if _, err := s.store.MarkProvisioning(ctx, cand.ID, callerPersonID, profile.UnitID); err != nil {
		return "", fmt.Errorf("markProvisioning: %w", err)
	}
	return profile.UnitID, nil
}

// ensureSite mirrors registration.ensureSite exactly: check-then-create, since createSite has no
// natural duplicate-conflict to rely on for a resumed retry.
func (s *Service) ensureSite(ctx context.Context, unitID string, cand domain.Candidate) error {
	sites, err := s.religion.ListSitesByUnit(ctx, unitID)
	if err != nil {
		return fmt.Errorf("listSitesByUnit: %w", err)
	}
	for _, site := range sites {
		if site.IsPrimary {
			return nil
		}
	}

	siteTypeID, err := s.churchSiteTypeID(ctx)
	if err != nil {
		return err
	}
	loc, err := s.location.CreateLocation(ctx, locationdomain.LocationInput{
		Latitude:    *cand.Latitude,
		Longitude:   *cand.Longitude,
		CountryID:   *cand.CountryID,
		AdminArea1:  ptrOrEmpty(cand.AdminArea1),
		Locality:    ptrOrEmpty(cand.Locality),
		Street:      ptrOrEmpty(cand.Street),
		HouseNumber: ptrOrEmpty(cand.HouseNumber),
		PostalCode:  ptrOrEmpty(cand.PostalCode),
	})
	if err != nil {
		return fmt.Errorf("createLocation: %w", err)
	}

	if _, err := s.religion.CreateSite(ctx, religionadapters.CreateSiteInput{
		OrgUnitID: unitID, LocationID: loc.ID, SiteTypeID: siteTypeID, IsPrimary: true,
	}); err != nil {
		return fmt.Errorf("createSite: %w", err)
	}
	return nil
}

func ptrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// churchSiteTypeID and slugCode are deliberate, minimal duplicates of registration's own
// identically-named unexported helpers — this repo's established convention (unexported symbols
// aren't importable across packages, and every module already hand-duplicates small shared shapes
// rather than promoting them to a common package).
func (s *Service) churchSiteTypeID(ctx context.Context) (string, error) {
	types, err := s.religion.ListSiteTypes(ctx)
	if err != nil {
		return "", fmt.Errorf("listSiteTypes: %w", err)
	}
	for _, t := range types {
		if t.Code == "church" {
			return t.ID, nil
		}
	}
	if len(types) > 0 {
		return types[0].ID, nil
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
