export interface IExclusionCheckResult {
    'excluded': boolean;
    /** The named D-Exclusions taxon code that matched, set only when excluded = true. */
    'excludedTaxonCode'?: string | null;
}
