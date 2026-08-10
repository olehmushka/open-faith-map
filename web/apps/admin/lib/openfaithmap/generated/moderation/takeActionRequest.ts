import { ActionKind } from "./actionKind";
import { TargetKind } from "./targetKind";

/** A proactive action with no prior report — the caller names the target directly. */
export interface ITakeActionRequest {
    'actionKind': ActionKind;
    'targetKind': TargetKind;
    'targetRef': string;
    'reason': string;
}
