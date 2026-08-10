// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"errors"

	genmoderation "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/moderation"
	"github.com/olehmushka/open-faith-map/internal/moderation/domain"
)

// errCtx carries whatever safe-args a given call site has on hand — domain sentinels don't carry
// their own (mirrors internal/content/transport's errCtx pattern).
type errCtx struct {
	ReportID string
	ActionID string
	AppealID string
	TaxonID  string
}

// mapErr maps a domain/store error to this module's typed Conjure error. Any other error (a
// go-oikumenea call failure, an unexpected store error) passes through unchanged.
func mapErr(err error, c errCtx) error {
	switch {
	case errors.Is(err, domain.ErrReportNotFound):
		return genmoderation.NewReportNotFound(c.ReportID)
	case errors.Is(err, domain.ErrActionNotFound):
		return genmoderation.NewActionNotFound(c.ActionID)
	case errors.Is(err, domain.ErrAppealNotFound):
		return genmoderation.NewAppealNotFound(c.AppealID)
	case errors.Is(err, domain.ErrForbidden):
		return genmoderation.NewForbidden()
	case errors.Is(err, domain.ErrActionNotReversible):
		return genmoderation.NewActionNotReversible(c.ActionID)
	case errors.Is(err, domain.ErrAppealActorConflict):
		return genmoderation.NewAppealActorConflict(c.AppealID)
	case errors.Is(err, domain.ErrTaxonNotFound):
		return genmoderation.NewTaxonNotFound(c.TaxonID)
	case errors.Is(err, domain.ErrDoctrinalReasonNotAllowed):
		return genmoderation.NewDoctrinalReasonNotAllowed()
	case errors.Is(err, domain.ErrInvalidPageToken):
		// Defense in depth — ListReports/ListAppeals already return this directly at decode time
		// (transport/service.go), before any store call, but route it through mapErr too in case a
		// future call path produces it deeper in the stack.
		return genmoderation.NewInvalidPageToken()
	default:
		return err
	}
}
