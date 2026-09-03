export interface IFormSubmissionInvalid {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:FormSubmissionInvalid";
    'parameters': {
        field: string;
    };
}

export function isFormSubmissionInvalid(arg: any): arg is IFormSubmissionInvalid {
    return arg && arg.errorName === "Content:FormSubmissionInvalid";
}
