import { BlockTypeStatus } from "./blockTypeStatus";

/**
 * M14.13 owner decision: jsonSchema/uiSchema are deliberately absent here — a block type's schema is locked after creation, so a runtime catalog edit can never silently break already-saved blocks of that type or the admin form built from its old schema. A moderator wanting a different shape retires the old type (status) and creates a new one.
 *
 */
export interface IUpdateBlockTypeRequest {
    'name'?: string | null;
    'status'?: BlockTypeStatus | null;
    'sortOrder'?: number | null;
}
