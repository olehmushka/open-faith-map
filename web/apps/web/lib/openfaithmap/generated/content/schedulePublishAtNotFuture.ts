export interface ISchedulePublishAtNotFuture {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:SchedulePublishAtNotFuture";
    'parameters': {
    };
}

export function isSchedulePublishAtNotFuture(arg: any): arg is ISchedulePublishAtNotFuture {
    return arg && arg.errorName === "Content:SchedulePublishAtNotFuture";
}
