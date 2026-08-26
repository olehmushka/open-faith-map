export interface IEdgeCycle {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Core:EdgeCycle";
    'parameters': {
        unitId: string;
    };
}

export function isEdgeCycle(arg: any): arg is IEdgeCycle {
    return arg && arg.errorName === "Core:EdgeCycle";
}
