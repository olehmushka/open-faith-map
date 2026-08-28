export interface ISlugReserved {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:SlugReserved";
    'parameters': {
        slug: string;
    };
}

export function isSlugReserved(arg: any): arg is ISlugReserved {
    return arg && arg.errorName === "Content:SlugReserved";
}
