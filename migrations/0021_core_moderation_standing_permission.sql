-- 0021_core_moderation_standing_permission — M12.0. Splits platform-moderator's identity-marker
-- permission out of unit.lifecycle into its own dedicated moderation.standing code.
--
-- D-PlatformModerator's M5 addendum granted platform-moderator unit.lifecycle purely as a PDP marker
-- ("does this caller hold platform-moderator standing"), reused only because go-oikumenea's
-- permission catalog was closed pre-port and couldn't mint a new moderation.* code. That constraint
-- is gone now that the catalog is this repo's own Go code (D-InProcessAuthz). Left unsplit,
-- unit.lifecycle would do double duty once M12.1 wires it to a real setUnitState/deleteUnit
-- endpoint — silently handing every platform-moderator archive/suspend/delete power over every unit
-- under root, as a side effect of two unrelated features sharing one permission code, not an
-- intended widening of moderator authority. registration-operator's unit.edges.manage is
-- deliberately left untouched here — its scope question is deferred to M12.2, when
-- internal/registration.Service.Reparent is actually refactored to call the new generic Move.

DELETE FROM openfaithmap.authz_role_permissions
  WHERE role_id = '01989e26-ce03-8101-8201-03a4b1becbd8' AND permission_code = 'unit.lifecycle';

INSERT INTO openfaithmap.authz_role_permissions (role_id, permission_code) VALUES
  ('01989e26-ce03-8101-8201-03a4b1becbd8', 'moderation.standing');
