// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	gencongregationimport "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/congregationimport"
	"github.com/palantir/pkg/datetime"
)

func toAPIRun(r domain.Run) gencongregationimport.ImportRun {
	var finishedAt *datetime.DateTime
	if r.FinishedAt != nil {
		dt := datetime.DateTime(*r.FinishedAt)
		finishedAt = &dt
	}
	return gencongregationimport.ImportRun{
		Id:                     r.ID,
		SourceCode:             r.SourceCode,
		Status:                 gencongregationimport.New_RunStatus(gencongregationimport.RunStatus_Value(r.Status)),
		TriggeredByPersonId:    r.TriggeredByPersonRID,
		CursorAtStart:          r.CursorAtStart,
		CursorAtEnd:            r.CursorAtEnd,
		RecordsFetched:         r.RecordsFetched,
		CandidatesCreated:      r.CandidatesCreated,
		CandidatesUpdated:      r.CandidatesUpdated,
		CandidatesAutoRejected: r.CandidatesAutoRejected,
		Error:                  r.Error,
		StartedAt:              datetime.DateTime(r.StartedAt),
		FinishedAt:             finishedAt,
	}
}

func toAPICandidate(c domain.Candidate) gencongregationimport.Candidate {
	var reviewedAt *datetime.DateTime
	if c.ReviewedAt != nil {
		dt := datetime.DateTime(*c.ReviewedAt)
		reviewedAt = &dt
	}
	return gencongregationimport.Candidate{
		Id:                             c.ID,
		ImportRunId:                    c.ImportRunID,
		SourceCode:                     c.SourceCode,
		SourceRecordId:                 c.SourceRecordID,
		Name:                           c.Name,
		TaxonHint:                      c.TaxonHint,
		TaxonId:                        c.TaxonID,
		JurisdictionHint:               c.JurisdictionHint,
		SuggestedJurisdictionUnitId:    c.SuggestedJurisdictionUnitID,
		CountryId:                      c.CountryID,
		AdminArea1:                     c.AdminArea1,
		Locality:                       c.Locality,
		Street:                         c.Street,
		HouseNumber:                    c.HouseNumber,
		PostalCode:                     c.PostalCode,
		Latitude:                       c.Latitude,
		Longitude:                      c.Longitude,
		GeocodePrecision:               c.GeocodePrecision,
		Status:                         gencongregationimport.New_CandidateStatus(gencongregationimport.CandidateStatus_Value(c.Status)),
		PossibleDuplicateOfCandidateId: c.PossibleDuplicateCandidateID,
		PossibleDuplicateOfUnitId:      c.PossibleDuplicateUnitID,
		RejectionReason:                c.RejectionReason,
		ReviewedByPersonId:             c.ReviewedByPersonRID,
		ReviewedAt:                     reviewedAt,
		CreatedUnitId:                  c.CreatedUnitID,
		CreatedAt:                      datetime.DateTime(c.CreatedAt),
		UpdatedAt:                      datetime.DateTime(c.UpdatedAt),
	}
}

func toAPITaxonAlias(a domain.TaxonAlias) gencongregationimport.TaxonAlias {
	return gencongregationimport.TaxonAlias{
		Id:                a.ID,
		SourceCode:        a.SourceCode,
		AliasText:         a.AliasText,
		TaxonId:           a.TaxonID,
		CreatedByPersonId: a.CreatedByPersonRID,
		CreatedAt:         datetime.DateTime(a.CreatedAt),
		UpdatedAt:         datetime.DateTime(a.UpdatedAt),
	}
}

func toAPIJurisdictionAlias(a domain.JurisdictionAlias) gencongregationimport.JurisdictionAlias {
	return gencongregationimport.JurisdictionAlias{
		Id:                 a.ID,
		SourceCode:         a.SourceCode,
		AliasText:          a.AliasText,
		JurisdictionUnitId: a.JurisdictionUnitID,
		CreatedByPersonId:  a.CreatedByPersonRID,
		CreatedAt:          datetime.DateTime(a.CreatedAt),
		UpdatedAt:          datetime.DateTime(a.UpdatedAt),
	}
}

func toDomainEdit(r gencongregationimport.EditCandidateRequest) domain.EditInput {
	return domain.EditInput{
		Name:        r.Name,
		TaxonID:     r.TaxonId,
		CountryID:   r.CountryId,
		AdminArea1:  r.AdminArea1,
		Locality:    r.Locality,
		Street:      r.Street,
		HouseNumber: r.HouseNumber,
		PostalCode:  r.PostalCode,
		Latitude:    r.Latitude,
		Longitude:   r.Longitude,
	}
}
