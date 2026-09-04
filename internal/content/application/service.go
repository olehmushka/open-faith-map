// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application holds the content module's business logic: the content.manage target-scoped
// gate (authorize.go), block json-schema validation (blockvalidation.go), the parent-depth walk,
// the document transition state machine, and the public/admin read split.
//
// M10.6 cutover: write authority is decided by internal/authz.Require against the request's
// context-resolved subject (internal/identity's authenticator middleware), not a per-call
// go-oikumenea client built from the caller's forwarded token. No behaviour change beyond
// D-InProcessAuthz's documented removal of the assignment.read meta-check (see
// internal/registration/application/service.go's own cutover comment for the general reasoning).
package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/olehmushka/open-faith-map/internal/content/adapters"
	"github.com/olehmushka/open-faith-map/internal/content/domain"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
	religiondomain "github.com/olehmushka/open-faith-map/internal/religion/domain"
)

// Config carries settings resolved once at wiring time (cmd/openfaithmap-api) rather than derived
// per-request.
type Config struct {
	// RootUnitID is the same shared root unit registration/moderation/discovery already use
	// (internal/platform/seed.Resolve's RootUnitID) — M14.13's requireCatalogManage target, mirroring
	// internal/moderation/application.Config's own field exactly.
	RootUnitID string
}

type Service struct {
	store          *adapters.Repository
	authzSvc       *authz.Service
	religion       *religionapplication.Service
	previewHMACKey string
	cfg            Config
}

// religion is injected the same direct-interface-call shape internal/discovery already uses
// against internal/religion (docs/architecture/conventions.md) — M14.11's site-chrome footer is
// the first place content itself, not just discovery, reads religion's live data.
func NewService(store *adapters.Repository, authzSvc *authz.Service, religionSvc *religionapplication.Service, previewHMACKey string, cfg Config) *Service {
	return &Service{store: store, authzSvc: authzSvc, religion: religionSvc, previewHMACKey: previewHMACKey, cfg: cfg}
}

// ---- sites ----

// CreateSite's target is the request's own claimed congregation unit — no site row exists yet to
// load one from.
func (s *Service) CreateSite(ctx context.Context, in domain.CreateSiteInput) (domain.Site, error) {
	if err := s.requireManage(ctx, in.CongregationUnitRID); err != nil {
		return domain.Site{}, err
	}
	if isReservedSlug(in.Slug) {
		return domain.Site{}, &domain.SlugReservedError{Slug: in.Slug}
	}
	return s.store.InsertSite(ctx, in)
}

// UpdateSiteTheme validates theme against D-CuratedTheme's fixed vocabulary and write-time WCAG
// contrast gate (M14.12, validateTheme) before storing — the only write path in this module with
// no validation until this milestone.
func (s *Service) UpdateSiteTheme(ctx context.Context, siteID string, theme json.RawMessage) (domain.Site, error) {
	site, err := s.store.GetSiteByID(ctx, siteID)
	if err != nil {
		return domain.Site{}, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return domain.Site{}, err
	}
	if _, err := validateTheme(theme); err != nil {
		return domain.Site{}, err
	}
	return s.store.UpdateSiteTheme(ctx, siteID, theme)
}

// UpdateSiteChrome overwrites logoUrl/socialLinks wholesale (M14.11) — content.manage-gated, same
// authorize path as UpdateSiteTheme; these are content_sites' own settings, never a content
// document (docs/modules/content.md's own M14.11 framing).
func (s *Service) UpdateSiteChrome(ctx context.Context, siteID string, logoURL *string, socialLinks domain.SocialLinks) (domain.Site, error) {
	site, err := s.store.GetSiteByID(ctx, siteID)
	if err != nil {
		return domain.Site{}, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return domain.Site{}, err
	}
	return s.store.UpdateSiteChrome(ctx, siteID, logoURL, socialLinks)
}

