// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

// M15 (DS-OFM-15): action/target constants for this module's internal/auditlog.Record calls.
// Own private const block, same convention internal/core/application already established — the
// identity_audit_log action/target_kind columns are free text with no DB CHECK, so there is no
// shared enum to import.
const (
	auditActionCreateSite       = "CREATE_SITE"
	auditActionUpdateSiteTheme  = "UPDATE_SITE_THEME"
	auditActionUpdateSiteChrome = "UPDATE_SITE_CHROME"
	auditActionPutNavItems      = "PUT_NAV_ITEMS"
	auditActionCreateBlockType  = "CREATE_BLOCK_TYPE"
	auditActionUpdateBlockType  = "UPDATE_BLOCK_TYPE"
	auditActionCreatePattern    = "CREATE_PATTERN"
	auditActionUpdatePattern    = "UPDATE_PATTERN"
	auditActionDeletePattern    = "DELETE_PATTERN"

	auditTargetSite      = "SITE"
	auditTargetNavItems  = "NAV_ITEMS"
	auditTargetBlockType = "BLOCK_TYPE"
	auditTargetPattern   = "PATTERN"
)
