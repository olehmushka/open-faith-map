import { DocumentKind } from "./documentKind";

export interface ICreateDocumentRequest {
    /** M3 only accepts PAGE — POST/EVENT return Content:KindNotSupported until M4. */
    'kind': DocumentKind;
    /** Omit to start a new translation group; set to join an existing one as another locale variant. */
    'translationGroupId'?: string | null;
    'locale': string;
    'parentDocumentId'?: string | null;
    'slug': string;
}
