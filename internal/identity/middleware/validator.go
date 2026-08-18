// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package middleware is identity's inbound-token validation seam: it verifies a bearer JWT against
// the configured issuer(s) — OIDC discovery + JWKS for production RS256 (Google, D-GoogleDirect), or
// a symmetric HS256 key for local-dev — maps the verified (issuer, subject) to a PDP subject, and
// attaches it to the request context (internal/authz). This validates an identity issued elsewhere;
// it never stores tokens. Ported in full from
// ../go-oikumenea/internal/identityfederation/middleware/validator.go per D-DirectTokenVerification's
// amendment — this file carries no service-principal/RLS baggage, so it needed no trimming.
package middleware

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
)

// errInvalidToken is the uniform validation failure — callers map every cause (bad signature, wrong
// issuer, expired, unknown identity) to the same Unauthorized, leaking no oracle about which check
// failed.
var errInvalidToken = errors.New("invalid inbound token")

// JIT match modes (D-JIT: "a token claim -> person.code OR a designated attribute"). The code arm is
// the default; the account-email arm lets an operator who knows only a person's address prepare the
// link in advance.
const (
	JITMatchCode         = "code"          // JITClaim's value is a person.code
	JITMatchAccountEmail = "account-email" // JITClaim's value is an identity_accounts.email
)

// IssuerType selects how an issuer's tokens are verified.
const (
	IssuerOIDC  = "oidc"  // production: OIDC discovery + JWKS (RS256/asymmetric) — Google
	IssuerHS256 = "hs256" // local/dev only: symmetric HMAC key from install config; refused at boot elsewhere (GuardSymmetricIssuers)
)

// GuardSymmetricIssuers refuses HS256 (symmetric) issuers outside the local/dev environments. A
// symmetric verification key is a credential-equivalent the service would then hold (anyone with the
// install secret can mint valid tokens for any subject), so it is permitted only where minting test
// tokens is the point. Fail-closed: any environment other than "local"/"dev" rejects, including an
// empty or unknown value. Called once at boot.
func GuardSymmetricIssuers(issuers []IssuerConfig, environment string) error {
	if environment == "local" || environment == "dev" {
		return nil
	}
	for _, ic := range issuers {
		if ic.Type == IssuerHS256 {
			return fmt.Errorf("issuer %q uses symmetric HS256, permitted only in local/dev (environment=%q)", ic.Issuer, environment)
		}
	}
	return nil
}

// GuardReservedIssuer refuses a configuration that claims the synthetic issuer no real IdP may ever
// present. Without this an operator could point a real IdP at that issuer value. Fails closed at
// boot, like GuardSymmetricIssuers.
func GuardReservedIssuer(issuers []IssuerConfig) error {
	for _, ic := range issuers {
		if ic.Issuer == ReservedLocalIssuer {
			return fmt.Errorf("issuer %q is reserved and cannot be configured", ic.Issuer)
		}
	}
	return nil
}

// GuardIssuerAudience refuses an `oidc` issuer that pins no audience. This is not a hygiene rule but
// a fail-closed authentication guard: a public IdP's `iss` is shared by every application registered
// with it (`https://accounts.google.com` is the same string for every Google OAuth client on earth,
// and a Google `sub` is stable per Google account, not per client). With no `aud` check, an ID token
// minted for an unrelated third-party application would carry an `iss`/`sub` this instance accepts
// and would resolve straight to the linked person. Pinning the audience to this deployment's own
// client id(s) is what binds a token to THIS relying party.
//
// It applies to every `oidc` issuer, not just the known-public ones. `hs256` issuers are exempt —
// they are local/dev only (GuardSymmetricIssuers) and their key is already deployment-private. Called
// once at boot, alongside the other guards.
func GuardIssuerAudience(issuers []IssuerConfig) error {
	for _, ic := range issuers {
		if ic.Type == IssuerHS256 {
			continue
		}
		if len(ic.Audiences) == 0 {
			return fmt.Errorf("issuer %q (type oidc) pins no audience: set its audience(s) to this deployment's client id(s), or any token from that issuer — including one minted for an unrelated application — would be accepted", ic.Issuer)
		}
	}
	return nil
}

