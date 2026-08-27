export interface IRevisionNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Content:RevisionNotFound";
    'parameters': {
        revisionId: string;
    };
}

export function isRevisionNotFound(arg: any): arg is IRevisionNotFound {
    return arg && arg.errorName === "Content:RevisionNotFound";
}
