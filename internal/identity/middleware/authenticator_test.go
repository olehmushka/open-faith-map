// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/olehmushka/open-faith-map/internal/identity/domain"
)

func newRequestWithAuth(authHeader string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", authHeader)
	return req
}

type fakeResolver struct {
	byIssuerSubject map[[2]string]domain.Resolution
	byEmail         map[string]string // email -> personID
	linked          []domain.ExternalIdentity
	// byAPIKeyToken maps a raw API-key token to what it resolves to (M11.9). Absent = unknown/revoked.
	byAPIKeyToken map[string]apiKeyResolution
}

type apiKeyResolution struct {
	res             domain.Resolution
	permissionCodes []string
}

func (f *fakeResolver) Resolve(_ context.Context, issuer, subject string) (domain.Resolution, error) {
	if r, ok := f.byIssuerSubject[[2]string{issuer, subject}]; ok {
		return r, nil
	}
	return domain.Resolution{}, domain.ErrIdentityNotFound
}

func (f *fakeResolver) LinkOnMatch(_ context.Context, personID, issuer, subject, email string) (domain.Resolution, error) {
	f.linked = append(f.linked, domain.ExternalIdentity{AccountID: personID, Issuer: issuer, Subject: subject})
	res := domain.Resolution{PersonID: personID, AccountID: "acct-" + personID, Email: email}
	if f.byIssuerSubject == nil {
		f.byIssuerSubject = map[[2]string]domain.Resolution{}
	}
	f.byIssuerSubject[[2]string{issuer, subject}] = res
	return res, nil
}

func (f *fakeResolver) PersonIDByAccountEmail(_ context.Context, email string) (string, bool, error) {
	p, ok := f.byEmail[email]
	return p, ok, nil
}

func (f *fakeResolver) ResolveByAPIKey(_ context.Context, rawToken string) (domain.Resolution, []string, error) {
	r, ok := f.byAPIKeyToken[rawToken]
	if !ok {
		return domain.Resolution{}, nil, domain.ErrIdentityNotFound
	}
	return r.res, r.permissionCodes, nil
}

type fakePersonDirectory struct {
	byCode map[string]string
}

func (f fakePersonDirectory) PersonIDByCode(_ context.Context, code string) (string, bool, error) {
	p, ok := f.byCode[code]
	return p, ok, nil
}

// fakeSessionChecker is the M11.3 SessionChecker test double: bySessionID maps a session id to the
// account it belongs to; ok=false (session missing from the map) simulates domain.ErrSessionNotFound.
// touchCalls counts invocations — M11.9's API-key path asserts this stays zero, proving the
// X-Session-Id check is skipped by construction rather than merely tolerated when absent.
type fakeSessionChecker struct {
	bySessionID map[string]string
	touchCalls  int
}

func (f *fakeSessionChecker) Touch(_ context.Context, sessionID string) (string, error) {
	f.touchCalls++
	accountID, ok := f.bySessionID[sessionID]
	if !ok {
		return "", domain.ErrSessionNotFound
	}
	return accountID, nil
}

