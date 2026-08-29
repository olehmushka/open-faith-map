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

func (s *Service) UpdateSiteChrome(ctx context.Context, authHeader bearertoken.Token, siteIdArg string, requestArg gencontent.UpdateSiteChromeRequest) (gencontent.Site, error) {
	site, err := s.appService.UpdateSiteChrome(ctx, siteIdArg, requestArg.LogoUrl, fromAPISocialLinks(requestArg.SocialLinks))
	if err != nil {
		return gencontent.Site{}, mapErr(err, errCtx{SiteID: siteIdArg})
	}
	return toAPISite(site), nil
}

func (s *Service) CreatePreviewLink(ctx context.Context, authHeader bearertoken.Token, siteIdArg string) (gencontent.PreviewLink, error) {
	token, err := s.appService.CreatePreviewLink(ctx, siteIdArg)
	if err != nil {
		return gencontent.PreviewLink{}, mapErr(err, errCtx{SiteID: siteIdArg})
	}
	return gencontent.PreviewLink{Token: token}, nil
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

func (s *Service) ListRevisions(ctx context.Context, authHeader bearertoken.Token, documentIdArg string) (gencontent.RevisionPage, error) {
	revisions, err := s.appService.ListRevisions(ctx, documentIdArg)
	if err != nil {
		return gencontent.RevisionPage{}, mapErr(err, errCtx{DocumentID: documentIdArg})
	}
	return gencontent.RevisionPage{Revisions: toAPIRevisions(revisions)}, nil
}

func (s *Service) RestoreRevision(ctx context.Context, authHeader bearertoken.Token, documentIdArg string, revisionIdArg string) (gencontent.BlockList, error) {
	blocks, err := s.appService.RestoreRevision(ctx, documentIdArg, revisionIdArg)
	if err != nil {
		return gencontent.BlockList{}, mapErr(err, errCtx{DocumentID: documentIdArg, RevisionID: revisionIdArg})
	}
	return gencontent.BlockList{Blocks: toAPIBlocks(blocks)}, nil
}

func (s *Service) ListNavItems(ctx context.Context, authHeader bearertoken.Token, siteIdArg string) (gencontent.NavItemList, error) {
	items, err := s.appService.ListNavItems(ctx, siteIdArg)
	if err != nil {
		return gencontent.NavItemList{}, mapErr(err, errCtx{SiteID: siteIdArg})
	}
	return gencontent.NavItemList{Items: toAPINavItems(items)}, nil
}

func (s *Service) PutNavItems(ctx context.Context, authHeader bearertoken.Token, siteIdArg string, requestArg gencontent.PutNavItemsRequest) (gencontent.NavItemList, error) {
	inputs := make([]domain.NavItemInput, 0, len(requestArg.Items))
	for _, item := range requestArg.Items {
		inputs = append(inputs, domain.NavItemInput{
			Label: item.Label, TargetDocumentID: item.TargetDocumentId, TargetURL: item.TargetUrl, SortOrder: item.SortOrder,
		})
	}
	items, err := s.appService.PutNavItems(ctx, siteIdArg, inputs)
	if err != nil {
		return gencontent.NavItemList{}, mapErr(err, errCtx{SiteID: siteIdArg})
	}
	return gencontent.NavItemList{Items: toAPINavItems(items)}, nil
}

func toAPINavItems(items []domain.NavItem) []gencontent.NavItem {
	out := make([]gencontent.NavItem, 0, len(items))
	for _, item := range items {
		out = append(out, gencontent.NavItem{
			Id: item.ID, SiteId: item.SiteID, Label: item.Label,
			TargetDocumentId: item.TargetDocumentID, TargetUrl: item.TargetURL, SortOrder: item.SortOrder,
		})
	}
	return out
}

func toAPISite(site domain.Site) gencontent.Site {
	return gencontent.Site{
		Id:                 site.ID,
		CongregationUnitId: site.CongregationUnitRID,
		Slug:               site.Slug,
		Theme:              unmarshalAny(site.Theme),
		LogoUrl:            site.LogoURL,
		SocialLinks:        toAPISocialLinks(site.SocialLinks),
		CreatedAt:          datetime.DateTime(site.CreatedAt),
		UpdatedAt:          datetime.DateTime(site.UpdatedAt),
	}
}

func toAPISocialLinks(l domain.SocialLinks) gencontent.SocialLinks {
	return gencontent.SocialLinks{
		Facebook: l.Facebook, Instagram: l.Instagram, Youtube: l.YouTube, Twitter: l.Twitter, Website: l.Website,
	}
}

func fromAPISocialLinks(l gencontent.SocialLinks) domain.SocialLinks {
	return domain.SocialLinks{
		Facebook: l.Facebook, Instagram: l.Instagram, YouTube: l.Youtube, Twitter: l.Twitter, Website: l.Website,
	}
}

func toAPISiteChrome(c domain.SiteChrome) gencontent.SiteChrome {
	schedules := make([]gencontent.ServiceSchedule, len(c.Schedules))
	for i, sch := range c.Schedules {
		schedules[i] = gencontent.ServiceSchedule{
			DayOfWeek: sch.DayOfWeek, Rrule: sch.RRule, StartTime: sch.StartTime, EndTime: sch.EndTime,
			Timezone: sch.Timezone, Language: sch.Language, Mode: sch.Mode, MeetingUrl: sch.MeetingURL, Description: sch.Description,
		}
	}
	return gencontent.SiteChrome{
		CongregationName: c.CongregationName,
		Address:          c.Address,
		LogoUrl:          c.LogoURL,
		SocialLinks:      toAPISocialLinks(c.SocialLinks),
		Schedules:        schedules,
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

func toAPIRevisions(revisions []domain.DocumentRevision) []gencontent.DocumentRevision {
	out := make([]gencontent.DocumentRevision, 0, len(revisions))
	for _, r := range revisions {
		out = append(out, gencontent.DocumentRevision{
			RevisionId:     r.ID,
			RevisionNo:     r.RevisionNo,
			CreatedAt:      datetime.DateTime(r.CreatedAt),
			AuthorPersonId: r.AuthorPersonID,
			Label:          r.Label,
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
