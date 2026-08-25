export interface IRootUnitProtected {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Core:RootUnitProtected";
    'parameters': {
        unitId: string;
    };
}

export function isRootUnitProtected(arg: any): arg is IRootUnitProtected {
    return arg && arg.errorName === "Core:RootUnitProtected";
}
