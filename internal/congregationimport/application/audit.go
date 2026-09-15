// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

// M15 (DS-OFM-15): action/target constants for this module's internal/auditlog.Record calls.
//
// auditActionCreateChildOrg deliberately uses the SAME literal string
// internal/registration/application also uses for its own private constant of the same name — both
// modules call the identical religion.CreateChildOrg primitive to create a congregation-tier unit,
// so one shared action-name vocabulary makes action = 'CREATE_CHILD_ORG' a meaningful cross-module
// filter in the audit viewer, even though each module owns its own const block.
const (
	auditActionCreateChildOrg          = "CREATE_CHILD_ORG"
	auditActionCreateTaxonAlias        = "CREATE_TAXON_ALIAS"
	auditActionCreateJurisdictionAlias = "CREATE_JURISDICTION_ALIAS"

	auditTargetUnit              = "UNIT"
	auditTargetTaxonAlias        = "TAXON_ALIAS"
	auditTargetJurisdictionAlias = "JURISDICTION_ALIAS"
)
