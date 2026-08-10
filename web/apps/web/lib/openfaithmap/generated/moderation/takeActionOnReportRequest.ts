import { ActionKind } from "./actionKind";

/** Target is derived from the report itself, never re-specified by the caller. */
export interface ITakeActionOnReportRequest {
    'actionKind': ActionKind;
    'reason': string;
}
