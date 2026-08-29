/**
 * M14.11. One religion_service_schedules row, read live and composed into SiteChrome — never copied into content's own tables (docs/modules/content.md's standing invariant).
 *
 */
export interface IServiceSchedule {
    'dayOfWeek'?: number | null;
    'rrule'?: string | null;
    /** "HH:MM", 24h. Absent for an rrule-only schedule. */
    'startTime'?: string | null;
    'endTime'?: string | null;
    'timezone': string;
    'language'?: string | null;
    'mode': string;
    'meetingUrl'?: string | null;
    'description'?: string | null;
}
