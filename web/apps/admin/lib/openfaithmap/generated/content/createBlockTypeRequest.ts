/** content.catalog.manage (platform-moderator only). Finally builds what M3 left unbuilt. */
export interface ICreateBlockTypeRequest {
    'code': string;
    'name': string;
    'jsonSchema': any;
    'uiSchema': any;
    'sortOrder': number;
}
