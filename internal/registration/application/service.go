// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application holds the registration module's business logic: the D-Exclusions taxon
// check, the operator gate, and the real go-oikumenea writes an approval performs — always with
// the CALLER's own forwarded token (D-Facade: OpenFaithMap makes zero authorization decisions of
// its own over anything go-oikumenea owns; go-oikumenea's PDP decides every call for real).
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
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/membership"
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/religion"

	"github.com/olehmushka/open-faith-map/internal/coreintegration"
	"github.com/olehmushka/open-faith-map/internal/registration/adapters"
	"github.com/olehmushka/open-faith-map/internal/registration/domain"
)

// operatorPermission is the permission MyCapabilities is checked for to decide whether a caller
// sees every pending request (an operator) or only their own (a submitter). Cosmetic/UX gating
// only (D-SelfCapabilities) — the real enforcement is go-oikumenea's PDP re-deciding every write
// Approve/Reject actually makes.
const operatorPermission = "religionorg.manage"

type Config struct {
	OikumeneaBaseURL            string
	OikumeneaInsecureSkipVerify bool
	// RootUnitID is the single shared root unit every congregation is registered as a child of
	// (scripts/bootstrap-registration-org).
	RootUnitID string
	// CongregationAdminRoleID is the role granted to a submitter on their own new unit at approval
	// time (scripts/bootstrap-registration-org — created ahead of time because role.create is
	// instance-scope and a registration-operator does not hold it).
	CongregationAdminRoleID string
}

type Service struct {
	store *adapters.Store
	cfg   Config
}

func NewService(store *adapters.Store, cfg Config) *Service {
	return &Service{store: store, cfg: cfg}
}

func (s *Service) client(token string) (*oikumenea.Client, error) {
	return coreintegration.NewUserClient(s.cfg.OikumeneaBaseURL, token, s.cfg.OikumeneaInsecureSkipVerify)
}

// Submit runs the D-Exclusions taxon check (walking ancestors via the caller's own religion.read)
// before persisting the request.
func (s *Service) Submit(ctx context.Context, token, submittedByPersonID string, in domain.SubmitInput) (domain.Request, error) {
	c, err := s.client(token)
	if err != nil {
		return domain.Request{}, err
	}
	if err := checkNotExcluded(ctx, c, in.TaxonID); err != nil {
		return domain.Request{}, err
	}
	in.SubmittedByPersonID = submittedByPersonID
	return s.store.Insert(ctx, in)
}

// checkNotExcluded walks taxonID's ancestor chain (via Taxon.ParentId) checking each against
// D-Exclusions' named list (domain.ExcludedTaxonCodes) — the taxonomy is shallow (religion → branch
// → tradition → sub_tradition → denomination), so this is at most a handful of calls.
func checkNotExcluded(ctx context.Context, c *oikumenea.Client, taxonID string) error {
	id := taxonID
	for i := 0; i < 10; i++ { // hard cap: never loop forever on an unexpected cycle
		taxon, err := c.Religion.GetTaxon(ctx, id)
		if err != nil {
			return fmt.Errorf("%w: %s: %v", domain.ErrTaxonNotFound, taxonID, err)
		}
		if domain.ExcludedTaxonCodes[taxon.Code] {
			return domain.ErrExcluded
		}
		if taxon.ParentId == nil {
			return nil
		}
		id = *taxon.ParentId
	}
	return fmt.Errorf("%w: ancestor chain too deep for %s", domain.ErrTaxonNotFound, taxonID)
}

// IsOperator asks go-oikumenea what the caller's own permissions are (D-SelfCapabilities,
// authenticated-but-ungated) and checks for operatorPermission — a coarse, cosmetic-only gate; see
// the package doc.
func (s *Service) IsOperator(ctx context.Context, token string) (bool, error) {
	c, err := s.client(token)
	if err != nil {
		return false, err
	}
	caps, err := c.Authorization.MyCapabilities(ctx)
	if err != nil {
		return false, err
	}
	for _, p := range caps.Permissions {
		if p == operatorPermission {
			return true, nil
		}
	}
	return false, nil
}

// List returns every request for an operator, or just the caller's own otherwise.
func (s *Service) List(ctx context.Context, token, callerPersonID string, status *domain.Status) ([]domain.Request, error) {
	isOperator, err := s.IsOperator(ctx, token)
	if err != nil {
		return nil, err
	}
	const pageSize = 200
	if isOperator {
		return s.store.List(ctx, status, pageSize)
	}
	return s.store.ListBySubmitter(ctx, callerPersonID, pageSize)
}

