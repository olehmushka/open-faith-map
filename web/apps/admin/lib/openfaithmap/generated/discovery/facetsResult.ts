import { ITraditionFacet } from "./traditionFacet";

export interface IFacetsResult {
    /** Distinct tradition taxa actually classified on at least one public, non-hidden site. */
    'traditions': Array<ITraditionFacet>;
    /** Distinct service-schedule languages actually present on at least one public, non-hidden site. */
    'languages': Array<string>;
}
