import { ActionKind } from "./actionKind";
import { TargetKind } from "./targetKind";

export interface IModerationAction {
    /** OpenFaithMap-local RID (openfaithmap.moderation.action). */
    'id': string;
    /** Unset for a proactive action taken with no prior report. */
    'reportId'?: string | null;
    'actionKind': ActionKind;
    'targetKind': TargetKind;
    'targetRef': string;
    /** The moderator who took this action (a go-oikumenea Person RID). */
    'actorPersonId': string;
    'reason': string;
    /** Set on the original row once a REVERSE action targets it. */
    'reversedByActionId'?: string | null;
    'createdAt': string;
}
