// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
)

// ActiveGrant is one active assignment with its role's resolved permission set — the PDP's per-grant
// input (assembled by the application layer from the repository). GraphID/GraphCode are empty for
// unit scope. Perms is the role's full code membership.
type ActiveGrant struct {
	AssignmentID string
	RoleID       string
	RoleCode     string
	TargetUnitID string
	Scope        Scope
	GraphID      string
	GraphCode    string
	Perms        map[Permission]struct{}
}

// Has reports whether the grant's role carries permission p.
func (g ActiveGrant) Has(p Permission) bool {
	_, ok := g.Perms[p]
	return ok
}

// Contribution names one reason an ALLOW decision was reached (decision-explain). For an
// instance-plane allow only InstanceAdmin is set; otherwise the contributing assignment is named.
type Contribution struct {
	InstanceAdmin bool
	AssignmentID  string
	RoleCode      string
	TargetUnitID  string
	Scope         Scope
	GraphCode     string
}

// Decision is the PDP's answer. Allow is the verdict; Via (ALLOW) or DenyReason (DENY) carry the
// explanation when explain is requested. Via may name several contributions (union across graphs).
type Decision struct {
	Allow      bool
	Via        []Contribution
	DenyReason string
}

// DecisionInput is one authorize question plus the subject's already-fetched authority state.
type DecisionInput struct {
	Grants          []ActiveGrant
	IsInstanceAdmin bool
	Action          string
	UnitID          string
	Explain         bool
}

// PDP is the Policy Decision Point engine. It is pure logic over the supplied authority state plus
// the directory closure port; it reads no per-request state beyond its inputs — decisions are pure
// functions of current data (assignments + closure + the code catalog).
type PDP struct {
	closure ClosurePort
}

// NewPDP builds the engine over the directory closure port.
func NewPDP(closure ClosurePort) PDP { return PDP{closure: closure} }

// Decide answers authorize(subject, action, unit):
//
//  1. An active instance admin is allowed everything (the cluster-admin plane) — the first admin
//     must be able to grant unit assignments (assignment.grant is unit-scoped) before any unit
//     assignment exists, so the instance plane cannot be limited to instance-scope permissions only.
//  2. Instance-scope permissions are satisfiable ONLY on the instance plane (roles never carry
//     them — Role.Validate rejects it), so a non-admin is denied any instance-scope action outright.
//  3. Otherwise the action must appear in some active grant that REACHES unitID: a `unit` grant
//     whose target is unitID, or a `subtree` grant on an authority-bearing graph whose target is
//     unitID or an ancestor of it in that graph's closure. Union across graphs.
func (p PDP) Decide(ctx context.Context, in DecisionInput) (Decision, error) {
	if in.IsInstanceAdmin {
		return allow(in.Explain, Contribution{InstanceAdmin: true}), nil
	}
	if IsInstanceScope(in.Action) {
		return deny(in.Explain, "action is instance-scope and the subject is not an instance admin"), nil
	}

	action := Permission(in.Action)
	authority := map[string]bool{} // graphID -> is_authority_bearing (memoized within this decision)
	var via []Contribution

	for _, g := range in.Grants {
		if !g.Has(action) {
			continue
		}
		switch g.Scope {
		case ScopeUnit:
			if g.TargetUnitID == in.UnitID {
				if !in.Explain {
					return allow(false, Contribution{}), nil
				}
				via = append(via, contributionOf(g))
			}
		case ScopeSubtree:
			bearing, ok := authority[g.GraphID]
			if !ok {
				b, err := p.closure.IsAuthorityBearing(ctx, g.GraphID)
				if err != nil {
					return Decision{}, err
				}
				bearing = b
				authority[g.GraphID] = b
			}
			if !bearing {
				continue // directory-only graph cascades nothing (D-DirectoryGraphs)
			}
			reaches := g.TargetUnitID == in.UnitID // self: a subtree includes its root
			if !reaches {
				r, err := p.closure.IsAncestorOrSelf(ctx, g.GraphID, g.TargetUnitID, in.UnitID)
				if err != nil {
					return Decision{}, err
				}
				reaches = r
			}
			if reaches {
				if !in.Explain {
					return allow(false, Contribution{}), nil
				}
				via = append(via, contributionOf(g))
			}
		}
	}

	if len(via) > 0 {
		return Decision{Allow: true, Via: via}, nil
	}
	return deny(in.Explain, "the requested permission is not in the subject's effective set for this unit"), nil
}

func contributionOf(g ActiveGrant) Contribution {
	return Contribution{
		AssignmentID: g.AssignmentID,
		RoleCode:     g.RoleCode,
		TargetUnitID: g.TargetUnitID,
		Scope:        g.Scope,
		GraphCode:    g.GraphCode,
	}
}

func allow(explain bool, c Contribution) Decision {
	d := Decision{Allow: true}
	if explain {
		d.Via = []Contribution{c}
	}
	return d
}

func deny(explain bool, reason string) Decision {
	d := Decision{Allow: false}
	if explain {
		d.DenyReason = reason
	}
	return d
}
