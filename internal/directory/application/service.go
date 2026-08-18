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

func (s *Service) inTx(ctx context.Context, fn func(store *adapters.Store) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(adapters.NewStore(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GetUnit reads a single unit outside any transaction (read-only, pool-bound).
func (s *Service) GetUnit(ctx context.Context, id string) (domain.Unit, error) {
	return adapters.NewStore(s.pool).GetUnit(ctx, id)
}

// ListUnits searches units by code/name — M10.7's core.conjure.yml ListUnits, read-only, pool-bound.
func (s *Service) ListUnits(ctx context.Context, query string, limit int) ([]domain.Unit, error) {
	return adapters.NewStore(s.pool).SearchUnits(ctx, query, limit)
}

// CreateUnit creates a root unit — no parent, no closure work. Use CreateUnitWithEdge when the
// caller already has a parent at creation time.
func (s *Service) CreateUnit(ctx context.Context, u domain.Unit) (domain.Unit, error) {
	var out domain.Unit
	err := s.inTx(ctx, func(store *adapters.Store) error {
		created, err := store.InsertUnit(ctx, u)
		if err != nil {
			return err
		}
		out = created
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
	err := s.inTx(ctx, func(store *adapters.Store) error {
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
	err := s.inTx(ctx, func(store *adapters.Store) error {
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
	return s.inTx(ctx, func(store *adapters.Store) error {
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
	store := adapters.NewStore(s.pool)
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
	store := adapters.NewStore(s.pool)
	g, err := store.GetGraphByCode(ctx, defaultGraph(graphCode))
	if err != nil {
		return nil, err
	}
	return store.ListDescendants(ctx, g.ID, unitID, limit)
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
		if err := s.inTx(ctx, func(store *adapters.Store) error {
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
	err = s.inTx(ctx, func(store *adapters.Store) error {
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
	store := adapters.NewStore(s.pool)
	if graphCode != nil {
		g, err := store.GetGraphByCode(ctx, *graphCode)
		if err != nil {
			return nil, err
		}
		return []domain.Graph{g}, nil
	}
	return store.ListGraphIDs(ctx)
}
