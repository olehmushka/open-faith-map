export interface ISubtreeGrantRequiresGraph {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Core:SubtreeGrantRequiresGraph";
    'parameters': {
    };
}

export function isSubtreeGrantRequiresGraph(arg: any): arg is ISubtreeGrantRequiresGraph {
    return arg && arg.errorName === "Core:SubtreeGrantRequiresGraph";
}
