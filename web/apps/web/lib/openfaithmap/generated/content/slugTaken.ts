export interface ISlugTaken {
    'errorCode': "CONFLICT";
    'errorInstanceId': string;
    'errorName': "Content:SlugTaken";
    'parameters': {
        slug: string;
        scope: string;
    };
}

export function isSlugTaken(arg: any): arg is ISlugTaken {
    return arg && arg.errorName === "Content:SlugTaken";
}
