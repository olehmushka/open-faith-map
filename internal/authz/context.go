// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package authz is the in-process policy decision point (D-InProcessAuthz): the PDP engine lives in
// internal/authz/domain, the Postgres grant store in internal/authz/adapters, and this package is the
// public seam every other module calls through — Service.Require(ctx, action, unitID), the
// request-scoped Subject carrier, and the unforgeable SystemContext escape hatch for the handful of
// background paths that have no human subject.
package authz

import (
	"context"
)

// Subject is the resolved PDP subject attached to an authenticated request by the identity
// middleware (internal/identity/middleware). Unlike go-oikumenea's own authn.Subject, there is no
// machine-subject arm here — D-DirectTokenVerification deletes service principals outright, so every
// subject is a person.
type Subject struct {
	PersonID  string
	AccountID string
	Email     string
	// SessionID is the caller's own identity_sessions row id (M11.3, D-SessionTracking) — set by
	// the identity middleware from the request's X-Session-Id header once validated. Lets a
	// self-scoped session list mark/disable-revoke on the session the caller is presently using
	// without a second header read anywhere downstream.
	SessionID string
	// Issuer is the verified bearer's `iss` claim — set by the identity middleware alongside
	// SessionID. RegisterSession (M11.3) records this on the new identity_sessions row server-side,
	// rather than trusting a client-supplied issuer field on the request.
	Issuer string
}

type ctxKey struct{}

// NewContext returns a copy of ctx carrying the resolved subject. The identity middleware calls this
// once, after mapping a verified token to a person.
func NewContext(ctx context.Context, s Subject) context.Context {
	return context.WithValue(ctx, ctxKey{}, s)
}

// SubjectFromContext returns the subject attached to ctx and whether one was present. Absent means
// the request was not authenticated — Require treats that as no subject and denies; there is no
// implicit "authenticated ⇒ may act" exemption.
func SubjectFromContext(ctx context.Context) (Subject, bool) {
	s, ok := ctx.Value(ctxKey{}).(Subject)
	return s, ok
}

// systemMarker is the unexported type SystemContext stamps onto a context — unforgeable because it
// can only be constructed inside this package.
type systemMarker struct{}

type systemKey struct{}

// SystemContext returns a copy of parent marked as a trusted, subject-less system action (the
// in-process equivalent of upstream's RunAsSystem). Reserved for background paths with no human
// caller — discovery cache refresh, moderation's exclusion check, congregationimport's connector
// loop, country-name resolution, and RunJurisdictionSync's write (D-InProcessAuthz amendment #5).
// Wiring those five call sites onto this is M10.6 scope (consumer cutover); the primitive is built
// now so it exists to wire onto.
//
// The identity middleware strips any system marker from every inbound request context
// unconditionally, so this can never arrive on a real request by forgery — only by a bug in this
// package's own callers. Require panics rather than denies if it ever sees one on what looks like a
// request-scoped context, so such a bug fails loud in dev and test rather than silently degrading to
// a deny.
func SystemContext(parent context.Context) context.Context {
	return context.WithValue(parent, systemKey{}, systemMarker{})
}

// isSystemContext reports whether ctx was marked via SystemContext.
func isSystemContext(ctx context.Context) bool {
	_, ok := ctx.Value(systemKey{}).(systemMarker)
	return ok
}

// MustBeSystemContext panics if ctx was not marked via SystemContext — the inverse of Require's own
// guard, for the rare method that exposes something ONLY a trusted background caller should reach
// (e.g. internal/religion.SearchSitesExact, which returns uncoarsened coordinates the public
// position-oracle fix deliberately withholds). Like Require, this fails loud rather than silently:
// a caller reaching this method with a request-scoped context is a bug in that caller, not a normal
// deny.
func MustBeSystemContext(ctx context.Context) {
	if !isSystemContext(ctx) {
		panic("authz: MustBeSystemContext called with a non-system context — this method is reserved for trusted background callers")
	}
}

// StripSystemMarker returns a copy of ctx with any SystemContext marker removed. The identity
// middleware calls this unconditionally on every inbound request, before anything else touches ctx,
// so a system marker can never survive from one request into another's context chain.
func StripSystemMarker(ctx context.Context) context.Context {
	if !isSystemContext(ctx) {
		return ctx
	}
	return context.WithValue(ctx, systemKey{}, nil)
}
