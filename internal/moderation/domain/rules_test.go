// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"testing"
	"time"
)

func TestValidateReasonCode(t *testing.T) {
	tests := []struct {
		name    string
		code    ReasonCode
		wantErr error
	}{
		{name: "spam is allowed", code: ReasonSpam, wantErr: nil},
		{name: "other is allowed", code: ReasonOther, wantErr: nil},
		{name: "doctrinal concern is rejected", code: doctrinalReasonCode, wantErr: ErrDoctrinalReasonNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReasonCode(tt.code)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateReasonCode(%q) = %v, want %v", tt.code, err, tt.wantErr)
			}
		})
	}
}

func TestCanReverse(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	reverserID := "action-2"

	tests := []struct {
		name    string
		action  Action
		wantErr error
	}{
		{
			name:    "fresh action within grace window is reversible",
			action:  Action{ID: "action-1", ActionKind: ActionSuspend, CreatedAt: now.Add(-1 * time.Hour)},
			wantErr: nil,
		},
		{
			name:    "action right at the edge of the grace window is reversible",
			action:  Action{ID: "action-1", ActionKind: ActionSuspend, CreatedAt: now.Add(-ReversalGracePeriod)},
			wantErr: nil,
		},
		{
			name:    "action past the grace window is not reversible",
			action:  Action{ID: "action-1", ActionKind: ActionSuspend, CreatedAt: now.Add(-ReversalGracePeriod - time.Minute)},
			wantErr: ErrActionNotReversible,
		},
		{
			name:    "a REVERSE action cannot itself be reversed",
			action:  Action{ID: "action-1", ActionKind: ActionReverse, CreatedAt: now.Add(-time.Minute)},
			wantErr: ErrActionNotReversible,
		},
		{
			name:    "an already-reversed action cannot be reversed again",
			action:  Action{ID: "action-1", ActionKind: ActionSuspend, CreatedAt: now.Add(-time.Minute), ReversedByActionID: &reverserID},
			wantErr: ErrActionNotReversible,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CanReverse(tt.action, now)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("CanReverse() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCanDecideAppeal(t *testing.T) {
	tests := []struct {
		name          string
		originalActor string
		decider       string
		wantErr       error
	}{
		{name: "a different moderator may decide", originalActor: "person-1", decider: "person-2", wantErr: nil},
		{name: "the original actor may not decide their own action's appeal", originalActor: "person-1", decider: "person-1", wantErr: ErrAppealActorConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CanDecideAppeal(tt.originalActor, tt.decider)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("CanDecideAppeal() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
