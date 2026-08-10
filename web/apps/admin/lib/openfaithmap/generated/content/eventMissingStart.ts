export interface IEventMissingStart {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:EventMissingStart";
    'parameters': {
    };
}

export function isEventMissingStart(arg: any): arg is IEventMissingStart {
    return arg && arg.errorName === "Content:EventMissingStart";
}
