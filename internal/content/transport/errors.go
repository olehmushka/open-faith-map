// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"errors"

	gencontent "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/content"
	"github.com/olehmushka/open-faith-map/internal/content/domain"
)

// errCtx carries whatever safe-args a given call site has on hand — domain sentinels don't carry
// their own (mirrors internal/registration/transport's mapErr(err, requestID, status) pattern,
// widened for this module's larger error surface).
type errCtx struct {
	SiteID           string
	DocumentID       string
	Kind             string
	ParentDocumentID string
	FromState        string
	Action           string
	BlockTypeCode    string
	RevisionID       string
}

// mapErr maps a domain/store error to this module's typed Conjure error. Any other error (a
// go-oikumenea call failure, an unexpected store error) passes through unchanged.
func mapErr(err error, c errCtx) error {
	var slugTaken *domain.SlugTakenError
	var slugReserved *domain.SlugReservedError
	var blockInvalid *domain.BlockDataInvalidError
	var dupPosition *domain.DuplicateBlockPositionError
	var urlNotAllowed *domain.BlockUrlNotAllowedError
	var themeInvalid *domain.ThemeInvalidError
	var themeContrastFailed *domain.ThemeContrastFailedError
	var navTargetInvalid *domain.NavTargetInvalidError
	var navTargetAmbiguous *domain.NavTargetAmbiguousError
	var dupNavSortOrder *domain.DuplicateNavItemSortOrderError
	var patternNotFound *domain.PatternNotFoundError
	var blockTypeCodeTaken *domain.BlockTypeCodeTakenError
	var translationLocaleTaken *domain.TranslationLocaleTakenError
	var translationGroupNotFound *domain.TranslationGroupNotFoundError

	switch {
	case errors.As(err, &slugTaken):
		return gencontent.NewSlugTaken(slugTaken.Slug, slugTaken.Scope)
	case errors.As(err, &slugReserved):
		return gencontent.NewSlugReserved(slugReserved.Slug)
	case errors.As(err, &blockInvalid):
		return gencontent.NewBlockDataInvalid(blockInvalid.BlockTypeCode, blockInvalid.Position, blockInvalid.Field)
	case errors.As(err, &dupPosition):
		return gencontent.NewDuplicateBlockPosition(dupPosition.Position)
	case errors.As(err, &urlNotAllowed):
		return gencontent.NewBlockUrlNotAllowed(urlNotAllowed.BlockTypeCode, urlNotAllowed.Position, urlNotAllowed.Field)
	case errors.As(err, &themeInvalid):
		return gencontent.NewThemeInvalid(themeInvalid.Field)
	case errors.As(err, &themeContrastFailed):
		return gencontent.NewThemeContrastFailed(themeContrastFailed.Accent, themeContrastFailed.Mode)
	case errors.As(err, &navTargetInvalid):
		return gencontent.NewNavTargetInvalid(navTargetInvalid.TargetDocumentID)
	case errors.As(err, &navTargetAmbiguous):
		return gencontent.NewNavTargetAmbiguous(navTargetAmbiguous.SortOrder)
	case errors.As(err, &dupNavSortOrder):
		return gencontent.NewDuplicateNavItemSortOrder(dupNavSortOrder.SortOrder)
	case errors.As(err, &patternNotFound):
		return gencontent.NewPatternNotFound(patternNotFound.PatternID)
	case errors.As(err, &blockTypeCodeTaken):
		return gencontent.NewBlockTypeCodeTaken(blockTypeCodeTaken.Code)
	case errors.As(err, &translationLocaleTaken):
		return gencontent.NewTranslationLocaleTaken(translationLocaleTaken.TranslationGroupID, translationLocaleTaken.Locale)
	case errors.As(err, &translationGroupNotFound):
		return gencontent.NewTranslationGroupNotFound(translationGroupNotFound.TranslationGroupID)
	case errors.Is(err, domain.ErrSiteNotFound):
		return gencontent.NewSiteNotFound(c.SiteID)
	case errors.Is(err, domain.ErrDocumentNotFound):
		return gencontent.NewDocumentNotFound(c.DocumentID)
	case errors.Is(err, domain.ErrForbidden):
		return gencontent.NewForbidden()
	case errors.Is(err, domain.ErrEventMissingStart):
		return gencontent.NewEventMissingStart()
	case errors.Is(err, domain.ErrParentTooDeep):
		return gencontent.NewParentTooDeep(c.ParentDocumentID)
	case errors.Is(err, domain.ErrInvalidTransition):
		return gencontent.NewInvalidTransition(c.DocumentID, c.FromState, c.Action)
	case errors.Is(err, domain.ErrScheduleMissingPublishAt):
		return gencontent.NewScheduleMissingPublishAt()
	case errors.Is(err, domain.ErrSchedulePublishAtNotFuture):
		return gencontent.NewSchedulePublishAtNotFuture()
	case errors.Is(err, domain.ErrBlockTypeNotFound):
		return gencontent.NewBlockTypeNotFound(c.BlockTypeCode)
	case errors.Is(err, domain.ErrRevisionNotFound):
		return gencontent.NewRevisionNotFound(c.RevisionID)
	case errors.Is(err, domain.ErrPreviewTokenInvalid):
		return gencontent.NewPreviewTokenInvalid()
	default:
		return err
	}
}
