export interface ICountry {
    'id': string;
    /** ISO-3166-1 alpha-2. */
    'code': string;
    /** English name. */
    'name': string;
    /** locale code -> localized name. */
    'names': { [key: string]: string };
}
