/** The shape Ancestors returns — enough to render a breadcrumb without a second round trip. */
export interface IUnitRef {
    'id': string;
    'code': string;
    'name': string;
    'depth': number;
}
