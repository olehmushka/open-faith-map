export interface ITranslationLocaleTaken {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Content:TranslationLocaleTaken";
    'parameters': {
        translationGroupId: string;
        locale: string;
    };
}

export function isTranslationLocaleTaken(arg: any): arg is ITranslationLocaleTaken {
    return arg && arg.errorName === "Content:TranslationLocaleTaken";
}
