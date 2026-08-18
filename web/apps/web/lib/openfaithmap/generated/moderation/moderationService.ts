import { IAppeal } from "./appeal";
import { IAppealPage } from "./appealPage";
import { AppealStatus } from "./appealStatus";
import { IDecideAppealRequest } from "./decideAppealRequest";
import { IFileAppealRequest } from "./fileAppealRequest";
import { IModerationAction } from "./moderationAction";
import { QueueScope } from "./queueScope";
import { IReportPage } from "./reportPage";
import { ReportStatus } from "./reportStatus";
import { IReverseActionRequest } from "./reverseActionRequest";
import { ITakeActionOnReportRequest } from "./takeActionOnReportRequest";
import { ITakeActionRequest } from "./takeActionRequest";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * Moderator queue, actions, and appeals. moderation.read/moderation.act (docs/modules/moderation.md) both resolve to one live PDP check: does the caller hold platform-moderator's grant on the shared root unit (D-PlatformModerator)? No OpenFaithMap-owned moderator roster — verified live against internal/authz's PDP, same discipline registration/content/discovery already follow.
 *
 */
export interface IModerationService {
    /** List reports in the given queue scope (default PLATFORM). Requires platform-moderator standing. */
    listReports(scope?: QueueScope | null, status?: ReportStatus | null, pageSize?: number | null, pageToken?: string | null): Promise<IReportPage>;
    /**
     * Take an action against a report's target. The moderation_actions row is written before any local effect is applied (D-Moderation's Correction replacement invariant). Marks the report ACTIONED.
     *
     */
    takeActionOnReport(reportId: string, request: ITakeActionOnReportRequest): Promise<IModerationAction>;
    /** Take a proactive action with no prior report (e.g. enforcing D-Exclusions directly). */
    takeAction(request: ITakeActionRequest): Promise<IModerationAction>;
    /**
     * Reverse a prior action within its grace window. Writes a new, append-only REVERSE row and sets reversedByActionId on the original — the original row is never edited or deleted.
     *
     */
    reverseAction(actionId: string, request: IReverseActionRequest): Promise<IModerationAction>;
    /**
     * File an appeal against an action, as the affected congregation admin (verified live via an internal/authz.Require check on the target unit, not a platform-moderator check).
     *
     */
    fileAppeal(actionId: string, request: IFileAppealRequest): Promise<IAppeal>;
    /** List appeals. Requires platform-moderator standing. */
    listAppeals(status?: AppealStatus | null, pageSize?: number | null, pageToken?: string | null): Promise<IAppealPage>;
    /**
     * Decide an appeal. Rejects with Moderation:AppealActorConflict if the caller is the original action's actor — enforced at write time, never left to moderator discipline.
     *
     */
    decideAppeal(appealId: string, request: IDecideAppealRequest): Promise<IAppeal>;
}

export class ModerationService implements IModerationService {
    constructor(private bridge: IHttpApiBridge) {
    }

    /** List reports in the given queue scope (default PLATFORM). Requires platform-moderator standing. */
    public listReports(scope?: QueueScope | null, status?: ReportStatus | null, pageSize?: number | null, pageToken?: string | null): Promise<IReportPage> {
        return this.bridge.call<IReportPage>(
            "ModerationService",
            "listReports",
            "GET",
            "/moderation/v1/reports",
            __undefined,
            __undefined,
            {
                "scope": scope,
                "status": status,
                "pageSize": pageSize,
                "pageToken": pageToken,
            },
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * Take an action against a report's target. The moderation_actions row is written before any local effect is applied (D-Moderation's Correction replacement invariant). Marks the report ACTIONED.
     *
     */
    public takeActionOnReport(reportId: string, request: ITakeActionOnReportRequest): Promise<IModerationAction> {
        return this.bridge.call<IModerationAction>(
            "ModerationService",
            "takeActionOnReport",
            "POST",
            "/moderation/v1/reports/{reportId}/actions",
            request,
            __undefined,
            __undefined,
            [
                reportId,
            ],
            __undefined,
            __undefined
        );
    }

    /** Take a proactive action with no prior report (e.g. enforcing D-Exclusions directly). */
    public takeAction(request: ITakeActionRequest): Promise<IModerationAction> {
        return this.bridge.call<IModerationAction>(
            "ModerationService",
            "takeAction",
            "POST",
            "/moderation/v1/actions",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * Reverse a prior action within its grace window. Writes a new, append-only REVERSE row and sets reversedByActionId on the original — the original row is never edited or deleted.
     *
     */
    public reverseAction(actionId: string, request: IReverseActionRequest): Promise<IModerationAction> {
        return this.bridge.call<IModerationAction>(
            "ModerationService",
            "reverseAction",
            "POST",
            "/moderation/v1/actions/{actionId}/reverse",
            request,
            __undefined,
            __undefined,
            [
                actionId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * File an appeal against an action, as the affected congregation admin (verified live via an internal/authz.Require check on the target unit, not a platform-moderator check).
     *
     */
    public fileAppeal(actionId: string, request: IFileAppealRequest): Promise<IAppeal> {
        return this.bridge.call<IAppeal>(
            "ModerationService",
            "fileAppeal",
            "POST",
            "/moderation/v1/actions/{actionId}/appeals",
            request,
            __undefined,
            __undefined,
            [
                actionId,
            ],
            __undefined,
            __undefined
        );
    }

    /** List appeals. Requires platform-moderator standing. */
    public listAppeals(status?: AppealStatus | null, pageSize?: number | null, pageToken?: string | null): Promise<IAppealPage> {
        return this.bridge.call<IAppealPage>(
            "ModerationService",
            "listAppeals",
            "GET",
            "/moderation/v1/appeals",
            __undefined,
            __undefined,
            {
                "status": status,
                "pageSize": pageSize,
                "pageToken": pageToken,
            },
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * Decide an appeal. Rejects with Moderation:AppealActorConflict if the caller is the original action's actor — enforced at write time, never left to moderator discipline.
     *
     */
    public decideAppeal(appealId: string, request: IDecideAppealRequest): Promise<IAppeal> {
        return this.bridge.call<IAppeal>(
            "ModerationService",
            "decideAppeal",
            "POST",
            "/moderation/v1/appeals/{appealId}/decide",
            request,
            __undefined,
            __undefined,
            [
                appealId,
            ],
            __undefined,
            __undefined
        );
    }
}
