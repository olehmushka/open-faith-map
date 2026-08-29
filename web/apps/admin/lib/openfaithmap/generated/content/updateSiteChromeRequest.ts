import { ISocialLinks } from "./socialLinks";

export interface IUpdateSiteChromeRequest {
    'logoUrl'?: string | null;
    'socialLinks': ISocialLinks;
}
