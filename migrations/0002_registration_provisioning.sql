-- 0002_registration_provisioning — M2.3 item 3 (docs/milestones.md).
--
-- Adds a PROVISIONING status between PENDING and APPROVED so approveRequest can persist the
-- go-oikumenea unit createChildOrg produced before the remaining writes run — the step that cannot
-- be re-derived on a retry after a crash. Expand-only: widens both CHECK constraints, no data loss.

ALTER TABLE openfaithmap.registration_requests
  DROP CONSTRAINT registration_requests_status_check;

ALTER TABLE openfaithmap.registration_requests
  ADD CONSTRAINT registration_requests_status_check
  CHECK (status IN ('PENDING', 'PROVISIONING', 'APPROVED', 'REJECTED'));

ALTER TABLE openfaithmap.registration_requests
  DROP CONSTRAINT registration_requests_decision_shape;

ALTER TABLE openfaithmap.registration_requests
  ADD CONSTRAINT registration_requests_decision_shape CHECK (
    (status = 'PENDING'      AND decided_by_person_id IS NULL     AND decided_at IS NULL) OR
    (status = 'PROVISIONING' AND decided_by_person_id IS NOT NULL AND created_unit_id IS NOT NULL) OR
    (status = 'APPROVED'     AND decided_by_person_id IS NOT NULL AND decided_at IS NOT NULL AND created_unit_id IS NOT NULL) OR
    (status = 'REJECTED'     AND decided_by_person_id IS NOT NULL AND decided_at IS NOT NULL AND rejection_reason IS NOT NULL)
  );
