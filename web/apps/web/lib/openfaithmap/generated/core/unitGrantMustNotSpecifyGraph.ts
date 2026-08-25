export interface IUnitGrantMustNotSpecifyGraph {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Core:UnitGrantMustNotSpecifyGraph";
    'parameters': {
    };
}

export function isUnitGrantMustNotSpecifyGraph(arg: any): arg is IUnitGrantMustNotSpecifyGraph {
    return arg && arg.errorName === "Core:UnitGrantMustNotSpecifyGraph";
}
