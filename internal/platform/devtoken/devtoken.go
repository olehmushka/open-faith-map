// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package devtoken mints local-dev-only HS256 bearer tokens for openfaithmap-api's own synthetic
// issuer (identitymiddleware.ReservedLocalIssuer + ":dev", accepted only when the target server was
// booted with DEV_ISSUER_HMAC_KEY set — GuardSymmetricIssuers refuses it outside local/dev
// regardless). Used by scripts/mint-local-token (a human operator's CLI) and by M10.9's
// authorization-matrix test (cmd/openfaithmap-api/authorization_matrix_test.go) to authenticate as
// an arbitrary test identity without a real browser Google OAuth round trip.
package devtoken

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	identitymiddleware "github.com/olehmushka/open-faith-map/internal/identity/middleware"
)

// Issuer is the `iss` value openfaithmap-api's own identity middleware routes to its HS256 verifier
// (cmd/openfaithmap-api/register_identity.go) once DEV_ISSUER_HMAC_KEY is set — never accepted from
// a real IdP (GuardReservedIssuer refuses any operator config that claims this prefix).
const Issuer = identitymiddleware.ReservedLocalIssuer + ":dev"

// Mint signs a bearer token for Issuer. hmacKey must match the target server's own
// DEV_ISSUER_HMAC_KEY. The token alone does not create an identity — a matching
// identity_external_identities row (issuer=Issuer, subject) must already resolve to a real person,
// either via a JIT link-on-match (IDENTITY_JIT_ENABLED) or a row inserted directly, the same way a
// real IdP's JIT/pre-provisioned identity would; an unresolved subject 401s like any other unknown
// token, by design (internal/identity/middleware.Authenticator.resolve).
func Mint(subject, email string, ttl time.Duration, hmacKey string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":            Issuer,
		"sub":            subject,
		"iat":            now.Unix(),
		"exp":            now.Add(ttl).Unix(),
		"email":          email,
		"email_verified": true,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(hmacKey))
}
