import { ICoordinate } from "./coordinate";

export interface ISubmitRegistrationRequest {
    'taxonId': string;
    'congregationName': string;
    'countryId': string;
    'adminArea1'?: string | null;
    'locality'?: string | null;
    'street'?: string | null;
    'houseNumber'?: string | null;
    'postalCode'?: string | null;
    'coordinate': ICoordinate;
}
