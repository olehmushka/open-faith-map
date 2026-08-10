export interface IBlockTypeNotFound {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:BlockTypeNotFound";
    'parameters': {
        blockTypeCode: string;
    };
}

export function isBlockTypeNotFound(arg: any): arg is IBlockTypeNotFound {
    return arg && arg.errorName === "Content:BlockTypeNotFound";
}
