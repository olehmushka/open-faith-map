export interface IDocumentNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Content:DocumentNotFound";
    'parameters': {
        documentId: string;
    };
}

export function isDocumentNotFound(arg: any): arg is IDocumentNotFound {
    return arg && arg.errorName === "Content:DocumentNotFound";
}
