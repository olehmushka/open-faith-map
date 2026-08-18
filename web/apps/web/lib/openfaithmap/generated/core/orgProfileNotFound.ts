export interface IOrgProfileNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Core:OrgProfileNotFound";
    'parameters': {
        unitId: string;
    };
}

export function isOrgProfileNotFound(arg: any): arg is IOrgProfileNotFound {
    return arg && arg.errorName === "Core:OrgProfileNotFound";
}
