export interface IScheduleMissingPublishAt {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:ScheduleMissingPublishAt";
    'parameters': {
    };
}

export function isScheduleMissingPublishAt(arg: any): arg is IScheduleMissingPublishAt {
    return arg && arg.errorName === "Content:ScheduleMissingPublishAt";
}