// GetSiteChrome is the public read (ContentPublicService) a tenant site's header/footer fetches
// once (M14.11): logoUrl/socialLinks are content_sites' own persisted columns; congregationName/
// address/schedules are composed live from religion's application service — never copied into
// content's own tables (docs/modules/content.md's standing invariant, restated by this milestone).
// No auth: same anonymous shape as GetSite/GetSiteBySlug. Degrades gracefully rather than erroring
// if the unit has no religion site yet (address/schedules empty, name falls back to the site's own
// slug) — the same "found bool, no hard fail" shape ContentResolver already establishes cross-module.
func (s *Service) GetSiteChrome(ctx context.Context, siteID string) (domain.SiteChrome, error) {
	site, err := s.store.GetSiteByID(ctx, siteID)
	if err != nil {
		return domain.SiteChrome{}, err
	}
	chrome := domain.SiteChrome{
		CongregationName: site.Slug,
		LogoURL:          site.LogoURL,
		SocialLinks:      site.SocialLinks,
	}

	rsite, found, err := s.religion.GetPrimarySiteByUnit(ctx, site.CongregationUnitRID)
	if err != nil {
		return domain.SiteChrome{}, err
	}
	if !found {
		return chrome, nil
	}
	chrome.CongregationName = rsite.Name
	if line, ok := religiondomain.CoarsenAddress(rsite.Locality, rsite.AdminArea1, rsite.AdminArea2, rsite.Street, rsite.HouseNumber, rsite.PostalCode, rsite.PublicPrecision); ok {
		chrome.Address = &line
	}
	// M14.17: Church JSON-LD's geo field. Must go through the exact same religiondomain.Coarsen a
	// SearchSites/DiscoverySite read already applies (internal/religion/application/service.go) —
	// GetSiteChrome's own precise rsite.Latitude/Longitude bypasses it entirely otherwise, which
	// would leak an exact coordinate through this anonymous endpoint even for a `hidden`-precision
	// site whose address CoarsenAddress just suppressed two lines above. ok=false (hidden) leaves
	// both fields nil, same as Address's own hidden case.
	if lat, lng, ok := religiondomain.Coarsen(rsite.Latitude, rsite.Longitude, rsite.PublicPrecision); ok {
		chrome.Latitude = &lat
		chrome.Longitude = &lng
	}

	schedules, err := s.religion.ListServiceSchedulesByUnit(ctx, site.CongregationUnitRID)
	if err != nil {
		return domain.SiteChrome{}, err
	}
	chrome.Schedules = make([]domain.ServiceSchedule, len(schedules))
	for i, sch := range schedules {
		chrome.Schedules[i] = domain.ServiceSchedule{
			DayOfWeek: sch.DayOfWeek, RRule: sch.RRule, StartTime: sch.StartTime, EndTime: sch.EndTime,
			Timezone: sch.Timezone, Language: sch.Language, Mode: sch.Mode, MeetingURL: sch.MeetingURL, Description: sch.Description,
		}
	}
	return chrome, nil
}

// CreatePreviewLink mints a short-lived, site-scoped preview token (M14.7) — content.manage-gated
// like every other draft-adjacent read on this service. The token itself carries no document id: a
// draft is content, not a special code path (D-ContentRevisions), so previewing means rendering the
// whole site's current draft state, the same one-pager shape the real public renderer already uses.
func (s *Service) CreatePreviewLink(ctx context.Context, siteID string) (string, error) {
	site, err := s.store.GetSiteByID(ctx, siteID)
	if err != nil {
		return "", err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return "", err
	}
	return mintPreviewToken(site.ID, s.previewHMACKey)
}

// GetSite is the public read (ContentPublicService) — no auth, keyed by the congregation unit RID.
func (s *Service) GetSite(ctx context.Context, congregationUnitRID string) (domain.Site, error) {
	return s.store.GetSiteByUnit(ctx, congregationUnitRID)
}

// GetSiteBySlug is the public read (ContentPublicService) the tenant-subdomain proxy resolves a
// Host header's slug through (M14.9) — no auth, same anonymous shape as GetSite.
func (s *Service) GetSiteBySlug(ctx context.Context, slug string) (domain.Site, error) {
	return s.store.GetSiteBySlug(ctx, slug)
}

// ---- documents ----

// ListDocuments is the admin read (ContentService) — every state, content.manage-gated.
func (s *Service) ListDocuments(ctx context.Context, siteID string, kind, locale, state *string) ([]domain.Document, error) {
	site, err := s.store.GetSiteByID(ctx, siteID)
	if err != nil {
		return nil, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return nil, err
	}
	return s.store.ListDocuments(ctx, siteID, kind, locale, state)
}

// ListPublicDocuments is the public read (ContentPublicService) — always published/unlisted only,
// no auth.
func (s *Service) ListPublicDocuments(ctx context.Context, siteID string, kind, locale *string) ([]domain.Document, error) {
	return s.store.ListPublicDocuments(ctx, siteID, kind, locale)
}

