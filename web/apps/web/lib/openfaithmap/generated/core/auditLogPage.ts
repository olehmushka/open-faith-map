import { IAuditLogEntry } from "./auditLogEntry";

export interface IAuditLogPage {
    'entries': Array<IAuditLogEntry>;
    'nextPageToken'?: string | null;
}
