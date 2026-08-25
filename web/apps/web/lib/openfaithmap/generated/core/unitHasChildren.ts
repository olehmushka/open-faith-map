export interface IUnitHasChildren {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Core:UnitHasChildren";
    'parameters': {
        unitId: string;
    };
}

export function isUnitHasChildren(arg: any): arg is IUnitHasChildren {
    return arg && arg.errorName === "Core:UnitHasChildren";
}
