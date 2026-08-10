import { IReport } from "./report";

export interface IReportPage {
    'reports': Array<IReport>;
    'nextPageToken'?: string | null;
}
