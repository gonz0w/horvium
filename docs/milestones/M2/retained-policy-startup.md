# Adopted slice: clean restart with retained ownership policies

Status: adopted in [PR 306](https://github.com/gonz0w/horvium/pull/306); implementation and qualification tracked in [issue 305](https://github.com/gonz0w/horvium/issues/305).
Date: 2026-10-10. Owner: Cam (`@gonz0w`).
Design pickup: [issue 288](https://github.com/gonz0w/horvium/issues/288).
Parent: [M2-RUNTIME-002 gate](https://github.com/gonz0w/horvium/issues/139).
Authority: [PRD](PRD.md), [EDD](EDD.md), [Test Design](TDD.md),
[recovery mechanism](property-recovery-mechanism.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).

## Question and bounded outcome

Which nonempty ledger can the next private startup consumer accept without
assuming executor recovery or releasing retained job holds? This design pickup
is time-boxed to four hours for one review pass over the merged startup consumers,
their contracts and receipts. It includes no runtime implementation or native
experiment.

Propose one implementation outcome: an original clean closed incarnation with
one or more valid member-bearing ownership policies and **zero job records** can
be reviewed, activated once, independently observed and durably retired while
unused. Preserve the existing empty-ledger path. Both paths retain the original
journal-first leases and the accepted publication protocol. Neither grants a
native launch permit or enrolls a new resource.

This addresses the first useful nonempty case: ownership can survive a clean
restart even before any work has been admitted. Any retained job, including a
succeeded, failed or cancelled record, denies this proposed activation and idle
retirement extension. Supporting such jobs needs a separately reviewed history
consumer; a terminal label cannot establish quiescence or release a hold.

GitHub owns live status and dependencies. The proposal and test criteria remain
authoritative in Git; approval and execution evidence must be linked from the
implementation issue rather than inferred from this document's existence.

## Accepted starting point and evidence limits

| Accepted component | Relevant contract and evidence | Limit retained here |
|---|---|---|
| PR 126 original closed review | [Contract](../../testing/M2/execution-startup-review.md), [receipt](../../testing/M2/execution-startup-review-manifest.json) | Native fixtures have empty history; a complete census is not per-policy native validation |
| PR 127 closed-startup activation | [Contract](../../testing/M2/closed-startup-activation.md), [receipt](../../testing/M2/closed-startup-activation-manifest.json) | New active publication requires the exact empty census |
| PR 128 unused restarted retirement | [Contract](../../testing/M2/restart-activation-retirement.md), [receipt](../../testing/M2/restart-activation-retirement-manifest.json) | Clean close requires the exact empty census and an original never-used lifetime |
| Existing property safety/readiness | [Policy binding](../../testing/M2/property-policy-binding.md), [member enrollment](../../testing/M2/property-member-enrollment.md), [store readiness](../../testing/M2/property-store-readiness.md) | Selected-pool observations remain ineligible; legacy policies are readable but cannot pass member safety |
| Existing mixed job projection | [Restart contract and receipt](../../testing/M2/job-restart.md) | Read-only metadata projection; it grants no replay, hold release or recovered authority |

Inspection baseline is PR 128 merge
`f3fc4a34ffa210f9512fde3b88a470943cd2a51f`. In that tree,
`execution_startup_freebsd.go` validates the exact journal receipt, complete
census, boot, objects and system backing. `execution_domains_freebsd.go` reads
the full catalog but returns only domain metadata, not the policies needed for
fresh member/scope validation. `execution_restart_freebsd.go` and
`execution_restart_retirement_freebsd.go` independently require an empty census.
Removing those comparisons alone would bypass the missing policy review.

The existing fixed policy/safety sources validate one policy with native
pool/filesystem, provider/member and system-use facts. Their ordinary ownership
readers acquire the ledger mutex. They cannot simply be invoked while startup
already holds that mutex. The implementation must preserve their checks through
an explicitly borrowed, already-locked read seam; it must neither reopen the
ledger nor drop the original leases to avoid the nested lock.

Historical failures stay attached to their original candidates: PR 126's
same-inode directory drift, PR 127's initial release/teardown omission and
fixed-root deadline exhaustion, and PR 128's initial fixed-root deadline failure
remain unqualified in the linked receipts. Their later successful candidates do
not qualify this proposed nonempty path. The PR 128 storage/fixed-root evidence
used separate checksum-bound artifacts and executed sources, as its receipt
records; do not collapse them into a fictitious single tested candidate.

## Admission boundary

| Retained input | Proposed result |
|---|---|
| Original closed, confirmed empty ledger | Preserve the accepted empty path and its denials |
| Original closed, bounded policies only, every policy passes fresh native checks | Eligible only for this private active/unused-close lifecycle |
| Legacy ownership lacking members; missing, renamed or drifted scope/member; unavailable provider/system facts | Deny the extension; preserve all records and observations available through existing diagnostics |
| Any job in either accepted format, in any lifecycle state or on another pool | Deny; no exception for terminal records or matching properties |
| Active journal on the same or a changed boot | Deny this path; explicit unclean recovery remains separate |
| Metadata/native pending, disarmed restore barrier, inconsistent halves or third census | Deny; do not clear, initialize, repair or replay |
| Unknown/restored provenance or joint journal/ledger rollback | No new clearance rule or anti-rollback claim; retain the accepted recovery limitations |

Require the entire namespace, not a caller-selected pool subset. Keep the existing
ownership count, record size, 64 MiB total byte, journal size and generation bounds.
Do not introduce configuration, public routes, wire fields, persistent formats,
indexes or runtime dependencies for this slice. A valid version-1 ownership record
remains readable through existing diagnostics, but does not become enrolled or
silently migrate. There is no missing-pool import or baseline refresh fallback.

## Original lifecycle and policy validation

1. Reserve the existing single-use root helper registration before opening any
   domain. Acquire and retain the same original journal-then-ledger leases.
   Keep the accepted current raw boot, actual disjoint backing, system-use,
   journal file receipt and original directory/lock/mount checks.
2. Stream a complete validated catalog and census through the original locked
   ledger reader. Require zero jobs, including terminal jobs. Retain bounded
   per-policy canonical bindings from that same complete observation. A copied
   catalog, decoded journal or caller-supplied policy grants no authority.
3. For every retained policy, collect the existing fixed native pool/filesystem,
   provider/member and system-use facts. Reuse the accepted validators for exact
   GUID/name/scope/member binding and system/device exclusions. Require nonempty
   enrolled members. Never infer policy from an imported pool or mutable name.
   Repeat the policy and complete census checks around native collection.
4. Revalidate all original facts before and after each active publication,
   observation and unused close. Match the retained policies exactly; equal
   census hashes alone do not replace native checks. Native drift returns no
   partial successful lifecycle result. Observed property state remains
   ineligible; this is no backend or global-launch authorization.
5. Activate once using fresh entropy and the existing durable journal successor
   pipeline. Retain exact policy bytes and census, installation binding and both
   original leases. Reserve generation capacity for the eventual close. Capture
   only the acknowledged publication's own directory change inside its sync
   boundary; later producer/fault callbacks cannot redefine that snapshot.
6. Permit unused retirement only for that original active acknowledgement, with
   no admission, metadata change, native launch or worker attached. Publish only
   the exact generation+1 closed successor, repeat all facts, then retire
   observation/finish authority on every entered finish outcome. Explicit close
   releases leases, preserving visible uncertain history without a clean marker.

Preserve pointer lineage, transfer/release exclusion, pre-entry cancellation,
sticky entered failures and idempotent original release. Neither re-registration,
another path, reopen, decoded metadata nor changed boot can resurrect authority.
The existing mutex order stays owner, pair, journal, ledger; borrowed readers must
not reenter it. Context deadlines are shared across the complete bounded operation,
never renewed per policy. Preserve the current 90-second startup operation bound.
At valid capacity, timeout fails conservatively; no supported commercial capacity
or new performance claim follows until measured separately.

## Proposed failing-first Test Design

These named cases refine M2-RUNTIME-002, M2-STATE-001, M2-RECOVERY-001,
M2-OWN-001, M2-DRIFT-001 and M2-PRIV-001. They are planned, unexecuted cases,
not additional product requirements or fabricated evidence.

| Case family | Observable acceptance criteria |
|---|---|
| Policy-only lifecycle | Valid one/multiple-policy fixtures activate once, observe and close; two generations are added, owner identity changes, all ledger names/bytes and data sentinels remain exact; empty behavior remains accepted |
| Complete history refusal | Both job formats and every valid state, same/other pool and mixed jobs deny active publication and unused close; bytes, submission history and all holds remain unchanged; no job is omitted from the catalog |
| Policy binding | Legacy/missing members, changed revision/name/scope, foreign GUID/descendant, missing pool, member/provider drift and system overlap each deny with no partial grant; the last policy failing cannot leave earlier pools eligible |
| Original lineage | Copy/decoded/reopened review or activation, old-owner acknowledgement, alternate paths and concurrent factory attempts grant nothing; one process registration remains consumed after failure/release |
| Lease and lock lifetime | Both actual locks remain held throughout policy collection, activation and finish; competing openers fail; observation, transfer, finish and release exclude each other; original release waits and does not deadlock the borrowed reader |
| Cancellation and retirement | Nil/already-cancelled contexts enter no producer; cancellation after entry or native/ledger read failure retires authority; renewed contexts/repaired objects cannot reset it; every entered finish outcome prevents observe/finish reuse |
| Exact census and objects | Added/deleted/replaced ownership or any job, same-inode touches, unknown/orphan entries, partial catalog, lock/directory/mount changes and unconfirmed/third census deny; preserved journal/ledger bytes prove no repair or hidden pruning |
| Publication uncertainty | Actual file-sync, rename, directory-sync, acknowledgement and post-publication cuts retain visible before/after state conservatively; no retry, rollback or reconstructed ack; policy/boot/system drift after a successful sync cannot be absorbed |
| Boot and generation | Stable current boot with historical closed boot follows the accepted rule; missing/changed current boot, active prior owner on either boot, pending/disarmed records and activation without close capacity deny; final-capacity close remains single-use |
| Bounds | Existing record/count/byte caps refuse oversized/incomplete input; complete valid policy sets remain bounded without copied inventories or per-policy deadline renewal; zero-on-error and unchanged empty/census codecs remain enforced |

Require behavioral red before implementation, then focused portable lifecycle,
policy/lock/census and action-sequence coverage. Actual OS-root checks must include
a credential-dropped child retaining inherited root descriptors. Mocks cannot
qualify kernel UID, advisory locks, filesystem sync or native pool/provider facts.
No convenience reset of the production process registry is permitted in tests.

### Native fixture contract

Use the existing serial disposable QEMU lab on both pinned FreeBSD/amd64 15.1 and
14.5 targets. Qualify actual UFS without softupdates and ZFS metadata domains,
independent journal/ledger restore scopes and independently owned enrolled data
pools. Pool/filesystem GUIDs, member/provider facts and system exclusions must
come from real fixed producers in the root composition gate. Synthetic transfers
in storage publication tests must remain explicitly marked synthetic.

Nonempty closed fixture seeds are a critical boundary. Before the first red run,
document their construction using accepted codecs and real publishers under the
original leases, confirming the exact complete census and sync acknowledgements.
The existing `execution_publication_freebsd_test.go` demonstrates original
journaled ownership publication: pending journal, ledger publication, then census
confirmation. Extend that fixture pattern from an actual native commissioned
seed; close its confirmed original journal with the existing strict successor
publisher, exclusively as fixture setup. Its low-level closed successor can
preserve a nonempty census without weakening the empty-only product consumers.
Use a fixture-only controlled metadata history, with no admitted worker or native
executor. Never weaken the production empty-only commissioning/retirement guards
to manufacture a seed, hand-edit JSON or reconstruct a publication receipt.
Such a seed tests the consumer of closed metadata, **not** a qualified producer
for shutdown after real work. A second process must independently inspect the
published policy-only cycle using its original factory and normally waited child.

Retain the affected PR 126–128 controls on both targets: the current fixed-root
gate has four parents/394 controls; the storage gate has six parents/544 controls.
The implementation must inventory every new named assertion and exact selector
before execution. Use the same checksum-bound artifact per suite across releases;
record each actually executed source. Preserve the existing eight-minute root
parent, 34-minute root process and eighteen-minute storage process bounds.
Optimize duplicate fixture setup only with preserved cases and independent cleanup
assertions; a timeout remains an unqualified result rather than a larger deadline.

Add policy-only lifecycle, all-job refusal, actual member/scope drift, UID/lock
checks and publication cuts to the relevant suites. Parent observations must
independently verify exact histories, both released leases, data sentinels,
disjoint backing and complete owned pool/mount/device/guest/artifact/supervisor
teardown. Denial failures and expected red runs still require cleanup evidence.
The new strict evidence consumer must reject missing controls, skips, stale
candidates, synthetic root-policy positives and incomplete teardown.

Run the mandatory baseline and affected-package review. Shared runner/checker
changes require the existing full portable policy; prose-only proposal review
requires baseline/title/signature/DCO/teardown checks without fresh native or deep
execution. No new Actions workflow, daemon, synchronization service or paid runner.

## Review, remaining gates and pickup

Acceptance of this proposal requires review of the policy-only boundary, locked
borrowing design, fixture seed limitations and the test inventory obligations.
Then pick up one bounded implementation issue under the existing runtime gate,
blocked by this design issue. Do not split out dozens of file-level tasks.
If the seed or lock integration requires a materially different mechanism, update
this proposal before coding rather than treating an experiment as accepted design.

Later retained-job activation must explicitly review the complete original
mixed-job view, conservative restart projection, independent holds and uncertain
execution provenance before any clearance. Nonterminal recovery, terminal hold
resolution, changed-boot/restore decisions, orderly worker/supervisor draining,
journaled backend consumption, global native launch, configured startup and
public workflows remain open under the parent gate and related acceptance issues.
This proposal supplies no successful nonempty native receipt, real reboot or
power-loss recovery, active kernel-interruption, file-client continuity, browser,
performance, commercial hardware or full M2/release acceptance evidence.
