export interface IApproveRegistrationRequest {
    /**
     * Short, unique go-oikumenea unit code. Defaults to a slug derived from congregationName + a short random suffix if omitted.
     *
     */
    'unitCode'?: string | null;
    /**
     * The go-oikumenea unit RID to create the congregation under (D-JurisdictionUnits, M4.1) — operator-chosen, never inferred from taxonId. Omitted = the current flat-root behavior, unchanged: the congregation is created directly under the configured root unit. On a resumed PROVISIONING retry, the ORIGINAL choice persisted at first approval is reused regardless of what this field carries on the retry call, exactly like unitCode's existing once-PROVISIONING-ignore-further-input discipline.
     *
     */
    'jurisdictionUnitId'?: string | null;
}
