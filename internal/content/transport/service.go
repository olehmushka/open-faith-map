// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport implements the generated content.ContentService and content.ContentPublicService
// (Conjure server interfaces): translates Conjure structs <-> domain types and maps domain errors to
// this module's typed Conjure errors.
//
// M10.6: the caller's identity no longer arrives via a per-request whoami round-trip — internal/authz
// resolves the subject from context (populated by internal/identity's authenticator middleware) at
// the point requireManage decides, so this layer no longer needs it at all.
package transport

import (
	"context"
	"time"

	gencontent "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/content"
	"github.com/olehmushka/open-faith-map/internal/content/application"
	"github.com/olehmushka/open-faith-map/internal/content/domain"
	"github.com/palantir/pkg/bearertoken"
	"github.com/palantir/pkg/datetime"
)

type Service struct {
	appService *application.Service
}

func NewService(appService *application.Service) *Service {
	return &Service{appService: appService}
}

var _ gencontent.ContentService = (*Service)(nil)

func (s *Service) CreateSite(ctx context.Context, authHeader bearertoken.Token, requestArg gencontent.CreateSiteRequest) (gencontent.Site, error) {
	site, err := s.appService.CreateSite(ctx, domain.CreateSiteInput{
		CongregationUnitRID: requestArg.CongregationUnitId,
		Slug:                requestArg.Slug,
	})
	if err != nil {
		return gencontent.Site{}, mapErr(err, errCtx{})
	}
	return toAPISite(site), nil
}

func (s *Service) UpdateSiteTheme(ctx context.Context, authHeader bearertoken.Token, siteIdArg string, requestArg gencontent.UpdateSiteThemeRequest) (gencontent.Site, error) {
	themeBytes, err := marshalAny(requestArg.Theme)
	if err != nil {
		return gencontent.Site{}, err
	}
	site, err := s.appService.UpdateSiteTheme(ctx, siteIdArg, themeBytes)
	if err != nil {
		return gencontent.Site{}, mapErr(err, errCtx{SiteID: siteIdArg})
	}
	return toAPISite(site), nil
}

func (s *Service) ListDocuments(ctx context.Context, authHeader bearertoken.Token, siteIdArg string, kindArg, localeArg, stateArg *string) (gencontent.DocumentPage, error) {
	docs, err := s.appService.ListDocuments(ctx, siteIdArg, kindArg, localeArg, stateArg)
	if err != nil {
		return gencontent.DocumentPage{}, mapErr(err, errCtx{SiteID: siteIdArg})
	}
	return gencontent.DocumentPage{Documents: toAPIDocuments(docs)}, nil
}

func (s *Service) CreateDocument(ctx context.Context, authHeader bearertoken.Token, siteIdArg string, requestArg gencontent.CreateDocumentRequest) (gencontent.Document, error) {
	var eventStartsAt, eventEndsAt *time.Time
	if requestArg.EventStartsAt != nil {
		t := time.Time(*requestArg.EventStartsAt)
		eventStartsAt = &t
	}
	if requestArg.EventEndsAt != nil {
		t := time.Time(*requestArg.EventEndsAt)
		eventEndsAt = &t
	}
	doc, err := s.appService.CreateDocument(ctx, siteIdArg, domain.CreateDocumentInput{
		Kind:                 domain.DocumentKind(requestArg.Kind.Value()),
		TranslationGroupID:   requestArg.TranslationGroupId,
		Locale:               requestArg.Locale,
		ParentDocumentID:     requestArg.ParentDocumentId,
		Slug:                 requestArg.Slug,
		EventStartsAt:        eventStartsAt,
		EventEndsAt:          eventEndsAt,
		EventRecurrenceRRule: requestArg.EventRecurrenceRrule,
	})
	if err != nil {
		return gencontent.Document{}, mapErr(err, errCtx{SiteID: siteIdArg, Kind: string(requestArg.Kind.Value()), ParentDocumentID: derefOr(requestArg.ParentDocumentId, "")})
	}
	return toAPIDocument(doc), nil
}

func (s *Service) UpdateDocument(ctx context.Context, authHeader bearertoken.Token, documentIdArg string, requestArg gencontent.UpdateDocumentRequest) (gencontent.Document, error) {
	doc, err := s.appService.UpdateDocument(ctx, documentIdArg, domain.UpdateDocumentInput{
		Slug:             requestArg.Slug,
		ParentDocumentID: requestArg.ParentDocumentId,
		ClearParent:      requestArg.ClearParent,
	})
	if err != nil {
		return gencontent.Document{}, mapErr(err, errCtx{DocumentID: documentIdArg, ParentDocumentID: derefOr(requestArg.ParentDocumentId, "")})
	}
	return toAPIDocument(doc), nil
}

