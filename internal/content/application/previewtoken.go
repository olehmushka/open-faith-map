// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// previewTokenTTL is how long a minted preview link stays usable (M14.7's "short-lived signed
// token"). No revocation path exists — a token that leaks is only a problem until it expires — so
// this stays short rather than configurable.
const previewTokenTTL = 20 * time.Minute

// previewTokenPurpose guards against a token minted for some other future use of this same HMAC key
// ever being accepted here by accident.
const previewTokenPurpose = "content-preview"

// mintPreviewToken signs a stateless, site-scoped HS256 token (same library and shape as
// internal/platform/devtoken.Mint, applied to a different problem: no DB row, no revocation, just an
// embedded expiry and a subject internal/content's own preview reads check themselves). A draft is
// content, not a special code path (D-ContentRevisions) — so the token only needs to say "which
// site," never "which document."
func mintPreviewToken(siteID, hmacKey string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":     siteID,
		"purpose": previewTokenPurpose,
		"iat":     now.Unix(),
		"exp":     now.Add(previewTokenTTL).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(hmacKey))
}

// verifyPreviewToken checks signature, expiry, and purpose, returning the site id the token is
// scoped to. Every failure mode — missing, malformed, expired, wrong purpose — collapses to the same
// domain.ErrPreviewTokenInvalid at the call site; this function only distinguishes them internally
// long enough to fail closed.
func verifyPreviewToken(token, hmacKey string) (siteID string, ok bool) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, isHMAC := t.Method.(*jwt.SigningMethodHMAC); !isHMAC {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(hmacKey), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil || !parsed.Valid {
		return "", false
	}
	claims, isMap := parsed.Claims.(jwt.MapClaims)
	if !isMap {
		return "", false
	}
	if purpose, _ := claims["purpose"].(string); purpose != previewTokenPurpose {
		return "", false
	}
	sub, isString := claims["sub"].(string)
	if !isString || sub == "" {
		return "", false
	}
	return sub, true
}
