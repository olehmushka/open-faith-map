export interface IChildCreationExcluded {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Core:ChildCreationExcluded";
    'parameters': {
        parentUnitId: string;
    };
}

export function isChildCreationExcluded(arg: any): arg is IChildCreationExcluded {
    return arg && arg.errorName === "Core:ChildCreationExcluded";
}
