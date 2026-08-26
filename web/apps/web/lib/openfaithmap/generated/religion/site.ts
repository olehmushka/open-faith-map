import { ISiteAttributes } from "./siteAttributes";

export interface ISite {
    'id': string;
    'orgUnitId': string;
    'locationId': string;
    'siteTypeId': string;
    'siteTypeCode': string;
    'siteTypeName': string;
    'visibility': string;
    'publicPrecision': string;
    'isPrimary': boolean;
    'latitude': number | "NaN";
    'longitude': number | "NaN";
    'attributes': ISiteAttributes;
}
