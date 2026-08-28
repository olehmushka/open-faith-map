// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"context"

	gencontent "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/content"
	"github.com/olehmushka/open-faith-map/internal/content/application"
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

func (s *PublicService) ListBlockTypes(ctx context.Context) (gencontent.BlockTypePage, error) {
	blockTypes, err := s.appService.ListBlockTypes(ctx)
	if err != nil {
		return gencontent.BlockTypePage{}, mapErr(err, errCtx{})
	}
	out := make([]gencontent.BlockType, 0, len(blockTypes))
	for _, bt := range blockTypes {
		out = append(out, gencontent.BlockType{
			Id:         bt.ID,
			Code:       bt.Code,
			Name:       bt.Name,
			JsonSchema: unmarshalAny(bt.JSONSchema),
			UiSchema:   unmarshalAny(bt.UISchema),
			Status:     gencontent.New_BlockTypeStatus(gencontent.BlockTypeStatus_Value(bt.Status)),
			SortOrder:  bt.SortOrder,
		})
	}
	return gencontent.BlockTypePage{BlockTypes: out}, nil
}
