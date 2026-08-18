// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"testing"
)

// fakeClosure is an in-memory ClosurePort: paths is the set of (graph, ancestor, descendant) edges
// considered reachable (self-pairs are implicit and never queried — the PDP special-cases them).
// bearing lists which graph ids are authority-bearing.
type fakeClosure struct {
	paths   map[[3]string]bool
	bearing map[string]bool
}

func (f fakeClosure) IsAncestorOrSelf(_ context.Context, g, a, d string) (bool, error) {
	return f.paths[[3]string{g, a, d}], nil
}

func (f fakeClosure) IsAuthorityBearing(_ context.Context, g string) (bool, error) {
	return f.bearing[g], nil
}

const (
	unitA  = "unit-a"
	unitB  = "unit-b" // descendant of unitA in graph "canonical"
	unitC  = "unit-c" // unrelated unit
	graphC = "canonical"
	graphD = "directory-only"
)

func closure() fakeClosure {
	return fakeClosure{
		paths: map[[3]string]bool{
			{graphC, unitA, unitB}: true,
		},
		bearing: map[string]bool{
			graphC: true,
			graphD: false,
		},
	}
}

func TestPDPDecide(t *testing.T) {
	tests := []struct {
		name  string
		in    DecisionInput
		allow bool
	}{
		{
			name:  "instance admin allowed any action on any unit",
			in:    DecisionInput{IsInstanceAdmin: true, Action: string(PermUnitLifecycle), UnitID: unitC},
			allow: true,
		},
		{
			name:  "instance-scope action denied for non-admin even with no grants",
			in:    DecisionInput{IsInstanceAdmin: false, Action: string(PermInstanceAdminManage), UnitID: unitA},
			allow: false,
		},
		{
			name: "instance-scope action denied even when subject holds an unrelated unit grant",
			in: DecisionInput{
				Action: string(PermInstanceAdminManage), UnitID: unitA,
				Grants: []ActiveGrant{{TargetUnitID: unitA, Scope: ScopeUnit, Perms: perms(PermInstanceAdminManage)}},
			},
			allow: false,
		},
		{
			name: "unit-scope grant matches exact target",
			in: DecisionInput{
				Action: string(PermUnitRead), UnitID: unitA,
				Grants: []ActiveGrant{{TargetUnitID: unitA, Scope: ScopeUnit, Perms: perms(PermUnitRead)}},
			},
			allow: true,
		},
		{
			name: "unit-scope grant does not cascade to a different unit",
			in: DecisionInput{
				Action: string(PermUnitRead), UnitID: unitB,
				Grants: []ActiveGrant{{TargetUnitID: unitA, Scope: ScopeUnit, Perms: perms(PermUnitRead)}},
			},
			allow: false,
		},
		{
			name: "unit-scope grant without the requested permission denies",
			in: DecisionInput{
				Action: string(PermUnitRead), UnitID: unitA,
				Grants: []ActiveGrant{{TargetUnitID: unitA, Scope: ScopeUnit, Perms: perms(PermSiteManage)}},
			},
			allow: false,
		},
		{
			name: "subtree grant on authority-bearing graph reaches self",
			in: DecisionInput{
				Action: string(PermUnitRead), UnitID: unitA,
				Grants: []ActiveGrant{{TargetUnitID: unitA, Scope: ScopeSubtree, GraphID: graphC, Perms: perms(PermUnitRead)}},
			},
			allow: true,
		},
		{
			name: "subtree grant on authority-bearing graph reaches a descendant via closure",
			in: DecisionInput{
				Action: string(PermUnitRead), UnitID: unitB,
				Grants: []ActiveGrant{{TargetUnitID: unitA, Scope: ScopeSubtree, GraphID: graphC, Perms: perms(PermUnitRead)}},
			},
			allow: true,
		},
		{
			name: "subtree grant does not reach an unrelated unit",
			in: DecisionInput{
				Action: string(PermUnitRead), UnitID: unitC,
				Grants: []ActiveGrant{{TargetUnitID: unitA, Scope: ScopeSubtree, GraphID: graphC, Perms: perms(PermUnitRead)}},
			},
			allow: false,
		},
		{
			name: "subtree grant on a directory-only (non-authority-bearing) graph cascades nothing",
			in: DecisionInput{
				Action: string(PermUnitRead), UnitID: unitB,
				Grants: []ActiveGrant{{TargetUnitID: unitA, Scope: ScopeSubtree, GraphID: graphD, Perms: perms(PermUnitRead)}},
			},
			allow: false,
		},
		{
			name: "union across graphs: second grant reaches even though the first doesn't",
			in: DecisionInput{
				Action: string(PermUnitRead), UnitID: unitB,
				Grants: []ActiveGrant{
					{TargetUnitID: unitC, Scope: ScopeUnit, Perms: perms(PermUnitRead)},
					{TargetUnitID: unitA, Scope: ScopeSubtree, GraphID: graphC, Perms: perms(PermUnitRead)},
				},
			},
			allow: true,
		},
		{
			name:  "no grants at all denies a unit-scope action",
			in:    DecisionInput{Action: string(PermUnitRead), UnitID: unitA},
			allow: false,
		},
	}

	pdp := NewPDP(closure())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pdp.Decide(context.Background(), tt.in)
			if err != nil {
				t.Fatalf("Decide: %v", err)
			}
			if got.Allow != tt.allow {
				t.Errorf("Decide(%+v) = %v, want allow=%v (denyReason=%q)", tt.in, got.Allow, tt.allow, got.DenyReason)
			}
		})
	}
}

