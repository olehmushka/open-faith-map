-- 0009_hardening — M7 (docs/milestones.md), docs/modules/hardening.md, D-Hardening.
--
-- Two new composite indexes backing the moderation-queue keyset pagination fix — the (created_at,
-- id) tiebreak the real cursor now needs, on top of the (queue_scope, status)/(status) prefix the
-- existing moderation_reports_scope_status_idx/moderation_appeals_status_idx already cover
-- (0007_moderation.sql). Postgres can't ALTER INDEX to add a column, so these land as new,
-- differently-named indexes rather than replacing the old ones — expand-only
-- (architecture/conventions.md); dropping the now-prefix-subsumed old indexes is a documented
-- future follow-up, not attempted here. Plain CREATE INDEX, not CONCURRENTLY — no
-- txmode/transaction-wrapper precedent exists anywhere in this repo's migrations, and every
-- existing CREATE INDEX (0007/0008) is plain, inside the migration's default transaction; matched
-- here rather than introducing a new convention for a low-traffic table.

CREATE INDEX moderation_reports_scope_status_created_id_idx
    ON openfaithmap.moderation_reports (queue_scope, status, created_at DESC, id DESC);

CREATE INDEX moderation_appeals_status_created_id_idx
    ON openfaithmap.moderation_appeals (status, created_at DESC, id DESC);