// ListPreviewDocuments is ContentPublicService's one token-gated exception (M14.7) to "always
// published/unlisted only": every document in every state for the token's own site, reusing the
// exact same store query the admin's ListDocuments already calls with an unset state filter — this
// is a different caller (a token, not a session), not a different read.
func (s *Service) ListPreviewDocuments(ctx context.Context, siteID, token string, kind, locale *string) ([]domain.Document, error) {
	subjectSiteID, ok := verifyPreviewToken(token, s.previewHMACKey)
	if !ok || subjectSiteID != siteID {
		return nil, domain.ErrPreviewTokenInvalid
	}
	return s.store.ListDocuments(ctx, siteID, kind, locale, nil)
}

func (s *Service) CreateDocument(ctx context.Context, siteID string, in domain.CreateDocumentInput) (domain.Document, error) {
	site, err := s.store.GetSiteByID(ctx, siteID)
	if err != nil {
		return domain.Document{}, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return domain.Document{}, err
	}
	if in.Kind == domain.KindEvent && in.EventStartsAt == nil {
		return domain.Document{}, domain.ErrEventMissingStart
	}
	if in.ParentDocumentID != nil {
		if err := s.checkParentDepth(ctx, *in.ParentDocumentID); err != nil {
			return domain.Document{}, err
		}
	}
	if in.TranslationGroupID != nil {
		if err := s.checkTranslationGroup(ctx, siteID, *in.TranslationGroupID, in.Locale); err != nil {
			return domain.Document{}, err
		}
	}
	return s.store.InsertDocument(ctx, siteID, in)
}

// checkTranslationGroup (M14.14) guards CreateDocument's "join an existing translation group"
// path: there's no DB constraint enforcing either of these, so both are app-level here. Rejects a
// group that belongs to a different site (translation_group_id has no FK to site, so a caller
// could otherwise link an arbitrary site's documents together) and a locale already present in the
// group (content.md's own invariant is that a group's documents share nothing but the group id —
// nothing in the DB stops two rows in one group at the same locale otherwise).
func (s *Service) checkTranslationGroup(ctx context.Context, siteID, translationGroupID, locale string) error {
	siblings, err := s.store.ListDocumentsByTranslationGroup(ctx, translationGroupID)
	if err != nil {
		return err
	}
	if len(siblings) == 0 {
		return &domain.TranslationGroupNotFoundError{TranslationGroupID: translationGroupID}
	}
	for _, sibling := range siblings {
		if sibling.SiteID != siteID {
			return &domain.TranslationGroupNotFoundError{TranslationGroupID: translationGroupID}
		}
		if sibling.Locale == locale {
			return &domain.TranslationLocaleTakenError{TranslationGroupID: translationGroupID, Locale: locale}
		}
	}
	return nil
}

func (s *Service) UpdateDocument(ctx context.Context, documentID string, in domain.UpdateDocumentInput) (domain.Document, error) {
	doc, err := s.store.GetDocument(ctx, documentID)
	if err != nil {
		return domain.Document{}, err
	}
	site, err := s.store.GetSiteByID(ctx, doc.SiteID)
	if err != nil {
		return domain.Document{}, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return domain.Document{}, err
	}
	if in.ParentDocumentID != nil {
		if *in.ParentDocumentID == documentID {
			return domain.Document{}, domain.ErrParentTooDeep
		}
		if err := s.checkParentDepth(ctx, *in.ParentDocumentID); err != nil {
			return domain.Document{}, err
		}
	}
	return s.store.UpdateDocument(ctx, documentID, in)
}

// checkParentDepth walks parentID's own ancestor chain, erroring if a 4th ancestor would exist
// once documentID is attached beneath it (content.md: pages nest up to 3 levels, app-enforced) —
// same bounded-walk shape as internal/registration/application's checkNotExcluded.
func (s *Service) checkParentDepth(ctx context.Context, parentID string) error {
	id := parentID
	for depth := 0; depth < 3; depth++ {
		doc, err := s.store.GetDocument(ctx, id)
		if err != nil {
			return err
		}
		if doc.ParentDocumentID == nil {
			return nil
		}
		id = *doc.ParentDocumentID
	}
	return domain.ErrParentTooDeep
}

// transitions is the fixed document-state machine (content.md's invariants). Looked up by a
// document's *effective* state (Document.EffectiveState), not its raw one — M14.15/D-PublishOnRead:
// a SCHEDULED document past its PublishAt is, for the purposes of what actions are legal, already
// PUBLISHED, so an admin can UNLIST or REVERT_TO_DRAFT it without a scheduler ever having touched
// the row. Taking either action is what settles the stale column to something real.
var transitions = map[domain.DocumentState]map[domain.TransitionAction]domain.DocumentState{
	domain.StateDraft: {
		domain.ActionPublish:  domain.StatePublished,
		domain.ActionSchedule: domain.StateScheduled,
	},
	domain.StatePublished: {
		domain.ActionRevertToDraft: domain.StateDraft,
		domain.ActionUnlist:        domain.StateUnlisted,
	},
	domain.StateUnlisted: {
		domain.ActionPublish:  domain.StatePublished,
		domain.ActionSchedule: domain.StateScheduled,
	},
	domain.StateScheduled: {
		domain.ActionPublish:       domain.StatePublished,
		domain.ActionRevertToDraft: domain.StateDraft,
	},
}

