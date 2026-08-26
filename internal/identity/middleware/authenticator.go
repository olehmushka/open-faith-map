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
	// ResolveByAPIKey maps a raw, domain.APIKeyTokenPrefix-shaped bearer to the owning person's PDP
	// subject plus the key's stored permission-code allowlist (M11.9) — the "new resolution path
	// alongside ResolveBySubject" the milestone spec calls for. No JIT applies to this path: an API
	// key is already bound to a specific person at creation time.
	ResolveByAPIKey(ctx context.Context, rawToken string) (domain.Resolution, []string, error)
}

// PersonDirectory resolves a token claim value to an existing person (D-JIT: claim -> person.code).
type PersonDirectory interface {
	PersonIDByCode(ctx context.Context, code string) (string, bool, error)
}

// SessionChecker validates an X-Session-Id header value (M11.3, D-SessionTracking): present,
// unrevoked, and touched (last_seen_at bumped, throttled — see adapters.sessionTouchThrottle).
// Returns the session's account id so Handle can cross-check it against the bearer-resolved
// account. internal/identity/application.Service satisfies it.
type SessionChecker interface {
	Touch(ctx context.Context, sessionID string) (accountID string, err error)
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
	sessions   SessionChecker
	jitEnabled bool
	jitMatch   string // JITMatchCode | JITMatchAccountEmail; read off the validator, not a Bind arg
}

// NewUnbound builds an Authenticator whose validator/resolver are wired later via Bind.
func NewUnbound() *Authenticator { return &Authenticator{} }

