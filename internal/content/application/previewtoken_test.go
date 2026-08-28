// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestPreviewTokenRoundTrip(t *testing.T) {
	token, err := mintPreviewToken("site-1", "hmac-key")
	if err != nil {
		t.Fatalf("mintPreviewToken: %v", err)
	}
	siteID, ok := verifyPreviewToken(token, "hmac-key")
	if !ok {
		t.Fatalf("verifyPreviewToken(valid token) ok = false, want true")
	}
	if siteID != "site-1" {
		t.Errorf("verifyPreviewToken(valid token) siteID = %q, want %q", siteID, "site-1")
	}
}

func TestPreviewTokenWrongKey(t *testing.T) {
	token, err := mintPreviewToken("site-1", "hmac-key")
	if err != nil {
		t.Fatalf("mintPreviewToken: %v", err)
	}
	if _, ok := verifyPreviewToken(token, "a-different-key"); ok {
		t.Errorf("verifyPreviewToken(wrong key) ok = true, want false")
	}
}

func TestPreviewTokenExpired(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":     "site-1",
		"purpose": previewTokenPurpose,
		"iat":     time.Now().Add(-2 * previewTokenTTL).Unix(),
		"exp":     time.Now().Add(-previewTokenTTL).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte("hmac-key"))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}
	if _, ok := verifyPreviewToken(signed, "hmac-key"); ok {
		t.Errorf("verifyPreviewToken(expired token) ok = true, want false")
	}
}

func TestPreviewTokenWrongPurpose(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":     "site-1",
		"purpose": "something-else",
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(previewTokenTTL).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte("hmac-key"))
	if err != nil {
		t.Fatalf("sign wrong-purpose token: %v", err)
	}
	if _, ok := verifyPreviewToken(signed, "hmac-key"); ok {
		t.Errorf("verifyPreviewToken(wrong purpose) ok = true, want false")
	}
}

func TestPreviewTokenMalformed(t *testing.T) {
	if _, ok := verifyPreviewToken("not-a-jwt-at-all", "hmac-key"); ok {
		t.Errorf("verifyPreviewToken(malformed) ok = true, want false")
	}
	if _, ok := verifyPreviewToken("", "hmac-key"); ok {
		t.Errorf("verifyPreviewToken(empty) ok = true, want false")
	}
}

// TestPreviewTokenRejectsNoneAlgorithm confirms the alg-confusion attack (a token claiming
// "alg":"none", carrying no real signature) is rejected — WithValidMethods pins acceptance to HS256
// specifically, not just any HMAC-family or even none-signed token.
func TestPreviewTokenRejectsNoneAlgorithm(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":     "site-1",
		"purpose": previewTokenPurpose,
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(previewTokenTTL).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signed, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none-alg token: %v", err)
	}
	if _, ok := verifyPreviewToken(signed, "hmac-key"); ok {
		t.Errorf("verifyPreviewToken(alg=none) ok = true, want false")
	}
}
