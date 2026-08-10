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
	"github.com/olehmushka/go-oikumenea/clients/go/oikumenea/tenant"
	"github.com/olehmushka/open-faith-map/internal/coreintegration"
	"github.com/olehmushka/open-faith-map/internal/registration/adapters"
	"github.com/olehmushka/open-faith-map/internal/registration/domain"
)

// operatorPermission is the go-oikumenea permission IsOperator asks Authorize to decide, scoped to
// Config.RootUnitID: does the caller hold operatorPermission on the shared root unit specifically?
// For this module's own reads (List/Get) this check IS the entire access-control decision — there is
// no go-oikumenea PDP behind an OpenFaithMap-owned table to catch a wrong local answer here
// (D-PlatformModerator) — so it must never be treated as cosmetic for reads. It remains cosmetic only
// for the write paths (Approve/Reject): go-oikumenea's PDP re-decides createChildOrg/site.manage/
// grantAssignment for real against the caller's actual assignments, independent of what this said.
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

// IsOperator asks go-oikumenea's real PDP (Authorize) whether the caller holds operatorPermission
// specifically on Config.RootUnitID — not the flat, untargeted MyCapabilities() this used to call,
// which answers "does the caller hold this permission *anywhere*" and so also matched
// congregation-admin, which holds religionorg.manage on its own unit (D-PlatformModerator's fix for
// exactly this shape of bug).
//
// Authorize itself requires the caller to already hold assignment.read reaching the target unit, with
// no self-exemption (go-oikumenea's own OQ-5/D-SelfCapabilities framing, deliberate).
// scripts/bootstrap-registration-org grants registration-operator that reach; congregation-admin does
// not. So a real operator gets a real Allow/Deny, and anyone lacking assignment.read on RootUnitID —
// including every congregation-admin — gets the typed Authorization:PermissionDenied error, which
// this treats as "not an operator" (false, nil), not a failure to propagate.
func (s *Service) IsOperator(ctx context.Context, token, callerPersonID string) (bool, error) {
	c, err := s.client(token)
	if err != nil {
		return false, err
	}
	rootUnitID := s.cfg.RootUnitID
	resp, err := c.Authorization.Authorize(ctx, authorization.AuthorizeRequest{
		SubjectPersonId: callerPersonID,
		Action:          operatorPermission,
		UnitId:          &rootUnitID,
	})
	if err != nil {
		if authorization.IsPermissionDenied(err) {
			return false, nil
		}
		return false, err
	}
	return resp.Allow, nil
}

// List returns every request for an operator, or just the caller's own otherwise.
func (s *Service) List(ctx context.Context, token, callerPersonID string, status *domain.Status) ([]domain.Request, error) {
	isOperator, err := s.IsOperator(ctx, token, callerPersonID)
	if err != nil {
		return nil, err
	}
	const pageSize = 200
	if isOperator {
		return s.store.List(ctx, status, pageSize)
	}
	return s.store.ListBySubmitter(ctx, callerPersonID, pageSize)
}

// Get returns id iff the caller is its submitter or an operator (the same root-unit-scoped Authorize
// check List uses) — otherwise domain.ErrNotFound, never a distinct "forbidden": this endpoint must
// not confirm the existence of a request the caller isn't permitted to see (M2.3 item 2 acceptance
// criterion). transport.mapErr already maps domain.ErrNotFound to Registration:RequestNotFound.
func (s *Service) Get(ctx context.Context, token, callerPersonID, id string) (domain.Request, error) {
	req, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Request{}, err
	}
	if req.SubmittedByPersonID == callerPersonID {
		return req, nil
	}
	isOperator, err := s.IsOperator(ctx, token, callerPersonID)
	if err != nil {
		return domain.Request{}, err
	}
	if !isOperator {
		return domain.Request{}, domain.ErrNotFound
	}
	return req, nil
}

