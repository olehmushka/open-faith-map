import { AppealStatus } from "./appealStatus";

export interface IAppeal {
    /** OpenFaithMap-local RID (openfaithmap.moderation.appeal). */
    'id': string;
    'actionId': string;
    'congregationAdminPersonId': string;
    'statement': string;
    /** Never equal to the original action's actorPersonId. */
    'assignedModeratorPersonId'?: string | null;
    'status': AppealStatus;
    'createdAt': string;
    'updatedAt': string;
}
