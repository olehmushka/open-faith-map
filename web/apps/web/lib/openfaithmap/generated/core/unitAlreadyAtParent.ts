export interface IUnitAlreadyAtParent {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Core:UnitAlreadyAtParent";
    'parameters': {
        unitId: string;
    };
}

export function isUnitAlreadyAtParent(arg: any): arg is IUnitAlreadyAtParent {
    return arg && arg.errorName === "Core:UnitAlreadyAtParent";
}
