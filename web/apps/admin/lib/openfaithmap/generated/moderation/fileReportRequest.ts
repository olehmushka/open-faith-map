import { ReasonCode } from "./reasonCode";
import { TargetKind } from "./targetKind";

export interface IFileReportRequest {
    'targetKind': TargetKind;
    'targetRef': string;
    'reasonCode': ReasonCode;
    'detail'?: string | null;
}
