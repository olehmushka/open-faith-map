/**
 * M11.3 — issuer is deliberately NOT a field here: the backend records the caller's own already-verified bearer issuer (authz.Subject.Issuer), not a client-supplied value.
 *
 */
export interface IRegisterSessionRequest {
    'deviceLabel'?: string | null;
}
