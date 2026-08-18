export interface IInstanceAdminGrantNotFound {
    'errorCode': "NOT_FOUND";
    'errorInstanceId': string;
    'errorName': "Core:InstanceAdminGrantNotFound";
    'parameters': {
        personId: string;
    };
}

export function isInstanceAdminGrantNotFound(arg: any): arg is IInstanceAdminGrantNotFound {
    return arg && arg.errorName === "Core:InstanceAdminGrantNotFound";
}
