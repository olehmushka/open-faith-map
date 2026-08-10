import { IAppeal } from "./appeal";

export interface IAppealPage {
    'appeals': Array<IAppeal>;
    'nextPageToken'?: string | null;
}
