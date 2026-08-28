// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"strings"
)

// reservedSlugs is D-TenantSubdomains' reserved-subdomain blocklist (M14.9): content_sites.slug
// becomes a hostname component (<slug>.<apex>) as of this milestone, so these names must be
// permanently unclaimable regardless of who's asking. Checked here, server-side, in Go — never
// only in the admin form's client-side format pattern (web/apps/admin's slug input only validates
// shape, not this list). A single source of truth, not duplicated into TypeScript.
var reservedSlugs = map[string]struct{}{
	// Named explicitly by the milestone/D-TenantSubdomains.
	"admin": {}, "api": {}, "auth": {}, "login": {}, "www": {}, "app": {},
	"mail": {}, "static": {}, "support": {}, "billing": {}, "help": {}, "status": {},

	// This repo's own other deployed surfaces.
	"admin-api": {}, "docs": {},

	// Mail/DNS infrastructure hygiene — no congregation can ever collide with real MX/DNS-role
	// hostnames, even though this platform hosts no mail today.
	"smtp": {}, "imap": {}, "pop": {}, "ftp": {}, "ns1": {}, "ns2": {}, "mx": {}, "autodiscover": {},

	// Auth/security-adjacent names, reserved defensively against phishing-lookalike subdomains.
	"sso": {}, "oauth": {}, "callback": {}, "webhook": {}, "webhooks": {},

	// Generic ops/infra names withheld regardless of current plans.
	"cdn": {}, "assets": {}, "media": {}, "cache": {}, "blog": {},

	// Environment/lifecycle names — must never be claimable, since these are exactly what a future
	// deploy script or CI job might stand up as a real internal environment.
	"dev": {}, "staging": {}, "test": {}, "sandbox": {}, "demo": {}, "beta": {}, "preview": {},

	// Defensive nonsense guards.
	"root": {}, "localhost": {}, "null": {}, "undefined": {},
}

// isReservedSlug reports whether slug (case-insensitively) is on D-TenantSubdomains' blocklist.
func isReservedSlug(slug string) bool {
	_, ok := reservedSlugs[strings.ToLower(slug)]
	return ok
}
