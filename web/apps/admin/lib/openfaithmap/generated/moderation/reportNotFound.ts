export interface IReportNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Moderation:ReportNotFound";
    'parameters': {
        reportId: string;
    };
}

export function isReportNotFound(arg: any): arg is IReportNotFound {
    return arg && arg.errorName === "Moderation:ReportNotFound";
}
