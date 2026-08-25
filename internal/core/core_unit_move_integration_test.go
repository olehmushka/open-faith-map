// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Proves M12.2's generic MoveUnit against a real Postgres instance: D-UnitMoveDualScope's dual-parent
// unit.edges.manage gate actually denies a missing or one-sided grant, and — the direct proof that
// U14's real subtree-grant-provisioning fix works — a scope="subtree" grant on a jurisdiction
// ancestor actually reaches BOTH of a descendant unit's old and new parents via the closure table,
// letting the move succeed where a scope="unit" grant on only one side cannot. See
// core_unit_lifecycle_integration_test.go's own header comment for the invocation:
//
//	DATABASE_URL="postgres://openfaithmap:dev@localhost:5432/postgres?sslmode=disable" \
//	  go test ./internal/core/... -run TestMoveUnitIntegration -v
package core_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	auditlogadapters "github.com/olehmushka/open-faith-map/internal/auditlog/adapters"
	auditlogapplication "github.com/olehmushka/open-faith-map/internal/auditlog/application"
	"github.com/olehmushka/open-faith-map/internal/authz"
	authzadapters "github.com/olehmushka/open-faith-map/internal/authz/adapters"
	authzdomain "github.com/olehmushka/open-faith-map/internal/authz/domain"
	coreapplication "github.com/olehmushka/open-faith-map/internal/core/application"
	directoryadapters "github.com/olehmushka/open-faith-map/internal/directory/adapters"
	directoryapplication "github.com/olehmushka/open-faith-map/internal/directory/application"
	directorydomain "github.com/olehmushka/open-faith-map/internal/directory/domain"
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
	religionapplication "github.com/olehmushka/open-faith-map/internal/religion/application"
)

func TestMoveUnitIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL to run against a live Postgres instance")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	dir := directoryapplication.NewService(pool)
	religionSvc := religionapplication.NewService(pool, dir)
	// Unlike this package's other integration tests (noopClosure) — MoveUnit's whole point under test
	// is the subtree-grant closure cascade, so this needs the real directory-backed ClosurePort, the
	// same wiring cmd/openfaithmap-api/register_core.go uses in production.
	authzSvc := authz.NewService(authzdomain.NewPDP(directoryadapters.NewStore(pool)), authzadapters.NewRepository(pool), pool)
	auditLogSvc := auditlogapplication.NewService(auditlogadapters.NewRepository(pool))
	coreApp := coreapplication.NewService(dir, religionSvc, nil, nil, nil, authzSvc, auditLogSvc, pool, seed.RootUnitID)

	var personSubtreeID, personOneSidedID, personUngrantedID string
	var unitIDs []string
	graphID, graphCode := createMoveTestGraph(ctx, t, pool)
	t.Cleanup(func() {
		bg := context.Background()
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log DISABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: disable reject_mutation: %v", err)
		}
		for _, pid := range []string{personSubtreeID, personOneSidedID, personUngrantedID} {
			if pid == "" {
				continue
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_audit_log WHERE actor_person_id = $1`, pid); err != nil {
				t.Errorf("cleanup: delete audit rows for %s: %v", pid, err)
			}
		}
		if _, err := pool.Exec(bg, `ALTER TABLE openfaithmap.identity_audit_log ENABLE TRIGGER identity_audit_log_reject_mutation`); err != nil {
			t.Errorf("cleanup: re-enable reject_mutation: %v", err)
		}
		for _, pid := range []string{personSubtreeID, personOneSidedID, personUngrantedID} {
			if pid == "" {
				continue
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.authz_role_assignments WHERE subject_person_id = $1`, pid); err != nil {
				t.Errorf("cleanup: delete role assignments for %s: %v", pid, err)
			}
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_move_jobs WHERE graph_id = $1`, graphID); err != nil {
			t.Errorf("cleanup: delete move jobs: %v", err)
		}
		for _, pid := range []string{personSubtreeID, personOneSidedID, personUngrantedID} {
			if pid == "" {
				continue
			}
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.identity_persons WHERE id = $1`, pid); err != nil {
				t.Errorf("cleanup: delete person %s: %v", pid, err)
			}
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_closure WHERE graph_id = $1`, graphID); err != nil {
			t.Errorf("cleanup: delete closure rows: %v", err)
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_unit_edges WHERE graph_id = $1`, graphID); err != nil {
			t.Errorf("cleanup: delete edges: %v", err)
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_closure_status WHERE graph_id = $1`, graphID); err != nil {
			t.Errorf("cleanup: delete closure status: %v", err)
		}
		for _, id := range unitIDs {
			if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_units WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: delete unit %s: %v", id, err)
			}
		}
		if _, err := pool.Exec(bg, `DELETE FROM openfaithmap.directory_graphs WHERE id = $1`, graphID); err != nil {
			t.Errorf("cleanup: delete graph: %v", err)
		}
	})

	// ---------------------------------------------------------------- setup: jurisdiction with two
	// children (oldParent, newParent), and the moving unit under oldParent.
	jurisdiction, err := dir.CreateUnit(ctx, directorydomain.Unit{Name: "M12.2 Jurisdiction"})
	if err != nil {
		t.Fatalf("CreateUnit(jurisdiction): %v", err)
	}
	unitIDs = append(unitIDs, jurisdiction.ID)

	oldParent, err := dir.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "M12.2 Old Parent"}, jurisdiction.ID, graphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge(oldParent): %v", err)
	}
	unitIDs = append(unitIDs, oldParent.ID)

	newParent, err := dir.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "M12.2 New Parent"}, jurisdiction.ID, graphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge(newParent): %v", err)
	}
	unitIDs = append(unitIDs, newParent.ID)

	child, err := dir.CreateUnitWithEdge(ctx, directorydomain.Unit{Name: "M12.2 Child"}, oldParent.ID, graphCode)
	if err != nil {
		t.Fatalf("CreateUnitWithEdge(child): %v", err)
	}
	unitIDs = append(unitIDs, child.ID)

	insertPerson := func(name string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO openfaithmap.identity_persons (display_name, given, surname)
			VALUES ($1, 'M12.2', $1) RETURNING id`, name).Scan(&id); err != nil {
			t.Fatalf("insert person %s: %v", name, err)
		}
		return id
	}
	personUngrantedID = insertPerson("Ungranted")
	personOneSidedID = insertPerson("OneSided")
	personSubtreeID = insertPerson("Subtree")

	// ---------------------------------------------------------------- no grant at all -> denied.
	ungrantedCtx := authz.NewContext(ctx, authz.Subject{PersonID: personUngrantedID})
	if _, err := coreApp.MoveUnit(ungrantedCtx, child.ID, newParent.ID, graphCode); !errorsIs(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("MoveUnit(ungranted) = %v, want ErrPermissionDenied", err)
	}

	// ---------------------------------------------------------------- unit.edges.manage on the OLD
	// parent only (scope="unit") satisfies half of D-UnitMoveDualScope's check but not the new-parent
	// half -> still denied. Proves the check is genuinely on BOTH sides, not just one.
	if _, err := authzSvc.GrantUnitRole(ctx, personOneSidedID, seed.RegistrationOperatorRoleID, oldParent.ID, authzdomain.ScopeUnit, "", ""); err != nil {
		t.Fatalf("GrantUnitRole(oneSided, unit scope on oldParent): %v", err)
	}
	oneSidedCtx := authz.NewContext(ctx, authz.Subject{PersonID: personOneSidedID})
	if _, err := coreApp.MoveUnit(oneSidedCtx, child.ID, newParent.ID, graphCode); !errorsIs(err, authzdomain.ErrPermissionDenied) {
		t.Errorf("MoveUnit(one-sided grant, oldParent only) = %v, want ErrPermissionDenied", err)
	}
	// Confirm the one-sided attempt genuinely didn't mutate anything (Require fails before Move is
	// ever called) — child's ancestor chain is still [oldParent, jurisdiction], nearest first.
	if ancestors, err := dir.Ancestors(ctx, child.ID, graphCode); err != nil || len(ancestors) != 2 || ancestors[0].ID != oldParent.ID {
		t.Fatalf("Ancestors(child) after denied one-sided move = %+v, %v, want unchanged [oldParent, jurisdiction]", ancestors, err)
	}

	// ---------------------------------------------------------------- unit.edges.manage at SUBTREE
	// scope on the jurisdiction ancestor reaches BOTH oldParent and newParent (both its descendants) —
	// this is the direct proof that U14's real subtree-grant-provisioning fix works: before M12.2,
	// scope="subtree" was fully implemented in the PDP but unprovisionable through any surface, so
	// this grant could not even be created, let alone reach a non-root target.
	if _, err := authzSvc.GrantUnitRole(ctx, personSubtreeID, seed.RegistrationOperatorRoleID, jurisdiction.ID, authzdomain.ScopeSubtree, graphID, ""); err != nil {
		t.Fatalf("GrantUnitRole(subtree, subtree scope on jurisdiction): %v", err)
	}
	subtreeCtx := authz.NewContext(ctx, authz.Subject{PersonID: personSubtreeID})
	job, err := coreApp.MoveUnit(subtreeCtx, child.ID, newParent.ID, graphCode)
	if err != nil {
		t.Fatalf("MoveUnit(subtree grant): %v", err)
	}
	if job.Status != directorydomain.MoveVerified || job.OldParentUnitID != oldParent.ID || job.NewParentUnitID != newParent.ID {
		t.Fatalf("MoveUnit(subtree grant) result = %+v, want Status=VERIFIED, OldParentUnitID=%s, NewParentUnitID=%s", job, oldParent.ID, newParent.ID)
	}
	if ancestors, err := dir.Ancestors(ctx, child.ID, graphCode); err != nil || len(ancestors) != 2 || ancestors[0].ID != newParent.ID {
		t.Fatalf("Ancestors(child) after successful move = %+v, %v, want [newParent, jurisdiction]", ancestors, err)
	}

	moveRow := mustAuditRow(ctx, t, pool, "MOVE_UNIT", child.ID)
	if moveRow.actorPersonID != personSubtreeID || moveRow.targetKind != "UNIT" {
		t.Errorf("MOVE_UNIT audit row = %+v, want actor=%s target_kind=UNIT", moveRow, personSubtreeID)
	}
	assertJSONField(t, moveRow.before, "parentUnitId", oldParent.ID)
	assertJSONField(t, moveRow.after, "parentUnitId", newParent.ID)

	// ---------------------------------------------------------------- GetUnitMoveStatus mirrors the
	// same job back.
	status, err := coreApp.GetUnitMoveStatus(ctx, child.ID, graphCode)
	if err != nil {
		t.Fatalf("GetUnitMoveStatus: %v", err)
	}
	if status == nil || status.ID != job.ID {
		t.Fatalf("GetUnitMoveStatus = %+v, want the same job (id %s)", status, job.ID)
	}

	// ---------------------------------------------------------------- the root unit refuses every
	// move outright, same hard guard SetUnitState/DeleteUnit already use.
	if _, err := coreApp.MoveUnit(subtreeCtx, seed.RootUnitID, newParent.ID, graphCode); !errorsIs(err, coreapplication.ErrRootUnitProtected) {
		t.Errorf("MoveUnit(root) = %v, want ErrRootUnitProtected", err)
	}
}

// createMoveTestGraph inserts a throwaway authority-bearing graph directly via SQL — mirrors
// internal/directory's own createTestGraph helper (unexported, different package).
func createMoveTestGraph(ctx context.Context, t *testing.T, pool *pgxpool.Pool) (id, code string) {
	t.Helper()
	code = "test-core-move-" + t.Name()
	if err := pool.QueryRow(ctx, `
		INSERT INTO openfaithmap.directory_graphs (id, code, name, is_authority_bearing)
		VALUES (openfaithmap.new_id(3,1,2), $1, $1, true)
		RETURNING id`, code).Scan(&id); err != nil {
		t.Fatalf("createMoveTestGraph: %v", err)
	}
	return id, code
}
