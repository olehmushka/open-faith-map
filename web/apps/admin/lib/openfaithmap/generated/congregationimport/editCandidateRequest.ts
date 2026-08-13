export interface IEditCandidateRequest {
    'name'?: string | null;
    'taxonId'?: string | null;
    'countryId'?: string | null;
    'adminArea1'?: string | null;
    'locality'?: string | null;
    'street'?: string | null;
    'houseNumber'?: string | null;
    'postalCode'?: string | null;
    'latitude'?: number | "NaN" | null;
    'longitude'?: number | "NaN" | null;
}
