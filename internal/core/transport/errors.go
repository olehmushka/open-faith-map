// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"errors"

	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	gencore "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/core"
	"github.com/olehmushka/open-faith-map/internal/core/application"
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
	ApiKeyID     string
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
	case errors.Is(err, directorydomain.ErrUnitHasChildren):
		return gencore.NewUnitHasChildren(c.UnitID)
	case errors.Is(err, application.ErrUnitHasActiveRoleAssignments):
		return gencore.NewUnitHasActiveRoleAssignments(c.UnitID)
	case errors.Is(err, application.ErrUnitHasOrgProfile):
		return gencore.NewUnitHasOrgProfile(c.UnitID)
	case errors.Is(err, application.ErrRootUnitProtected):
		return gencore.NewRootUnitProtected(c.UnitID)
	case errors.Is(err, application.ErrInvalidGrantScope):
		return gencore.NewInvalidGrantScope()
	case errors.Is(err, application.ErrSubtreeGrantRequiresGraph):
		return gencore.NewSubtreeGrantRequiresGraph()
	case errors.Is(err, application.ErrUnitGrantMustNotSpecifyGraph):
		return gencore.NewUnitGrantMustNotSpecifyGraph()
	case errors.Is(err, directorydomain.ErrUnitHasNoCurrentParent):
		return gencore.NewUnitHasNoCurrentParent(c.UnitID)
	case errors.Is(err, directorydomain.ErrUnitMoveConflict):
		return gencore.NewUnitMoveConflict(c.UnitID)
	case errors.Is(err, directorydomain.ErrEdgeCycle):
		return gencore.NewEdgeCycle(c.UnitID)
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
	case errors.Is(err, authzdomain.ErrEmptyPersonIDs):
		return gencore.NewEmptyPersonIdsList()
	case errors.Is(err, identitydomain.ErrSessionNotFound):
		return gencore.NewSessionNotFound(c.SessionID)
	case errors.Is(err, identitydomain.ErrAccountAlreadyExists):
		return gencore.NewAccountAlreadyExists()
	case errors.Is(err, identitydomain.ErrInviteAlreadyAccepted):
		return gencore.NewInviteAlreadyAccepted()
	case errors.Is(err, identitydomain.ErrInviteExpired):
		return gencore.NewInviteExpired()
	case errors.Is(err, identitydomain.ErrInviteNotFound):
		return gencore.NewInviteNotFound()
	case errors.Is(err, identitydomain.ErrCannotMergeSelf):
		return gencore.NewCannotMergeSelf()
	case errors.Is(err, identitydomain.ErrAPIKeyNotFound):
		return gencore.NewApiKeyNotFound(c.ApiKeyID)
	case errors.Is(err, identitydomain.ErrUnknownPermissionCode):
		return gencore.NewUnknownPermissionCode()
	case errors.Is(err, identitydomain.ErrAccountDisabled):
		// No oracle leak to an anonymous invitee holding just a token guess (same reasoning
		// D-AccountStatusEnforcement's own ResolveBySubject already applies): a disabled account
		// behind a real invite reads identically to a nonexistent one.
		return gencore.NewInviteNotFound()
	default:
		return err
	}
}
