/**
 * M14.16, D-InAppInbox. An anonymous contact-form entry, read through openfaithmap-admin's Messages screen. name/email are whatever the visitor entered, or absent — never validated beyond presence. message is untrusted plain text, rendered as such, never as a block or rich text.
 *
 */
export interface IContactFormSubmission {
    'id': string;
    'name'?: string | null;
    'email'?: string | null;
    'message': string;
    'createdAt': string;
}
