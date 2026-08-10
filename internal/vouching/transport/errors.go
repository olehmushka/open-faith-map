// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"errors"

	genvouching "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/vouching"
	"github.com/olehmushka/open-faith-map/internal/vouching/domain"
)

// errCtx carries whatever safe-args a given call site has on hand — domain sentinels don't carry
// their own (mirrors internal/moderation/transport's errCtx pattern).
type errCtx struct {
	GuarantorPersonID string
}

// mapErr maps a domain/store error to this module's typed Conjure error. Any other error (a
// go-oikumenea call failure, an unexpected store error, or the wrapped
// ErrGuarantorRevokeFanoutIncomplete — still a successful revocation, not rejected to the caller)
// passes through unchanged; RevokeGuarantor's transport method handles the fan-out-incomplete case
// itself rather than through mapErr, since it isn't a rejection.
func mapErr(err error, c errCtx) error {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		return genvouching.NewForbidden()
	case errors.Is(err, domain.ErrGuarantorRevoked):
		return genvouching.NewGuarantorRevoked(c.GuarantorPersonID)
	default:
		return err
	}
}
