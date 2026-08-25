export interface IInvalidUnitState {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Core:InvalidUnitState";
    'parameters': {
    };
}

export function isInvalidUnitState(arg: any): arg is IInvalidUnitState {
    return arg && arg.errorName === "Core:InvalidUnitState";
}
