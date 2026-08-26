// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"errors"

	genreligion "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/religion"
	"github.com/olehmushka/open-faith-map/internal/religion/domain"
)

// mapErr maps a domain error to this module's typed Conjure error. Any other error (an unexpected
// store error) passes through unchanged.
func mapErr(err error) error {
	if errors.Is(err, domain.ErrForbidden) {
		return genreligion.NewForbidden()
	}
	if errors.Is(err, domain.ErrSiteNotFound) {
		return genreligion.NewSiteNotFound()
	}
	return err
}
