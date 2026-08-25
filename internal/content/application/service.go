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
	store    *adapters.Repository
	authzSvc *authz.Service
}

func NewService(store *adapters.Repository, authzSvc *authz.Service) *Service {
	return &Service{store: store, authzSvc: authzSvc}
}

// ---- sites ----

// CreateSite's target is the request's own claimed congregation unit — no site row exists yet to
// load one from.
func (s *Service) CreateSite(ctx context.Context, in domain.CreateSiteInput) (domain.Site, error) {
	if err := s.requireManage(ctx, in.CongregationUnitRID); err != nil {
		return domain.Site{}, err
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

// GetSite is the public read (ContentPublicService) — no auth, keyed by the congregation unit RID.
func (s *Service) GetSite(ctx context.Context, congregationUnitRID string) (domain.Site, error) {
	return s.store.GetSiteByUnit(ctx, congregationUnitRID)
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
	return s.store.UpdateDocumentState(ctx, documentID, next, firstPublish)
}

// ---- blocks ----

// GetBlocks is the admin read (ContentService) — works regardless of document state.
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
	return s.store.ListBlocks(ctx, documentID)
}

// GetPublicBlocks is the public read (ContentPublicService) — Content:DocumentNotFound for a draft
// document, never distinguishing "draft" from "doesn't exist" (content.md).
func (s *Service) GetPublicBlocks(ctx context.Context, documentID string) ([]domain.Block, error) {
	doc, err := s.store.GetDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if doc.State == domain.StateDraft {
		return nil, domain.ErrDocumentNotFound
	}
	return s.store.ListBlocks(ctx, documentID)
}

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
	for _, b := range blocks {
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
		if err := validateBlockData(blockType, b.Position, b.Data); err != nil {
			return nil, err
		}
	}

	return s.store.ReplaceBlocks(ctx, documentID, blocks)
}

// ---- block-type catalog ----

// ListBlockTypes is the public read (ContentPublicService) — active types only, no auth.
func (s *Service) ListBlockTypes(ctx context.Context) ([]domain.BlockType, error) {
	return s.store.ListActiveBlockTypes(ctx)
}