func TestPDPDecideExplain(t *testing.T) {
	pdp := NewPDP(closure())

	t.Run("explained deny carries a reason", func(t *testing.T) {
		got, err := pdp.Decide(context.Background(), DecisionInput{
			Action: string(PermUnitRead), UnitID: unitA, Explain: true,
		})
		if err != nil {
			t.Fatalf("Decide: %v", err)
		}
		if got.Allow {
			t.Fatal("expected deny")
		}
		if got.DenyReason == "" {
			t.Error("expected a non-empty deny reason when Explain is set")
		}
	})

	t.Run("explained instance-admin allow names the plane, not a grant", func(t *testing.T) {
		got, err := pdp.Decide(context.Background(), DecisionInput{
			IsInstanceAdmin: true, Action: string(PermUnitRead), UnitID: unitA, Explain: true,
		})
		if err != nil {
			t.Fatalf("Decide: %v", err)
		}
		if len(got.Via) != 1 || !got.Via[0].InstanceAdmin {
			t.Errorf("Via = %+v, want a single InstanceAdmin contribution", got.Via)
		}
	})

	t.Run("explained allow unions contributions across graphs", func(t *testing.T) {
		got, err := pdp.Decide(context.Background(), DecisionInput{
			Action: string(PermUnitRead), UnitID: unitB, Explain: true,
			Grants: []ActiveGrant{
				{AssignmentID: "g1", TargetUnitID: unitB, Scope: ScopeUnit, Perms: perms(PermUnitRead)},
				{AssignmentID: "g2", TargetUnitID: unitA, Scope: ScopeSubtree, GraphID: graphC, Perms: perms(PermUnitRead)},
			},
		})
		if err != nil {
			t.Fatalf("Decide: %v", err)
		}
		if len(got.Via) != 2 {
			t.Errorf("Via = %+v, want 2 contributions (one per matching grant)", got.Via)
		}
	})

	t.Run("non-explain call short-circuits on first match (no Via)", func(t *testing.T) {
		got, err := pdp.Decide(context.Background(), DecisionInput{
			Action: string(PermUnitRead), UnitID: unitA,
			Grants: []ActiveGrant{{TargetUnitID: unitA, Scope: ScopeUnit, Perms: perms(PermUnitRead)}},
		})
		if err != nil {
			t.Fatalf("Decide: %v", err)
		}
		if len(got.Via) != 0 {
			t.Errorf("Via = %+v, want empty when Explain is false", got.Via)
		}
	})
}

func perms(ps ...Permission) map[Permission]struct{} {
	out := make(map[Permission]struct{}, len(ps))
	for _, p := range ps {
		out[p] = struct{}{}
	}
	return out
}
