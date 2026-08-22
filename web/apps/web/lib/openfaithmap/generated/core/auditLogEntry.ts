/**
 * M11.2 — one append-only row of identity_audit_log. targetKind/targetId is an opaque discriminator+ref pair (mirrors Moderation's target_kind/target_ref) since targets span role assignments, instance-admin grants, and accounts today, and will span sessions/persons later. before/after are optional — absent for a create/delete side respectively.
 *
 */
export interface IAuditLogEntry {
    'id': string;
    /** Empty if the acting person was later deleted (the FK is ON DELETE SET NULL). */
    'actorPersonId'?: string | null;
    /** Denormalized at read time; empty alongside actorPersonId. */
    'actorPersonName'?: string | null;
    'action': string;
    'targetKind': string;
    'targetId': string;
    'before'?: any | null;
    'after'?: any | null;
    'createdAt': string;
}
