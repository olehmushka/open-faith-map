// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"context"
	"strings"
	"time"

	gencontent "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/content"
	"github.com/olehmushka/open-faith-map/internal/content/application"
	"github.com/olehmushka/open-faith-map/internal/content/domain"
)

// PublicService implements the generated ContentPublicService — genuinely anonymous, no
// bearertoken.Token parameter anywhere (confirmed by the generated interface: this service
// declares no default-auth in api/content.conjure.yml, so conjure-go emits no auth arg at all).
// Always filters to published/unlisted; never discloses draft documents or their blocks.
type PublicService struct {
	appService *application.Service
}

func NewPublicService(appService *application.Service) *PublicService {
	return &PublicService{appService: appService}
}

var _ gencontent.ContentPublicService = (*PublicService)(nil)

func (s *PublicService) GetSite(ctx context.Context, congregationUnitIdArg string) (gencontent.Site, error) {
	site, err := s.appService.GetSite(ctx, congregationUnitIdArg)
	if err != nil {
		return gencontent.Site{}, mapErr(err, errCtx{SiteID: congregationUnitIdArg})
	}
	return toAPISite(site), nil
}

// GetSiteBySlug is what the tenant-subdomain proxy resolves a Host header's slug through (M14.9).
// Same SiteNotFound-reuse precedent as GetSite: the safe-arg slot is whatever the lookup key
// actually was, not necessarily content_sites.id.
func (s *PublicService) GetSiteBySlug(ctx context.Context, slugArg string) (gencontent.Site, error) {
	site, err := s.appService.GetSiteBySlug(ctx, slugArg)
	if err != nil {
		return gencontent.Site{}, mapErr(err, errCtx{SiteID: slugArg})
	}
	return toAPISite(site), nil
}

// GetSiteChrome is the tenant layout's one call for header/footer data (M14.11) — logoUrl/
// socialLinks from content_sites, congregationName/address/schedules composed live from religion.
func (s *PublicService) GetSiteChrome(ctx context.Context, siteIdArg string) (gencontent.SiteChrome, error) {
	chrome, err := s.appService.GetSiteChrome(ctx, siteIdArg)
	if err != nil {
		return gencontent.SiteChrome{}, mapErr(err, errCtx{SiteID: siteIdArg})
	}
	return toAPISiteChrome(chrome), nil
}

func (s *PublicService) ListPublicDocuments(ctx context.Context, siteIdArg string, kindArg, localeArg *string) (gencontent.DocumentPage, error) {
	docs, err := s.appService.ListPublicDocuments(ctx, siteIdArg, kindArg, localeArg)
	if err != nil {
		return gencontent.DocumentPage{}, mapErr(err, errCtx{SiteID: siteIdArg})
	}
	return gencontent.DocumentPage{Documents: toAPIDocuments(docs)}, nil
}

func (s *PublicService) GetPublicBlocks(ctx context.Context, documentIdArg string) (gencontent.BlockList, error) {
	blocks, err := s.appService.GetPublicBlocks(ctx, documentIdArg)
	if err != nil {
		return gencontent.BlockList{}, mapErr(err, errCtx{DocumentID: documentIdArg})
	}
	return gencontent.BlockList{Blocks: toAPIBlocks(blocks)}, nil
}

// ListPreviewDocuments and GetPreviewBlocks (M14.7) are this service's one deliberate exception to
// "always published/unlisted only" — gated by a site-scoped token instead of a session, since this
// service's caller never holds one. See ContentPublicService's own doc comment above.
func (s *PublicService) ListPreviewDocuments(ctx context.Context, siteIdArg string, tokenArg string, kindArg, localeArg *string) (gencontent.DocumentPage, error) {
	docs, err := s.appService.ListPreviewDocuments(ctx, siteIdArg, tokenArg, kindArg, localeArg)
	if err != nil {
		return gencontent.DocumentPage{}, mapErr(err, errCtx{SiteID: siteIdArg})
	}
	return gencontent.DocumentPage{Documents: toAPIDocuments(docs)}, nil
}

func (s *PublicService) GetPreviewBlocks(ctx context.Context, documentIdArg string, tokenArg string) (gencontent.BlockList, error) {
	blocks, err := s.appService.GetPreviewBlocks(ctx, documentIdArg, tokenArg)
	if err != nil {
		return gencontent.BlockList{}, mapErr(err, errCtx{DocumentID: documentIdArg})
	}
	return gencontent.BlockList{Blocks: toAPIBlocks(blocks)}, nil
}

