export interface IUnitNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Core:UnitNotFound";
    'parameters': {
        unitId: string;
    };
}

export function isUnitNotFound(arg: any): arg is IUnitNotFound {
    return arg && arg.errorName === "Core:UnitNotFound";
}
