// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"testing"
)

func TestCanVouch(t *testing.T) {
	tests := []struct {
		name    string
		status  GuarantorStatus
		wantErr error
	}{
		{name: "trusted guarantor may vouch", status: GuarantorStatus{Status: StatusTrusted}, wantErr: nil},
		{name: "revoked guarantor may not vouch", status: GuarantorStatus{Status: StatusRevoked}, wantErr: ErrGuarantorRevoked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CanVouch(tt.status)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("CanVouch() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
