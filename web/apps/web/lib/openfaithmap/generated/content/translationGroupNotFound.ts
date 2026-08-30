export interface ITranslationGroupNotFound {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:TranslationGroupNotFound";
    'parameters': {
        translationGroupId: string;
    };
}

export function isTranslationGroupNotFound(arg: any): arg is ITranslationGroupNotFound {
    return arg && arg.errorName === "Content:TranslationGroupNotFound";
}
