export interface ISuggestCoordinatesResponse {
    'latitude': number | "NaN";
    'longitude': number | "NaN";
    /** The provider's own reported precision/place type — shown to the operator, never trusted blindly. */
    'precision'?: string | null;
    /** The provider's resolved address, so the operator can sanity-check the match before trusting it. */
    'displayName': string;
    /** Which geocoding provider produced this (its Code()) — shown alongside the suggestion. */
    'provider': string;
}
