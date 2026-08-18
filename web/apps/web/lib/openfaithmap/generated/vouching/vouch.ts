export interface IVouch {
    /** OpenFaithMap-local uuid (openfaithmap.vouching_edges.id). */
    'id': string;
    /** The identity_persons RID of the admin who vouched. */
    'guarantorPersonId': string;
    /** The identity_persons RID of the person being vouched for. */
    'claimantPersonId': string;
    /** The directory_units RID of the congregation the claim is about. */
    'congregationUnitId': string;
    'statement'?: string | null;
    'createdAt': string;
}
