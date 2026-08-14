// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"errors"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	gencongregationimport "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/congregationimport"
)

// mapErr maps a domain/store error to this module's typed Conjure error. candidateID fills in the
// safe-args the domain sentinel errors don't carry themselves. Any other error (including a
// go-oikumenea call failure during ApproveCandidate) passes through unchanged — the caller sees
// exactly what go-oikumenea's PDP said, matching every other module's own convention.
func mapErr(err error, candidateID, status string) error {
	switch {
	case errors.Is(err, domain.ErrCandidateNotFound):
		return gencongregationimport.NewCandidateNotFound(candidateID)
	case errors.Is(err, domain.ErrRunNotFound):
		return gencongregationimport.NewRunNotFound(candidateID)
	case errors.Is(err, domain.ErrForbidden):
		return gencongregationimport.NewForbidden()
	case errors.Is(err, domain.ErrNotEditable):
		return gencongregationimport.NewNotEditable(candidateID, status)
	case errors.Is(err, domain.ErrNotApprovable):
		return gencongregationimport.NewNotApprovable(candidateID)
	case errors.Is(err, domain.ErrGeocodeNoMatch):
		return gencongregationimport.NewGeocodeNoMatch(candidateID)
	default:
		return err
	}
}

// mapUpstreamErr wraps a failure resolving the caller's own identity (whoami) — always an upstream
// go-oikumenea/network error, never one of this module's own typed errors.
func mapUpstreamErr(err error) error {
	return err
}

// mapRunErr is mapErr's counterpart for runConnector, which needs sourceCode (not a candidateId/
// status) filled into RunParametersNotSupported's own safe-arg.
func mapRunErr(err error, sourceCode string) error {
	switch {
	case errors.Is(err, domain.ErrRunNotFound):
		return gencongregationimport.NewRunNotFound(sourceCode)
	case errors.Is(err, domain.ErrRunParametersNotSupported):
		return gencongregationimport.NewRunParametersNotSupported(sourceCode)
	default:
		return err
	}
}

// mapAliasErr is mapErr's counterpart for the alias-creation endpoints, which need aliasText (not
// a candidateId/status) filled into their one alias-specific typed error.
func mapAliasErr(err error, aliasText string) error {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		return gencongregationimport.NewForbidden()
	case errors.Is(err, domain.ErrAliasConflict):
		return gencongregationimport.NewAliasConflict(aliasText)
	default:
		return err
	}
}
