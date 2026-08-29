export interface IDuplicateNavItemSortOrder {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Content:DuplicateNavItemSortOrder";
    'parameters': {
        sortOrder: number;
    };
}

export function isDuplicateNavItemSortOrder(arg: any): arg is IDuplicateNavItemSortOrder {
    return arg && arg.errorName === "Content:DuplicateNavItemSortOrder";
}
