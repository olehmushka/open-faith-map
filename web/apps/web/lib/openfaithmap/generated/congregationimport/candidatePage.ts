import { ICandidate } from "./candidate";

export interface ICandidatePage {
    'candidates': Array<ICandidate>;
    'nextPageToken'?: string | null;
}
