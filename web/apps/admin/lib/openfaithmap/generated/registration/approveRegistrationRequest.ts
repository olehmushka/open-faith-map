export interface IApproveRegistrationRequest {
    /**
     * Short, unique go-oikumenea unit code. Defaults to a slug derived from congregationName + a short random suffix if omitted.
     *
     */
    'unitCode'?: string | null;
}
