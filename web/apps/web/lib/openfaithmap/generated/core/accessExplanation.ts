import { IAccessExplanationContribution } from "./accessExplanationContribution";

/**
 * M12.4 — explainAccess's return value; mirrors internal/authz/domain.Decision exactly. via is empty when allow is false; denyReason is the empty string when allow is true.
 *
 */
export interface IAccessExplanation {
    'allow': boolean;
    'via': Array<IAccessExplanationContribution>;
    'denyReason': string;
}
