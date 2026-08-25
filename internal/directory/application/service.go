// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package application is the directory module's orchestration layer: unit/edge creation, the
// incremental closure maintenance on attach/detach, and closure rebuild/verify. Ported from
// ../go-oikumenea/internal/tenant/application/service.go, trimmed of organizations/domains/
// unit-kinds/visibility/audit logging per D-CorePortScope.
package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/directory/adapters"
	"github.com/olehmushka/open-faith-map/internal/directory/domain"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func defaultGraph(code string) string {
	if code == "" {
		return domain.CanonicalGraphCode
	}
	return code
}

func (s *Service) inTx(ctx context.Context, fn func(store *adapters.Repository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(adapters.NewRepository(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GetUnit reads a single unit outside any transaction (read-only, pool-bound).
func (s *Service) GetUnit(ctx context.Context, id string) (domain.Unit, error) {
	return adapters.NewRepository(s.pool).GetUnit(ctx, id)
}

// ListUnits searches units by code/name — M10.7's core.conjure.yml ListUnits, read-only, pool-bound.
func (s *Service) ListUnits(ctx context.Context, query string, limit int) ([]domain.Unit, error) {
	return adapters.NewRepository(s.pool).SearchUnits(ctx, query, limit)
}

// CreateUnit creates a root unit — no parent, no closure work. Use CreateUnitWithEdge when the
// caller already has a parent at creation time.
func (s *Service) CreateUnit(ctx context.Context, u domain.Unit) (domain.Unit, error) {
	var out domain.Unit
	err := s.inTx(ctx, func(store *adapters.Repository) error {
		created, err := store.InsertUnit(ctx, u)
		if err != nil {
			return err
		}
		out = created
		return nil
	})
	return out, err
}

// UpdateUnit rewrites id's name/code/level (M12.1). Cross-module authorization and the audit-log
// before/after snapshot are internal/core's job (D-InProcessAuthz) — this method just writes.
func (s *Service) UpdateUnit(ctx context.Context, id, name string, code *string, level *int16) (domain.Unit, error) {
	var out domain.Unit
	err := s.inTx(ctx, func(store *adapters.Repository) error {
		updated, err := store.UpdateUnit(ctx, id, name, code, level)
		if err != nil {
			return err
		}
		out = updated
		return nil
	})
	return out, err
}

// SetUnitState transitions id to state (M12.1) — archive/suspend/reactivate. Root-unit protection
// and the unit.lifecycle gate live in internal/core, one layer up.
func (s *Service) SetUnitState(ctx context.Context, id string, state domain.State) (domain.Unit, error) {
	var out domain.Unit
	err := s.inTx(ctx, func(store *adapters.Repository) error {
		updated, err := store.SetUnitState(ctx, id, state)
		if err != nil {
			return err
		}
		out = updated
		return nil
	})
	return out, err
}

// HasChildren reports whether id has any live child edge, in any graph (M12.1). Exposed at the
// Service level, not just used internally by DeleteUnit, so internal/core.DeleteUnit can run this
// cheap, purely-structural check before its own two cross-module orphan checks (active role
// assignments, an existing religion org profile) — deterministic ordering, and avoids two extra
// round-trips to another module's store when the delete was always going to fail on this one.
func (s *Service) HasChildren(ctx context.Context, id string) (bool, error) {
	return adapters.NewRepository(s.pool).HasChildren(ctx, id)
}

// DeleteUnit soft-deletes id (M12.1), refusing if it has any live child edge (ErrUnitHasChildren).
// The other two orphan checks milestones.md's M12.1 row calls for — active role assignments, an
// existing religion org profile — are cross-module and enforced by internal/core before this is ever
// called (D-InProcessAuthz: this module cannot import authz or religion). The HasChildren check
// below runs again inside the same transaction as the delete itself — belt-and-suspenders against a
// child being added between internal/core's own upfront HasChildren call and this method running.
func (s *Service) DeleteUnit(ctx context.Context, id string) (domain.Unit, error) {
	var out domain.Unit
	err := s.inTx(ctx, func(store *adapters.Repository) error {
		hasChildren, err := store.HasChildren(ctx, id)
		if err != nil {
			return err
		}
		if hasChildren {
			return domain.ErrUnitHasChildren
		}
		deleted, err := store.SoftDeleteUnit(ctx, id)
		if err != nil {
			return err
		}
		out = deleted
		return nil
	})
	return out, err
}

// CreateUnitWithEdge atomically creates a unit under parentID and attaches the parent->child edge
// in ONE transaction (ported from GH-36's own fix, upstream service.go:453-519). Mints the child's
// id up front and seeds its closure BEFORE the unit's own INSERT — see the package doc and
// docs/architecture/decisions.md's D-CorePortScope amendment for why the ordering is kept even
// though this port has no RLS for it to matter to.
func (s *Service) CreateUnitWithEdge(ctx context.Context, u domain.Unit, parentID, graphCode string) (domain.Unit, error) {
	graphCode = defaultGraph(graphCode)
	var out domain.Unit
	err := s.inTx(ctx, func(store *adapters.Repository) error {
		if _, err := store.GetUnit(ctx, parentID); err != nil {
			return err
		}
		g, err := store.GetGraphByCode(ctx, graphCode)
		if err != nil {
			return err
		}
		if err := store.LockGraphForClosure(ctx, g.ID); err != nil {
			return err
		}
		childID, err := store.MintUnitID(ctx)
		if err != nil {
			return err
		}
		// Closure BEFORE the unit's own row exists — a freshly minted id can't already be anyone's
		// ancestor, so (unlike AddEdge) no cycle guard is needed.
		if err := store.ExtendClosureForEdge(ctx, g.ID, parentID, childID); err != nil {
			return err
		}
		created, err := store.InsertUnitWithID(ctx, childID, u)
		if err != nil {
			return err
		}
		if _, err := store.InsertEdge(ctx, g.ID, parentID, childID); err != nil {
			return err
		}
		out = created
		return nil
	})
	return out, err
}

// AddEdge attaches childID as a child of parentID within a graph (default "canonical"), guarding
// against cycles, then incrementally extends the graph's closure — all in one transaction. The
// per-graph closure lock is taken before the cycle guard: two concurrent guard-then-insert attaches
// could otherwise each pass the guard and jointly close a cycle.
func (s *Service) AddEdge(ctx context.Context, childID, parentID, graphCode string) (domain.Edge, error) {
	graphCode = defaultGraph(graphCode)
	if parentID == childID {
		return domain.Edge{}, domain.ErrEdgeCycle
	}
	var out domain.Edge
	err := s.inTx(ctx, func(store *adapters.Repository) error {
		if _, err := store.GetUnit(ctx, childID); err != nil {
			return err
		}
		if _, err := store.GetUnit(ctx, parentID); err != nil {
			return err
		}
		g, err := store.GetGraphByCode(ctx, graphCode)
		if err != nil {
			return err
		}
		if err := store.LockGraphForClosure(ctx, g.ID); err != nil {
			return err
		}
		// A new parent->child edge closes a cycle iff the child already reaches the parent in g.
		cyclic, err := store.ClosureHasPath(ctx, g.ID, childID, parentID)
		if err != nil {
			return err
		}
		if cyclic {
			return domain.ErrEdgeCycle
		}
		edge, err := store.InsertEdge(ctx, g.ID, parentID, childID)
		if err != nil {
			return err
		}
		if err := store.ExtendClosureForEdge(ctx, g.ID, parentID, childID); err != nil {
			return err
		}
		edge.GraphCode = g.Code
		out = edge
		return nil
	})
	return out, err
}

// RemoveEdge detaches childID from parentID within a graph (default "canonical") and, only if an
// edge was actually removed, incrementally shrinks the closure. Detaching an absent edge is a
// documented no-op (idempotent) — skipping the shrink then is load-bearing, not an optimization; see
// adapters.Store.ShrinkClosureForEdge's own doc comment.
func (s *Service) RemoveEdge(ctx context.Context, childID, parentID, graphCode string) error {
	graphCode = defaultGraph(graphCode)
	return s.inTx(ctx, func(store *adapters.Repository) error {
		g, err := store.GetGraphByCode(ctx, graphCode)
		if err != nil {
			return err
		}
		if err := store.LockGraphForClosure(ctx, g.ID); err != nil {
			return err
		}
		deleted, err := store.DeleteEdge(ctx, g.ID, parentID, childID)
		if err != nil {
			return err
		}
		if deleted == 0 {
			return nil
		}
		return store.ShrinkClosureForEdge(ctx, g.ID, parentID, childID)
	})
}

// Ancestors returns unitID's ancestors in graphCode (default "canonical"), nearest first.
func (s *Service) Ancestors(ctx context.Context, unitID, graphCode string) ([]domain.UnitRef, error) {
	store := adapters.NewRepository(s.pool)
	g, err := store.GetGraphByCode(ctx, defaultGraph(graphCode))
	if err != nil {
		return nil, err
	}
	return store.ListAncestors(ctx, g.ID, unitID)
}

// Descendants returns a bounded (non-paginated — see adapters.Store.ListDescendants) list of
// unitID's subtree in graphCode (default "canonical").
func (s *Service) Descendants(ctx context.Context, unitID, graphCode string, limit int) ([]domain.UnitRef, error) {
	if limit <= 0 {
		limit = 100
	}
	store := adapters.NewRepository(s.pool)
	g, err := store.GetGraphByCode(ctx, defaultGraph(graphCode))
	if err != nil {
		return nil, err
	}
	return store.ListDescendants(ctx, g.ID, unitID, limit)
}

// Move starts or resumes moving unitID onto newParentUnitID within graphCode (default "canonical") —
// M12.2, generalized out of internal/registration's former private reparent state machine (the same
// add-before-remove, resumable, closure-safe algorithm; see runMoveSteps). Re-entrant on
// (graphCode, unitID): a repeat call while a live job targets the same newParentUnitID resumes it
// from whichever step last durably landed; targeting a different parent while one is live is a
// conflict the caller must resolve first.
//
// Add-before-remove by design (ported from D-JurisdictionUnits): unitID briefly has two parents
// mid-move rather than momentarily zero, so a subtree-scoped grant never loses reach to it.
func (s *Service) Move(ctx context.Context, graphCode, unitID, newParentUnitID, performedByPersonID string) (domain.MoveJob, error) {
	graphCode = defaultGraph(graphCode)
	store := adapters.NewRepository(s.pool)
	g, err := store.GetGraphByCode(ctx, graphCode)
	if err != nil {
		return domain.MoveJob{}, err
	}

	job, err := store.GetLiveMoveJob(ctx, g.ID, unitID)
	if err != nil {
		return domain.MoveJob{}, fmt.Errorf("getLiveMoveJob: %w", err)
	}
	if job == nil {
		oldParentUnitID, err := s.currentParent(ctx, store, g.ID, unitID)
		if err != nil {
			return domain.MoveJob{}, fmt.Errorf("resolve current parent: %w", err)
		}
		created, err := store.CreateMoveJob(ctx, g.ID, unitID, oldParentUnitID, newParentUnitID, performedByPersonID)
		if err != nil {
			return domain.MoveJob{}, fmt.Errorf("createMoveJob: %w", err)
		}
		job = &created
	} else if job.NewParentUnitID != newParentUnitID {
		return domain.MoveJob{}, fmt.Errorf("%w: unit %s already targets %s, requested %s", domain.ErrUnitMoveConflict, unitID, job.NewParentUnitID, newParentUnitID)
	}

	return s.runMoveSteps(ctx, store, graphCode, *job)
}

// GetMoveStatus returns the most recent move job for (graphCode, unitID), or nil if none has ever
// been started.
func (s *Service) GetMoveStatus(ctx context.Context, graphCode, unitID string) (*domain.MoveJob, error) {
	store := adapters.NewRepository(s.pool)
	g, err := store.GetGraphByCode(ctx, defaultGraph(graphCode))
	if err != nil {
		return nil, err
	}
	return store.GetLatestMoveJob(ctx, g.ID, unitID)
}

// CurrentParent resolves unitID's actual current parent in graphCode (default "canonical") — the
// same resolution Move uses internally, exposed so a caller that must authorize a move BEFORE
// calling it (internal/core's dual-parent unit.edges.manage check, D-UnitMoveDualScope) can learn the
// old parent without duplicating Move's own job-history-aware logic.
func (s *Service) CurrentParent(ctx context.Context, graphCode, unitID string) (string, error) {
	graphCode = defaultGraph(graphCode)
	store := adapters.NewRepository(s.pool)
	g, err := store.GetGraphByCode(ctx, graphCode)
	if err != nil {
		return "", err
	}
	return s.currentParent(ctx, store, g.ID, unitID)
}

// currentParent resolves unitID's actual current parent in graphID: the most recent VERIFIED move
// job's target if one exists (this store IS the record of every successful move ever performed on
// this unit in this graph), else the nearest ancestor read straight from the graph itself — the
// first-ever-moved case, where there is no job history yet to trust instead.
func (s *Service) currentParent(ctx context.Context, store *adapters.Repository, graphID, unitID string) (string, error) {
	latest, err := store.GetLatestMoveJob(ctx, graphID, unitID)
	if err != nil {
		return "", err
	}
	if latest != nil && latest.Status == domain.MoveVerified {
		return latest.NewParentUnitID, nil
	}
	ancestors, err := store.ListAncestors(ctx, graphID, unitID)
	if err != nil {
		return "", err
	}
	if len(ancestors) == 0 {
		return "", fmt.Errorf("%w: unit %s, graph %s", domain.ErrUnitHasNoCurrentParent, unitID, graphID)
	}
	return ancestors[0].ID, nil
}

// runMoveSteps drives job through whichever steps haven't durably landed yet. Each step persists its
// own completion before the next runs, so a crash between any two steps resumes exactly here on the
// next call rather than repeating or skipping work. Each step is its own transaction (via AddEdge/
// RemoveEdge/Ancestors below) — not one transaction spanning the whole job — which is exactly why the
// job needs this resumable state machine instead of relying on directory-level atomicity.
func (s *Service) runMoveSteps(ctx context.Context, store *adapters.Repository, graphCode string, job domain.MoveJob) (domain.MoveJob, error) {
	if job.Status == domain.MovePending {
		if _, err := s.AddEdge(ctx, job.UnitID, job.NewParentUnitID, graphCode); err != nil && !errors.Is(err, domain.ErrEdgeExists) {
			return store.FailMoveJob(ctx, job.ID, fmt.Sprintf("addEdge(new parent): %v", err))
		}
		updated, err := store.AdvanceMoveJob(ctx, job.ID, domain.MoveNewEdgeAdded)
		if err != nil {
			return domain.MoveJob{}, err
		}
		job = updated
	}

	if job.Status == domain.MoveNewEdgeAdded {
		// RemoveEdge on an already-absent edge is a documented no-op (idempotent by design), so a
		// resumed retry re-calling this needs no special-case error handling.
		if err := s.RemoveEdge(ctx, job.UnitID, job.OldParentUnitID, graphCode); err != nil {
			return store.FailMoveJob(ctx, job.ID, fmt.Sprintf("removeEdge(old parent): %v", err))
		}
		updated, err := store.AdvanceMoveJob(ctx, job.ID, domain.MoveOldEdgeRemoved)
		if err != nil {
			return domain.MoveJob{}, err
		}
		job = updated
	}

	if job.Status == domain.MoveOldEdgeRemoved {
		ancestors, err := s.Ancestors(ctx, job.UnitID, graphCode)
		if err != nil {
			return store.FailMoveJob(ctx, job.ID, fmt.Sprintf("ancestors (verify): %v", err))
		}
		if !ancestorsInclude(ancestors, job.NewParentUnitID) {
			return store.FailMoveJob(ctx, job.ID, fmt.Sprintf("verify: %s not found in %s's ancestors after move", job.NewParentUnitID, job.UnitID))
		}
		updated, err := store.AdvanceMoveJob(ctx, job.ID, domain.MoveVerified)
		if err != nil {
			return domain.MoveJob{}, err
		}
		job = updated
	}

	return job, nil
}

func ancestorsInclude(refs []domain.UnitRef, unitID string) bool {
	for _, u := range refs {
		if u.ID == unitID {
			return true
		}
	}
	return false
}

// RebuildClosure recomputes graphCode's closure from scratch (or every graph, if graphCode is nil).
// One transaction per graph; the row lock is held for the whole operation (delete-all, re-derive,
// status upsert) — unlike edge-add/remove there is no "as late as possible" here, since the entire
// point is a from-scratch recompute.
func (s *Service) RebuildClosure(ctx context.Context, graphCode *string) ([]domain.ClosureReport, error) {
	graphs, err := s.resolveGraphs(ctx, graphCode)
	if err != nil {
		return nil, err
	}
	reports := make([]domain.ClosureReport, 0, len(graphs))
	for _, g := range graphs {
		if err := s.inTx(ctx, func(store *adapters.Repository) error {
			if err := store.LockGraphForClosure(ctx, g.ID); err != nil {
				return err
			}
			if err := store.DeleteClosureForGraph(ctx, g.ID); err != nil {
				return err
			}
			if err := store.RebuildClosureForGraph(ctx, g.ID); err != nil {
				return err
			}
			return store.UpsertClosureStatus(ctx, g.ID, 0, 0, false, json.RawMessage("[]"))
		}); err != nil {
			return nil, err
		}
		reports = append(reports, domain.ClosureReport{GraphCode: g.Code})
	}
	return reports, nil
}

// VerifyClosure diffs the stored closure against what the edges alone imply, per graph (or every
// graph, if graphCode is nil), and upserts directory_closure_status. One transaction across ALL
// graphs, deliberately NOT locked — it's read-only, and a transient false-drift reading during a
// concurrent edit is accepted (acceptable for a health probe).
func (s *Service) VerifyClosure(ctx context.Context, graphCode *string) ([]domain.ClosureReport, error) {
	graphs, err := s.resolveGraphs(ctx, graphCode)
	if err != nil {
		return nil, err
	}
	var reports []domain.ClosureReport
	err = s.inTx(ctx, func(store *adapters.Repository) error {
		for _, g := range graphs {
			missing, extra, sample, err := store.VerifyClosureForGraph(ctx, g.ID)
			if err != nil {
				return err
			}
			inDrift := missing > 0 || extra > 0
			if err := store.UpsertClosureStatus(ctx, g.ID, missing, extra, inDrift, sample); err != nil {
				return err
			}
			reports = append(reports, domain.ClosureReport{
				GraphCode: g.Code, MissingCount: missing, ExtraCount: extra, InDrift: inDrift, Sample: sample,
			})
		}
		return nil
	})
	return reports, err
}

func (s *Service) resolveGraphs(ctx context.Context, graphCode *string) ([]domain.Graph, error) {
	store := adapters.NewRepository(s.pool)
	if graphCode != nil {
		g, err := store.GetGraphByCode(ctx, *graphCode)
		if err != nil {
			return nil, err
		}
		return []domain.Graph{g}, nil
	}
	return store.ListGraphIDs(ctx)
}
