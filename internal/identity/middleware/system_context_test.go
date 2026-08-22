// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/olehmushka/open-faith-map/internal/authz"
	"github.com/olehmushka/open-faith-map/internal/identity/domain"
)

// M10.9's unforgeable-SystemContext proof: Handle's very first line
// (authz.StripSystemMarker(r.Context())) must strip any SystemContext marker unconditionally,
// before anything else touches the request — proving the strip is unconditional, not merely that
// the marker type is unexported (already true by construction, and not what this proves).
func TestHandleStripsSystemContextUnconditionally(t *testing.T) {
	const issuer = "urn:test:hs256"
	const hmacKey = "test-hmac-key"

	validator := NewValidator(Config{
		Issuers:   []IssuerConfig{{Issuer: issuer, Type: IssuerHS256, HMACKey: hmacKey}},
		ClockSkew: 5 * time.Second,
	})
	resolver := &fakeResolver{byIssuerSubject: map[[2]string]domain.Resolution{
		{issuer, "sub-1"}: {PersonID: "p1", AccountID: "a1", Email: "a@example.com"},
	}}
	sessions := fakeSessionChecker{bySessionID: map[string]string{"sess-1": "a1"}}
	auth := NewUnbound()
	auth.Bind(validator, resolver, fakePersonDirectory{}, sessions, false)

	claims := jwt.MapClaims{"iss": issuer, "sub": "sub-1", "exp": time.Now().Add(time.Minute).Unix()}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(hmacKey))
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}

	req := newRequestWithAuth("Bearer " + signed)
	req.Header.Set(SessionIDHeader, "sess-1")
	// An attacker (or a bug elsewhere) cannot construct this marker directly — systemMarker/systemKey
	// are unexported to this package's sibling internal/authz — but simulate the one way a system
	// context COULD arrive on a request context: a caller that (wrongly) reused a background
	// SystemContext across requests. Handle must still strip it.
	req = req.WithContext(authz.SystemContext(req.Context()))

	var nextCtx context.Context
	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nextCalled = true
		nextCtx = r.Context()
	})

	rw := httptest.NewRecorder()
	auth.Handle(rw, req, next)

	if !nextCalled {
		t.Fatalf("next was never called — request was rejected (status %d, body %q)", rw.Code, rw.Body.String())
	}

	// authz.MustBeSystemContext panics on a context that is NOT marked as a system context — used
	// here as the concrete, checkable proof the marker is gone (the only exported way to observe
	// isSystemContext's private state from outside internal/authz).
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("SystemContext marker survived Handle — MustBeSystemContext did not panic, meaning ctx is still marked as a system context")
			}
		}()
		authz.MustBeSystemContext(nextCtx)
	}()
}
