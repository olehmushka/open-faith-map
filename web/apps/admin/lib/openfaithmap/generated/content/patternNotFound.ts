export interface IPatternNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Content:PatternNotFound";
    'parameters': {
        patternId: string;
    };
}

export function isPatternNotFound(arg: any): arg is IPatternNotFound {
    return arg && arg.errorName === "Content:PatternNotFound";
}
