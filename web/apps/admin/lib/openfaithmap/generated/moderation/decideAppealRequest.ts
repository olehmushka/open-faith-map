import { AppealDecision } from "./appealDecision";

export interface IDecideAppealRequest {
    'decision': AppealDecision;
    'note'?: string | null;
}
