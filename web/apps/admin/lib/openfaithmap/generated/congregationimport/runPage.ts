import { IImportRun } from "./importRun";

export interface IRunPage {
    'runs': Array<IImportRun>;
    'nextPageToken'?: string | null;
}
