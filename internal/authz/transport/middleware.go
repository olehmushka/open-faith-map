// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package transport is the wrouter glue for internal/authz's route-group-level enforcement — kept out
// of internal/authz itself (which has zero HTTP/wrouter dependency by design), the same way
// internal/platform/ratelimit is kept out of the modules that use it.
package transport

import (
	"net/http"

	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

// RequireInstanceAdmin returns wrouter middleware that denies any request whose subject is not an
// active instance admin, before the request reaches the generated Conjure handler — the M10.7/
// D-SuperAdminFold "shared, hard-to-misuse enforcer" gate. Attach it once, to the whole
// CoreSuperAdminService route group (wrouter.RouteMiddleware(RequireInstanceAdmin(svc)) passed into
// gencore.RegisterRoutesCoreSuperAdminService), not per-handler — every route registered through that
// call inherits the check structurally, so no future super-admin endpoint can be added without it.
func RequireInstanceAdmin(svc *authz.Service) wrouter.RouteHandlerMiddleware {
	return func(rw http.ResponseWriter, r *http.Request, reqVals wrouter.RequestVals, next wrouter.RouteRequestHandler) {
		if err := svc.RequireInstanceAdmin(r.Context()); err != nil {
			forbidden(rw)
			return
		}
		next(rw, r, reqVals)
	}
}

// forbidden writes a Conjure-shaped 403 — matches internal/identity/middleware's own pre-dispatch
// unauthorized() convention. A real generated Conjure error constructor isn't reachable from
// middleware (it runs before the generated handler's own error-translation path), so the body is
// hand-shaped to the same wire format a generated PERMISSION_DENIED error produces.
func forbidden(rw http.ResponseWriter) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusForbidden)
	_, _ = rw.Write([]byte(`{"errorCode":"PERMISSION_DENIED","errorName":"Authz:InstanceAdminRequired","parameters":{}}`))
}
