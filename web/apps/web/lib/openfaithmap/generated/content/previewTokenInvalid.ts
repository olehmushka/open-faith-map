export interface IPreviewTokenInvalid {
    'errorCode': "PERMISSION_DENIED";
    'errorInstanceId': string;
    'errorName': "Content:PreviewTokenInvalid";
    'parameters': {
    };
}

export function isPreviewTokenInvalid(arg: any): arg is IPreviewTokenInvalid {
    return arg && arg.errorName === "Content:PreviewTokenInvalid";
}
