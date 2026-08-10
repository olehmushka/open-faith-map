// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"errors"

	gendiscovery "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/discovery"
	"github.com/olehmushka/open-faith-map/internal/discovery/domain"
)

// mapErr maps a domain error to this module's typed Conjure error. Any other error (a
// go-oikumenea call failure, an unexpected store error) passes through unchanged.
func mapErr(err error) error {
	if errors.Is(err, domain.ErrForbidden) {
		return gendiscovery.NewForbidden()
	}
	return err
}