// Bind wires the validator, the (issuer, subject) resolver, the person directory (for JIT), the
// session checker (M11.3), and the JIT-enabled flag. Called once at boot.
func (a *Authenticator) Bind(validator *Validator, resolver Resolver, persons PersonDirectory, sessions SessionChecker, jitEnabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.bound = &bound{
		validator: validator, resolver: resolver, persons: persons, sessions: sessions, jitEnabled: jitEnabled,
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

	if isBypassPath(r.Method, r.URL.Path) {
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

	// M11.9: an API-key-shaped bearer (domain.APIKeyTokenPrefix, "ofm_") is never a JWT — branch here,
	// before b.validator.Validate ever attempts to parse it as one. This is a different resolution
	// path entirely, not an issuer-based carve-out of the JWT path below (the M11.3 comment on the
	// session-id check two blocks down is about THAT path, not this one): an API key is itself the
	// credential, has no NextAuth session to forward, and so is unconditionally exempt from the
	// X-Session-Id check by construction — it never reaches that block at all. No JIT fallback applies
	// either (JIT is specifically (issuer, subject)-claim matching; an API key is already bound to one
	// person at creation time).
	if strings.HasPrefix(raw, domain.APIKeyTokenPrefix) {
		res, permCodes, err := b.resolver.ResolveByAPIKey(r.Context(), raw)
		if err != nil {
			unauthorized(rw)
			return
		}
		ctx := authz.NewContext(r.Context(), authz.Subject{
			PersonID: res.PersonID, AccountID: res.AccountID, Email: res.Email,
			APIKeyPermissionCodes: permCodes,
		})
		next.ServeHTTP(rw, r.WithContext(ctx))
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

	// M11.3, D-SessionTracking: a valid, unrevoked session id must accompany the bearer on every
	// authenticated request — except registerSession itself (sessionExemptRoutes), which is what
	// CREATES the row a session id would otherwise need to already exist. No issuer-based carve-out:
	// this applies to dev/local-issuer tokens the same as real Google ID tokens (confirmed decision,
	// docs/milestones.md's M11.3 row) — cmd/openfaithmap-api/authorization_matrix_test.go and
	// scripts/mint-local-token both insert a real identity_sessions row for every token they mint.
	var sessionID string
	if !isSessionExemptPath(r.Method, r.URL.Path) {
		sessionID = r.Header.Get(SessionIDHeader)
		if sessionID == "" {
			unauthorized(rw)
			return
		}
		sessAccountID, err := b.sessions.Touch(r.Context(), sessionID)
		if err != nil || sessAccountID != res.AccountID {
			unauthorized(rw)
			return
		}
	}

	ctx := authz.NewContext(r.Context(), authz.Subject{
		PersonID: res.PersonID, AccountID: res.AccountID, Email: res.Email,
		SessionID: sessionID, Issuer: claims.Issuer,
	})
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

// anonymousRoute is one exact (method, path) pair a real anonymous product endpoint uses — not a
// path prefix, because DiscoveryPublicService/DiscoveryService share /discovery/v1 and
// ModerationPublicService/ModerationService share /moderation/v1 (a prefix bypass could not tell
// the anonymous arm from the authenticated one sharing its base path). Sourced directly from each
// contract's own http: line (api/discovery.conjure.yml, api/moderation.conjure.yml) — not
// rediscovered by guessing at runtime behaviour.
type anonymousRoute struct {
	method string
	path   string
}

// anonymousRoutes is the M10.6 resolution of the deliberately-deferred middleware/bypass-list
// blocker: attaching server.WithMiddleware(Handle) needed visibility into all six consumer modules'
// route shapes at once, which only existed once every module was cut over. Every entry here is a
// genuinely anonymous product endpoint (D-AdminSurface: openfaithmap-web holds no session to
// forward) — content's own public arm doesn't need an entry because /content/v1/public already has
// a distinct path prefix (isBypassPath's own prefix check below), the only one of the three
// affected modules where that was true.
var anonymousRoutes = []anonymousRoute{
	{http.MethodGet, "/discovery/v1/search"},
	{http.MethodPost, "/moderation/v1/reports"},
	{http.MethodPost, "/moderation/v1/exclusion-check"},
}

// SessionIDHeader carries the NextAuth-issued session id (M11.3, D-SessionTracking) — a separate
// signal from the Authorization bearer, since the bearer is Google's own signed ID token and cannot
// carry a custom claim NextAuth controls. See web/apps/admin/lib/core.ts's client() for the sender
// side.
const SessionIDHeader = "X-Session-Id"

// sessionExemptRoutes is the one endpoint allowed to skip the session-presence check every other
// authenticated request now goes through: registerSession creates the very identity_sessions row a
// session id would otherwise need to already exist, so requiring one here would make it
// unreachable. Still requires a fully valid bearer — only the NEW session check is skipped, not
// authentication itself (unlike anonymousRoutes/isBypassPath above, which skip both).
var sessionExemptRoutes = []anonymousRoute{
	{http.MethodPost, "/core/v1/sessions"},
}

// isSessionExemptPath reports whether (method, path) is exempt from the per-request session-id
// check (sessionExemptRoutes) — not from authentication itself.
func isSessionExemptPath(method, path string) bool {
	for _, r := range sessionExemptRoutes {
		if r.method == method && r.path == path {
			return true
		}
	}
	return false
}

// isBypassPath reports whether (method, path) belongs to either the management/diagnostic surface
// (readiness/liveness/health, debug diagnostics — a path prefix is safe here, nothing else lives
// under /status or /debug) or a genuinely anonymous product endpoint (method+path exact match, or
// a distinct .../public path prefix — content's own /content/v1/public, and M11.6's
// /core/v1/public, the admin-side counterpart: the invitee genuinely has no session yet, not "this
// app never forwards one," but the mechanism is identical).
func isBypassPath(method, path string) bool {
	if strings.HasPrefix(path, "/status") || strings.HasPrefix(path, "/debug") {
		return true
	}
	if strings.HasPrefix(path, "/content/v1/public") {
		return true
	}
	if strings.HasPrefix(path, "/core/v1/public") {
		return true
	}
	// M13.0's GetSite carries a path parameter (/discovery/v1/sites/{unitId}), so it can't be an
	// anonymousRoutes exact-match entry the way /discovery/v1/search is — a prefix is safe here
	// (unlike a blanket /discovery/v1/ prefix would be) since it's distinct from the sibling
	// DiscoveryService's own /discovery/v1/refresh, the one authenticated endpoint sharing the base
	// path this module's own anonymousRoute doc comment already flags as the reason a base-path
	// prefix bypass can't be used for this pair of services.
	if method == http.MethodGet && strings.HasPrefix(path, "/discovery/v1/sites/") {
		return true
	}
	for _, r := range anonymousRoutes {
		if r.method == method && r.path == path {
			return true
		}
	}
	return false
}

// unauthorized writes a uniform 401 (no detail about which check failed).
func unauthorized(rw http.ResponseWriter) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusUnauthorized)
	_, _ = rw.Write([]byte(`{"errorCode":"CUSTOM_CLIENT","errorName":"Identity:Unauthorized","parameters":{}}`))
}
