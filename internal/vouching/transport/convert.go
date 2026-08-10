// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"strings"

	genvouching "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/vouching"
	"github.com/olehmushka/open-faith-map/internal/vouching/domain"
	"github.com/palantir/pkg/datetime"
)

func toAPIVouch(v domain.Vouch) genvouching.Vouch {
	return genvouching.Vouch{
		Id:                 v.ID,
		GuarantorPersonId:  v.GuarantorPersonRID,
		ClaimantPersonId:   v.ClaimantPersonRID,
		CongregationUnitId: v.CongregationUnitID,
		Statement:          v.Statement,
		CreatedAt:          datetime.DateTime(v.CreatedAt),
	}
}

func toAPIVouches(vouches []domain.Vouch) []genvouching.Vouch {
	out := make([]genvouching.Vouch, 0, len(vouches))
	for _, v := range vouches {
		out = append(out, toAPIVouch(v))
	}
	return out
}

// toAPIGuarantorStatusValue shims the one deliberate, scoped case deviation in this module: the DB
// CHECK constraint (migrations/0008_vouching.sql) and domain.GuarantorStatusValue are lowercase
// (vouching.md's own literal SQL text), while the Conjure GuarantorStatus enum is uppercase,
// matching every other Conjure enum in this repo. No endpoint takes a GuarantorStatus as input —
// it is only ever returned — so only this direction is needed; no other module needs this shim at
// all (see domain/vouching.go's comment on GuarantorStatusValue).
func toAPIGuarantorStatusValue(v domain.GuarantorStatusValue) genvouching.GuarantorStatus {
	return genvouching.New_GuarantorStatus(genvouching.GuarantorStatus_Value(strings.ToUpper(string(v))))
}

func toAPIGuarantorStatus(g domain.GuarantorStatus) genvouching.GuarantorStatusRecord {
	var updatedAt *datetime.DateTime
	if g.UpdatedAt != nil {
		v := datetime.DateTime(*g.UpdatedAt)
		updatedAt = &v
	}
	var revokedAt *datetime.DateTime
	if g.RevokedAt != nil {
		v := datetime.DateTime(*g.RevokedAt)
		revokedAt = &v
	}
	return genvouching.GuarantorStatusRecord{
		GuarantorPersonId: g.GuarantorPersonRID,
		Status:            toAPIGuarantorStatusValue(g.Status),
		RevokedAt:         revokedAt,
		RevokedReason:     g.RevokedReason,
		RevokedByPersonId: g.RevokedByPersonRID,
		UpdatedAt:         updatedAt,
	}
}