func (s *Service) Get(ctx context.Context, id string) (domain.Request, error) {
	return s.store.Get(ctx, id)
}

// Approve performs the real go-oikumenea writes with the caller's own forwarded token — a child org
// under the configured root unit (with the request's taxon as its primary classification), a
// location + site over it, a filled Position, and a unit-scoped grant of CongregationAdminRoleID to
// the submitter. go-oikumenea's PDP decides for real: if the caller lacks religionorg.manage /
// assignment.grant on RootUnitID, these calls fail and Approve returns that failure unchanged.
func (s *Service) Approve(ctx context.Context, token, decidedByPersonID, id string, unitCodeOverride *string) (domain.Request, error) {
	req, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Request{}, err
	}
	if req.Status != domain.StatusPending {
		return domain.Request{}, domain.ErrNotPending
	}

	c, err := s.client(token)
	if err != nil {
		return domain.Request{}, err
	}

	unitCode := slugCode(req.CongregationName)
	if unitCodeOverride != nil && *unitCodeOverride != "" {
		unitCode = *unitCodeOverride
	}

	profile, err := c.Religion.CreateChildOrg(ctx, s.cfg.RootUnitID, religion.CreateChildOrgRequest{
		Code:           unitCode,
		Name:           req.CongregationName,
		PrimaryTaxonId: &req.TaxonID,
	})
	if err != nil {
		return domain.Request{}, fmt.Errorf("createChildOrg: %w", err)
	}
	unitID := profile.UnitId

	siteTypeID, err := churchSiteTypeID(ctx, c)
	if err != nil {
		return domain.Request{}, err
	}

	loc, err := c.Location.CreateLocation(ctx, location.LocationWrite{
		Coordinate: &location.CoordinateInput{
			Format:    "latlon",
			Latitude:  &req.Coordinate.Latitude,
			Longitude: &req.Coordinate.Longitude,
		},
		CountryId:   req.CountryID,
		AdminArea1:  req.AdminArea1,
		Locality:    req.Locality,
		Street:      req.Street,
		HouseNumber: req.HouseNumber,
		PostalCode:  req.PostalCode,
	})
	if err != nil {
		return domain.Request{}, fmt.Errorf("createLocation: %w", err)
	}

	isPrimary := true
	if _, err := c.Religion.CreateSite(ctx, unitID, religion.CreateSiteRequest{
		LocationId: loc.Id,
		SiteTypeId: siteTypeID,
		IsPrimary:  &isPrimary,
	}); err != nil {
		return domain.Request{}, fmt.Errorf("createSite: %w", err)
	}

	position, err := c.Membership.CreatePosition(ctx, unitID, membership.CreatePositionRequest{
		Code:  "admin",
		Title: "Congregation Admin",
	})
	if err != nil {
		return domain.Request{}, fmt.Errorf("createPosition: %w", err)
	}
	if _, err := c.Membership.FillPosition(ctx, position.Id, membership.FillPositionRequest{
		PersonId: req.SubmittedByPersonID,
	}); err != nil {
		return domain.Request{}, fmt.Errorf("fillPosition: %w", err)
	}

	if _, err := c.Authorization.GrantAssignment(ctx, authorization.GrantAssignmentRequest{
		SubjectPersonId: req.SubmittedByPersonID,
		RoleId:          s.cfg.CongregationAdminRoleID,
		TargetUnitId:    unitID,
		Scope:           "unit", // never subtree — a congregation admin never reaches another congregation
	}); err != nil {
		return domain.Request{}, fmt.Errorf("grantAssignment: %w", err)
	}

	return s.store.Approve(ctx, id, decidedByPersonID, unitID)
}

func (s *Service) Reject(ctx context.Context, decidedByPersonID, id, reason string) (domain.Request, error) {
	req, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Request{}, err
	}
	if req.Status != domain.StatusPending {
		return domain.Request{}, domain.ErrNotPending
	}
	return s.store.Reject(ctx, id, decidedByPersonID, reason)
}

// churchSiteTypeID finds go-oikumenea's seeded "church" religion_site_types row (D-Scope: Christian
// only). Falls back to the first available site type if the seed ever changes shape, so approval
// doesn't hard-fail on a catalog rename.
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

// slugCode derives a short, unique-enough unit code from a congregation name (e.g. "St. Mary's
// Chapel" -> "st-marys-chapel-a1b2") — a default when the operator doesn't override it.
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