func TestResolveDirectMatch(t *testing.T) {
	resolver := &fakeResolver{byIssuerSubject: map[[2]string]domain.Resolution{
		{"iss", "sub"}: {PersonID: "p1", AccountID: "a1"},
	}}
	b := &bound{resolver: resolver, jitEnabled: false}
	res, jitLinked, err := b.resolve(context.Background(), Claims{Issuer: "iss", Subject: "sub"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if jitLinked {
		t.Error("direct match should not report jitLinked")
	}
	if res.PersonID != "p1" {
		t.Errorf("PersonID = %q, want p1", res.PersonID)
	}
}

func TestResolveJITDisabledRejectsUnknownIdentity(t *testing.T) {
	b := &bound{resolver: &fakeResolver{}, jitEnabled: false}
	_, _, err := b.resolve(context.Background(), Claims{Issuer: "iss", Subject: "unknown", JITValue: "code-1"})
	if !errors.Is(err, errInvalidToken) {
		t.Errorf("err = %v, want errInvalidToken", err)
	}
}

func TestResolveJITCodeMatchLinksAndNeverCreatesAPerson(t *testing.T) {
	resolver := &fakeResolver{}
	persons := fakePersonDirectory{byCode: map[string]string{"admin-code": "p1"}}
	b := &bound{resolver: resolver, persons: persons, jitEnabled: true, jitMatch: JITMatchCode}

	res, jitLinked, err := b.resolve(context.Background(), Claims{Issuer: "iss", Subject: "sub", JITValue: "admin-code"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !jitLinked {
		t.Error("expected jitLinked=true on a JIT match")
	}
	if res.PersonID != "p1" {
		t.Errorf("PersonID = %q, want p1 (the EXISTING person matched by code)", res.PersonID)
	}
	if len(resolver.linked) != 1 {
		t.Fatalf("expected exactly one LinkOnMatch call, got %d", len(resolver.linked))
	}
}

func TestResolveJITCodeNoMatchRejectsRatherThanCreating(t *testing.T) {
	b := &bound{resolver: &fakeResolver{}, persons: fakePersonDirectory{}, jitEnabled: true, jitMatch: JITMatchCode}
	_, _, err := b.resolve(context.Background(), Claims{Issuer: "iss", Subject: "sub", JITValue: "no-such-code"})
	if !errors.Is(err, errInvalidToken) {
		t.Errorf("err = %v, want errInvalidToken (JIT never creates a person on no match)", err)
	}
}

func TestMatchPersonAccountEmailRejectsUnverifiedEmail(t *testing.T) {
	resolver := &fakeResolver{byEmail: map[string]string{"admin@example.org": "p1"}}
	b := &bound{resolver: resolver, jitMatch: JITMatchAccountEmail}

	_, ok, err := b.matchPerson(context.Background(), Claims{JITValue: "admin@example.org", EmailVerified: false})
	if err != nil {
		t.Fatalf("matchPerson: %v", err)
	}
	if ok {
		t.Error("expected no match for an unverified email, even though it exists in the resolver")
	}
}

func TestMatchPersonAccountEmailAcceptsVerifiedEmail(t *testing.T) {
	resolver := &fakeResolver{byEmail: map[string]string{"admin@example.org": "p1"}}
	b := &bound{resolver: resolver, jitMatch: JITMatchAccountEmail}

	personID, ok, err := b.matchPerson(context.Background(), Claims{JITValue: "admin@example.org", EmailVerified: true})
	if err != nil {
		t.Fatalf("matchPerson: %v", err)
	}
	if !ok || personID != "p1" {
		t.Errorf("matchPerson = (%q, %v), want (p1, true)", personID, ok)
	}
}

func TestBearerToken(t *testing.T) {
	// header parsing is exercised indirectly via Handle in an integration setting; unit-test the
	// pure helper directly for the exact case-insensitive-prefix behavior.
	req := newRequestWithAuth("Bearer abc123")
	if got := bearerToken(req); got != "abc123" {
		t.Errorf("bearerToken = %q, want abc123", got)
	}
	req = newRequestWithAuth("bearer abc123")
	if got := bearerToken(req); got != "abc123" {
		t.Errorf("bearerToken (lowercase scheme) = %q, want abc123", got)
	}
	req = newRequestWithAuth("Basic xyz")
	if got := bearerToken(req); got != "" {
		t.Errorf("bearerToken (wrong scheme) = %q, want empty", got)
	}
}

func TestIsBypassPath(t *testing.T) {
	for _, p := range []string{"/status/liveness", "/debug/pprof"} {
		if !isBypassPath(http.MethodGet, p) {
			t.Errorf("isBypassPath(GET, %q) = false, want true", p)
		}
	}
	if isBypassPath(http.MethodGet, "/api/registration") {
		t.Error("isBypassPath should not treat an app route as a bypass path")
	}

	// M10.6: the extended, method+path-exact anonymous allowlist — resolves the deferred
	// middleware/bypass-list blocker now that all six consumer modules' route shapes are known.
	for _, tt := range []struct {
		method, path string
		want         bool
	}{
		{http.MethodGet, "/discovery/v1/search", true},
		{http.MethodPost, "/discovery/v1/refresh", false}, // shares a base path with search, but is header-authed
		{http.MethodPost, "/moderation/v1/reports", true},
		{http.MethodGet, "/moderation/v1/reports", false}, // ModerationService.listReports shares the PATH, not the method
		{http.MethodPost, "/moderation/v1/exclusion-check", true},
		{http.MethodPost, "/moderation/v1/reports/abc/actions", false},
		{http.MethodGet, "/content/v1/public/sites/abc", true},
		{http.MethodPost, "/content/v1/sites", false},
	} {
		if got := isBypassPath(tt.method, tt.path); got != tt.want {
			t.Errorf("isBypassPath(%s, %q) = %v, want %v", tt.method, tt.path, got, tt.want)
		}
	}
}

// -------------------------------------------------------------------- M11.9: API-key Handle path

// newRejectAllValidator returns a Validator with no configured issuers, so any JWT-parsing attempt
// against it fails. Used to prove the API-key branch never reaches b.validator.Validate at all — if
// it did, a request would 401 against this validator regardless of the resolver's own state.
func newRejectAllValidator() *Validator {
	return NewValidator(Config{})
}

func TestHandleAPIKeyRoutesBeforeJWTValidationAndSkipsSessionCheck(t *testing.T) {
	const rawKey = domain.APIKeyTokenPrefix + "test-key-1"
	resolver := &fakeResolver{byAPIKeyToken: map[string]apiKeyResolution{
		rawKey: {res: domain.Resolution{PersonID: "p1", AccountID: "a1", Email: "a@example.com"}, permissionCodes: []string{"person.read"}},
	}}
	sessions := &fakeSessionChecker{}
	auth := NewUnbound()
	auth.Bind(newRejectAllValidator(), resolver, fakePersonDirectory{}, sessions, false)

	req := newRequestWithAuth("Bearer " + rawKey)
	// Deliberately no X-Session-Id header — an API-key request has no NextAuth session to forward.

	var nextCtx context.Context
	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nextCalled = true
		nextCtx = r.Context()
	})
	rw := httptest.NewRecorder()
	auth.Handle(rw, req, next)

	if !nextCalled {
		t.Fatalf("next was never called — request was rejected (status %d, body %q); the API-key branch must bypass JWT validation entirely", rw.Code, rw.Body.String())
	}
	if sessions.touchCalls != 0 {
		t.Errorf("SessionChecker.Touch was called %d times, want 0 — an API-key request must skip the session check by construction", sessions.touchCalls)
	}
	subject, ok := authz.SubjectFromContext(nextCtx)
	if !ok {
		t.Fatal("no authz.Subject attached to the request context")
	}
	if subject.PersonID != "p1" || subject.AccountID != "a1" {
		t.Errorf("subject = %+v, want PersonID=p1 AccountID=a1", subject)
	}
	if len(subject.APIKeyPermissionCodes) != 1 || subject.APIKeyPermissionCodes[0] != "person.read" {
		t.Errorf("subject.APIKeyPermissionCodes = %v, want [person.read]", subject.APIKeyPermissionCodes)
	}
	if subject.SessionID != "" {
		t.Errorf("subject.SessionID = %q, want empty — no NextAuth session backs an API-key request", subject.SessionID)
	}
}

func TestHandleAPIKeyUnknownTokenRejects(t *testing.T) {
	resolver := &fakeResolver{}
	auth := NewUnbound()
	auth.Bind(newRejectAllValidator(), resolver, fakePersonDirectory{}, &fakeSessionChecker{}, false)

	req := newRequestWithAuth("Bearer " + domain.APIKeyTokenPrefix + "unknown")
	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { nextCalled = true })
	rw := httptest.NewRecorder()
	auth.Handle(rw, req, next)

	if nextCalled {
		t.Error("next was called for an unknown API key token, want rejected")
	}
	if rw.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rw.Code, http.StatusUnauthorized)
	}
}