// TransitionDocument applies action to documentID's document. publishAt is only meaningful (and
// required, and must be strictly future) for ActionSchedule — every other action ignores it.
func (s *Service) TransitionDocument(ctx context.Context, documentID string, action domain.TransitionAction, publishAt *time.Time) (domain.Document, error) {
	doc, err := s.store.GetDocument(ctx, documentID)
	if err != nil {
		return domain.Document{}, err
	}
	site, err := s.store.GetSiteByID(ctx, doc.SiteID)
	if err != nil {
		return domain.Document{}, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return domain.Document{}, err
	}
	now := time.Now()
	next, ok := transitions[doc.EffectiveState(now)][action]
	if !ok {
		return domain.Document{}, fmt.Errorf("%w: %s -> %s not allowed", domain.ErrInvalidTransition, doc.State, action)
	}
	if action == domain.ActionSchedule {
		if publishAt == nil {
			return domain.Document{}, domain.ErrScheduleMissingPublishAt
		}
		if !publishAt.After(now) {
			return domain.Document{}, domain.ErrSchedulePublishAtNotFuture
		}
		return s.store.ScheduleDocument(ctx, documentID, *doc.DraftRevisionID, currentPersonID(ctx), *publishAt)
	}
	firstPublish := next == domain.StatePublished && doc.PublishedAt == nil
	// M14.6: transitioning into PUBLISHED also snapshots the current draft into a new immutable
	// checkpoint revision and repoints published_revision_id — see PublishDocument's own comment.
	// Every other transition (revert-to-draft, unlist) is a pure state flip, unchanged from before
	// M14.6: published_revision_id keeps pointing at whatever was last published, which is exactly
	// what an UNLISTED document's direct (not listed) reads should keep serving.
	if next == domain.StatePublished {
		return s.store.PublishDocument(ctx, documentID, *doc.DraftRevisionID, currentPersonID(ctx), firstPublish)
	}
	// M14.15: clear a stale publish_at whenever a document leaves SCHEDULED by any path other than
	// publishing (which already routes through PublishDocument above and never touches publish_at).
	return s.store.UpdateDocumentState(ctx, documentID, next, firstPublish, nil)
}

// ---- blocks ----

// currentPersonID returns the request's authenticated subject's person id, or nil if none is
// attached to ctx. Only ever used for revision authorship metadata — never for an authorization
// decision, which requireManage alone decides.
func currentPersonID(ctx context.Context) *string {
	subject, ok := authz.SubjectFromContext(ctx)
	if !ok || subject.PersonID == "" {
		return nil
	}
	return &subject.PersonID
}

// GetBlocks is the admin read (ContentService) — works regardless of document state. M14.6: reads
// the draft revision's snapshot rather than live content_blocks rows.
func (s *Service) GetBlocks(ctx context.Context, documentID string) ([]domain.Block, error) {
	doc, err := s.store.GetDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}
	site, err := s.store.GetSiteByID(ctx, doc.SiteID)
	if err != nil {
		return nil, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return nil, err
	}
	draft, err := s.store.GetRevision(ctx, *doc.DraftRevisionID)
	if err != nil {
		return nil, err
	}
	return unmarshalBlocksSnapshot(documentID, draft.CreatedAt, draft.Data)
}

// GetPublicBlocks is the public read (ContentPublicService) — Content:DocumentNotFound for a draft
// document or one that has never been published, never distinguishing either from "doesn't exist"
// (content.md's invariant, unchanged by M14.6). Reads the published revision's snapshot, completely
// decoupled from whatever the draft currently holds — the whole point of forward revisions.
func (s *Service) GetPublicBlocks(ctx context.Context, documentID string) ([]domain.Block, error) {
	doc, err := s.store.GetDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if !doc.IsPubliclyVisible(time.Now()) || doc.PublishedRevisionID == nil {
		return nil, domain.ErrDocumentNotFound
	}
	published, err := s.store.GetRevision(ctx, *doc.PublishedRevisionID)
	if err != nil {
		return nil, err
	}
	return unmarshalBlocksSnapshot(documentID, published.CreatedAt, published.Data)
}

