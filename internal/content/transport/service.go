// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport implements the generated content.ContentService and content.ContentPublicService
// (Conjure server interfaces): translates Conjure structs <-> domain types, resolves the caller's
// own person RID from their forwarded token (never a client-supplied id — always asked of
// go-oikumenea itself via whoami), and maps domain errors to this module's typed Conjure errors.
package transport

import (
	"context"

	gencontent "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/content"
	"github.com/olehmushka/open-faith-map/internal/content/application"
	"github.com/olehmushka/open-faith-map/internal/content/domain"
	"github.com/olehmushka/open-faith-map/internal/coreintegration"
	"github.com/palantir/pkg/bearertoken"
	"github.com/palantir/pkg/datetime"
)

type Config struct {
	OikumeneaBaseURL            string
	OikumeneaInsecureSkipVerify bool
}

type Service struct {
	oikumeneaBaseURL  string
	oikumeneaInsecure bool
	appService        *application.Service
}

func NewService(appService *application.Service, cfg Config) *Service {
	return &Service{appService: appService, oikumeneaBaseURL: cfg.OikumeneaBaseURL, oikumeneaInsecure: cfg.OikumeneaInsecureSkipVerify}
}

var _ gencontent.ContentService = (*Service)(nil)

// whoami resolves the caller's own go-oikumenea person RID from their forwarded token — never
// trusts a client-supplied id.
func (s *Service) whoami(ctx context.Context, token bearertoken.Token) (string, error) {
	c, err := coreintegration.NewUserClient(s.oikumeneaBaseURL, string(token), s.oikumeneaInsecure)
	if err != nil {
		return "", err
	}
	who, err := c.IdentityFederation.Whoami(ctx)
	if err != nil {
		return "", err
	}
	return who.PersonId, nil
}

func (s *Service) CreateSite(ctx context.Context, authHeader bearertoken.Token, requestArg gencontent.CreateSiteRequest) (gencontent.Site, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencontent.Site{}, mapUpstreamErr(err)
	}
	site, err := s.appService.CreateSite(ctx, string(authHeader), personID, domain.CreateSiteInput{
		CongregationUnitRID: requestArg.CongregationUnitId,
		Slug:                requestArg.Slug,
	})
	if err != nil {
		return gencontent.Site{}, mapErr(err, errCtx{})
	}
	return toAPISite(site), nil
}

func (s *Service) UpdateSiteTheme(ctx context.Context, authHeader bearertoken.Token, siteIdArg string, requestArg gencontent.UpdateSiteThemeRequest) (gencontent.Site, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencontent.Site{}, mapUpstreamErr(err)
	}
	themeBytes, err := marshalAny(requestArg.Theme)
	if err != nil {
		return gencontent.Site{}, err
	}
	site, err := s.appService.UpdateSiteTheme(ctx, string(authHeader), personID, siteIdArg, themeBytes)
	if err != nil {
		return gencontent.Site{}, mapErr(err, errCtx{SiteID: siteIdArg})
	}
	return toAPISite(site), nil
}

func (s *Service) ListDocuments(ctx context.Context, authHeader bearertoken.Token, siteIdArg string, kindArg, localeArg, stateArg *string) (gencontent.DocumentPage, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencontent.DocumentPage{}, mapUpstreamErr(err)
	}
	docs, err := s.appService.ListDocuments(ctx, string(authHeader), personID, siteIdArg, kindArg, localeArg, stateArg)
	if err != nil {
		return gencontent.DocumentPage{}, mapErr(err, errCtx{SiteID: siteIdArg})
	}
	return gencontent.DocumentPage{Documents: toAPIDocuments(docs)}, nil
}

func (s *Service) CreateDocument(ctx context.Context, authHeader bearertoken.Token, siteIdArg string, requestArg gencontent.CreateDocumentRequest) (gencontent.Document, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencontent.Document{}, mapUpstreamErr(err)
	}
	doc, err := s.appService.CreateDocument(ctx, string(authHeader), personID, siteIdArg, domain.CreateDocumentInput{
		Kind:               domain.DocumentKind(requestArg.Kind.Value()),
		TranslationGroupID: requestArg.TranslationGroupId,
		Locale:             requestArg.Locale,
		ParentDocumentID:   requestArg.ParentDocumentId,
		Slug:               requestArg.Slug,
	})
	if err != nil {
		return gencontent.Document{}, mapErr(err, errCtx{SiteID: siteIdArg, Kind: string(requestArg.Kind.Value()), ParentDocumentID: derefOr(requestArg.ParentDocumentId, "")})
	}
	return toAPIDocument(doc), nil
}

func (s *Service) UpdateDocument(ctx context.Context, authHeader bearertoken.Token, documentIdArg string, requestArg gencontent.UpdateDocumentRequest) (gencontent.Document, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencontent.Document{}, mapUpstreamErr(err)
	}
	doc, err := s.appService.UpdateDocument(ctx, string(authHeader), personID, documentIdArg, domain.UpdateDocumentInput{
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
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencontent.Document{}, mapUpstreamErr(err)
	}
	action := domain.TransitionAction(requestArg.Action.Value())
	doc, err := s.appService.TransitionDocument(ctx, string(authHeader), personID, documentIdArg, action)
	if err != nil {
		return gencontent.Document{}, mapErr(err, errCtx{DocumentID: documentIdArg, Action: string(action)})
	}
	return toAPIDocument(doc), nil
}

func (s *Service) GetBlocks(ctx context.Context, authHeader bearertoken.Token, documentIdArg string) (gencontent.BlockList, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencontent.BlockList{}, mapUpstreamErr(err)
	}
	blocks, err := s.appService.GetBlocks(ctx, string(authHeader), personID, documentIdArg)
	if err != nil {
		return gencontent.BlockList{}, mapErr(err, errCtx{DocumentID: documentIdArg})
	}
	return gencontent.BlockList{Blocks: toAPIBlocks(blocks)}, nil
}

func (s *Service) PutBlocks(ctx context.Context, authHeader bearertoken.Token, documentIdArg string, requestArg gencontent.PutBlocksRequest) (gencontent.BlockList, error) {
	personID, err := s.whoami(ctx, authHeader)
	if err != nil {
		return gencontent.BlockList{}, mapUpstreamErr(err)
	}
	inputs := make([]domain.BlockInput, 0, len(requestArg.Blocks))
	for _, b := range requestArg.Blocks {
		data, err := marshalAny(b.Data)
		if err != nil {
			return gencontent.BlockList{}, err
		}
		inputs = append(inputs, domain.BlockInput{BlockTypeCode: b.BlockTypeCode, Position: b.Position, Data: data})
	}
	blocks, err := s.appService.PutBlocks(ctx, string(authHeader), personID, documentIdArg, inputs)
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
