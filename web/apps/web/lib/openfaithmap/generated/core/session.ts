/**
 * M11.3 — one row of identity_sessions: a NextAuth sign-in the backend can revoke independently of every other session on the same account (D-SessionTracking — the reason this exists at all rather than reusing account-status-disable's all-or-nothing shape).
 *
 */
export interface ISession {
    'id': string;
    /** Best-effort User-Agent captured at sign-in; absent if it couldn't be read. */
    'deviceLabel'?: string | null;
    'createdAt': string;
    'lastSeenAt': string;
    /** True for the one session the caller's own request is presently authenticated with. */
    'isCurrent': boolean;
}
