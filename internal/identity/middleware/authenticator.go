// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/olehmushka/open-faith-map/internal/identity/domain"
)

// ReservedLocalIssuer is a synthetic issuer string no real IdP may ever present — GuardReservedIssuer
// refuses any operator configuration that claims it, so a value this repo might later use for a
// local convenience token can never be shadowed by a real IdP.
const ReservedLocalIssuer = "urn:openfaithmap:local"

// Resolver maps a verified (issuer, subject) to a PDP subject, and performs the just-in-time
// link-on-match. internal/identity/application.Service satisfies it.
type Resolver interface {
	Resolve(ctx context.Context, issuer, subject string) (domain.Resolution, error)
	LinkOnMatch(ctx context.Context, personID, issuer, subject, email string) (domain.Resolution, error)
	// PersonIDByAccountEmail backs D-JIT's attribute arm: the person behind the single active
	// account carrying this IdP-asserted email, and whether one was found.
	PersonIDByAccountEmail(ctx context.Context, email string) (string, bool, error)
}

// PersonDirectory resolves a token claim value to an existing person (D-JIT: claim -> person.code).
type PersonDirectory interface {
	PersonIDByCode(ctx context.Context, code string) (string, bool, error)
}

// Authenticator is the inbound-token validation middleware, matching wrouter's
// RequestHandlerMiddleware signature exactly. It supports late binding: the composition root
// registers Handle on the server before Start, then Binds the validator + resolver once the DB pool
// and services exist inside the boot InitFunc — all before any request is served.
type Authenticator struct {
	mu    sync.RWMutex
	bound *bound
}

type bound struct {
	validator  *Validator
	resolver   Resolver
	persons    PersonDirectory
	jitEnabled bool
	jitMatch   string // JITMatchCode | JITMatchAccountEmail; read off the validator, not a Bind arg
}

// NewUnbound builds an Authenticator whose validator/resolver are wired later via Bind.
func NewUnbound() *Authenticator { return &Authenticator{} }

// Bind wires the validator, the (issuer, subject) resolver, the person directory (for JIT), and the
// JIT-enabled flag. Called once at boot.
func (a *Authenticator) Bind(validator *Validator, resolver Resolver, persons PersonDirectory, jitEnabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.bound = &bound{
		validator: validator, resolver: resolver, persons: persons, jitEnabled: jitEnabled,
		jitMatch: validator.JITMatch(),
	}
}

func (a *Authenticator) snapshot() *bound {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.bound
}

// MustBeBound reports whether Bind has wired the validator/resolver. The composition root calls it
// at the end of boot so a forgotten Bind fails startup instead of 401-ing every request forever.
func (a *Authenticator) MustBeBound() error {
	if a.snapshot() == nil {
		return errors.New("identity authenticator not bound: call Bind before serving")
	}
	return nil
}

// Handle is the wrouter.RequestHandlerMiddleware. It validates the bearer token, resolves the PDP
// subject, attaches it to the request context, and calls next. Management/diagnostic paths
// (/status, /debug) are passed through unauthenticated — health/readiness probes must stay open.
//
// NOT wired onto the live server yet (M10.2 is additive, matching M10.1's own precedent) — the
// server.WithMiddleware(authenticator.Handle) attachment that would make this gate real traffic is
// M10.6 scope, once the six consumer modules stop depending on go-oikumenea's own Whoami/Authorize
// SDK calls. This method is fully real and unit-tested; it's just not load-bearing yet.
func (a *Authenticator) Handle(rw http.ResponseWriter, r *http.Request, next http.Handler) {
	// Strip any system marker unconditionally, before anything else touches ctx — a system context
	// must never survive from one request into another's context chain (authz.SystemContext's doc).
	r = r.WithContext(authz.StripSystemMarker(r.Context()))

	if isBypassPath(r.URL.Path) {
		next.ServeHTTP(rw, r)
		return
	}
	b := a.snapshot()
	if b == nil {
		unauthorized(rw)
		return
	}
	raw := bearerToken(r)
	if raw == "" {
		unauthorized(rw)
		return
	}

	claims, err := b.validator.Validate(r.Context(), raw)
	if err != nil {
		unauthorized(rw)
		return
	}

	res, _, err := b.resolve(r.Context(), claims)
	if err != nil {
		unauthorized(rw)
		return
	}
	ctx := authz.NewContext(r.Context(), authz.Subject{PersonID: res.PersonID, AccountID: res.AccountID, Email: res.Email})
	next.ServeHTTP(rw, r.WithContext(ctx))
}

// resolve turns verified claims into a PDP subject: first a direct (issuer, subject) lookup; on an
// unknown identity, just-in-time link-on-match (D-JIT) when enabled — match the configured claim to
// an EXISTING person.code (or verified account email) and link; otherwise reject. JIT never creates
// a person. The bool return is jitLinked: true when the subject was linked via JIT this request.
func (b *bound) resolve(ctx context.Context, claims Claims) (domain.Resolution, bool, error) {
	res, err := b.resolver.Resolve(ctx, claims.Issuer, claims.Subject)
	if err == nil {
		return res, false, nil
	}
	if !errors.Is(err, domain.ErrIdentityNotFound) {
		return domain.Resolution{}, false, err
	}
	if !b.jitEnabled || claims.JITValue == "" {
		return domain.Resolution{}, false, errInvalidToken
	}
	personID, ok, err := b.matchPerson(ctx, claims)
	if err != nil {
		return domain.Resolution{}, false, err
	}
	if !ok {
		return domain.Resolution{}, false, errInvalidToken // no match -> reject
	}
	linked, err := b.resolver.LinkOnMatch(ctx, personID, claims.Issuer, claims.Subject, claims.Email)
	if err != nil {
		return domain.Resolution{}, false, err
	}
	return linked, true, nil
}

// matchPerson resolves the JIT claim value to an existing person by the configured match mode
// (D-JIT: "a token claim -> person.code or a designated attribute").
//
//   - code (default): the claim value IS a person.code.
//   - account-email: the claim value is an email, matched against the single active account
//     carrying it. Requires a verified email — an unverified address is an unproven claim, and
//     matching on it would let anyone who can assert someone else's address at the IdP take over
//     that account.
func (b *bound) matchPerson(ctx context.Context, claims Claims) (string, bool, error) {
	if b.jitMatch == JITMatchAccountEmail {
		if !claims.EmailVerified {
			return "", false, nil
		}
		return b.resolver.PersonIDByAccountEmail(ctx, claims.JITValue)
	}
	return b.persons.PersonIDByCode(ctx, claims.JITValue)
}

// bearerToken extracts the token from the Authorization header (case-insensitive "Bearer " scheme).
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

// isBypassPath reports whether a path belongs to the management/diagnostic surface that must remain
// reachable without authentication (readiness/liveness/health, debug diagnostics).
func isBypassPath(path string) bool {
	return strings.HasPrefix(path, "/status") || strings.HasPrefix(path, "/debug")
}

// unauthorized writes a uniform 401 (no detail about which check failed).
func unauthorized(rw http.ResponseWriter) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusUnauthorized)
	_, _ = rw.Write([]byte(`{"errorCode":"CUSTOM_CLIENT","errorName":"Identity:Unauthorized","parameters":{}}`))
}
