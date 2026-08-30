import { IBlockInput } from "./blockInput";

export interface IUpdatePatternRequest {
    'name'?: string | null;
    'description'?: string | null;
    'blocks'?: Array<IBlockInput> | null;
    'sortOrder'?: number | null;
}
