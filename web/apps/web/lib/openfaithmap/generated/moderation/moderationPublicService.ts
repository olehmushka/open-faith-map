import { IExclusionCheckRequest } from "./exclusionCheckRequest";
import { IExclusionCheckResult } from "./exclusionCheckResult";
import { IFileReportRequest } from "./fileReportRequest";
import { IReport } from "./report";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * Anonymous public surface (openfaithmap-web holds no session — D-AdminSurface): file a report against any target, and run the D-Exclusions taxon check ahead of registration as a standalone dry-run. See docs/modules/moderation.md.
 *
 */
export interface IModerationPublicService {
    /** File a report. reporterPersonId is unset — this endpoint never asks for identity. */
    fileReport(request: IFileReportRequest): Promise<IReport>;
    /**
     * Dry-run the D-Exclusions taxon check via the server's own service-principal token — the caller here is anonymous and has no token of its own to forward. Re-run at registration time itself (internal/registration), never cached from a prior visit here.
     *
     */
    checkExclusion(request: IExclusionCheckRequest): Promise<IExclusionCheckResult>;
}

export class ModerationPublicService implements IModerationPublicService {
    constructor(private bridge: IHttpApiBridge) {
    }

    /** File a report. reporterPersonId is unset — this endpoint never asks for identity. */
    public fileReport(request: IFileReportRequest): Promise<IReport> {
        return this.bridge.call<IReport>(
            "ModerationPublicService",
            "fileReport",
            "POST",
            "/moderation/v1/reports",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * Dry-run the D-Exclusions taxon check via the server's own service-principal token — the caller here is anonymous and has no token of its own to forward. Re-run at registration time itself (internal/registration), never cached from a prior visit here.
     *
     */
    public checkExclusion(request: IExclusionCheckRequest): Promise<IExclusionCheckResult> {
        return this.bridge.call<IExclusionCheckResult>(
            "ModerationPublicService",
            "checkExclusion",
            "POST",
            "/moderation/v1/exclusion-check",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }
}
