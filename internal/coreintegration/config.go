// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package coreintegration

// Config configures OpenFaithMap's service-principal client to go-oikumenea (M1, D-CoreDependency,
// D-ServiceIdentities). No Keycloak: authentication is Google directly (see
// deploy/oikumenea-install.yml's header comment for the full rationale) — a GCP service account
// mints its own Google-signed ID token per call, targeted at Audience, rather than an OAuth2
// client-credentials grant against a self-hosted IdP.
type Config struct {
	// BaseURL is go-oikumenea's base URL, e.g. "https://oikumenea-app:8443" on the compose network
	// (it publishes no host port — D-HeadlessTopology).
	BaseURL string `yaml:"base-url"`
	// CredentialsFile is the path to the GCP service-account key JSON (GOOGLE_APPLICATION_CREDENTIALS
	// convention). Never committed — see .gitignore's var/*.json.
	CredentialsFile string `yaml:"credentials-file"`
	// Audience is the target_audience OpenFaithMap's self-minted Google ID tokens carry. Must match
	// one of the Google issuer's `audiences` in deploy/oikumenea-install.yml.
	Audience string `yaml:"audience"`
	// InsecureSkipVerify accepts go-oikumenea's self-signed local-dev certificate. Never in prod.
	InsecureSkipVerify bool `yaml:"insecure-skip-verify"`
}
