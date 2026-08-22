// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application is the audit-log module's business logic: Record (the shared logging helper
// M11.2's ticket calls for) and List (keyset-paginated, filterable reads for the viewer UI).
//
// This is a new module, not folded into internal/core: internal/core/application's own header scopes
// itself as owning "no new domain logic of its own" beyond one gate, and identity_audit_log carries
// real domain logic (pagination/filtering, the actor-resolution contract below) that M11.3 (session
// revocation) needs to reuse without reaching into internal/core, which is a consumer-facing wiring
// layer, not a place other modules should import from.
package application

import (
	"context"
	"encoding/json"

	"github.com/olehmushka/open-faith-map/internal/authz"
	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"

	"github.com/olehmushka/open-faith-map/internal/auditlog/domain"
)

type Store interface {
	InsertEntry(ctx context.Context, actorPersonID, action, targetKind, targetID string, before, after []byte) error
	ListEntries(ctx context.Context, filter domain.Filter, pageSize int, after *domain.PageCursor) ([]domain.Entry, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Record writes one identity_audit_log row for the caller resolved from ctx. It hard-fails with
// authzdomain.ErrPermissionDenied if ctx carries no subject, rather than silently logging an empty
// actor — a missing subject here means a bug in the calling mutation path (every mutating
// super-admin endpoint is reached only through an authenticated, RequireInstanceAdmin-gated request),
// and a log entry with no actor would be worse than no log entry at all.
//
// before/after are marshaled as given — nil is a legitimate value (e.g. before is nil for a create,
// after is nil for a delete/revoke) and marshals to SQL NULL, not the JSON literal "null"
// (adapters.nullableJSON). Callers pass plain Go values (structs or map[string]any), not
// pre-marshaled JSON, mirroring how every other free-form JSON field in this codebase
// (content.Block.Data) is handled at its module boundary.
func (s *Service) Record(ctx context.Context, action, targetKind, targetID string, before, after any) error {
	subject, ok := authz.SubjectFromContext(ctx)
	if !ok || subject.PersonID == "" {
		return authzdomain.ErrPermissionDenied
	}
	beforeJSON, err := marshalOrNil(before)
	if err != nil {
		return err
	}
	afterJSON, err := marshalOrNil(after)
	if err != nil {
		return err
	}
	return s.store.InsertEntry(ctx, subject.PersonID, action, targetKind, targetID, beforeJSON, afterJSON)
}

func marshalOrNil(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

// List answers the audit-log viewer's listAuditLog — pageSize+1 is the caller's responsibility (same
// convention as moderation.Service.ListReports: the transport layer trims the extra row and encodes
// nextPageToken from it).
func (s *Service) List(ctx context.Context, filter domain.Filter, pageSize int, after *domain.PageCursor) ([]domain.Entry, error) {
	return s.store.ListEntries(ctx, filter, pageSize, after)
}
