/** M12.2 — moveUnit's request. graphCode defaults to "canonical" when unset. */
export interface IMoveUnitRequest {
    'newParentUnitId': string;
    'graphCode'?: string | null;
}
