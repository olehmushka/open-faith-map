// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

// CanVouch enforces vouching.md's "a guarantor cannot vouch while revoked" invariant. Pure — no
// store, no go-oikumenea call — so it's unit-testable without either; the caller is responsible for
// getting a live (never cached) GuarantorStatus before calling this.
func CanVouch(status GuarantorStatus) error {
	if status.Status == StatusRevoked {
		return ErrGuarantorRevoked
	}
	return nil
}
