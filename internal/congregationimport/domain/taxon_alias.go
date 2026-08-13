// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"time"
)

// TaxonAlias is an operator-maintained free-text-hint -> religion_taxa RID mapping
// (congregationimport_taxon_aliases). SourceCode nil means the alias applies across every source.
type TaxonAlias struct {
	ID                 string
	SourceCode         *string
	AliasText          string
	TaxonID            string
	CreatedByPersonRID string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
