export interface IDuplicateBlockPosition {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:DuplicateBlockPosition";
    'parameters': {
        position: number;
    };
}

export function isDuplicateBlockPosition(arg: any): arg is IDuplicateBlockPosition {
    return arg && arg.errorName === "Content:DuplicateBlockPosition";
}
