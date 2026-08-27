export interface IBlockDataInvalid {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:BlockDataInvalid";
    'parameters': {
        blockTypeCode: string;
        position: number;
        field: string;
    };
}

export function isBlockDataInvalid(arg: any): arg is IBlockDataInvalid {
    return arg && arg.errorName === "Content:BlockDataInvalid";
}
