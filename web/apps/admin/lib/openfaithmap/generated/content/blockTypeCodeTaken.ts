export interface IBlockTypeCodeTaken {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Content:BlockTypeCodeTaken";
    'parameters': {
        code: string;
    };
}

export function isBlockTypeCodeTaken(arg: any): arg is IBlockTypeCodeTaken {
    return arg && arg.errorName === "Content:BlockTypeCodeTaken";
}
