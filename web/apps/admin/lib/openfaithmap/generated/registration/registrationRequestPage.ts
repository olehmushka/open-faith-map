import { IRegistrationRequest } from "./registrationRequest";

export interface IRegistrationRequestPage {
    'requests': Array<IRegistrationRequest>;
    'nextPageToken'?: string | null;
}
