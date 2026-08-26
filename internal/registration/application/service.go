// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application holds the registration module's business logic: the D-Exclusions taxon
// check, the operator gate, and the real writes an approval performs.
//
// M10.6 cutover: every go-oikumenea SDK call is replaced by a direct, in-process call into
// internal/{religion,location,membership,directory,authz} — the caller's forwarded token and the
// go-oikumenea round-trip both disappear. Two real behaviour changes, both deliberate (see
// docs/architecture/decisions.md's D-InProcessAuthz amendment #3 and D-OwnCore's own text):
//  1. The `Authorize` meta-check disappears — go-oikumenea required the caller to already hold
//     assignment.read reaching the target unit before it would even evaluate the requested
//     permission; internal/authz.Require has no such meta-check, it is a pure function of the
//     subject (already in context, never a parameter).
//  2. IsOperator no longer needs a token at all — authz.Require reads the subject from ctx.
package application

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/olehmushka/open-faith-map/internal/authz"
	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	directoryapplication "github.com/olehmushka/open-faith-map/internal/directory/application"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
	locationapplication "github.com/olehmushka/open-faith-map/internal/location/application"
	locationdomain "github.com/olehmushka/open-faith-map/internal/location/domain"
	membershipapplication "github.com/olehmushka/open-faith-map/internal/membership/application"
	membershipdomain "github.com/olehmushka/open-faith-map/internal/membership/domain"
	"github.com/olehmushka/open-faith-map/internal/registration/adapters"
	"github.com/olehmushka/open-faith-map/internal/registration/domain"
	religionadapters "github.com/olehmushka/open-faith-map/internal/religion/adapters"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
)

// operatorPermission is what IsOperator asks authz.Require to decide, scoped to Config.RootUnitID
// specifically: does the caller's grant reach the root unit? Registration-operator's seeded grant
// targets the root unit directly; congregation-admin's own grant targets their own congregation
// unit, which is a DESCENDANT of root, never an ancestor-or-self of it — so the PDP's own
// ancestor-or-self direction check (never the reverse) is what keeps a congregation admin from
// reading as an operator, the same shape of defect D-PlatformModerator fixed once already
// (M2.3/M3's untargeted-permission bug class).
const operatorPermission = authzdomain.PermReligionOrgManage

type Config struct {
	// RootUnitID is the single shared root unit every congregation is registered as a child of
	// (internal/platform/seed.Resolve's RootUnitID — a fixed structural RID since D-SeedBootstrap, not an env
	// var).
	RootUnitID string
	// CongregationAdminRoleID is the role granted to a submitter on their own new unit at approval
	// time (internal/platform/seed.Resolve's CongregationAdminRoleID).
	CongregationAdminRoleID string
}

type Service struct {
	store      *adapters.Repository
	religion   *religionapplication.Service
	location   *locationapplication.Service
	membership *membershipapplication.Service
	directory  *directoryapplication.Service
	authzSvc   *authz.Service
	cfg        Config
}

func NewService(
	store *adapters.Repository,
	religionSvc *religionapplication.Service,
	locationSvc *locationapplication.Service,
	membershipSvc *membershipapplication.Service,
	directorySvc *directoryapplication.Service,
	authzSvc *authz.Service,
	cfg Config,
) *Service {
	return &Service{
		store: store, religion: religionSvc, location: locationSvc,
		membership: membershipSvc, directory: directorySvc, authzSvc: authzSvc, cfg: cfg,
	}
}

// Submit runs the D-Exclusions taxon check (walking ancestors via internal/religion) before
// persisting the request.
func (s *Service) Submit(ctx context.Context, submittedByPersonID string, in domain.SubmitInput) (domain.Request, error) {
	if err := s.checkNotExcluded(ctx, in.TaxonID); err != nil {
		return domain.Request{}, err
	}
	in.SubmittedByPersonID = submittedByPersonID
	return s.store.Insert(ctx, in)
}

