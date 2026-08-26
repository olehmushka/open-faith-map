export interface IUnitHasNoCurrentParent {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Core:UnitHasNoCurrentParent";
    'parameters': {
        unitId: string;
    };
}

export function isUnitHasNoCurrentParent(arg: any): arg is IUnitHasNoCurrentParent {
    return arg && arg.errorName === "Core:UnitHasNoCurrentParent";
}
