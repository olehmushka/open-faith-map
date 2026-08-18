// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"testing"
)

func TestGuardSymmetricIssuers(t *testing.T) {
	hs256 := []IssuerConfig{{Issuer: "urn:test:hs256", Type: IssuerHS256}}
	oidcOnly := []IssuerConfig{{Issuer: "https://accounts.google.com", Type: IssuerOIDC}}

	tests := []struct {
		name    string
		issuers []IssuerConfig
		env     string
		wantErr bool
	}{
		{"local allows HS256", hs256, "local", false},
		{"dev allows HS256", hs256, "dev", false},
		{"empty environment refuses HS256", hs256, "", true},
		{"staging refuses HS256", hs256, "staging", true},
		{"prod refuses HS256", hs256, "prod", true},
		{"unrecognized value refuses HS256", hs256, "production", true},
		{"no HS256 issuer configured: any environment passes", oidcOnly, "prod", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GuardSymmetricIssuers(tt.issuers, tt.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("GuardSymmetricIssuers(env=%q) error = %v, wantErr %v", tt.env, err, tt.wantErr)
			}
		})
	}
}

func TestGuardReservedIssuer(t *testing.T) {
	if err := GuardReservedIssuer([]IssuerConfig{{Issuer: ReservedLocalIssuer}}); err == nil {
		t.Error("expected an error when an issuer claims the reserved local issuer string")
	}
	if err := GuardReservedIssuer([]IssuerConfig{{Issuer: "https://accounts.google.com"}}); err != nil {
		t.Errorf("unexpected error for a non-reserved issuer: %v", err)
	}
}

func TestGuardIssuerAudience(t *testing.T) {
	tests := []struct {
		name    string
		issuers []IssuerConfig
		wantErr bool
	}{
		{"oidc issuer with an audience passes", []IssuerConfig{{Issuer: "x", Type: IssuerOIDC, Audiences: []string{"client-1"}}}, false},
		{"oidc issuer with no audience refused", []IssuerConfig{{Issuer: "x", Type: IssuerOIDC}}, true},
		{"defaulted (empty) type treated as oidc, refused with no audience", []IssuerConfig{{Issuer: "x"}}, true},
		{"hs256 issuer exempt even with no audience", []IssuerConfig{{Issuer: "x", Type: IssuerHS256}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GuardIssuerAudience(tt.issuers)
			if (err != nil) != tt.wantErr {
				t.Errorf("GuardIssuerAudience() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAudienceAccepted(t *testing.T) {
	tests := []struct {
		name       string
		tokenAud   []string
		configured []string
		want       bool
	}{
		{"empty configured set accepts anything (hs256-only legality)", []string{"anything"}, nil, true},
		{"exact single match", []string{"client-1"}, []string{"client-1"}, true},
		{"token carries several audiences, one matches", []string{"unrelated", "client-1"}, []string{"client-1"}, true},
		{"issuer accepts several clients, token matches the second", []string{"client-2"}, []string{"client-1", "client-2"}, true},
		{"no intersection rejected", []string{"third-party"}, []string{"client-1"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := audienceAccepted(tt.tokenAud, tt.configured); got != tt.want {
				t.Errorf("audienceAccepted(%v, %v) = %v, want %v", tt.tokenAud, tt.configured, got, tt.want)
			}
		})
	}
}
