// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"time"
)

// JurisdictionAlias is an operator-maintained free-text-hint -> go-oikumenea jurisdiction Unit RID
// mapping (congregationimport_jurisdiction_aliases). Same shape as TaxonAlias, deliberately not
// merged with it: a taxon alias resolves denomination, a jurisdiction alias resolves a specific
// diocese/eparchy/synod unit — unrelated ID spaces. SourceCode nil means the alias applies across
// every source.
type JurisdictionAlias struct {
	ID                 string
	SourceCode         *string
	AliasText          string
	JurisdictionUnitID string
	CreatedByPersonRID string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
