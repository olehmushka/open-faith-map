export interface IBlockUrlNotAllowed {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:BlockUrlNotAllowed";
    'parameters': {
        blockTypeCode: string;
        position: number;
        field: string;
    };
}

export function isBlockUrlNotAllowed(arg: any): arg is IBlockUrlNotAllowed {
    return arg && arg.errorName === "Content:BlockUrlNotAllowed";
}
