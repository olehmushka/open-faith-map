/**
 * M11.5 — no personId field here, deliberately: the backend always updates the caller's own person row, resolved from the already-verified request subject, never a client-supplied id.
 *
 */
export interface IUpdateMyProfileRequest {
    'displayName': string;
}
