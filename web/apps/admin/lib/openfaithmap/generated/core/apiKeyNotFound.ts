export interface IApiKeyNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Core:ApiKeyNotFound";
    'parameters': {
        apiKeyId: string;
    };
}

export function isApiKeyNotFound(arg: any): arg is IApiKeyNotFound {
    return arg && arg.errorName === "Core:ApiKeyNotFound";
}