// GetPreviewBlocks is ContentPublicService's other token-gated exception (M14.7): reads the draft
// revision regardless of document state, exactly like the admin's GetBlocks, but authorized by a
// site-scoped token instead of content.manage — the document's own site must match the token's
// subject, checked here rather than trusted from the caller.
func (s *Service) GetPreviewBlocks(ctx context.Context, documentID, token string) ([]domain.Block, error) {
	doc, err := s.store.GetDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}
	subjectSiteID, ok := verifyPreviewToken(token, s.previewHMACKey)
	if !ok || subjectSiteID != doc.SiteID {
		return nil, domain.ErrPreviewTokenInvalid
	}
	draft, err := s.store.GetRevision(ctx, *doc.DraftRevisionID)
	if err != nil {
		return nil, err
	}
	return unmarshalBlocksSnapshot(documentID, draft.CreatedAt, draft.Data)
}

// PutBlocks is every draft save — the manual "Save" action and the debounced autosave call alike
// (web/apps/admin), both hitting this one endpoint. Validation is unchanged from before M14.6; only
// the final persistence step moved from a delete-then-insert into content_blocks to an in-place
// update of the draft revision's snapshot.
func (s *Service) PutBlocks(ctx context.Context, documentID string, blocks []domain.BlockInput) ([]domain.Block, error) {
	doc, err := s.store.GetDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}
	site, err := s.store.GetSiteByID(ctx, doc.SiteID)
	if err != nil {
		return nil, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return nil, err
	}

	seen := make(map[int]bool, len(blocks))
	for i := range blocks {
		b := &blocks[i]
		if seen[b.Position] {
			return nil, &domain.DuplicateBlockPositionError{Position: b.Position}
		}
		seen[b.Position] = true

		blockType, err := s.store.GetBlockTypeByCode(ctx, b.BlockTypeCode)
		if err != nil {
			return nil, err
		}
		if blockType.Status != domain.BlockTypeActive {
			return nil, domain.ErrBlockTypeNotFound
		}
		// M14.3: normalize known share-link hosts (Google Drive/Dropbox/OneDrive) before
		// validation, so the stored url is the direct-content form and the schema's own
		// "format":"uri" / scheme allowlist both see the normalized value.
		normalized, err := normalizeBlockMediaURLs(b.BlockTypeCode, b.Data)
		if err != nil {
			return nil, err
		}
		b.Data = normalized
		if err := validateBlockData(blockType, b.Position, b.Data); err != nil {
			return nil, err
		}
	}

	snapshot, err := marshalBlocksSnapshot(blocks)
	if err != nil {
		return nil, err
	}
	draft, err := s.store.SaveDraftRevisionData(ctx, *doc.DraftRevisionID, snapshot)
	if err != nil {
		return nil, err
	}
	return unmarshalBlocksSnapshot(documentID, draft.CreatedAt, draft.Data)
}

// ListRevisions is the history list (ContentService, M14.6) — every checkpoint but the draft
// itself, newest first.
func (s *Service) ListRevisions(ctx context.Context, documentID string) ([]domain.DocumentRevision, error) {
	doc, err := s.store.GetDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}
	site, err := s.store.GetSiteByID(ctx, doc.SiteID)
	if err != nil {
		return nil, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return nil, err
	}
	return s.store.ListCheckpointRevisions(ctx, documentID, *doc.DraftRevisionID)
}

// RestoreRevision copies a past checkpoint's snapshot into the draft — into the draft only, per the
// owner's decision: it never touches published_revision_id, so the public site is unaffected until
// an explicit Publish. revisionID is verified to belong to documentID before use: a client-supplied
// id is never trusted to already be scoped correctly (content.md's own authorization-touchpoints
// note on the same discipline elsewhere in this module).
func (s *Service) RestoreRevision(ctx context.Context, documentID, revisionID string) ([]domain.Block, error) {
	doc, err := s.store.GetDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}
	site, err := s.store.GetSiteByID(ctx, doc.SiteID)
	if err != nil {
		return nil, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return nil, err
	}
	target, err := s.store.GetRevision(ctx, revisionID)
	if err != nil {
		return nil, err
	}
	if target.DocumentID != documentID {
		return nil, domain.ErrRevisionNotFound
	}
	draft, err := s.store.SaveDraftRevisionData(ctx, *doc.DraftRevisionID, target.Data)
	if err != nil {
		return nil, err
	}
	return unmarshalBlocksSnapshot(documentID, draft.CreatedAt, draft.Data)
}

// ---- nav items (M14.10) ----

