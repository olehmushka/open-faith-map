export interface IGuarantorRevoked {
    'errorCode': "INVALID_ARGUMENT";
    'errorInstanceId': string;
    'errorName': "Vouching:GuarantorRevoked";
    'parameters': {
        guarantorPersonId: string;
    };
}

export function isGuarantorRevoked(arg: any): arg is IGuarantorRevoked {
    return arg && arg.errorName === "Vouching:GuarantorRevoked";
}
