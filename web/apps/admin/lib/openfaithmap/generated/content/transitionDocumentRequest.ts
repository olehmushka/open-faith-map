import { DocumentTransitionAction } from "./documentTransitionAction";

export interface ITransitionDocumentRequest {
    'action': DocumentTransitionAction;
    /**
     * M14.15. Required (and must be in the future) when action is SCHEDULE; ignored for every other action.
     *
     */
    'publishAt'?: string | null;
}
