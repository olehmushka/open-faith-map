-- 0012_congregationimport_run_parameters — supports manually triggering a connector run from the
-- admin UI with run-specific parameters (e.g. osm's countryCodes), docs/modules/congregationimport.md.
--
-- Expand-only (architecture/conventions.md): a nullable jsonb column, NULL for the common
-- no-parameters case (every ua-edr/ar-rnc run, and most osm runs), a JSON object
-- (e.g. {"countryCodes": "UY,PY"}) when the operator actually supplied one. Persisted so
-- listRuns/getRun show what a past run really used, not just its sourceCode.

ALTER TABLE openfaithmap.congregationimport_runs
    ADD COLUMN parameters jsonb;