// PutNavItems is a full replace, content.manage-gated like every other write on this service —
// a small, hand-curated list edited as a batch, the same shape putBlocks used before M14.6's
// revision refactor moved it to an in-place update.
func (s *Service) PutNavItems(ctx context.Context, siteID string, items []domain.NavItemInput) ([]domain.NavItem, error) {
	site, err := s.store.GetSiteByID(ctx, siteID)
	if err != nil {
		return nil, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return nil, err
	}

	seen := make(map[int]bool, len(items))
	for _, item := range items {
		if seen[item.SortOrder] {
			return nil, &domain.DuplicateNavItemSortOrderError{SortOrder: item.SortOrder}
		}
		seen[item.SortOrder] = true

		hasDoc, hasURL := item.TargetDocumentID != nil, item.TargetURL != nil
		if hasDoc == hasURL {
			return nil, &domain.NavTargetAmbiguousError{SortOrder: item.SortOrder}
		}
		if hasDoc {
			target, err := s.store.GetDocument(ctx, *item.TargetDocumentID)
			if errors.Is(err, domain.ErrDocumentNotFound) {
				return nil, &domain.NavTargetInvalidError{TargetDocumentID: *item.TargetDocumentID}
			}
			if err != nil {
				return nil, err
			}
			if target.SiteID != siteID || target.Kind != domain.KindPage {
				return nil, &domain.NavTargetInvalidError{TargetDocumentID: *item.TargetDocumentID}
			}
		}
	}
	return s.store.ReplaceNavItems(ctx, siteID, items)
}

// ListNavItems is the admin read (ContentService) — content.manage-gated.
func (s *Service) ListNavItems(ctx context.Context, siteID string) ([]domain.NavItem, error) {
	site, err := s.store.GetSiteByID(ctx, siteID)
	if err != nil {
		return nil, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return nil, err
	}
	return s.store.ListNavItems(ctx, siteID)
}

// ListPublicNavItems is the public read (ContentPublicService) — no auth. An item whose target
// document is missing or DRAFT is silently omitted, never surfaced as a broken link (D-
// ContentRevisions' own consequence: a draft is never public).
func (s *Service) ListPublicNavItems(ctx context.Context, siteID string) ([]domain.PublicNavItem, error) {
	items, err := s.store.ListNavItems(ctx, siteID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PublicNavItem, 0, len(items))
	for _, item := range items {
		if item.TargetURL != nil {
			out = append(out, domain.PublicNavItem{Label: item.Label, Href: *item.TargetURL, External: true})
			continue
		}
		doc, err := s.store.GetDocument(ctx, *item.TargetDocumentID)
		if errors.Is(err, domain.ErrDocumentNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !doc.IsPubliclyVisible(time.Now()) {
			continue
		}
		ancestors, err := s.resolveAncestorChain(ctx, doc)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.PublicNavItem{Label: item.Label, Href: buildPublicHref(doc, ancestors), External: false})
	}
	return out, nil
}

// ListSitemapEntries (M14.17) is the public read web/apps/web's app/sitemap.ts calls — every
// effectively-PUBLISHED PAGE document, hrefs already resolved server-side via the same
// resolveAncestorChain/buildPublicHref pair ListPublicNavItems already uses, so the caller never
// walks N ancestor chains itself. Reuses ListPublicDocuments's own store query (kind=PAGE) rather
// than a new one; the UNLISTED exclusion happens here, in Go, since that query's own WHERE clause
// intentionally includes UNLISTED for every other caller.
func (s *Service) ListSitemapEntries(ctx context.Context, siteID string) ([]domain.SitemapEntry, error) {
	kind := string(domain.KindPage)
	docs, err := s.store.ListPublicDocuments(ctx, siteID, &kind, nil)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]domain.SitemapEntry, 0, len(docs))
	for _, doc := range docs {
		if doc.EffectiveState(now) != domain.StatePublished {
			continue
		}
		ancestors, err := s.resolveAncestorChain(ctx, doc)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.SitemapEntry{Href: buildPublicHref(doc, ancestors), UpdatedAt: doc.UpdatedAt})
	}
	return out, nil
}