// ListPublicNavItems returns the site's nav menu with every target already resolved to a
// ready-to-render href (see PublicNavItem's own docs in api/content.conjure.yml).
func (s *PublicService) ListPublicNavItems(ctx context.Context, siteIdArg string) (gencontent.PublicNavItemList, error) {
	items, err := s.appService.ListPublicNavItems(ctx, siteIdArg)
	if err != nil {
		return gencontent.PublicNavItemList{}, mapErr(err, errCtx{SiteID: siteIdArg})
	}
	out := make([]gencontent.PublicNavItem, 0, len(items))
	for _, item := range items {
		out = append(out, gencontent.PublicNavItem{Label: item.Label, Href: item.Href, External: item.External})
	}
	return gencontent.PublicNavItemList{Items: out}, nil
}

// GetPublicDocumentByPath splits the slash-joined path query param into ordered slug segments
// (filtering empty segments, so a leading/trailing "/" is tolerated) and resolves the leaf PAGE
// document plus its real ancestor chain — see the endpoint's own docs in api/content.conjure.yml.
func (s *PublicService) GetPublicDocumentByPath(ctx context.Context, siteIdArg string, localeArg string, pathArg string) (gencontent.DocumentWithAncestors, error) {
	var segments []string
	for _, seg := range strings.Split(pathArg, "/") {
		if seg != "" {
			segments = append(segments, seg)
		}
	}
	doc, ancestors, translations, err := s.appService.GetPublicDocumentByPath(ctx, siteIdArg, localeArg, segments)
	if err != nil {
		return gencontent.DocumentWithAncestors{}, mapErr(err, errCtx{SiteID: siteIdArg})
	}
	outTranslations := make([]gencontent.DocumentTranslation, 0, len(translations))
	for _, t := range translations {
		outTranslations = append(outTranslations, gencontent.DocumentTranslation{Locale: t.Locale, Href: t.Href})
	}
	return gencontent.DocumentWithAncestors{Document: toAPIDocument(doc), Ancestors: toAPIDocuments(ancestors), Translations: outTranslations}, nil
}

func (s *PublicService) ListBlockTypes(ctx context.Context) (gencontent.BlockTypePage, error) {
	blockTypes, err := s.appService.ListBlockTypes(ctx)
	if err != nil {
		return gencontent.BlockTypePage{}, mapErr(err, errCtx{})
	}
	return gencontent.BlockTypePage{BlockTypes: toAPIBlockTypes(blockTypes)}, nil
}

// ListPatterns is not sensitive data (same reasoning ListBlockTypes above already uses for having
// no auth) — every pattern, in sortOrder. The admin editor's insert-a-pattern UI calls this exact
// endpoint to fetch blocks to copy client-side (M14.13, D-SitePatterns).
func (s *PublicService) ListPatterns(ctx context.Context) (gencontent.PatternPage, error) {
	patterns, err := s.appService.ListPatterns(ctx)
	if err != nil {
		return gencontent.PatternPage{}, mapErr(err, errCtx{})
	}
	return gencontent.PatternPage{Patterns: toAPIPatterns(patterns)}, nil
}

// SubmitContactForm is genuinely anonymous (no bearertoken.Token arg at all — this service
// declares no default-auth). Honeypot/too-fast submissions are handled entirely in
// application.Service.SubmitContactForm, which returns nil for both exactly like a real insert —
// this method has no way to tell the two apart, by design (D-InAppInbox: "an error teaches the
// bot").
func (s *PublicService) SubmitContactForm(ctx context.Context, siteIdArg string, requestArg gencontent.SubmitContactFormRequest) error {
	honeypot := ""
	if requestArg.Honeypot != nil {
		honeypot = *requestArg.Honeypot
	}
	err := s.appService.SubmitContactForm(ctx, domain.SubmitContactFormInput{
		SiteID:         siteIdArg,
		Name:           requestArg.Name,
		Email:          requestArg.Email,
		Message:        requestArg.Message,
		Honeypot:       honeypot,
		FormRenderedAt: time.Time(requestArg.FormRenderedAt),
	})
	if err != nil {
		return mapErr(err, errCtx{SiteID: siteIdArg})
	}
	return nil
}
