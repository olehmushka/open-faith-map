/**
 * M11.9 — the closed unit-scoped permission catalog (internal/authz/domain/permissions.go), excluding instance-scope codes: an API key can never exercise one (RequireInstanceAdmin hard-denies every API-key-authenticated subject, allowlist or not), so offering them in the creation picker would be misleading. Self-scoped, not admin-only: every person needs this for their own createApiKey picker.
 *
 */
export interface IPermissionCodePage {
    'codes': Array<string>;
}