// Approve performs the real go-oikumenea writes with the caller's own forwarded token — a child org
// under the configured root unit (with the request's taxon as its primary classification), a
// location + site over it, a filled Position, and a unit-scoped grant of CongregationAdminRoleID to
// the submitter. go-oikumenea's PDP decides for real: if the caller lacks religionorg.manage /
// assignment.grant on RootUnitID, these calls fail and Approve returns that failure unchanged.
//
// Resumable (M2.3 item 3): a request already in PROVISIONING (a prior attempt died after
// createChildOrg, the one step that can't be re-derived) resumes from its persisted
// created_unit_id instead of creating a second org. The remaining steps are re-runnable: ensureSite
// checks for an existing primary site first (createSite has no duplicate-conflict to catch), and
// ensurePosition/ensureFilled/ensureGrant treat go-oikumenea's own conflict errors for a repeat call
// as success.
func (s *Service) Approve(ctx context.Context, token, decidedByPersonID, id string, unitCodeOverride, jurisdictionUnitID *string) (domain.Request, error) {
	req, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Request{}, err
	}
	if req.Status != domain.StatusPending && req.Status != domain.StatusProvisioning {
		return domain.Request{}, domain.ErrNotPending
	}

	c, err := s.client(token)
	if err != nil {
		return domain.Request{}, err
	}

	unitID, err := s.ensureUnit(ctx, c, decidedByPersonID, req, unitCodeOverride, jurisdictionUnitID)
	if err != nil {
		return domain.Request{}, err
	}
	if err := s.ensureSite(ctx, c, unitID, req); err != nil {
		return domain.Request{}, err
	}
	position, err := s.ensurePosition(ctx, c, unitID)
	if err != nil {
		return domain.Request{}, err
	}
	if err := s.ensureFilled(ctx, c, position, req.SubmittedByPersonID); err != nil {
		return domain.Request{}, err
	}
	if err := s.ensureGrant(ctx, c, req.SubmittedByPersonID, unitID); err != nil {
		return domain.Request{}, err
	}

	return s.store.Approve(ctx, id, decidedByPersonID, unitID)
}

// ensureUnit returns req's go-oikumenea unit, reusing the persisted created_unit_id on a resumed
// PROVISIONING request rather than calling createChildOrg a second time. jurisdictionUnitID is the
// operator's parent choice (M4.1, D-JurisdictionUnits) — nil falls back to the configured root unit,
// the original flat-root behavior. On a resumed retry this parameter is ignored entirely: the
// ORIGINAL choice, persisted by MarkProvisioning on the first pass, is what already produced
// req.CreatedUnitID, so there is nothing left to decide.
func (s *Service) ensureUnit(ctx context.Context, c *oikumenea.Client, decidedByPersonID string, req domain.Request, unitCodeOverride, jurisdictionUnitID *string) (string, error) {
	if req.Status == domain.StatusProvisioning && req.CreatedUnitID != nil {
		return *req.CreatedUnitID, nil
	}

	unitCode := slugCode(req.CongregationName)
	if unitCodeOverride != nil && *unitCodeOverride != "" {
		unitCode = *unitCodeOverride
	}
	parentUnitID := s.cfg.RootUnitID
	if jurisdictionUnitID != nil && *jurisdictionUnitID != "" {
		parentUnitID = *jurisdictionUnitID
	}
	profile, err := c.Religion.CreateChildOrg(ctx, parentUnitID, religion.CreateChildOrgRequest{
		Code:           unitCode,
		Name:           req.CongregationName,
		PrimaryTaxonId: &req.TaxonID,
	})
	if err != nil {
		return "", fmt.Errorf("createChildOrg: %w", err)
	}
	if _, err := s.store.MarkProvisioning(ctx, req.ID, decidedByPersonID, profile.UnitId, jurisdictionUnitID); err != nil {
		return "", fmt.Errorf("markProvisioning: %w", err)
	}
	return profile.UnitId, nil
}

