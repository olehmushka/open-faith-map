export interface ICreateVouchRequest {
    'claimantPersonId': string;
    'congregationUnitId': string;
    /**
     * The unit the CALLER (guarantor) proves their own religionorg.manage standing on — deliberately independent of congregationUnitId, the claim being vouched for. See this file's header comment.
     *
     */
    'guarantorCongregationUnitId': string;
    'statement'?: string | null;
}
