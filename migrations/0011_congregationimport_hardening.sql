-- 0011_congregationimport_hardening — production-hardening pass on M8 (docs/milestones.md),
-- docs/modules/congregationimport.md.
--
-- Two new composite indexes backing ListCandidates/ListRuns's new real keyset pagination — the
-- (created_at, id) / (started_at, id) tiebreak the cursor now needs, on top of the
-- (status)/(source_code, started_at) prefix the existing
-- congregationimport_candidates_status_idx/congregationimport_runs_source_idx already cover
-- (0010_congregationimport.sql). Mirrors 0009_hardening.sql's identical fix for
-- moderation_reports/moderation_appeals exactly, including its own reasoning: Postgres can't ALTER
-- INDEX to add a column, so these land as new, differently-named indexes rather than replacing the
-- old ones — expand-only (architecture/conventions.md); dropping the now-prefix-subsumed old
-- indexes is a documented future follow-up, not attempted here. Plain CREATE INDEX, matching every
-- existing index in this repo's migrations (no CONCURRENTLY precedent).

CREATE INDEX congregationimport_candidates_status_created_id_idx
    ON openfaithmap.congregationimport_candidates (status, created_at DESC, id DESC);

CREATE INDEX congregationimport_runs_source_started_id_idx
    ON openfaithmap.congregationimport_runs (source_code, started_at DESC, id DESC);
