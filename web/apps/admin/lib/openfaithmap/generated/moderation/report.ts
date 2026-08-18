import { QueueScope } from "./queueScope";
import { ReasonCode } from "./reasonCode";
import { ReportStatus } from "./reportStatus";
import { TargetKind } from "./targetKind";

export interface IReport {
    /** OpenFaithMap-local RID (openfaithmap.moderation.report). */
    'id': string;
    'targetKind': TargetKind;
    /** The RID of the reported thing — a local content_* RID, or a directory_units RID for CONGREGATION. */
    'targetRef': string;
    'reasonCode': ReasonCode;
    'detail'?: string | null;
    /** Unset when the reporter was anonymous. */
    'reporterPersonId'?: string | null;
    'queueScope': QueueScope;
    'status': ReportStatus;
    'createdAt': string;
    'updatedAt': string;
}
