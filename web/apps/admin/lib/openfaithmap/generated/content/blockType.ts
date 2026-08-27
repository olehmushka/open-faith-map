import { BlockTypeStatus } from "./blockTypeStatus";

export interface IBlockType {
    'id': string;
    'code': string;
    'name': string;
    'jsonSchema': any;
    'uiSchema': any;
    'status': BlockTypeStatus;
    'sortOrder': number;
}
