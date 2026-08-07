# Module: vouching

> Reads: [glossary](../glossary.md) · [core-integration](core-integration.md) ·
> [moderation](moderation.md)
> Table prefix: `openfaithmap.vouching_*`

## Purpose

A lightweight web-of-trust mechanism: an already-verified congregation admin can vouch that a new
admin genuinely represents the congregation they claim, reducing how often a platform moderator
must manually verify a claim by hand. A vouch **raises trust, never grants authority** — the actual
authority to administer a congregation still comes only from a go-oikumenea role assignment
(D-Vouching). No go-oikumenea equivalent exists; this module is entirely OpenFaithMap-owned.

## Entities & aggregates

- **Vouching edge** — an immutable event-log entry: guarantor vouched for a claimant, at a point
  in time, for a specific congregation claim.
- **Guarantor status** — the mutable overlay recording whether a guarantor is currently trusted.
  Revoking a guarantor triggers moderator review of every vouch they made — it never
  auto-revokes the vouched congregation's access.

## Data model

Conventions per [conventions.md](../architecture/conventions.md).

**`vouching_edges`** (append-only, `reject_mutation()` guarded)
- `id` PK (RID)
- `guarantor_person_rid TEXT NOT NULL` — a go-oikumenea `Person` RID
- `claimant_person_rid TEXT NOT NULL`
- `congregation_unit_rid TEXT NOT NULL` — the congregation the claim is about
- `statement TEXT` — optional free text ("I've known this pastor for five years")
- `created_at`

**`vouching_guarantor_status`** (mutable overlay, one row per guarantor)
- `guarantor_person_rid TEXT PRIMARY KEY`
- `status TEXT NOT NULL DEFAULT 'trusted' CHECK (status IN ('trusted','revoked'))`
- `revoked_at TIMESTAMPTZ`
- `revoked_reason TEXT`
- `revoked_by_person_rid TEXT` — a moderator's RID
- `updated_at`

## Conjure API surface

`VouchingService` (`/vouching/v1`), `api/vouching.conjure.yml`:

| Op | Intent | Perm |
|---|---|---|
| `POST /vouches` | Guarantor vouches for a claimant on a congregation | caller must hold an active `content.manage`-equivalent authority on **some** congregation (proven the same way [content.md](content.md) proves it — a live go-oikumenea authority check), and must not themselves be `revoked` |
| `GET /vouches?claimant=&congregation=` | List vouches for a claim (moderator/support tooling) | `moderation.read` |
| `POST /guarantors/{personRid}/revoke` | Revoke a guarantor | `moderation.act` |
| `GET /guarantors/{personRid}/status` | Read current status | `moderation.read` |

## Dependencies

- **Calls:** [core-integration.md](core-integration.md) (proving the guarantor's own standing
  before accepting a vouch); [moderation.md](moderation.md) (a guarantor revocation queues review
  of their outstanding vouches as `moderation_reports`, `reason_code = guarantor_revoked`, one per
  affected claim).
- **Called by:** the [web-admin](web-admin.md) congregation-claim flow (offers "ask an existing
  admin to vouch for you" as an alternative/supplement to manual moderator verification, requires
  being logged in) and the moderator console.

## Authorization touchpoints

`moderation.read`/`moderation.act` gate the guarantor-management endpoints (same platform-moderator
roster as [moderation.md](moderation.md)). Filing a vouch itself is gated by proof of the
guarantor's *own* standing over a real congregation — proven live against go-oikumenea, never
cached (same rule as [core-integration.md](core-integration.md#invariants)).

## Invariants

- **A vouch is evidence, never authority.** Nothing in this module ever creates or modifies a
  go-oikumenea role assignment. A heavily-vouched claim still requires the claimant to actually
  receive congregation authority through the normal go-oikumenea-mediated flow
  ([core-integration.md](core-integration.md), step 5) — vouching only shapes how much moderator
  scrutiny that grant gets before or after the fact.
- **The vouching graph is append-only.** `vouching_edges` rows are never edited or deleted
  (`reject_mutation()`-guarded); a mistaken vouch is addressed by moderator action on the
  *claim*, not by erasing the vouch.
- **A revoked guarantor's past vouches are never silently invalidated.** Revocation only queues
  moderator review — automatically undoing every downstream effect of a revoked guarantor's vouches
  would be a bigger, riskier action than the original problem (a bad-faith guarantor) usually
  warrants.
- **A guarantor cannot vouch while revoked.** Checked at write time on every `POST /vouches`.

## Open seams

- **Guarantor eligibility threshold** (must the guarantor have been verified for N days, or vouched
  for M prior successful claims, before they can vouch themselves?) is undecided — MVP ships with
  "any admin with current standing on a congregation may vouch," to be tightened once real abuse
  patterns (if any) are observed.
- **Vouching-graph visualization** for moderators (who vouched for whom, transitively) is a
  candidate console feature, not designed yet.