// checkNotExcluded walks taxonID's ancestor chain (via Taxon.ParentID) checking each against
// D-Exclusions' named list (domain.ExcludedTaxonCodes) — the taxonomy is shallow (religion → branch
// → tradition → sub_tradition → denomination), so this is at most a handful of calls.
func (s *Service) checkNotExcluded(ctx context.Context, taxonID string) error {
	id := taxonID
	for i := 0; i < 10; i++ { // hard cap: never loop forever on an unexpected cycle
		taxon, err := s.religion.GetTaxon(ctx, id)
		if err != nil {
			return fmt.Errorf("%w: %s: %v", domain.ErrTaxonNotFound, taxonID, err)
		}
		if domain.ExcludedTaxonCodes[taxon.Code] {
			return domain.ErrExcluded
		}
		if taxon.ParentID == nil {
			return nil
		}
		id = *taxon.ParentID
	}
	return fmt.Errorf("%w: ancestor chain too deep for %s", domain.ErrTaxonNotFound, taxonID)
}

// IsOperator asks internal/authz's PDP whether the request's subject (from ctx) holds
// operatorPermission specifically on Config.RootUnitID. A denial is reported as (false, nil), not a
// failure to propagate — matching the pre-cutover behaviour of treating "not an operator" and
// "explicitly denied" as the same outcome for this read-time gate.
func (s *Service) IsOperator(ctx context.Context) (bool, error) {
	err := s.requireOperator(ctx)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, authzdomain.ErrPermissionDenied) {
		return false, nil
	}
	return false, err
}

// requireOperator is IsOperator's write-path form: it propagates a denial as an error instead of
// swallowing it to a bool. Pre-cutover, Approve/Reject/Reparent had no explicit gate of their own —
// each real go-oikumenea write (createChildOrg, grantAssignment, ...) was independently authorized
// by go-oikumenea's own PDP against the caller's forwarded token, so the check was implicit and
// happened per-write. internal/religion/internal/location/internal/membership/internal/directory
// carry no authorization logic of their own (D-InProcessAuthz: that's internal/authz's job,
// exclusively) — without this explicit call, any authenticated caller could approve/reject/re-parent
// any request, a real regression this cutover must not introduce. One check at the top of each write
// path, scoped to root (the same reach IsOperator already tests), stands in for what used to be
// several independent per-write checks against a possibly-deeper jurisdiction unit target — a
// deliberate simplification: a registration-operator's real-world grant is subtree-scoped on root
// specifically so it reaches every jurisdiction underneath, so reach-to-root is the correct proxy for
// "may approve into any jurisdiction," not an approximation of a narrower thing.
func (s *Service) requireOperator(ctx context.Context) error {
	return s.authzSvc.Require(ctx, operatorPermission, s.cfg.RootUnitID)
}

// List returns every request for an operator, or just the caller's own otherwise.
func (s *Service) List(ctx context.Context, callerPersonID string, status *domain.Status) ([]domain.Request, error) {
	isOperator, err := s.IsOperator(ctx)
	if err != nil {
		return nil, err
	}
	const pageSize = 200
	if isOperator {
		return s.store.List(ctx, status, pageSize)
	}
	return s.store.ListBySubmitter(ctx, callerPersonID, pageSize)
}

// Get returns id iff the caller is its submitter or an operator — otherwise domain.ErrNotFound,
// never a distinct "forbidden": this endpoint must not confirm the existence of a request the
// caller isn't permitted to see (M2.3 item 2 acceptance criterion).
func (s *Service) Get(ctx context.Context, callerPersonID, id string) (domain.Request, error) {
	req, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Request{}, err
	}
	if req.SubmittedByPersonID == callerPersonID {
		return req, nil
	}
	isOperator, err := s.IsOperator(ctx)
	if err != nil {
		return domain.Request{}, err
	}
	if !isOperator {
		return domain.Request{}, domain.ErrNotFound
	}
	return req, nil
}

