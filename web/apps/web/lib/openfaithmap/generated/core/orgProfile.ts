import { IOrgClassification } from "./orgClassification";

export interface IOrgProfile {
    'unitId': string;
    'orgKindId'?: string | null;
    'shortCode'?: string | null;
    'classifications': Array<IOrgClassification>;
}
