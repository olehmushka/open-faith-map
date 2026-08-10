import { IVouch } from "./vouch";

export interface IVouchPage {
    'vouches': Array<IVouch>;
    'nextPageToken'?: string | null;
}
