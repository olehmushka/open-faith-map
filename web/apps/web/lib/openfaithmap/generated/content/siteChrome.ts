import { IServiceSchedule } from "./serviceSchedule";
import { ISocialLinks } from "./socialLinks";

/**
 * M14.11's header/footer bundle, fetched once by the tenant layout. logoUrl/socialLinks are content_sites' own persisted settings; congregationName/address/schedules are composed live from religion_sites/religion_service_schedules at read time — never copied here.
 *
 */
export interface ISiteChrome {
    'congregationName': string;
    /** Coarsened per the religion site's own publish precision; absent if hidden or unset. */
    'address'?: string | null;
    'logoUrl'?: string | null;
    'socialLinks': ISocialLinks;
    'schedules': Array<IServiceSchedule>;
}
