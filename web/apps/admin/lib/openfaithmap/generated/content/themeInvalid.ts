export interface IThemeInvalid {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:ThemeInvalid";
    'parameters': {
        field: string;
    };
}

export function isThemeInvalid(arg: any): arg is IThemeInvalid {
    return arg && arg.errorName === "Content:ThemeInvalid";
}
