export interface IUnitHasOrgProfile {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Core:UnitHasOrgProfile";
    'parameters': {
        unitId: string;
    };
}

export function isUnitHasOrgProfile(arg: any): arg is IUnitHasOrgProfile {
    return arg && arg.errorName === "Core:UnitHasOrgProfile";
}