// Approve performs the real writes: a child org under the configured root unit (or the operator's
// chosen jurisdiction unit) with the request's taxon as its primary classification, a location +
// site over it, a filled Position, and a unit-scoped grant of CongregationAdminRoleID to the
// submitter. internal/authz's PDP decides for real via each of these modules' own callers; unlike
// pre-cutover there is no meta-permission to fail on independently of the actual action.
//
// Resumable (M2.3 item 3, preserved verbatim): a request already in PROVISIONING (a prior attempt
// died after createChildOrg, the one step that can't be re-derived) resumes from its persisted
// created_unit_id instead of creating a second org. ensureSite checks for an existing primary site
// first; ensurePosition/ensureFilled/ensureGrant treat this module's own conflict errors for a
// repeat call as success.
func (s *Service) Approve(ctx context.Context, decidedByPersonID, id string, unitCodeOverride, jurisdictionUnitID *string) (domain.Request, error) {
	req, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Request{}, err
	}
	if req.Status != domain.StatusPending && req.Status != domain.StatusProvisioning {
		return domain.Request{}, domain.ErrNotPending
	}
	if err := s.requireOperator(ctx); err != nil {
		return domain.Request{}, err
	}

	unitID, err := s.ensureUnit(ctx, decidedByPersonID, req, unitCodeOverride, jurisdictionUnitID)
	if err != nil {
		return domain.Request{}, err
	}
	if err := s.ensureSite(ctx, unitID, req); err != nil {
		return domain.Request{}, err
	}
	position, err := s.ensurePosition(ctx, unitID)
	if err != nil {
		return domain.Request{}, err
	}
	if err := s.ensureFilled(ctx, position, req.SubmittedByPersonID); err != nil {
		return domain.Request{}, err
	}
	if err := s.ensureGrant(ctx, req.SubmittedByPersonID, unitID, decidedByPersonID); err != nil {
		return domain.Request{}, err
	}

	return s.store.Approve(ctx, id, decidedByPersonID, unitID)
}

// ensureUnit returns req's unit, reusing the persisted created_unit_id on a resumed PROVISIONING
// request rather than calling CreateChildOrg a second time. jurisdictionUnitID is the operator's
// parent choice (M4.1, D-JurisdictionUnits) — nil falls back to the configured root unit.
func (s *Service) ensureUnit(ctx context.Context, decidedByPersonID string, req domain.Request, unitCodeOverride, jurisdictionUnitID *string) (string, error) {
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
	profile, err := s.religion.CreateChildOrg(ctx, parentUnitID, unitCode, req.CongregationName, nil, &req.TaxonID)
	if err != nil {
		return "", fmt.Errorf("createChildOrg: %w", err)
	}
	if _, err := s.store.MarkProvisioning(ctx, req.ID, decidedByPersonID, profile.UnitID, jurisdictionUnitID); err != nil {
		return "", fmt.Errorf("markProvisioning: %w", err)
	}
	return profile.UnitID, nil
}

