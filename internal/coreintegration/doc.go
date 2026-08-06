// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package coreintegration is the bridge to go-oikumenea (D-CoreDependency, D-Facade): the
// generated Go SDK client wiring, service-principal auth, and (not yet built) the
// congregation-provisioning flow. It owns no schema of its own. See docs/modules/core-integration.md
// — read it before any other module doc.
//
// M1 status: the service-principal client (NewServiceClient) is real and proven end-to-end against
// a live go-oikumenea instance — see scripts/bootstrap-service-principal, which registers the
// principal this package authenticates as. The user-token-passthrough path (a congregation admin's
// own bearer, forwarded from openfaithmap-web) is not built yet — that's M2's provisioning flow, and
// needs a real openfaithmap-web session layer (also not built — M1's own "login working" exit
// criterion is intentionally deferred, see docs/milestones.md).
package coreintegration
