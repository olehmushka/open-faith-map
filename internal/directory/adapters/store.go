// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the directory module's Postgres store: unit/graph/edge CRUD plus the closure
// maintenance queries ported from ../go-oikumenea/internal/tenant/adapters/queries/tenant.sql
// (renamed tenant_* -> directory_*, org-scoping dropped per D-CorePortScope). Hand-written pgx,
// matching this repo's own convention.
package adapters

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/olehmushka/open-faith-map/internal/directory/domain"
)

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx, so a Store can be bound either to the pool
// for read-only calls (Ancestors/Descendants/VerifyClosure's read side, the ClosurePort methods) or
// to a single pgx.Tx for the multi-statement writes (CreateUnitWithEdge/AddEdge/RemoveEdge/
// RebuildClosure all need one transaction — see internal/directory/application).
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Store struct {
	q Querier
}

func NewStore(q Querier) *Store {
	return &Store{q: q}
}

// ---------------------------------------------------------------- units

func (s *Store) GetUnit(ctx context.Context, id string) (domain.Unit, error) {
	return s.scanUnit(s.q.QueryRow(ctx, `
		SELECT id, code, name, level, state, metadata, created_at, updated_at
		FROM openfaithmap.directory_units
		WHERE id = $1 AND deleted_at IS NULL`, id))
}

// MintUnitID mints a Unit RID ahead of the unit's own INSERT — CreateUnitWithEdge needs the id
// before the row exists so it can seed the closure for it first (GH-36's own fix, ported).
func (s *Store) MintUnitID(ctx context.Context) (string, error) {
	var id string
	err := s.q.QueryRow(ctx, `SELECT openfaithmap.new_id(3, 1, 1)`).Scan(&id)
	return id, err
}

// InsertUnit inserts a root unit — the database mints its id. Use InsertUnitWithID when the id must
// be known before the closure is seeded (CreateUnitWithEdge).
func (s *Store) InsertUnit(ctx context.Context, u domain.Unit) (domain.Unit, error) {
	return s.scanUnit(s.q.QueryRow(ctx, `
		INSERT INTO openfaithmap.directory_units (code, name, level, state, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, code, name, level, state, metadata, created_at, updated_at`,
		nullableText(u.Code), u.Name, u.Level, string(orDefaultState(u.State)), orDefaultMetadata(u.Metadata)))
}

// InsertUnitWithID inserts a unit with a caller-supplied id (already minted via MintUnitID).
func (s *Store) InsertUnitWithID(ctx context.Context, id string, u domain.Unit) (domain.Unit, error) {
	return s.scanUnit(s.q.QueryRow(ctx, `
		INSERT INTO openfaithmap.directory_units (id, code, name, level, state, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, code, name, level, state, metadata, created_at, updated_at`,
		id, nullableText(u.Code), u.Name, u.Level, string(orDefaultState(u.State)), orDefaultMetadata(u.Metadata)))
}

func (s *Store) scanUnit(row pgx.Row) (domain.Unit, error) {
	var u domain.Unit
	var code *string
	var state string
	var metadata []byte
	if err := row.Scan(&u.ID, &code, &u.Name, &u.Level, &state, &metadata, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Unit{}, domain.ErrUnitNotFound
		}
		return domain.Unit{}, err
	}
	if code != nil {
		u.Code = *code
	}
	u.State = domain.State(state)
	u.Metadata = metadata
	return u, nil
}

// ---------------------------------------------------------------- graphs

func (s *Store) GetGraphByCode(ctx context.Context, code string) (domain.Graph, error) {
	var g domain.Graph
	err := s.q.QueryRow(ctx, `
		SELECT id, code, name, is_default, is_authority_bearing
		FROM openfaithmap.directory_graphs
		WHERE code = $1 AND deleted_at IS NULL`, code,
	).Scan(&g.ID, &g.Code, &g.Name, &g.IsDefault, &g.IsAuthorityBearing)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Graph{}, domain.ErrGraphNotFound
	}
	return g, err
}