// ensureSite makes sure unitID has a primary site, creating one from req's location fields only if
// none exists yet. CreateSite has no natural duplicate-conflict (unlike the position/fill/grant
// steps below), so a resumed retry must check first rather than rely on an error to catch.
func (s *Service) ensureSite(ctx context.Context, unitID string, req domain.Request) error {
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
		Latitude:    req.Coordinate.Latitude,
		Longitude:   req.Coordinate.Longitude,
		CountryID:   req.CountryID,
		AdminArea1:  ptrOrEmpty(req.AdminArea1),
		Locality:    ptrOrEmpty(req.Locality),
		Street:      ptrOrEmpty(req.Street),
		HouseNumber: ptrOrEmpty(req.HouseNumber),
		PostalCode:  ptrOrEmpty(req.PostalCode),
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

// churchSiteTypeID finds the seeded "church" religion_site_types row (D-Scope: Christian only).
// Falls back to the first available site type if the seed ever changes shape, so approval doesn't
// hard-fail on a catalog rename — the same silent-fallback shape the pre-cutover code had (a real,
// separately-tracked defect, U11 in docs/milestones.md's unresolved-unknowns table; not this
// milestone's job to fix).
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

const congregationAdminPositionCode = "admin"

// ensurePosition returns unitID's "admin" position, creating it on a first attempt and reusing the
// existing one on a resumed retry — CreatePosition rejects a repeat code with
// domain.ErrPositionConflict, which is exactly the signal a resume needs.
func (s *Service) ensurePosition(ctx context.Context, unitID string) (membershipdomain.Position, error) {
	position, err := s.membership.CreatePosition(ctx, unitID, congregationAdminPositionCode, "Congregation Admin")
	if err == nil {
		return position, nil
	}
	if !errors.Is(err, membershipdomain.ErrPositionConflict) {
		return membershipdomain.Position{}, fmt.Errorf("createPosition: %w", err)
	}

	positions, err := s.membership.ListPositionsByUnit(ctx, unitID)
	if err != nil {
		return membershipdomain.Position{}, fmt.Errorf("listPositions after conflict: %w", err)
	}
	for _, p := range positions {
		if p.Code == congregationAdminPositionCode {
			return p, nil
		}
	}
	return membershipdomain.Position{}, fmt.Errorf("createPosition: conflict reported but %q not found in unit %s", congregationAdminPositionCode, unitID)
}

// ensureFilled fills position with personID, treating an already-filled position (a resumed retry —
// this position is created fresh per request, so no other actor can have filled it) as success.
func (s *Service) ensureFilled(ctx context.Context, position membershipdomain.Position, personID string) error {
	if _, err := s.membership.FillPosition(ctx, personID, position.UnitID, position.ID); err != nil {
		if errors.Is(err, membershipdomain.ErrPositionAlreadyFilled) {
			return nil
		}
		return fmt.Errorf("fillPosition: %w", err)
	}
	return nil
}

// ensureGrant grants CongregationAdminRoleID to personID on unitID, idempotent on a resumed retry
// (internal/authz.Service.GrantUnitRole's own unique-index-conflict-as-success handling).
func (s *Service) ensureGrant(ctx context.Context, personID, unitID, grantedByPersonID string) error {
	if _, err := s.authzSvc.GrantUnitRole(ctx, personID, s.cfg.CongregationAdminRoleID, unitID, authzdomain.ScopeUnit, "", grantedByPersonID, nil); err != nil {
		return fmt.Errorf("grantUnitRole: %w", err)
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
	if err := s.requireOperator(ctx); err != nil {
		return domain.Request{}, err
	}
	return s.store.Reject(ctx, id, decidedByPersonID, reason)
}

// GetReparentStatus returns the most recent re-parenting job for id, or nil if none has ever been
// started (M12.2: a thin view over internal/directory.GetMoveStatus — see Reparent's own doc for
// why registration no longer owns this job's storage).
func (s *Service) GetReparentStatus(ctx context.Context, id string) (*domain.ReparentingJob, error) {
	req, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.CreatedUnitID == nil {
		return nil, nil
	}
	mj, err := s.directory.GetMoveStatus(ctx, directorydomain.CanonicalGraphCode, *req.CreatedUnitID)
	if err != nil {
		return nil, err
	}
	if mj == nil {
		return nil, nil
	}
	job := moveJobToReparentingJob(req, *mj)
	return &job, nil
}

// Reparent starts or resumes moving an APPROVED request's congregation unit onto newParentUnitID
// (M4.1, D-JurisdictionUnits). M12.2: the actual add-before-remove, resumable, closure-safe move —
// formerly a state machine private to this service, backed by its own jurisdiction_reparenting_jobs
// table — is now internal/directory.Move, generalized so internal/registration is a caller rather
// than the sole owner (docs/milestones.md's M12.2 row); this method is now just the
// approval/operator-gate wrapper around it. jurisdiction_reparenting_jobs itself is left in place,
// untouched, as a frozen historical log — this service no longer writes to it.
func (s *Service) Reparent(ctx context.Context, performedByPersonID, id, newParentUnitID string) (domain.ReparentingJob, error) {
	req, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.ReparentingJob{}, err
	}
	if req.Status != domain.StatusApproved || req.CreatedUnitID == nil {
		return domain.ReparentingJob{}, domain.ErrNotApproved
	}
	if err := s.requireOperator(ctx); err != nil {
		return domain.ReparentingJob{}, err
	}
	mj, err := s.directory.Move(ctx, directorydomain.CanonicalGraphCode, *req.CreatedUnitID, newParentUnitID, performedByPersonID)
	if err != nil {
		return domain.ReparentingJob{}, err
	}
	return moveJobToReparentingJob(req, mj), nil
}

// moveJobToReparentingJob adapts a directorydomain.MoveJob onto this module's own ReparentingJob wire
// shape (M12.2), so registration's public API contract is unchanged for its own callers even though
// the underlying job storage moved to internal/directory.
func moveJobToReparentingJob(req domain.Request, mj directorydomain.MoveJob) domain.ReparentingJob {
	return domain.ReparentingJob{
		ID:                    mj.ID,
		RegistrationRequestID: req.ID,
		CongregationUnitID:    mj.UnitID,
		OldParentUnitID:       mj.OldParentUnitID,
		NewParentUnitID:       mj.NewParentUnitID,
		Status:                domain.ReparentStatus(mj.Status),
		PerformedByPersonID:   mj.PerformedByPersonID,
		Error:                 mj.Error,
		CreatedAt:             mj.CreatedAt,
		UpdatedAt:             mj.UpdatedAt,
	}
}

func ptrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
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
