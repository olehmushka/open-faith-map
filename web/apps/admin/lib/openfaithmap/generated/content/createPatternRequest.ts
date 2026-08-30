import { IBlockInput } from "./blockInput";

export interface ICreatePatternRequest {
    'name': string;
    'description': string;
    'blocks': Array<IBlockInput>;
    'sortOrder': number;
}
