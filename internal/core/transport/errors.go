// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"errors"

	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	gencore "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/core"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
	identitydomain "github.com/olehmushka/open-faith-map/internal/identity/domain"
	religiondomain "github.com/olehmushka/open-faith-map/internal/religion/domain"
)

// errCtx carries whatever safe-args a given call site has on hand — domain sentinels don't carry
// their own, matching internal/content/transport's own mapErr(err, errCtx) shape.
type errCtx struct {
	PersonID     string
	UnitID       string
	TaxonID      string
	AssignmentID string
	SessionID    string
}

// mapErr maps a domain error to this contract's typed Conjure error. Any other error (an
// unexpected store error) passes through unchanged.
func mapErr(err error, c errCtx) error {
	switch {
	case errors.Is(err, identitydomain.ErrPersonNotFound):
		return gencore.NewPersonNotFound(c.PersonID)
	case errors.Is(err, identitydomain.ErrAccountNotFound):
		return gencore.NewAccountNotFound(c.PersonID)
	case errors.Is(err, directorydomain.ErrUnitNotFound):
		return gencore.NewUnitNotFound(c.UnitID)
	case errors.Is(err, religiondomain.ErrTaxonNotFound):
		return gencore.NewTaxonNotFound(c.TaxonID)
	case errors.Is(err, religiondomain.ErrProfileNotFound):
		return gencore.NewOrgProfileNotFound(c.UnitID)
	case errors.Is(err, religiondomain.ErrChildCreationExcluded):
		return gencore.NewChildCreationExcluded(c.UnitID)
	case errors.Is(err, authzdomain.ErrPermissionDenied):
		return gencore.NewForbidden()
	case errors.Is(err, authzdomain.ErrAssignmentNotFound):
		return gencore.NewAssignmentNotFound(c.AssignmentID)
	case errors.Is(err, authzdomain.ErrInstanceAdminGrantNotFound):
		return gencore.NewInstanceAdminGrantNotFound(c.PersonID)
	case errors.Is(err, identitydomain.ErrSessionNotFound):
		return gencore.NewSessionNotFound(c.SessionID)
	default:
		return err
	}
}
