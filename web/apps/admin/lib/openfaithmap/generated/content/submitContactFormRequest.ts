/**
 * honeypot and formRenderedAt are anti-spam signals, not content: a non-empty honeypot (a field a real visitor never sees or fills) or a formRenderedAt too close to now() causes the submission to be silently discarded — this endpoint returns success either way, so a probing bot learns nothing (D-InAppInbox's "an error teaches the bot").
 *
 */
export interface ISubmitContactFormRequest {
    'name'?: string | null;
    'email'?: string | null;
    'message': string;
    'honeypot'?: string | null;
    'formRenderedAt': string;
}