// IssuerConfig describes one accepted issuer (install config).
type IssuerConfig struct {
	Issuer string // the `iss` value; also the OIDC discovery base URL
	// Audiences are the accepted `aud` values; a token validates when its own audience intersects
	// this set. Empty is legal only for IssuerHS256; GuardIssuerAudience refuses it on an oidc issuer.
	Audiences []string
	Type      string // IssuerOIDC (default) | IssuerHS256
	HMACKey   string // symmetric verification key for IssuerHS256 (secret)
}

// Config is the validator's configuration: the accepted issuers + the JIT mapping.
type Config struct {
	Issuers    []IssuerConfig
	ClockSkew  time.Duration
	JITEnabled bool
	JITClaim   string // token claim whose value maps to the person key (D-JIT link-on-match)
	JITMatch   string // JITMatchCode (default) | JITMatchAccountEmail — WHICH person key JITClaim is matched against
}

// Claims is the minimal verified projection the middleware needs: the federation key
// (issuer, subject), the optional asserted email, and the optional JIT claim value.
type Claims struct {
	Issuer  string
	Subject string
	Email   string
	// EmailVerified is the `email_verified` claim. Load-bearing ONLY for D-JIT's attribute arm:
	// matching an unverified address would let anyone able to assert someone else's email at the
	// IdP claim their account, so that arm requires it to be present AND true (fail-closed).
	EmailVerified bool
	JITValue      string // the configured JIT claim's string value, "" when absent
}

// Validator verifies inbound tokens against the configured issuers. OIDC verifiers are built lazily
// on first use (so a fresh boot does not require the IdP to be reachable) and cached.
type Validator struct {
	cfg      Config
	byIssuer map[string]IssuerConfig

	mu            sync.Mutex
	oidcVerifiers map[string]*oidc.IDTokenVerifier
}

// NewValidator indexes the configured issuers by their `iss` value.
func NewValidator(cfg Config) *Validator {
	idx := make(map[string]IssuerConfig, len(cfg.Issuers))
	for _, ic := range cfg.Issuers {
		idx[ic.Issuer] = ic
	}
	return &Validator{cfg: cfg, byIssuer: idx, oidcVerifiers: map[string]*oidc.IDTokenVerifier{}}
}

// JITMatch reports the configured D-JIT match mode, defaulting to the code arm when unset.
func (v *Validator) JITMatch() string {
	if v.cfg.JITMatch == JITMatchAccountEmail {
		return JITMatchAccountEmail
	}
	return JITMatchCode
}

// Validate verifies a raw bearer token and returns its claims, or errInvalidToken on any failure. It
// routes on the token's (unverified) `iss` to the matching issuer config, then fully verifies.
func (v *Validator) Validate(ctx context.Context, raw string) (Claims, error) {
	iss, err := unverifiedIssuer(raw)
	if err != nil {
		return Claims{}, errInvalidToken
	}
	ic, ok := v.byIssuer[iss]
	if !ok {
		return Claims{}, errInvalidToken
	}
	switch ic.Type {
	case IssuerHS256:
		return v.validateHS256(raw, ic)
	default:
		return v.validateOIDC(ctx, raw, ic)
	}
}

func (v *Validator) validateHS256(raw string, ic IssuerConfig) (Claims, error) {
	claims := jwt.MapClaims{}
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(ic.Issuer),
		jwt.WithLeeway(v.cfg.ClockSkew),
		jwt.WithExpirationRequired(),
	}
	keyFunc := func(_ *jwt.Token) (interface{}, error) { return []byte(ic.HMACKey), nil }
	if _, err := jwt.ParseWithClaims(raw, claims, keyFunc, opts...); err != nil {
		return Claims{}, errInvalidToken
	}
	// Audience is checked here rather than via jwt.WithAudience: the parser option ANDs repeated
	// expectations, whereas an issuer's configured audiences are ALTERNATIVES (any one may match).
	aud, _ := claims.GetAudience()
	if !audienceAccepted(aud, ic.Audiences) {
		return Claims{}, errInvalidToken
	}
	sub, _ := claims.GetSubject()
	return v.project(ic.Issuer, sub, claims), nil
}

