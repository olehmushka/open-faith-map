export interface IApproveCandidateRequest {
    /**
     * The directory_units RID to create the congregation under (D-JurisdictionUnits precedent) — operator-chosen, never inferred. Omitted = the configured root unit.
     *
     */
    'jurisdictionUnitId'?: string | null;
}
