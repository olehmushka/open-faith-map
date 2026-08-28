-- 0026_content_manage_permission — M14.9 (docs/architecture/decisions.md#d-tenantsubdomains,
-- U16 ruling). content.manage becomes its own grantable permission code, so a congregation's site
-- content write access stops being a byproduct of registration-operator's religionorg.manage
-- subtree grant on the shared root — once content_sites.slug is a hostname, "any operator can edit
-- any congregation's website" is a materially different stake than it was when it was just an
-- unlinked blob of blocks.
--
-- Granted to congregation-admin only, same unit-scoped shape as its existing site.manage grant
-- (migrations/0015_core_seed.sql). registration-operator's existing grants (including
-- religionorg.manage, still needed for approving registrations and other subtree-scoped work) are
-- deliberately untouched here — the fix is entirely in which permission code
-- internal/content/application/authorize.go checks, not in revoking anything from that role.
INSERT INTO openfaithmap.authz_role_permissions (role_id, permission_code)
VALUES ((SELECT id FROM openfaithmap.authz_roles WHERE code = 'congregation-admin'), 'content.manage');