// ensureSite makes sure unitID has a primary site, creating one from req's location fields only if
// none exists yet. createSite has no natural duplicate-conflict (unlike the position/fill/grant
// steps below), so a resumed retry must check first rather than rely on an error to catch.
func (s *Service) ensureSite(ctx context.Context, c *oikumenea.Client, unitID string, req domain.Request) error {
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

const congregationAdminPositionCode = "admin"

// ensurePosition returns unitID's "admin" position, creating it on a first attempt and reusing the
// existing one on a resumed retry — createPosition rejects a repeat code with
// Position:PositionConflict, which is exactly the signal a resume needs.
func (s *Service) ensurePosition(ctx context.Context, c *oikumenea.Client, unitID string) (membership.Position, error) {
	position, err := c.Membership.CreatePosition(ctx, unitID, membership.CreatePositionRequest{
		Code:  congregationAdminPositionCode,
		Title: "Congregation Admin",
	})
	if err == nil {
		return position, nil
	}
	if !membership.IsPositionConflict(err) {
		return membership.Position{}, fmt.Errorf("createPosition: %w", err)
	}

	positions, err := c.Membership.ListPositions(ctx, unitID, nil, nil, nil)
	if err != nil {
		return membership.Position{}, fmt.Errorf("listPositions after PositionConflict: %w", err)
	}
	for _, p := range positions.Positions {
		if p.Code == congregationAdminPositionCode {
			return p, nil
		}
	}
	return membership.Position{}, fmt.Errorf("createPosition: PositionConflict reported but %q not found in unit %s", congregationAdminPositionCode, unitID)
}

// ensureFilled fills position with personID, treating an already-filled position (a resumed retry —
// this position is created fresh per request, so no other actor can have filled it) as success.
func (s *Service) ensureFilled(ctx context.Context, c *oikumenea.Client, position membership.Position, personID string) error {
	if _, err := c.Membership.FillPosition(ctx, position.Id, membership.FillPositionRequest{PersonId: personID}); err != nil {
		if membership.IsPositionAlreadyFilled(err) {
			return nil
		}
		return fmt.Errorf("fillPosition: %w", err)
	}
	return nil
}

// ensureGrant grants CongregationAdminRoleID to personID on unitID, treating an identical existing
// grant (a resumed retry) as success.
func (s *Service) ensureGrant(ctx context.Context, c *oikumenea.Client, personID, unitID string) error {
	if _, err := c.Authorization.GrantAssignment(ctx, authorization.GrantAssignmentRequest{
		SubjectPersonId: personID,
		RoleId:          s.cfg.CongregationAdminRoleID,
		TargetUnitId:    unitID,
		Scope:           "unit", // never subtree — a congregation admin never reaches another congregation
	}); err != nil {
		if authorization.IsAssignmentConflict(err) {
			return nil
		}
		return fmt.Errorf("grantAssignment: %w", err)
	}
	return nil
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

// GetReparentStatus returns the most recent re-parenting job for id, or nil if none has ever been
// started.
func (s *Service) GetReparentStatus(ctx context.Context, id string) (*domain.ReparentingJob, error) {
	if _, err := s.store.Get(ctx, id); err != nil {
		return nil, err
	}
	return s.store.GetLatestReparentJob(ctx, id)
}

// Reparent starts or resumes moving an APPROVED request's congregation unit onto newParentUnitID
// (M4.1, D-JurisdictionUnits) using the caller's own forwarded token — same D-Facade discipline as
// Approve. go-oikumenea has no single atomic "move" call for a religion Unit (verified live: only
// the generic tenant module's addEdge/removeEdge exist), so this is a resumable state machine,
// re-entrant on requestId: a repeat call with the request already mid-job resumes from whichever
// ReparentStatus step last durably landed, driven by runReparentSteps below.
//
// Add-before-remove by design (D-JurisdictionUnits): the congregation briefly has two canonical
// parents mid-migration rather than momentarily zero, so a subtree-scoped grant (registration-
// operator, platform-moderator) never loses reach to it during the move — live-verified against a
// real instance, not just reasoned about. Grant preservation itself (a unit-scoped grant on the
// congregation surviving the move unchanged) is a structural property of go-oikumenea's edge model,
// also live-verified once as a repo-wide fact rather than re-checked per job: edge mutations never
// touch authz_role_assignments at all.
func (s *Service) Reparent(ctx context.Context, token, performedByPersonID, id, newParentUnitID string) (domain.ReparentingJob, error) {
	req, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.ReparentingJob{}, err
	}
	if req.Status != domain.StatusApproved || req.CreatedUnitID == nil {
		return domain.ReparentingJob{}, domain.ErrNotApproved
	}
	congregationUnitID := *req.CreatedUnitID

	job, err := s.store.GetLiveReparentJob(ctx, congregationUnitID)
	if err != nil {
		return domain.ReparentingJob{}, fmt.Errorf("getLiveReparentJob: %w", err)
	}
	if job == nil {
		oldParentUnitID, err := s.currentParent(ctx, req)
		if err != nil {
			return domain.ReparentingJob{}, fmt.Errorf("resolve current parent: %w", err)
		}
		created, err := s.store.CreateReparentJob(ctx, req.ID, congregationUnitID, oldParentUnitID, newParentUnitID, performedByPersonID)
		if err != nil {
			return domain.ReparentingJob{}, fmt.Errorf("createReparentJob: %w", err)
		}
		job = &created
	} else if job.NewParentUnitID != newParentUnitID {
		return domain.ReparentingJob{}, fmt.Errorf("unit %s already has a live reparent job targeting %s — resolve it before starting a move to %s", congregationUnitID, job.NewParentUnitID, newParentUnitID)
	}

	c, err := s.client(token)
	if err != nil {
		return domain.ReparentingJob{}, err
	}
	return s.runReparentSteps(ctx, c, *job)
}

// currentParent resolves congregationUnitID's actual current parent: the most recent VERIFIED
// reparent job's target if one exists (this store IS the record of every successful move this
// service has ever performed), else the jurisdiction chosen at approval time, else the configured
// root unit.
func (s *Service) currentParent(ctx context.Context, req domain.Request) (string, error) {
	latest, err := s.store.GetLatestReparentJob(ctx, req.ID)
	if err != nil {
		return "", err
	}
	if latest != nil && latest.Status == domain.ReparentVerified {
		return latest.NewParentUnitID, nil
	}
	if req.JurisdictionUnitID != nil && *req.JurisdictionUnitID != "" {
		return *req.JurisdictionUnitID, nil
	}
	return s.cfg.RootUnitID, nil
}

// runReparentSteps drives job through whichever steps haven't durably landed yet. Each step
// persists its own completion before the next runs, so a crash between any two steps resumes
// exactly here on the next call rather than repeating or skipping work.
func (s *Service) runReparentSteps(ctx context.Context, c *oikumenea.Client, job domain.ReparentingJob) (domain.ReparentingJob, error) {
	canonical := "canonical"

	if job.Status == domain.ReparentPending {
		if _, err := c.Tenant.AddEdge(ctx, job.CongregationUnitID, tenant.AddEdgeRequest{
			ParentId: job.NewParentUnitID,
			Graph:    &canonical,
		}); err != nil && !isEdgeAlreadyExists(err) {
			return s.store.FailReparentJob(ctx, job.ID, fmt.Sprintf("addEdge(new parent): %v", err))
		}
		updated, err := s.store.AdvanceReparentJob(ctx, job.ID, domain.ReparentNewEdgeAdded)
		if err != nil {
			return domain.ReparentingJob{}, err
		}
		job = updated
	}

	if job.Status == domain.ReparentNewEdgeAdded {
		// RemoveEdge on an already-absent edge is a documented no-op in go-oikumenea (idempotent by
		// design), so a resumed retry re-calling this needs no special-case error handling.
		if err := c.Tenant.RemoveEdge(ctx, job.CongregationUnitID, job.OldParentUnitID, &canonical); err != nil {
			return s.store.FailReparentJob(ctx, job.ID, fmt.Sprintf("removeEdge(old parent): %v", err))
		}
		updated, err := s.store.AdvanceReparentJob(ctx, job.ID, domain.ReparentOldEdgeRemoved)
		if err != nil {
			return domain.ReparentingJob{}, err
		}
		job = updated
	}

	if job.Status == domain.ReparentOldEdgeRemoved {
		ancestors, err := c.Tenant.UnitAncestors(ctx, job.CongregationUnitID, &canonical)
		if err != nil {
			return s.store.FailReparentJob(ctx, job.ID, fmt.Sprintf("unitAncestors (verify): %v", err))
		}
		if !ancestorsInclude(ancestors, job.NewParentUnitID) {
			return s.store.FailReparentJob(ctx, job.ID, fmt.Sprintf("verify: %s not found in %s's ancestors after re-parenting", job.NewParentUnitID, job.CongregationUnitID))
		}
		updated, err := s.store.AdvanceReparentJob(ctx, job.ID, domain.ReparentVerified)
		if err != nil {
			return domain.ReparentingJob{}, err
		}
		job = updated
	}

	return job, nil
}

// isEdgeAlreadyExists reports whether err is AddEdge's specific "this exact (graph, parent, child)
// edge already exists" rejection — live-verified as Tenant:UnitInvalid with that reason text, not a
// dedicated conflict type — vs. some other UnitInvalid cause that should genuinely fail the job
// (e.g. a missing unit). A resumed retry hitting this specific case is success, not a failure.
func isEdgeAlreadyExists(err error) bool {
	return tenant.IsUnitInvalid(err) && strings.Contains(err.Error(), "already exists")
}

func ancestorsInclude(l tenant.UnitRefList, unitID string) bool {
	for _, u := range l.Units {
		if u.Id == unitID {
			return true
		}
	}
	return false
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