// GetPublicDocumentByPath resolves the leaf PAGE document (by locale + slug) plus its real
// ancestor chain, for the tenant-subdomain catch-all page route — no auth. segments is the
// ordered, 1-to-3-long list of URL path slugs; every segment before the last must match the
// leaf's real parent_document_id chain positionally (only the leaf's own state gates public
// visibility — an ancestor's state is never re-checked, only its slug identity). Any mismatch,
// including the segment count itself, collapses to the same Content:DocumentNotFound a missing
// document already returns — never resolved by the last segment alone.
func (s *Service) GetPublicDocumentByPath(ctx context.Context, siteID, locale string, segments []string) (domain.Document, []domain.Document, []domain.DocumentTranslation, error) {
	if len(segments) == 0 || len(segments) > 3 {
		return domain.Document{}, nil, nil, domain.ErrDocumentNotFound
	}
	leaf, err := s.store.GetDocumentBySlug(ctx, siteID, domain.KindPage, locale, segments[len(segments)-1])
	if err != nil {
		return domain.Document{}, nil, nil, err
	}
	if !leaf.IsPubliclyVisible(time.Now()) {
		return domain.Document{}, nil, nil, domain.ErrDocumentNotFound
	}
	ancestors, err := s.resolveAncestorChain(ctx, leaf)
	if err != nil {
		return domain.Document{}, nil, nil, err
	}
	if len(ancestors) != len(segments)-1 {
		return domain.Document{}, nil, nil, domain.ErrDocumentNotFound
	}
	for i, ancestor := range ancestors {
		if ancestor.Slug != segments[i] {
			return domain.Document{}, nil, nil, domain.ErrDocumentNotFound
		}
	}
	translations, err := s.resolvePublishedTranslations(ctx, leaf)
	if err != nil {
		return domain.Document{}, nil, nil, err
	}
	return leaf, ancestors, translations, nil
}

// resolvePublishedTranslations (M14.14, DS-OFM-7) lists every effectively-PUBLISHED document
// sharing leaf's translation group — leaf's own group membership means it's included in the result
// set already, no special-casing needed. Each sibling gets its own ancestor-chain walk and href:
// siblings can live at a different hierarchy/slug per locale, so a translation's href is never
// derived from the leaf's own href. M14.15: a due SCHEDULED sibling counts as PUBLISHED here just
// like everywhere else on the public read path; UNLISTED stays excluded, unchanged from M14.14 —
// only PUBLISHED (real or effective) siblings are worth linking to as a translation.
func (s *Service) resolvePublishedTranslations(ctx context.Context, leaf domain.Document) ([]domain.DocumentTranslation, error) {
	siblings, err := s.store.ListDocumentsByTranslationGroup(ctx, leaf.TranslationGroupID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	var out []domain.DocumentTranslation
	for _, sibling := range siblings {
		if sibling.EffectiveState(now) != domain.StatePublished {
			continue
		}
		ancestors, err := s.resolveAncestorChain(ctx, sibling)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.DocumentTranslation{Locale: sibling.Locale, Href: buildPublicHref(sibling, ancestors)})
	}
	return out, nil
}