// audienceAccepted reports whether a token's audience list intersects the issuer's configured set. An
// empty configured set skips the check — legal only for hs256 issuers, which GuardIssuerAudience
// enforces at boot.
func audienceAccepted(tokenAud []string, configured []string) bool {
	if len(configured) == 0 {
		return true
	}
	for _, want := range configured {
		for _, got := range tokenAud {
			if got == want {
				return true
			}
		}
	}
	return false
}

func (v *Validator) validateOIDC(ctx context.Context, raw string, ic IssuerConfig) (Claims, error) {
	verifier, err := v.oidcVerifier(ctx, ic)
	if err != nil {
		return Claims{}, errInvalidToken
	}
	tok, err := verifier.Verify(ctx, raw)
	if err != nil {
		return Claims{}, errInvalidToken
	}
	// go-oidc checks a single ClientID; an issuer here may accept several (one per client of this
	// deployment), so the audience is verified against the configured set instead — see oidcVerifier.
	if !audienceAccepted(tok.Audience, ic.Audiences) {
		return Claims{}, errInvalidToken
	}
	var all map[string]any
	if err := tok.Claims(&all); err != nil {
		return Claims{}, errInvalidToken
	}
	return v.project(tok.Issuer, tok.Subject, all), nil
}

// oidcVerifier lazily builds and caches the OIDC verifier for an issuer (discovery + JWKS).
func (v *Validator) oidcVerifier(ctx context.Context, ic IssuerConfig) (*oidc.IDTokenVerifier, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if ver, ok := v.oidcVerifiers[ic.Issuer]; ok {
		return ver, nil
	}
	provider, err := oidc.NewProvider(ctx, ic.Issuer)
	if err != nil {
		return nil, err
	}
	// SkipClientIDCheck delegates the audience decision to audienceAccepted, which supports a SET of
	// accepted audiences; go-oidc's own check takes one ClientID. The audience is still verified on
	// every token (validateOIDC), and GuardIssuerAudience guarantees the set is non-empty at boot, so
	// skipping here does not weaken verification.
	ver := provider.Verifier(&oidc.Config{SkipClientIDCheck: true})
	v.oidcVerifiers[ic.Issuer] = ver
	return ver, nil
}

// project extracts the fields the middleware needs from a verified claim set. claims is either a
// jwt.MapClaims (HS256) or a decoded map[string]any (OIDC) — both are map[string]any.
func (v *Validator) project(issuer, subject string, claims map[string]any) Claims {
	out := Claims{Issuer: issuer, Subject: subject}
	if e, ok := claims["email"].(string); ok {
		out.Email = e
	}
	// Some IdPs send email_verified as a JSON bool, others as the string "true"; both are accepted,
	// anything else counts as unverified.
	switch ev := claims["email_verified"].(type) {
	case bool:
		out.EmailVerified = ev
	case string:
		out.EmailVerified = ev == "true"
	}
	if v.cfg.JITClaim != "" {
		if val, ok := claims[v.cfg.JITClaim].(string); ok {
			out.JITValue = val
		}
	}
	return out
}

// unverifiedIssuer reads the `iss` claim WITHOUT verifying the signature — used only to route to the
// right issuer config, after which the token is fully verified. Safe because routing alone grants
// nothing.
func unverifiedIssuer(raw string) (string, error) {
	claims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(raw, claims); err != nil {
		return "", err
	}
	iss, err := claims.GetIssuer()
	if err != nil {
		return "", err
	}
	return iss, nil
}
