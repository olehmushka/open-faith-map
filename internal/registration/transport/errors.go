// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"errors"

	genregistration "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/registration"
	"github.com/olehmushka/open-faith-map/internal/registration/domain"
)

// mapErr maps a domain/store error to this module's typed Conjure error. requestID/status fill in
// the safe-args domain.ErrNotFound/ErrNotPending don't carry themselves. Any other error (including
// a go-oikumenea call failure during Approve) passes through unchanged — the caller sees exactly
// what go-oikumenea's PDP said, per this module's own docs.
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
	default:
		return err
	}
}

// mapUpstreamErr wraps a failure resolving the caller's own identity (whoami) — always an upstream
// go-oikumenea/network error, never one of this module's own typed errors.
func mapUpstreamErr(err error) error {
	return err
}
