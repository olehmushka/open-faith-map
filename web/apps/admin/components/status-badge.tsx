import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type {
  CandidateStatus,
} from "@/lib/openfaithmap/generated/congregationimport/candidateStatus";
import type { ReportStatus } from "@/lib/openfaithmap/generated/moderation/reportStatus";
import type { AppealStatus } from "@/lib/openfaithmap/generated/moderation/appealStatus";
import type { RegistrationStatus } from "@/lib/openfaithmap/generated/registration/registrationStatus";
import type { ReparentStatus } from "@/lib/openfaithmap/generated/registration/reparentStatus";
import type { GuarantorStatus } from "@/lib/openfaithmap/generated/vouching/guarantorStatus";

export type StatusTone = "neutral" | "info" | "warning" | "success" | "danger";

const TONE_CLASSES: Record<StatusTone, string> = {
  neutral: "border-border bg-muted text-muted-foreground",
  info: "border-transparent bg-info/15 text-info dark:bg-info/25",
  warning: "border-transparent bg-warning/15 text-warning dark:bg-warning/25",
  success: "border-transparent bg-success/15 text-success dark:bg-success/25",
  danger: "border-transparent bg-destructive/10 text-destructive dark:bg-destructive/20",
};

export function StatusBadge({
  status,
  tone,
  className,
}: {
  status: string;
  tone: StatusTone;
  className?: string;
}) {
  return (
    <Badge variant="outline" className={cn(TONE_CLASSES[tone], className)}>
      {status}
    </Badge>
  );
}

const CANDIDATE_TONE: Record<CandidateStatus, StatusTone> = {
  STAGED: "neutral",
  NEEDS_TAXON_REVIEW: "warning",
  NEEDS_GEOCODE: "warning",
  POSSIBLE_DUPLICATE: "danger",
  APPROVED: "success",
  PROVISIONING: "info",
  PROVISIONED: "success",
  REJECTED: "danger",
  REJECTED_EXCLUDED: "neutral",
};

const REPORT_TONE: Record<ReportStatus, StatusTone> = {
  OPEN: "warning",
  ACTIONED: "success",
  DISMISSED: "neutral",
};

// UPHELD/OVERTURNED aren't inherently "good/bad" the way approve/reject is — worth a quick
// confirm with whoever owns moderation UX if this reads wrong in practice.
const APPEAL_TONE: Record<AppealStatus, StatusTone> = {
  OPEN: "warning",
  UPHELD: "info",
  OVERTURNED: "success",
};

const REGISTRATION_TONE: Record<RegistrationStatus, StatusTone> = {
  PENDING: "warning",
  PROVISIONING: "info",
  APPROVED: "success",
  REJECTED: "danger",
};

const REPARENT_TONE: Record<ReparentStatus, StatusTone> = {
  PENDING: "warning",
  NEW_EDGE_ADDED: "info",
  OLD_EDGE_REMOVED: "info",
  VERIFIED: "success",
  FAILED: "danger",
};

const GUARANTOR_TONE: Record<GuarantorStatus, StatusTone> = {
  TRUSTED: "success",
  REVOKED: "danger",
};

// Unit.state (M10.1) is a plain wire string, not a generated conjure enum, unlike every other status
// above — the three values are directorydomain.State's own lowercase constants (M12.5).
const UNIT_TONE: Record<string, StatusTone> = {
  active: "success",
  suspended: "warning",
  archived: "neutral",
};

export const CandidateStatusBadge = ({ status }: { status: CandidateStatus }) => (
  <StatusBadge status={status} tone={CANDIDATE_TONE[status]} />
);
export const ReportStatusBadge = ({ status }: { status: ReportStatus }) => (
  <StatusBadge status={status} tone={REPORT_TONE[status]} />
);
export const AppealStatusBadge = ({ status }: { status: AppealStatus }) => (
  <StatusBadge status={status} tone={APPEAL_TONE[status]} />
);
export const RegistrationStatusBadge = ({ status }: { status: RegistrationStatus }) => (
  <StatusBadge status={status} tone={REGISTRATION_TONE[status]} />
);
export const ReparentStatusBadge = ({ status }: { status: ReparentStatus }) => (
  <StatusBadge status={status} tone={REPARENT_TONE[status]} />
);
export const GuarantorStatusBadge = ({ status }: { status: GuarantorStatus }) => (
  <StatusBadge status={status} tone={GUARANTOR_TONE[status]} />
);
export const UnitStatusBadge = ({ status }: { status: string }) => (
  <StatusBadge status={status} tone={UNIT_TONE[status] ?? "neutral"} />
);
