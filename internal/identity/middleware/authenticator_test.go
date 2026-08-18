// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"errors"
	"net/http"
	"testing"

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

type fakePersonDirectory struct {
	byCode map[string]string
}

func (f fakePersonDirectory) PersonIDByCode(_ context.Context, code string) (string, bool, error) {
	p, ok := f.byCode[code]
	return p, ok, nil
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
