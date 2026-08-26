/** M12.1 — createUnit's request. General unit creation under a parent, without createChildOrg's religion-profile side effects. */
export interface ICreateUnitRequest {
    'parentUnitId': string;
    'code': string;
    'name': string;
    'level'?: number | null;
}