func (s *Store) GetGraphByID(ctx context.Context, id string) (domain.Graph, error) {
	var g domain.Graph
	err := s.q.QueryRow(ctx, `
		SELECT id, code, name, is_default, is_authority_bearing
		FROM openfaithmap.directory_graphs
		WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&g.ID, &g.Code, &g.Name, &g.IsDefault, &g.IsAuthorityBearing)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Graph{}, domain.ErrGraphNotFound
	}
	return g, err
}

// ListGraphIDs returns every non-deleted graph's (id, code) — RebuildClosure/VerifyClosure's "all
// graphs" fan-out when no specific graph code is given.
func (s *Store) ListGraphIDs(ctx context.Context) ([]domain.Graph, error) {
	rows, err := s.q.Query(ctx, `
		SELECT id, code, name, is_default, is_authority_bearing
		FROM openfaithmap.directory_graphs WHERE deleted_at IS NULL ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Graph
	for rows.Next() {
		var g domain.Graph
		if err := rows.Scan(&g.ID, &g.Code, &g.Name, &g.IsDefault, &g.IsAuthorityBearing); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------- edges + the row lock

// LockGraphForClosure serializes closure maintenance on one graph for the rest of the caller's
// transaction (attach/detach/rebuild all take it first, before touching edges or closure rows —
// concurrent maintenance could otherwise miss compound paths through each other's edges, and the
// guard-then-insert cycle check in AddEdge needs it too). A ROW lock (FOR NO KEY UPDATE on
// directory_graphs), not the session-level advisory lock internal/platform/db.WithAdvisoryLock
// provides for boot seeding — a different mechanism for a different reason
// (docs/architecture/decisions.md's D-CorePortScope amendment). The returned id is discarded; this
// call exists purely for its locking side effect.
func (s *Store) LockGraphForClosure(ctx context.Context, graphID string) error {
	var id string
	return s.q.QueryRow(ctx, `SELECT id FROM openfaithmap.directory_graphs WHERE id = $1 FOR NO KEY UPDATE`, graphID).Scan(&id)
}

// ClosureHasPath reports whether ancestorID is an ancestor of descendantID in graphID — the cycle
// guard (a new parent->child edge closes a cycle iff the child already reaches the parent).
func (s *Store) ClosureHasPath(ctx context.Context, graphID, ancestorID, descendantID string) (bool, error) {
	var reachable bool
	err := s.q.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM openfaithmap.directory_unit_closure
			WHERE graph_id = $1 AND ancestor_id = $2 AND descendant_id = $3
		)`, graphID, ancestorID, descendantID).Scan(&reachable)
	return reachable, err
}

func (s *Store) InsertEdge(ctx context.Context, graphID, parentID, childID string) (domain.Edge, error) {
	var e domain.Edge
	e.GraphID = graphID
	err := s.q.QueryRow(ctx, `
		INSERT INTO openfaithmap.directory_unit_edges (graph_id, parent_id, child_id)
		VALUES ($1, $2, $3)
		RETURNING id, parent_id, child_id, created_at`, graphID, parentID, childID,
	).Scan(&e.ID, &e.ParentID, &e.ChildID, &e.CreatedAt)
	return e, err
}

// DeleteEdge hard-deletes the edge, returning the number of rows removed (0 or 1 — the unique index
// on (graph_id, parent_id, child_id) makes more impossible). valid_from/valid_to on
// directory_unit_edges stay vestigial exactly as upstream: never touched here or anywhere else —
// removal is a real DELETE, not a soft valid_to update.
func (s *Store) DeleteEdge(ctx context.Context, graphID, parentID, childID string) (int64, error) {
	tag, err := s.q.Exec(ctx, `
		DELETE FROM openfaithmap.directory_unit_edges
		WHERE graph_id = $1 AND parent_id = $2 AND child_id = $3`, graphID, parentID, childID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ---------------------------------------------------------------- closure: incremental extend (attach)

// SeedClosureSelfRows seeds the reflexive (u,u,0) rows for both edge endpoints if either has never
// appeared in an edge before — ExtendClosureForEdge's anc*/desc* joins need them to exist.
func (s *Store) SeedClosureSelfRows(ctx context.Context, graphID, parentID, childID string) error {
	_, err := s.q.Exec(ctx, `
		INSERT INTO openfaithmap.directory_unit_closure (graph_id, ancestor_id, descendant_id, depth)
		VALUES ($1, $2, $2, 0), ($1, $3, $3, 0)
		ON CONFLICT DO NOTHING`, graphID, parentID, childID)
	return err
}

// ExtendClosureForEdge incrementally attaches: every path created by a new parent->child edge is a
// path a->parent, the edge, then child->d, so the affected pairs are exactly
// anc*(parent) x desc*(child) (reflexive rows included via SeedClosureSelfRows, called first). Each
// output pair occurs exactly once (one anc row per ancestor, one dsc row per descendant, by the PK),
// so the multi-row ON CONFLICT is safe; LEAST keeps depth = shortest path. Must run after the cycle
// guard (ClosureHasPath), so acyclicity holds.
func (s *Store) ExtendClosureForEdge(ctx context.Context, graphID, parentID, childID string) error {
	if err := s.SeedClosureSelfRows(ctx, graphID, parentID, childID); err != nil {
		return err
	}
	_, err := s.q.Exec(ctx, `
		INSERT INTO openfaithmap.directory_unit_closure (graph_id, ancestor_id, descendant_id, depth)
		SELECT $1::uuid, anc.ancestor_id, dsc.descendant_id, anc.depth + dsc.depth + 1
		FROM openfaithmap.directory_unit_closure anc
		JOIN openfaithmap.directory_unit_closure dsc
		  ON dsc.graph_id = $1 AND dsc.ancestor_id = $3
		WHERE anc.graph_id = $1 AND anc.descendant_id = $2
		ON CONFLICT (graph_id, ancestor_id, descendant_id)
		DO UPDATE SET depth = LEAST(openfaithmap.directory_unit_closure.depth, EXCLUDED.depth)`,
		graphID, parentID, childID)
	return err
}

// ---------------------------------------------------------------- closure: incremental shrink (detach)

// ShrinkClosureForEdge incrementally detaches, in 3 steps that must run in this order:
//  1. DeleteClosureSlice — delete every pair in the affected slice A x D
//     (A = anc*(parent) ∪ {parent}, D = desc*(child) ∪ {child}), the only pairs a path through the
//     removed edge could have touched.
//  2. RederiveClosureSlice — re-derive that slice from the surviving edges.
//  3. PruneClosureSelfRows — drop the two endpoints' reflexive rows if they no longer appear in any
//     edge, keeping incremental output identical to a from-scratch rebuild's output.
//
// Callers must only invoke this when an edge was actually deleted (DeleteEdge returned > 0) — on a
// no-op delete, skipping the shrink is load-bearing, not an optimization: without the edge, this
// method's slice algebra can no longer assume acyclicity (a surviving child->...->parent path is
// then possible), so the derivation below would be unsound.
func (s *Store) ShrinkClosureForEdge(ctx context.Context, graphID, parentID, childID string) error {
	if err := s.deleteClosureSlice(ctx, graphID, parentID, childID); err != nil {
		return err
	}
	if err := s.rederiveClosureSlice(ctx, graphID, parentID, childID); err != nil {
		return err
	}
	return s.pruneClosureSelfRows(ctx, graphID, parentID, childID)
}

func (s *Store) deleteClosureSlice(ctx context.Context, graphID, parentID, childID string) error {
	_, err := s.q.Exec(ctx, `
		WITH anc AS (
		  SELECT tc.ancestor_id AS u FROM openfaithmap.directory_unit_closure tc
		  WHERE tc.graph_id = $1 AND tc.descendant_id = $2
		  UNION SELECT $2::uuid
		),
		dsc AS (
		  SELECT tc.descendant_id AS u FROM openfaithmap.directory_unit_closure tc
		  WHERE tc.graph_id = $1 AND tc.ancestor_id = $3
		  UNION SELECT $3::uuid
		)
		DELETE FROM openfaithmap.directory_unit_closure tc
		WHERE tc.graph_id = $1
		  AND tc.ancestor_id   IN (SELECT u FROM anc)
		  AND tc.descendant_id IN (SELECT u FROM dsc)`, graphID, parentID, childID)
	return err
}

func (s *Store) rederiveClosureSlice(ctx context.Context, graphID, parentID, childID string) error {
	_, err := s.q.Exec(ctx, `
		WITH RECURSIVE
		anc AS (
		  SELECT tc.ancestor_id AS u FROM openfaithmap.directory_unit_closure tc
		  WHERE tc.graph_id = $1 AND tc.descendant_id = $2
		  UNION SELECT $2::uuid
		),
		dsc AS (
		  SELECT tc.descendant_id AS u FROM openfaithmap.directory_unit_closure tc
		  WHERE tc.graph_id = $1 AND tc.ancestor_id = $3
		  UNION SELECT $3::uuid
		),
		walk AS (
		  SELECT a.u AS ancestor_id, a.u AS node, 0 AS depth FROM anc a
		  UNION ALL
		  SELECT w.ancestor_id, e.child_id, w.depth + 1
		  FROM walk w
		  JOIN openfaithmap.directory_unit_edges e ON e.graph_id = $1 AND e.parent_id = w.node
		  WHERE w.node IN (SELECT u FROM anc)
		),
		pairs AS (
		  SELECT w.ancestor_id, tc.descendant_id, w.depth + tc.depth AS depth
		  FROM walk w
		  JOIN openfaithmap.directory_unit_closure tc ON tc.graph_id = $1 AND tc.ancestor_id = w.node
		  WHERE w.node NOT IN (SELECT u FROM anc)
		    AND tc.descendant_id IN (SELECT u FROM dsc)
		)
		INSERT INTO openfaithmap.directory_unit_closure (graph_id, ancestor_id, descendant_id, depth)
		SELECT $1::uuid, ancestor_id, descendant_id, min(depth)::int
		FROM pairs GROUP BY ancestor_id, descendant_id`, graphID, parentID, childID)
	return err
}

func (s *Store) pruneClosureSelfRows(ctx context.Context, graphID, parentID, childID string) error {
	_, err := s.q.Exec(ctx, `
		DELETE FROM openfaithmap.directory_unit_closure tc
		WHERE tc.graph_id = $1
		  AND tc.ancestor_id = tc.descendant_id
		  AND tc.ancestor_id IN ($2::uuid, $3::uuid)
		  AND NOT EXISTS (
		    SELECT 1 FROM openfaithmap.directory_unit_edges e
		    WHERE e.graph_id = $1 AND (e.parent_id = tc.ancestor_id OR e.child_id = tc.ancestor_id)
		  )`, graphID, parentID, childID)
	return err
}

// ---------------------------------------------------------------- closure: rebuild + verify

func (s *Store) DeleteClosureForGraph(ctx context.Context, graphID string) error {
	_, err := s.q.Exec(ctx, `DELETE FROM openfaithmap.directory_unit_closure WHERE graph_id = $1`, graphID)
	return err
}

// RebuildClosureForGraph recomputes one graph's full transitive closure from its edges. Reflexive
// (g,u,u,0) rows only for units appearing in an edge, then descend; MIN(depth) collapses multi-path
// DAG depths to the shortest. Cycle-free by construction — acyclicity is guarded in Go (AddEdge's
// ClosureHasPath check), not re-verified here.
func (s *Store) RebuildClosureForGraph(ctx context.Context, graphID string) error {
	_, err := s.q.Exec(ctx, `
		WITH RECURSIVE
		nodes AS (
		  SELECT parent_id AS u FROM openfaithmap.directory_unit_edges WHERE graph_id = $1
		  UNION SELECT child_id FROM openfaithmap.directory_unit_edges WHERE graph_id = $1
		),
		reach AS (
		  SELECT u AS ancestor_id, u AS descendant_id, 0 AS depth FROM nodes
		  UNION ALL
		  SELECT r.ancestor_id, e.child_id, r.depth + 1
		  FROM reach r
		  JOIN openfaithmap.directory_unit_edges e ON e.graph_id = $1 AND e.parent_id = r.descendant_id
		)
		INSERT INTO openfaithmap.directory_unit_closure (graph_id, ancestor_id, descendant_id, depth)
		SELECT $1::uuid, ancestor_id, descendant_id, min(depth)::int
		FROM reach GROUP BY ancestor_id, descendant_id`, graphID)
	return err
}

// VerifyClosureForGraph diffs the stored closure against what the edges alone imply, including
// depth (EXCEPT over the full (ancestor_id, descendant_id, depth) triple, so a depth-only drift
// shows as one missing + one extra row for the same pair, not silently ignored). Sample capped at 5
// missing + 5 extra, tagged by kind.
func (s *Store) VerifyClosureForGraph(ctx context.Context, graphID string) (missing, extra int, sample json.RawMessage, err error) {
	err = s.q.QueryRow(ctx, `
		WITH RECURSIVE
		nodes AS (
		  SELECT parent_id AS u FROM openfaithmap.directory_unit_edges WHERE graph_id = $1
		  UNION SELECT child_id FROM openfaithmap.directory_unit_edges WHERE graph_id = $1
		),
		reach AS (
		  SELECT u AS ancestor_id, u AS descendant_id, 0 AS depth FROM nodes
		  UNION ALL
		  SELECT r.ancestor_id, e.child_id, r.depth + 1
		  FROM reach r
		  JOIN openfaithmap.directory_unit_edges e ON e.graph_id = $1 AND e.parent_id = r.descendant_id
		),
		expected AS (
		  SELECT ancestor_id, descendant_id, min(depth)::int AS depth FROM reach GROUP BY ancestor_id, descendant_id
		),
		stored AS (
		  SELECT tc.ancestor_id, tc.descendant_id, tc.depth FROM openfaithmap.directory_unit_closure tc WHERE tc.graph_id = $1
		),
		missing AS (SELECT ancestor_id, descendant_id, depth FROM expected EXCEPT SELECT ancestor_id, descendant_id, depth FROM stored),
		extra   AS (SELECT ancestor_id, descendant_id, depth FROM stored   EXCEPT SELECT ancestor_id, descendant_id, depth FROM expected)
		SELECT
		  (SELECT count(*) FROM missing)::int,
		  (SELECT count(*) FROM extra)::int,
		  (SELECT coalesce(jsonb_agg(s), '[]'::jsonb) FROM (
		     (SELECT 'missing'::text AS kind, ancestor_id, descendant_id FROM missing LIMIT 5)
		     UNION ALL
		     (SELECT 'extra'::text AS kind, ancestor_id, descendant_id FROM extra LIMIT 5)
		   ) s)`, graphID,
	).Scan(&missing, &extra, &sample)
	return missing, extra, sample, err
}

func (s *Store) UpsertClosureStatus(ctx context.Context, graphID string, missing, extra int, inDrift bool, sample json.RawMessage) error {
	_, err := s.q.Exec(ctx, `
		INSERT INTO openfaithmap.directory_closure_status (graph_id, missing_count, extra_count, in_drift, sample, last_checked_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (graph_id) DO UPDATE SET
		  missing_count = EXCLUDED.missing_count, extra_count = EXCLUDED.extra_count,
		  in_drift = EXCLUDED.in_drift, sample = EXCLUDED.sample, last_checked_at = now()`,
		graphID, missing, extra, inDrift, sample)
	return err
}

// ---------------------------------------------------------------- ancestors / descendants

// ListAncestors returns unitID's ancestors in graphID (strict; excludes self), nearest first.
func (s *Store) ListAncestors(ctx context.Context, graphID, unitID string) ([]domain.UnitRef, error) {
	rows, err := s.q.Query(ctx, `
		SELECT u.id, COALESCE(u.code, ''), u.name, c.depth
		FROM openfaithmap.directory_unit_closure c
		JOIN openfaithmap.directory_units u ON u.id = c.ancestor_id AND u.deleted_at IS NULL
		WHERE c.graph_id = $1 AND c.descendant_id = $2 AND c.depth > 0
		ORDER BY c.depth, u.code`, graphID, unitID)
	if err != nil {
		return nil, err
	}
	return scanUnitRefs(rows)
}

// ListDescendants returns unitID's subtree in graphID (strict; excludes self), bounded by limit — a
// simple LIMIT, no keyset cursor: no consumer needs paging yet (nothing calls this module this
// session); add a real cursor if a later milestone actually needs one.
func (s *Store) ListDescendants(ctx context.Context, graphID, unitID string, limit int) ([]domain.UnitRef, error) {
	rows, err := s.q.Query(ctx, `
		SELECT u.id, COALESCE(u.code, ''), u.name, c.depth
		FROM openfaithmap.directory_unit_closure c
		JOIN openfaithmap.directory_units u ON u.id = c.descendant_id AND u.deleted_at IS NULL
		WHERE c.graph_id = $1 AND c.ancestor_id = $2 AND c.depth > 0
		ORDER BY c.descendant_id
		LIMIT $3`, graphID, unitID, limit)
	if err != nil {
		return nil, err
	}
	return scanUnitRefs(rows)
}

func scanUnitRefs(rows pgx.Rows) ([]domain.UnitRef, error) {
	defer rows.Close()
	var out []domain.UnitRef
	for rows.Next() {
		var r domain.UnitRef
		if err := rows.Scan(&r.ID, &r.Code, &r.Name, &r.Depth); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------- domain.ClosurePort (internal/authz)

// IsAncestorOrSelf satisfies internal/authz/domain.ClosurePort structurally — no import of
// internal/authz here (D-InProcessAuthz amendment #4). Identical query to ClosureHasPath; the PDP
// itself already special-cases target==unit before calling this, so no self-row special-casing is
// needed here either (a reflexive row would answer it correctly anyway, if one exists).
func (s *Store) IsAncestorOrSelf(ctx context.Context, graphID, ancestorUnitID, descendantUnitID string) (bool, error) {
	return s.ClosureHasPath(ctx, graphID, ancestorUnitID, descendantUnitID)
}

// IsAuthorityBearing satisfies internal/authz/domain.ClosurePort.
func (s *Store) IsAuthorityBearing(ctx context.Context, graphID string) (bool, error) {
	g, err := s.GetGraphByID(ctx, graphID)
	if err != nil {
		if errors.Is(err, domain.ErrGraphNotFound) {
			return false, nil
		}
		return false, err
	}
	return g.IsAuthorityBearing, nil
}

// ---------------------------------------------------------------- helpers

func nullableText(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func orDefaultState(st domain.State) domain.State {
	if st == "" {
		return domain.StateActive
	}
	return st
}

func orDefaultMetadata(m json.RawMessage) json.RawMessage {
	if len(m) == 0 {
		return json.RawMessage("{}")
	}
	return m
}
