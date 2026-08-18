// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"errors"

	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	genregistration "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/registration"
	"github.com/olehmushka/open-faith-map/internal/registration/domain"
)

// mapErr maps a domain/store error to this module's typed Conjure error. requestID/status fill in
// the safe-args domain.ErrNotFound/ErrNotPending don't carry themselves. Any other error — a write
// failure inside Approve — passes through unchanged, the same "surface the real failure" default
// this module has always had (pre-cutover that meant go-oikumenea's own PDP error; post-cutover it
// means internal/authz's).
func mapErr(err error, requestID, status string) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return genregistration.NewRequestNotFound(requestID)
	case errors.Is(err, domain.ErrNotPending):
		return genregistration.NewRequestNotPending(requestID, status)
	case errors.Is(err, domain.ErrExcluded):
		return genregistration.NewTaxonExcluded("")
	case errors.Is(err, domain.ErrTaxonNotFound):
		return genregistration.NewTaxonNotFound("")
	case errors.Is(err, domain.ErrNotApproved):
		return genregistration.NewRequestNotApproved(requestID, status)
	case errors.Is(err, authzdomain.ErrPermissionDenied):
		return genregistration.NewForbidden()
	default:
		return err
	}
}