func (s *Service) TransitionDocument(ctx context.Context, authHeader bearertoken.Token, documentIdArg string, requestArg gencontent.TransitionDocumentRequest) (gencontent.Document, error) {
	action := domain.TransitionAction(requestArg.Action.Value())
	doc, err := s.appService.TransitionDocument(ctx, documentIdArg, action)
	if err != nil {
		return gencontent.Document{}, mapErr(err, errCtx{DocumentID: documentIdArg, Action: string(action)})
	}
	return toAPIDocument(doc), nil
}

func (s *Service) GetBlocks(ctx context.Context, authHeader bearertoken.Token, documentIdArg string) (gencontent.BlockList, error) {
	blocks, err := s.appService.GetBlocks(ctx, documentIdArg)
	if err != nil {
		return gencontent.BlockList{}, mapErr(err, errCtx{DocumentID: documentIdArg})
	}
	return gencontent.BlockList{Blocks: toAPIBlocks(blocks)}, nil
}

func (s *Service) PutBlocks(ctx context.Context, authHeader bearertoken.Token, documentIdArg string, requestArg gencontent.PutBlocksRequest) (gencontent.BlockList, error) {
	inputs := make([]domain.BlockInput, 0, len(requestArg.Blocks))
	for _, b := range requestArg.Blocks {
		data, err := marshalAny(b.Data)
		if err != nil {
			return gencontent.BlockList{}, err
		}
		inputs = append(inputs, domain.BlockInput{BlockTypeCode: b.BlockTypeCode, Position: b.Position, Data: data})
	}
	blocks, err := s.appService.PutBlocks(ctx, documentIdArg, inputs)
	if err != nil {
		return gencontent.BlockList{}, mapErr(err, errCtx{DocumentID: documentIdArg})
	}
	return gencontent.BlockList{Blocks: toAPIBlocks(blocks)}, nil
}

func toAPISite(site domain.Site) gencontent.Site {
	return gencontent.Site{
		Id:                 site.ID,
		CongregationUnitId: site.CongregationUnitRID,
		Slug:               site.Slug,
		Theme:              unmarshalAny(site.Theme),
		CreatedAt:          datetime.DateTime(site.CreatedAt),
		UpdatedAt:          datetime.DateTime(site.UpdatedAt),
	}
}

func toAPIDocuments(docs []domain.Document) []gencontent.Document {
	out := make([]gencontent.Document, 0, len(docs))
	for _, d := range docs {
		out = append(out, toAPIDocument(d))
	}
	return out
}

func toAPIDocument(d domain.Document) gencontent.Document {
	var publishedAt, eventStartsAt, eventEndsAt *datetime.DateTime
	if d.PublishedAt != nil {
		dt := datetime.DateTime(*d.PublishedAt)
		publishedAt = &dt
	}
	if d.EventStartsAt != nil {
		dt := datetime.DateTime(*d.EventStartsAt)
		eventStartsAt = &dt
	}
	if d.EventEndsAt != nil {
		dt := datetime.DateTime(*d.EventEndsAt)
		eventEndsAt = &dt
	}
	return gencontent.Document{
		Id:                   d.ID,
		SiteId:               d.SiteID,
		Kind:                 gencontent.New_DocumentKind(gencontent.DocumentKind_Value(d.Kind)),
		TranslationGroupId:   d.TranslationGroupID,
		Locale:               d.Locale,
		ParentDocumentId:     d.ParentDocumentID,
		Slug:                 d.Slug,
		State:                gencontent.New_DocumentState(gencontent.DocumentState_Value(d.State)),
		PublishedAt:          publishedAt,
		EventStartsAt:        eventStartsAt,
		EventEndsAt:          eventEndsAt,
		EventRecurrenceRrule: d.EventRecurrenceRRule,
		CreatedAt:            datetime.DateTime(d.CreatedAt),
		UpdatedAt:            datetime.DateTime(d.UpdatedAt),
	}
}

func toAPIBlocks(blocks []domain.Block) []gencontent.Block {
	out := make([]gencontent.Block, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, gencontent.Block{
			Id:            b.ID,
			DocumentId:    b.DocumentID,
			BlockTypeCode: b.BlockTypeCode,
			Position:      b.Position,
			Data:          unmarshalAny(b.Data),
			CreatedAt:     datetime.DateTime(b.CreatedAt),
			UpdatedAt:     datetime.DateTime(b.UpdatedAt),
		})
	}
	return out
}

func derefOr(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}