// resolveAncestorChain walks doc's real ParentDocumentID chain, root-first, leaf excluded —
// bounded at 3 (mirrors checkParentDepth's own bound), shared by ListPublicNavItems and
// GetPublicDocumentByPath so the walk is never duplicated.
func (s *Service) resolveAncestorChain(ctx context.Context, doc domain.Document) ([]domain.Document, error) {
	var chain []domain.Document
	id := doc.ParentDocumentID
	for depth := 0; id != nil && depth < 3; depth++ {
		ancestor, err := s.store.GetDocument(ctx, *id)
		if err != nil {
			return nil, err
		}
		chain = append(chain, ancestor)
		id = ancestor.ParentDocumentID
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, nil
}

// buildPublicHref assembles a document's hierarchical public path from its resolved ancestor
// chain (root-first) plus its own slug.
func buildPublicHref(doc domain.Document, ancestors []domain.Document) string {
	segments := make([]string, 0, len(ancestors)+1)
	for _, a := range ancestors {
		segments = append(segments, a.Slug)
	}
	segments = append(segments, doc.Slug)
	return "/" + doc.Locale + "/" + strings.Join(segments, "/")
}

// ---- block-type catalog ----

// ListBlockTypes is the public read (ContentPublicService) — active types only, no auth.
func (s *Service) ListBlockTypes(ctx context.Context) ([]domain.BlockType, error) {
	return s.store.ListActiveBlockTypes(ctx)
}

// ---- block-type catalog admin (M14.13, content.catalog.manage) ----

// ListAllBlockTypesForCatalog is the admin catalog read (ContentService) — every status, so a
// moderator can see (and un-retire) RETIRED types too, catalog.manage-gated rather than public.
func (s *Service) ListAllBlockTypesForCatalog(ctx context.Context) ([]domain.BlockType, error) {
	if err := s.requireCatalogManage(ctx); err != nil {
		return nil, err
	}
	return s.store.ListAllBlockTypes(ctx)
}

// CreateBlockType finally builds what M3 left unbuilt (content.catalog.manage). The submitted
// json_schema is smoke-tested for compile-validity up front — a broken schema would otherwise only
// surface on the first putBlocks call that references it, far from where the mistake was made.
func (s *Service) CreateBlockType(ctx context.Context, in domain.CreateBlockTypeInput) (domain.BlockType, error) {
	if err := s.requireCatalogManage(ctx); err != nil {
		return domain.BlockType{}, err
	}
	if err := compileBlockTypeSchema(in.Code, in.JSONSchema); err != nil {
		return domain.BlockType{}, err
	}
	return s.store.InsertBlockType(ctx, in)
}

// UpdateBlockType only ever touches name/status/sortOrder (domain.UpdateBlockTypeInput has no
// schema field at all) — see that type's own doc comment for the owner decision this encodes.
func (s *Service) UpdateBlockType(ctx context.Context, blockTypeID string, in domain.UpdateBlockTypeInput) (domain.BlockType, error) {
	if err := s.requireCatalogManage(ctx); err != nil {
		return domain.BlockType{}, err
	}
	return s.store.UpdateBlockType(ctx, blockTypeID, in)
}

// ---- patterns (M14.13, D-SitePatterns) ----

// ListPatterns is the public read (ContentPublicService) — not sensitive data (same reasoning
// ListBlockTypes already uses for having no auth); the admin editor calls this exact endpoint to
// browse patterns for insertion, the same way it already calls the public ListBlockTypes.
func (s *Service) ListPatterns(ctx context.Context) ([]domain.Pattern, error) {
	return s.store.ListPatterns(ctx)
}

// CreatePattern is catalog.manage-gated — platform-wide, no site/unit in scope.
func (s *Service) CreatePattern(ctx context.Context, in domain.CreatePatternInput) (domain.Pattern, error) {
	if err := s.requireCatalogManage(ctx); err != nil {
		return domain.Pattern{}, err
	}
	return s.store.InsertPattern(ctx, in)
}

func (s *Service) UpdatePattern(ctx context.Context, patternID string, in domain.UpdatePatternInput) (domain.Pattern, error) {
	if err := s.requireCatalogManage(ctx); err != nil {
		return domain.Pattern{}, err
	}
	return s.store.UpdatePattern(ctx, patternID, in)
}

// DeletePattern soft-deletes — a pattern already inserted into a document is unaffected (unsynced:
// no ongoing reference from any document back to this row).
func (s *Service) DeletePattern(ctx context.Context, patternID string) error {
	if err := s.requireCatalogManage(ctx); err != nil {
		return err
	}
	return s.store.DeletePattern(ctx, patternID)
}

// ---- form submissions (M14.16, D-InAppInbox) ----

// minContactFormSubmitDuration is the minimum time a real visitor needs between the page rendering
// and the form posting back — a submission faster than this is treated exactly like a filled
// honeypot: silently discarded, still reported as success. Deliberately generous (no legitimate
// human fills three fields and submits inside it) rather than tuned against real traffic, which
// doesn't exist yet.
const minContactFormSubmitDuration = 3 * time.Second

// SubmitContactForm is the anonymous write (ContentPublicService) — the third such endpoint in the
// codebase after moderation's two. A honeypot hit or a too-fast submission is never an error
// (D-InAppInbox: "an error teaches the bot") — both return success having inserted nothing, so a
// probing caller learns nothing about which check, if either, fired.
func (s *Service) SubmitContactForm(ctx context.Context, in domain.SubmitContactFormInput) error {
	if _, err := s.store.GetSiteByID(ctx, in.SiteID); err != nil {
		return err
	}
	if in.Honeypot != "" {
		return nil
	}
	if time.Since(in.FormRenderedAt) < minContactFormSubmitDuration {
		return nil
	}
	message := strings.TrimSpace(in.Message)
	if message == "" {
		return &domain.FormSubmissionInvalidError{Field: "message"}
	}
	in.Message = message
	_, err := s.store.InsertFormSubmission(ctx, in)
	return err
}

// ListFormSubmissions is the admin read (ContentService) backing the Messages screen —
// content.manage-gated, same shape as ListDocuments.
func (s *Service) ListFormSubmissions(ctx context.Context, siteID string) ([]domain.FormSubmission, error) {
	site, err := s.store.GetSiteByID(ctx, siteID)
	if err != nil {
		return nil, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return nil, err
	}
	return s.store.ListFormSubmissions(ctx, siteID)
}
