export interface IVouch {
    /** OpenFaithMap-local uuid (openfaithmap.vouching_edges.id). */
    'id': string;
    /** The go-oikumenea Person RID of the admin who vouched. */
    'guarantorPersonId': string;
    /** The go-oikumenea Person RID of the person being vouched for. */
    'claimantPersonId': string;
    /** The go-oikumenea Unit RID of the congregation the claim is about. */
    'congregationUnitId': string;
    'statement'?: string | null;
    'createdAt': string;
}
