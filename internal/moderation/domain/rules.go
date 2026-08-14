// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"time"
)

// doctrinalReasonCode is not a real ReasonCode value (ReasonCode's own const block has no
// DOCTRINAL_CONCERN) — this exists only so ValidateReasonCode can recognize the specific string a
// client might mistakenly send and point them at the right answer (file under OTHER) instead of a
// bare "invalid enum value" from the DB CHECK constraint.
const doctrinalReasonCode ReasonCode = "DOCTRINAL_CONCERN"

// ValidateReasonCode enforces moderation.md's invariant that doctrinal disputes are not adjudicated
// as their own reason — a caller sending DOCTRINAL_CONCERN gets a clear, on-point rejection rather
// than falling through to Postgres's CHECK constraint (which would reject it too, just with no
// guidance toward OTHER). Any other unrecognized string is left to that same CHECK constraint, same
// as every other module's enum handling in this repo (no IsUnknown() gate anywhere else either).
func ValidateReasonCode(code ReasonCode) error {
	if code == doctrinalReasonCode {
		return ErrDoctrinalReasonNotAllowed
	}
	return nil
}

// CanReverse enforces moderation.md's "every action is reversible within its grace window"
// invariant: a REVERSE action can't itself be reversed, an action already reversed can't be
// reversed again, and the grace window (ReversalGracePeriod) must not have elapsed. Pure — no store,
// no go-oikumenea call — so it's unit-testable without either.
func CanReverse(original Action, now time.Time) error {
	if original.ActionKind == ActionReverse {
		return ErrActionNotReversible
	}
	if original.ReversedByActionID != nil {
		return ErrActionNotReversible
	}
	if now.Sub(original.CreatedAt) > ReversalGracePeriod {
		return ErrActionNotReversible
	}
	return nil
}

// CanDecideAppeal enforces moderation.md's "an appeal is never decided by its action's original
// actor" invariant, at write time rather than left to moderator discipline.
func CanDecideAppeal(originalActorPersonID, deciderPersonID string) error {
	if originalActorPersonID == deciderPersonID {
		return ErrAppealActorConflict
	}
	return nil
}
