export interface IThemeContrastFailed {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:ThemeContrastFailed";
    'parameters': {
        accent: string;
        mode: string;
    };
}

export function isThemeContrastFailed(arg: any): arg is IThemeContrastFailed {
    return arg && arg.errorName === "Content:ThemeContrastFailed";
}
