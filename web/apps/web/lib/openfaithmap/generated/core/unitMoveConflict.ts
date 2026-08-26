export interface IUnitMoveConflict {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Core:UnitMoveConflict";
    'parameters': {
        unitId: string;
    };
}

export function isUnitMoveConflict(arg: any): arg is IUnitMoveConflict {
    return arg && arg.errorName === "Core:UnitMoveConflict";
}
