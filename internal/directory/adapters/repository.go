// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package adapters is the directory module's Postgres store: unit/graph/edge CRUD plus the closure
// maintenance queries ported from ../go-oikumenea/internal/tenant/adapters/queries/tenant.sql
// (renamed tenant_* -> directory_*, org-scoping dropped per D-CorePortScope). sqlc-generated
// (docs/architecture/decisions.md's D-Stack) — queries live in queries/directory.sql, generated code
// in directorysql/.
package adapters

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/olehmushka/open-faith-map/internal/directory/adapters/directorysql"
	"github.com/olehmushka/open-faith-map/internal/directory/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/db"
)

type Repository struct {
	q *directorysql.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{q: directorysql.New(conn)}
}

func nullableText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func fromNullableText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func nullableInt2(l *int16) pgtype.Int2 {
	if l == nil {
		return pgtype.Int2{}
	}
	return pgtype.Int2{Int16: *l, Valid: true}
}

func fromNullableInt2(i pgtype.Int2) *int16 {
	if !i.Valid {
		return nil
	}
	v := i.Int16
	return &v
}

// ---------------------------------------------------------------- units

func (r *Repository) GetUnit(ctx context.Context, id string) (domain.Unit, error) {
	row, err := r.q.GetUnit(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Unit{}, domain.ErrUnitNotFound
	}
	if err != nil {
		return domain.Unit{}, err
	}
	return domain.Unit{ID: row.ID, Code: fromNullableText(row.Code), Name: row.Name, Level: fromNullableInt2(row.Level), State: domain.State(row.State), Metadata: row.Metadata, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

// GetUnitByCode resolves a unit by its stable code — directory_units carries the same
// unique-while-active code index as authz_roles (directory_units_code_active_idx). Used by
// internal/platform/seed.Resolve at boot time instead of a hardcoded UUID constant (the root unit
// is seeded with code='root', migrations/0015_core_seed.sql).
func (r *Repository) GetUnitByCode(ctx context.Context, code string) (domain.Unit, error) {
	row, err := r.q.GetUnitByCode(ctx, nullableText(code))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Unit{}, domain.ErrUnitNotFound
	}
	if err != nil {
		return domain.Unit{}, err
	}
	return domain.Unit{ID: row.ID, Code: fromNullableText(row.Code), Name: row.Name, Level: fromNullableInt2(row.Level), State: domain.State(row.State), Metadata: row.Metadata, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

// MintUnitID mints a Unit RID ahead of the unit's own INSERT — CreateUnitWithEdge needs the id
// before the row exists so it can seed the closure for it first (GH-36's own fix, ported).
func (r *Repository) MintUnitID(ctx context.Context) (string, error) {
	return r.q.MintUnitID(ctx)
}

// InsertUnit inserts a root unit — the database mints its id. Use InsertUnitWithID when the id must
// be known before the closure is seeded (CreateUnitWithEdge).
func (r *Repository) InsertUnit(ctx context.Context, u domain.Unit) (domain.Unit, error) {
	row, err := r.q.InsertUnit(ctx, directorysql.InsertUnitParams{
		Code: nullableText(u.Code), Name: u.Name, Level: nullableInt2(u.Level), State: string(orDefaultState(u.State)), Metadata: orDefaultMetadata(u.Metadata),
	})
	if err != nil {
		return domain.Unit{}, err
	}
	return domain.Unit{ID: row.ID, Code: fromNullableText(row.Code), Name: row.Name, Level: fromNullableInt2(row.Level), State: domain.State(row.State), Metadata: row.Metadata, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

// InsertUnitWithID inserts a unit with a caller-supplied id (already minted via MintUnitID).
func (r *Repository) InsertUnitWithID(ctx context.Context, id string, u domain.Unit) (domain.Unit, error) {
	row, err := r.q.InsertUnitWithID(ctx, directorysql.InsertUnitWithIDParams{
		ID: id, Code: nullableText(u.Code), Name: u.Name, Level: nullableInt2(u.Level), State: string(orDefaultState(u.State)), Metadata: orDefaultMetadata(u.Metadata),
	})
	if err != nil {
		return domain.Unit{}, err
	}
	return domain.Unit{ID: row.ID, Code: fromNullableText(row.Code), Name: row.Name, Level: fromNullableInt2(row.Level), State: domain.State(row.State), Metadata: row.Metadata, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

// UpdateUnit rewrites id's name/code/level (M12.1) — metadata/state are left to their own dedicated
// callers (this method always sets all three).
func (r *Repository) UpdateUnit(ctx context.Context, id, name string, code *string, level *int16) (domain.Unit, error) {
	codeArg := pgtype.Text{}
	if code != nil {
		codeArg = pgtype.Text{String: *code, Valid: true}
	}
	row, err := r.q.UpdateUnit(ctx, directorysql.UpdateUnitParams{ID: id, Name: name, Code: codeArg, Level: nullableInt2(level)})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Unit{}, domain.ErrUnitNotFound
	}
	if err != nil {
		return domain.Unit{}, err
	}
	return domain.Unit{ID: row.ID, Code: fromNullableText(row.Code), Name: row.Name, Level: fromNullableInt2(row.Level), State: domain.State(row.State), Metadata: row.Metadata, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

// SetUnitState transitions id to state (M12.1) — active/suspended/archived.
func (r *Repository) SetUnitState(ctx context.Context, id string, state domain.State) (domain.Unit, error) {
	row, err := r.q.SetUnitState(ctx, directorysql.SetUnitStateParams{ID: id, State: string(state)})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Unit{}, domain.ErrUnitNotFound
	}
	if err != nil {
		return domain.Unit{}, err
	}
	return domain.Unit{ID: row.ID, Code: fromNullableText(row.Code), Name: row.Name, Level: fromNullableInt2(row.Level), State: domain.State(row.State), Metadata: row.Metadata, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

// HasChildren reports whether id has any live parent->child edge to a non-deleted unit, in any
// graph (M12.1's orphan-protection for DeleteUnit).
func (r *Repository) HasChildren(ctx context.Context, id string) (bool, error) {
	return r.q.HasChildren(ctx, id)
}

// SoftDeleteUnit sets id's deleted_at (M12.1) — the caller has already checked HasChildren.
func (r *Repository) SoftDeleteUnit(ctx context.Context, id string) (domain.Unit, error) {
	row, err := r.q.SoftDeleteUnit(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Unit{}, domain.ErrUnitNotFound
	}
	if err != nil {
		return domain.Unit{}, err
	}
	return domain.Unit{ID: row.ID, Code: fromNullableText(row.Code), Name: row.Name, Level: fromNullableInt2(row.Level), State: domain.State(row.State), Metadata: row.Metadata, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

// SearchUnits is M10.7's core.conjure.yml ListUnits — a plain ILIKE search over code/name, capped
// at limit (default/max 50).
func (r *Repository) SearchUnits(ctx context.Context, query string, limit int) ([]domain.Unit, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	rows, err := r.q.SearchUnits(ctx, directorysql.SearchUnitsParams{Query: query, LimitCount: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Unit, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Unit{ID: row.ID, Code: fromNullableText(row.Code), Name: row.Name, Level: fromNullableInt2(row.Level), State: domain.State(row.State), Metadata: row.Metadata, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return out, nil
}

// ---------------------------------------------------------------- graphs

func (r *Repository) GetGraphByCode(ctx context.Context, code string) (domain.Graph, error) {
	row, err := r.q.GetGraphByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Graph{}, domain.ErrGraphNotFound
	}
	if err != nil {
		return domain.Graph{}, err
	}
	return domain.Graph{ID: row.ID, Code: row.Code, Name: row.Name, IsDefault: row.IsDefault, IsAuthorityBearing: row.IsAuthorityBearing}, nil
}

func (r *Repository) GetGraphByID(ctx context.Context, id string) (domain.Graph, error) {
	row, err := r.q.GetGraphByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Graph{}, domain.ErrGraphNotFound
	}
	if err != nil {
		return domain.Graph{}, err
	}
	return domain.Graph{ID: row.ID, Code: row.Code, Name: row.Name, IsDefault: row.IsDefault, IsAuthorityBearing: row.IsAuthorityBearing}, nil
}

// ListGraphIDs returns every non-deleted graph's (id, code) — RebuildClosure/VerifyClosure's "all
// graphs" fan-out when no specific graph code is given.
func (r *Repository) ListGraphIDs(ctx context.Context) ([]domain.Graph, error) {
	rows, err := r.q.ListGraphs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Graph, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Graph{ID: row.ID, Code: row.Code, Name: row.Name, IsDefault: row.IsDefault, IsAuthorityBearing: row.IsAuthorityBearing})
	}
	return out, nil
}

// ---------------------------------------------------------------- edges + the row lock

// LockGraphForClosure serializes closure maintenance on one graph for the rest of the caller's
// transaction. The returned id is discarded; this call exists purely for its locking side effect.
func (r *Repository) LockGraphForClosure(ctx context.Context, graphID string) error {
	_, err := r.q.LockGraphForClosure(ctx, graphID)
	return err
}

// ClosureHasPath reports whether ancestorID is an ancestor of descendantID in graphID — the cycle
// guard (a new parent->child edge closes a cycle iff the child already reaches the parent).
func (r *Repository) ClosureHasPath(ctx context.Context, graphID, ancestorID, descendantID string) (bool, error) {
	return r.q.ClosureHasPath(ctx, directorysql.ClosureHasPathParams{GraphID: graphID, AncestorID: ancestorID, DescendantID: descendantID})
}

func (r *Repository) InsertEdge(ctx context.Context, graphID, parentID, childID string) (domain.Edge, error) {
	row, err := r.q.InsertEdge(ctx, directorysql.InsertEdgeParams{GraphID: graphID, ParentID: parentID, ChildID: childID})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "directory_unit_edges_unique" {
			return domain.Edge{}, domain.ErrEdgeExists
		}
		return domain.Edge{}, err
	}
	return domain.Edge{ID: row.ID, GraphID: graphID, ParentID: row.ParentID, ChildID: row.ChildID, CreatedAt: row.CreatedAt}, nil
}

// DeleteEdge hard-deletes the edge, returning the number of rows removed (0 or 1 — the unique index
// on (graph_id, parent_id, child_id) makes more impossible).
func (r *Repository) DeleteEdge(ctx context.Context, graphID, parentID, childID string) (int64, error) {
	return r.q.DeleteEdge(ctx, directorysql.DeleteEdgeParams{GraphID: graphID, ParentID: parentID, ChildID: childID})
}

// ---------------------------------------------------------------- closure: incremental extend (attach)

// SeedClosureSelfRows seeds the reflexive (u,u,0) rows for both edge endpoints if either has never
// appeared in an edge before — ExtendClosureForEdge's anc*/desc* joins need them to exist.
func (r *Repository) SeedClosureSelfRows(ctx context.Context, graphID, parentID, childID string) error {
	return r.q.SeedClosureSelfRows(ctx, directorysql.SeedClosureSelfRowsParams{GraphID: graphID, ParentID: parentID, ChildID: childID})
}

// ExtendClosureForEdge incrementally attaches: every path created by a new parent->child edge is a
// path a->parent, the edge, then child->d, so the affected pairs are exactly
// anc*(parent) x desc*(child) (reflexive rows included via SeedClosureSelfRows, called first). Must
// run after the cycle guard (ClosureHasPath), so acyclicity holds.
func (r *Repository) ExtendClosureForEdge(ctx context.Context, graphID, parentID, childID string) error {
	if err := r.SeedClosureSelfRows(ctx, graphID, parentID, childID); err != nil {
		return err
	}
	return r.q.ExtendClosureForEdge(ctx, directorysql.ExtendClosureForEdgeParams{GraphID: graphID, ParentID: parentID, ChildID: childID})
}

// ---------------------------------------------------------------- closure: incremental shrink (detach)

// ShrinkClosureForEdge incrementally detaches, in 3 steps that must run in this order — see each
// step's own query comment. Callers must only invoke this when an edge was actually deleted
// (DeleteEdge returned > 0).
func (r *Repository) ShrinkClosureForEdge(ctx context.Context, graphID, parentID, childID string) error {
	if err := r.q.DeleteClosureSlice(ctx, directorysql.DeleteClosureSliceParams{GraphID: graphID, ParentID: parentID, ChildID: childID}); err != nil {
		return err
	}
	if err := r.q.RederiveClosureSlice(ctx, directorysql.RederiveClosureSliceParams{GraphID: graphID, ParentID: parentID, ChildID: childID}); err != nil {
		return err
	}
	return r.q.PruneClosureSelfRows(ctx, directorysql.PruneClosureSelfRowsParams{GraphID: graphID, ParentID: parentID, ChildID: childID})
}

// ---------------------------------------------------------------- closure: rebuild + verify

func (r *Repository) DeleteClosureForGraph(ctx context.Context, graphID string) error {
	return r.q.DeleteClosureForGraph(ctx, graphID)
}

// RebuildClosureForGraph recomputes one graph's full transitive closure from its edges.
func (r *Repository) RebuildClosureForGraph(ctx context.Context, graphID string) error {
	return r.q.RebuildClosureForGraph(ctx, graphID)
}

// VerifyClosureForGraph diffs the stored closure against what the edges alone imply, including
// depth. Sample capped at 5 missing + 5 extra, tagged by kind.
func (r *Repository) VerifyClosureForGraph(ctx context.Context, graphID string) (missing, extra int, sample json.RawMessage, err error) {
	row, err := r.q.VerifyClosureForGraph(ctx, graphID)
	if err != nil {
		return 0, 0, nil, err
	}
	return int(row.MissingCount), int(row.ExtraCount), row.Sample, nil
}

func (r *Repository) UpsertClosureStatus(ctx context.Context, graphID string, missing, extra int, inDrift bool, sample json.RawMessage) error {
	return r.q.UpsertClosureStatus(ctx, directorysql.UpsertClosureStatusParams{
		GraphID: graphID, MissingCount: int32(missing), ExtraCount: int32(extra), InDrift: inDrift, Sample: sample,
	})
}

// ---------------------------------------------------------------- ancestors / descendants

// ListAncestors returns unitID's ancestors in graphID (strict; excludes self), nearest first.
func (r *Repository) ListAncestors(ctx context.Context, graphID, unitID string) ([]domain.UnitRef, error) {
	rows, err := r.q.ListAncestors(ctx, directorysql.ListAncestorsParams{GraphID: graphID, UnitID: unitID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.UnitRef, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.UnitRef{ID: row.ID, Code: row.Code, Name: row.Name, Depth: int(row.Depth)})
	}
	return out, nil
}

// ListDescendants returns unitID's subtree in graphID (strict; excludes self), bounded by limit.
func (r *Repository) ListDescendants(ctx context.Context, graphID, unitID string, limit int) ([]domain.UnitRef, error) {
	rows, err := r.q.ListDescendants(ctx, directorysql.ListDescendantsParams{GraphID: graphID, UnitID: unitID, LimitCount: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.UnitRef, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.UnitRef{ID: row.ID, Code: row.Code, Name: row.Name, Depth: int(row.Depth)})
	}
	return out, nil
}

// ---------------------------------------------------------------- move jobs (M12.2)

func toMoveJob(row directorysql.OpenfaithmapDirectoryUnitMoveJob) domain.MoveJob {
	j := domain.MoveJob{
		ID: row.ID, GraphID: row.GraphID, UnitID: row.UnitID, OldParentUnitID: row.OldParentUnitID, NewParentUnitID: row.NewParentUnitID,
		Status: domain.MoveStatus(row.Status), PerformedByPersonID: row.PerformedByPersonID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.Error.Valid {
		j.Error = &row.Error.String
	}
	return j
}

// CreateMoveJob starts a new PENDING move job. Fails with a unique-violation if a non-FAILED job
// already exists for (graphID, unitID) (directory_unit_move_jobs_live_idx).
func (r *Repository) CreateMoveJob(ctx context.Context, graphID, unitID, oldParentUnitID, newParentUnitID, performedByPersonID string) (domain.MoveJob, error) {
	row, err := r.q.CreateMoveJob(ctx, directorysql.CreateMoveJobParams{
		GraphID: graphID, UnitID: unitID, OldParentUnitID: oldParentUnitID, NewParentUnitID: newParentUnitID, PerformedByPersonID: performedByPersonID,
	})
	if err != nil {
		return domain.MoveJob{}, err
	}
	return toMoveJob(row), nil
}

// GetLiveMoveJob returns the current in-progress job for (graphID, unitID), if one exists.
func (r *Repository) GetLiveMoveJob(ctx context.Context, graphID, unitID string) (*domain.MoveJob, error) {
	row, err := r.q.GetLiveMoveJob(ctx, directorysql.GetLiveMoveJobParams{GraphID: graphID, UnitID: unitID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	job := toMoveJob(row)
	return &job, nil
}

// GetLatestMoveJob returns the most recently created job for (graphID, unitID) (FAILED or not).
func (r *Repository) GetLatestMoveJob(ctx context.Context, graphID, unitID string) (*domain.MoveJob, error) {
	row, err := r.q.GetLatestMoveJob(ctx, directorysql.GetLatestMoveJobParams{GraphID: graphID, UnitID: unitID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	job := toMoveJob(row)
	return &job, nil
}

// AdvanceMoveJob moves job id to status (NEW_EDGE_ADDED/OLD_EDGE_REMOVED/VERIFIED), clearing any
// prior error.
func (r *Repository) AdvanceMoveJob(ctx context.Context, id string, status domain.MoveStatus) (domain.MoveJob, error) {
	row, err := r.q.AdvanceMoveJob(ctx, directorysql.AdvanceMoveJobParams{ID: id, Status: string(status)})
	if err != nil {
		return domain.MoveJob{}, err
	}
	return toMoveJob(row), nil
}

// FailMoveJob moves job id to FAILED with the given error message.
func (r *Repository) FailMoveJob(ctx context.Context, id, errMsg string) (domain.MoveJob, error) {
	row, err := r.q.FailMoveJob(ctx, directorysql.FailMoveJobParams{ID: id, Error: nullableText(errMsg)})
	if err != nil {
		return domain.MoveJob{}, err
	}
	return toMoveJob(row), nil
}

// ---------------------------------------------------------------- domain.ClosurePort (internal/authz)

// IsAncestorOrSelf satisfies internal/authz/domain.ClosurePort structurally — no import of
// internal/authz here (D-InProcessAuthz amendment #4). Identical query to ClosureHasPath.
func (r *Repository) IsAncestorOrSelf(ctx context.Context, graphID, ancestorUnitID, descendantUnitID string) (bool, error) {
	return r.ClosureHasPath(ctx, graphID, ancestorUnitID, descendantUnitID)
}

// IsAuthorityBearing satisfies internal/authz/domain.ClosurePort.
func (r *Repository) IsAuthorityBearing(ctx context.Context, graphID string) (bool, error) {
	g, err := r.GetGraphByID(ctx, graphID)
	if err != nil {
		if errors.Is(err, domain.ErrGraphNotFound) {
			return false, nil
		}
		return false, err
	}
	return g.IsAuthorityBearing, nil
}

// ---------------------------------------------------------------- helpers

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
