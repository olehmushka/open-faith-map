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
	"fmt"

	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/olehmushka/open-faith-map/internal/content/adapters"
	"github.com/olehmushka/open-faith-map/internal/content/domain"
)

type Service struct {
	store          *adapters.Repository
	authzSvc       *authz.Service
	previewHMACKey string
}

func NewService(store *adapters.Repository, authzSvc *authz.Service, previewHMACKey string) *Service {
	return &Service{store: store, authzSvc: authzSvc, previewHMACKey: previewHMACKey}
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

func (s *Service) UpdateSiteTheme(ctx context.Context, siteID string, theme json.RawMessage) (domain.Site, error) {
	site, err := s.store.GetSiteByID(ctx, siteID)
	if err != nil {
		return domain.Site{}, err
	}
	if err := s.requireManage(ctx, site.CongregationUnitRID); err != nil {
		return domain.Site{}, err
	}
	return s.store.UpdateSiteTheme(ctx, siteID, theme)
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
	return s.store.InsertDocument(ctx, siteID, in)
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

// transitions is the fixed document-state machine (content.md's invariants).
var transitions = map[domain.DocumentState]map[domain.TransitionAction]domain.DocumentState{
	domain.StateDraft: {
		domain.ActionPublish: domain.StatePublished,
	},
	domain.StatePublished: {
		domain.ActionRevertToDraft: domain.StateDraft,
		domain.ActionUnlist:        domain.StateUnlisted,
	},
	domain.StateUnlisted: {
		domain.ActionPublish: domain.StatePublished,
	},
}

func (s *Service) TransitionDocument(ctx context.Context, documentID string, action domain.TransitionAction) (domain.Document, error) {
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
	next, ok := transitions[doc.State][action]
	if !ok {
		return domain.Document{}, fmt.Errorf("%w: %s -> %s not allowed", domain.ErrInvalidTransition, doc.State, action)
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
	return s.store.UpdateDocumentState(ctx, documentID, next, firstPublish)
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
	if doc.State == domain.StateDraft || doc.PublishedRevisionID == nil {
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

// ---- block-type catalog ----

// ListBlockTypes is the public read (ContentPublicService) — active types only, no auth.
func (s *Service) ListBlockTypes(ctx context.Context) ([]domain.BlockType, error) {
	return s.store.ListActiveBlockTypes(ctx)
}
