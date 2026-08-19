// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package devtoken

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	identitymiddleware "github.com/olehmushka/open-faith-map/internal/identity/middleware"
)

func TestMint(t *testing.T) {
	const hmacKey = "test-hmac-key"

	tok, err := Mint("test-subject", "test@example.com", time.Minute, hmacKey)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(tok, claims, func(*jwt.Token) (interface{}, error) {
		return []byte(hmacKey), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		t.Fatalf("parse minted token: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("minted token did not validate against its own hmac key")
	}

	iss, _ := claims.GetIssuer()
	if iss != Issuer {
		t.Errorf("iss = %q, want %q", iss, Issuer)
	}
	if iss != identitymiddleware.ReservedLocalIssuer+":dev" {
		t.Errorf("Issuer drifted from identitymiddleware.ReservedLocalIssuer: %q", iss)
	}

	sub, _ := claims.GetSubject()
	if sub != "test-subject" {
		t.Errorf("sub = %q, want %q", sub, "test-subject")
	}
	if email, _ := claims["email"].(string); email != "test@example.com" {
		t.Errorf("email = %q, want %q", email, "test@example.com")
	}
	if verified, _ := claims["email_verified"].(bool); !verified {
		t.Error("email_verified = false, want true")
	}
}

func TestMintWrongKeyFails(t *testing.T) {
	tok, err := Mint("test-subject", "test@example.com", time.Minute, "right-key")
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	_, err = jwt.Parse(tok, func(*jwt.Token) (interface{}, error) {
		return []byte("wrong-key"), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err == nil {
		t.Fatal("expected validation to fail against the wrong HMAC key")
	}
}
