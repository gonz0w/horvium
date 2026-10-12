# Adopted amendment: cumulative two-policy lifecycle fixture budgets

Status: Adopted through [PR 319](https://github.com/gonz0w/horvium/pull/319),
merge `aebfb2c8b85040154f6e70ffb3296bba543a9db9`, proposal source
`0802767509a23ec6309980462773ef8f6f034283`. Native qualification remains pending.
Date: 2026-10-11. Owner: Cam (`@gonz0w`). Time box: one design/review session.
Design decision: [issue 318](https://github.com/gonz0w/horvium/issues/318).
Implementation/qualification: [issue 305](https://github.com/gonz0w/horvium/issues/305).
Parent: [M2-RUNTIME-002 gate](https://github.com/gonz0w/horvium/issues/139).
Authority: [approved PRD](PRD.md), [EDD](EDD.md), [Test Design](TDD.md),
[adopted policy-only design](retained-policy-startup.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
Requirements: M2-RUNTIME-002, M2-STATE-001, M2-RECOVERY-001, M2-OWN-001,
M2-DRIFT-001, M2-PRIV-001 and M2-NATIVE-001.

The subsequent [caller composition amendment](../../research/M2/policy-lifecycle-caller.md)
was adopted through PR 332. Its 180/390/1200/44 profile applies only to subsequent
full-lifecycle candidates; the original adopted rows and historical receipts below
retain their own limits and outcomes. See the [implementation](../../testing/M2/retained-policy-lifecycle-caller.md).

## Problem and proposed outcome

The original two-policy fixture charges startup, observation, unused retirement,
assertions and denial probes to a 150-second child test. Its three positive caller
budgets are already 120 + 90 + 90 = 300 seconds. The aggregate child deadline can
interrupt a valid independent operation before its own allowed deadline.

Merged [PR 317](https://github.com/gonz0w/horvium/pull/317) verifies one preserved
failed stack: source `5336a7be00c59e55cd35faeb373889ab64b9ebea` reported
startup/observation at 91.04/28.02 seconds, then reached the 150-second child
limit during post-closed-publication revalidation. At most approximately 30.94
seconds remained for all other child work. That snapshot proves neither a
successful close nor command duration or emulation causality. The two other
startup-deadline failures have no equivalent stack and remain unresolved.
See the [analysis and witness](../../research/M2/policy-lifecycle-timeout.md).

Propose a fixed **fixture-only** hierarchy for this existing three-phase suite
on the currently selected Mac/amd64-TCG lab. Increase its whole-child, lifecycle
parent-work and lifecycle-phase allowances together. Preserve every caller,
product, collector, command and cleanup limit, original native checks and all
existing controls. This permits an operation to use its existing budget; it
cannot turn a product/caller timeout into success or guarantee a native pass.
This PR supplies a design decision and Test Design only, with no implementation
or new native experiment.

## Exact hierarchy and scope

| Boundary | Current | Proposed | Scope and reason |
|---|---:|---:|---|
| Lifecycle child Go test | 150 s | **330 s (5m30s)** | Only `TestMultiplePolicyRootLifecycleChild`, both `first` and `restart`; 300 s positive-call envelope plus 30 s for fixture work |
| Lifecycle parent work context | 480 s | **1,080 s (18m)** | Only `TestNativeMultiplePolicyLifecycleUFS` and `TestNativeMultiplePolicyLifecycleZFS`; both children and retained fact/data work share this one original absolute deadline |
| Lifecycle Go test process phase | 18m | **40m** | Only the existing combined UFS/ZFS lifecycle phase, including failure cleanup and Go test-process termination |
| Facts and review process phases | 18m each | 18m each | Existing selectors and controls retained |
| Standalone/embedded facts and review children | 150 s | 150 s | No unrelated child gains the lifecycle allowance |
| Other fact-harness parent work contexts | 480 s | 480 s | Standalone facts, UFS/ZFS review and independent diagnostics retain their existing bounds |
| Lifecycle startup caller | 120 s | 120 s | One caller context across original review plus activation, never renewed between policies or product stages |
| Observe, finish and denial caller contexts | 90 s each | 90 s each | Separate owner calls; denial contexts start unexpired, as before |
| Each product startup/activation/observe/finish operation | 90 s | 90 s | Every original bounded operation, shared across its complete checks |
| Native collector/command limits | Existing fixed limits | Unchanged | Original timeout reaper, descendant containment and zero-on-error remain mandatory |
| Each original metadata or fact/data cleanup invocation | 30 s | 30 s | Independent background context after work failure; no new retries or grace renewal |
| Remote suite command timeout | 7,200 s | 7,200 s | Existing SSH controller bound |
| Owned lab supervisor timeout / termination grace | 9,000 s / 15 s | 9,000 s / 15 s | Existing outer foreground ownership; no persistent runner |
| Child diagnostic output | 16 KiB | 16 KiB | Reject excessive output; keep raw inventories private |

These increases are explicit changes to the effective **test** envelope, not an
optimization or an unchanged-deadline claim. The current contracts and failed
receipts retain their historical bounds. Upon explicit adoption, this amendment
supersedes only the three changed rows for the original two-policy lifecycle;
it does not supersede the [earlier single-policy child correction](../../testing/M2/retained-policy-child-budget.md),
the adopted runtime design or any historical qualification claim.

### Cumulative accounting and enforcement

Two lifecycle children can consume 2 × 330 = 660 seconds, and the independently
waited retained facts child still has 150 seconds. Their 810-second envelope
leaves 270 seconds within the 1,080-second parent work context for pool/metadata
setup, strict original seed publication, parent readbacks, sentinels and assertions.
That is a planning allowance, not a newly renewable per-stage timeout. Each
child command is also subject to the original parent deadline; a late child does
not get to extend its parent. No child retry, reseed between cycles or checkpoint
substitute is permitted.

The current metadata harness can invoke its independent 30-second cleanup once.
The fact harness has its existing explicit cleanup and registered `t.Cleanup`
callback; conservatively reserve another two 30-second invocations when explicit
cleanup fails. Preserve that existing failure behavior, including errors; add no
retry or deletion loop. Two parent work envelopes plus this cleanup reserve use
2 × (1,080 + 90) = 2,340 seconds. The proposed 2,400-second lifecycle process
limit leaves 60 seconds for process/affinity overhead. Normal successful cleanup
need not consume those reserves.

A context deadline is not a hard kill of every local filesystem syscall, publisher
or kernel wait. Do not claim an exact parent wall-time ceiling from this accounting.
The Go test process, unchanged remote timeout and owned supervisor still terminate
an overrun and require independent teardown; missing parent cleanup stays
unaccepted. Neither timeout nor process termination grants storage authority.

The three serial process allowances total 18 + 18 + 40 = **76 minutes**, up from
54 minutes. They fit inside the unchanged 120-minute remote command timeout with
44 minutes of unused envelope. Guest boot/copy are separately bounded before that
command; the unchanged 150-minute supervisor includes the complete owned run and
host teardown. These are ceilings, not expected durations or a supported workload.
Do not spend the envelope by continuing after a failed phase. Keep releases and
deep portable checks serial; an initial failed target is preserved before deciding
on correction or a separately recorded second-release attempt.

The tagged harness must select the longer parent context through a fixed internal
lifecycle-only choice. Preserve the existing default for every other caller;
reject unknown selections. Do not expose arbitrary timeouts through an environment
variable, public flag, product configuration or operator endpoint. The existing
one-CPU qemu64/TCG, 6 GiB, seven-device disposable profile, kernel/CPU 0 checks,
root-excluded metadata spare and topology restrictions remain unchanged. No new
CPU experiment, architecture, host infrastructure or caching of native facts.

## Retained and new acceptance inventory

Keep all existing phase selectors and their order:

1. Facts: `^TestNativeMultiplePolicyFacts$`, process timeout 18m.
2. Review: `^TestNativeMultiplePolicyReview(UFS|ZFS)$`, process timeout 18m.
3. Lifecycle: `^TestNativeMultiplePolicyLifecycle(UFS|ZFS)$`, process timeout 40m.

Retain exactly five completed parents and thirteen normally waited children per
release. Preserve all **283 existing controls**. Add only these six assertions:

- Each of the four original lifecycle children checks `budget_profile`: its
  actual Go test timeout flag is 330 seconds and its live `testing.T.Deadline`
  exists, is unexpired and has no more than that remaining allowance. Its parent
  carries this result as `first_budget_profile` or `restart_budget_profile`.
- Each lifecycle parent checks `parent_budget`: the selected original harness
  work context was created with 1,080 seconds, retained one absolute deadline
  across callbacks/children and has not expired at successful completion. Do
  not fabricate a fresh deadline for the end assertion.

The child inventory becomes **21 controls**, retaining all original twenty:
`cpu_affinity`, `registered`, `active_successor`, `active_private_record`,
`current_boot`, `leases_held`, `nonjournal_unchanged`, `original_observation`,
`observation_preserves_history`, `copied_denied`, `decoded_denied`,
`cancelled_no_effect`, `closed_successor`, `ledger_unchanged`, `close_retains_leases`,
`retired_observation`, `retired_finish`, `release_once`, `locks_released`,
`registration_consumed`, plus `budget_profile`.

Each lifecycle parent retains `seed_closed`, `domain_teardown`, `mounts_restored`;
for each `first`/`restart` child it retains `child_waited`, `child_inventory`,
`exact_closed`, `ledger_preserved`, `leases_released` and every child control.
Add `parent_budget`: **56 root controls**, plus the unchanged 21 fact/data controls
= **77 controls per lifecycle parent**. The retained standalone facts and both
reviews still contribute 135 controls. Total: **135 + 2 × 77 = 289 controls**.
The original [fact inventory](../../testing/M2/retained-policy-native-facts.md)
and [review inventory](../../testing/M2/retained-policy-native-review.md) remain
required in every affected parent, not borrowed from older passes.

Require an exact scalar lifecycle budget-profile marker inside its lifecycle
phase: child seconds 330, parent-work seconds 1,080 and phase minutes 40. Preserve
the CPU-profile marker before facts and CPU 0 affinity marker inside lifecycle.
Record the actual selected profile in the host receipt, bound to the same signed
source and executable checksum. The scalar marker is provenance, not a substitute
for the six actual assertions. Normal builds must still exclude tagged fixtures.

The strict consumer must reject old 18m lifecycle markers or 283-control logs,
missing/extra/duplicate/unknown budget profiles, extra parents or children,
misplaced markers, skipped/failed phases or controls and incomplete independent
cleanup. No compatibility fallback may promote old failed or partial evidence.
All UFS/ZFS seeds, fixed factory/producers, borrowed original leases, generation
and owner/boot readbacks, exact histories, denied copies/decodes/cancellation,
terminal retirement, explicit release and consumed registry checks remain exact.
The second child starts from the first's actual closed successor, without reseeding.

## Failing-first Test Design and stop rules

These fixture test IDs refine M2-NATIVE-001/M2-RUNTIME-002; they are planned,
unexecuted acceptance, not new product requirements or claimed native passes.

| Test ID | Required observable behavior |
|---|---|
| M2-POLICY-FIXTURE-BUDGET-001 | Before implementation, strict-consumer tests fail for the proposed complete 289-control/40m fixture log; after implementation, accept only exact new profiles and refuse every missing/new control and altered profile or phase |
| M2-POLICY-FIXTURE-BUDGET-002 | Selection regression refuses unknown parent-budget choices; existing non-lifecycle fixture callers retain the original 480-second context and 150-second children; only lifecycle selects the new envelope |
| M2-POLICY-FIXTURE-BUDGET-003 | Native child/parent assertions establish actual live deadlines and fixed timeout selection, original shared absolute parent deadline and all unchanged caller/product/native bounds; no per-policy renewal, stale-denial context or synthetic factory positive |
| M2-POLICY-FIXTURE-BUDGET-004 | Both pinned FreeBSD/amd64 15.1 and 14.5 complete all three phases, five parents, 289 controls and thirteen waited children using one checksum-bound artifact; all prior receipt bytes and their failed/skipped limits remain exact |
| M2-POLICY-FIXTURE-BUDGET-005 | Invalid profiles, operation failure or child/process timeout invalidate the run; original cleanup contexts do not borrow an expired work deadline; independent host/device/guest/key/artifact/supervisor teardown and base/runtime preservation are verified, while incomplete in-guest vectors stay unaccepted |

Before native execution, inventory the exact selectors/control vectors/profile
markers in Git and sign the clean implementation candidate. Record failing-first
consumer/selection observations with their actual source; no fabricated red of
unchanged product behavior is required. Shared tooling changes require the normal
full portable fallback, not just the documentation plan. Evidence-only followups
may reuse genuinely executed deep checks only under the existing exact-input
policy, with fresh baseline review.

Record source/base/title, compiler/artifact/input/log/result hashes, actual
profile, failed/skipped attempts and independent teardown. Preserve all eleven
prior policy receipts byte-for-byte, including the unchanged failed lifecycle,
primitive diagnostic and offline stack witness. Native acceptance still requires
complete original UFS/ZFS lifecycle and restart on **both** pinned releases.

If startup again reaches the existing 120-second caller or any 90-second product
limit, this amendment has not solved that failure. Stop, preserve uncertainty and
teardown evidence and investigate that distinct path. Do not increase those limits,
change the CPU profile again, suppress checks, add caching/retries or infer that
an owner was never published. If the 330-second child or changed parent/process
ceiling fails, preserve that failure too; no automatic further increase is adopted.
A suitable accelerated amd64 lab remains an alternative needing its own declared
preservation/containment profile and actual receipts before use.

## Adoption, implementation and remaining gates

This proposal requires an explicit maintainer decision through review and merge
of its design PR. Research PR 317's merge is not that decision. Keep issue 318
open in Review until adopted or explicitly superseded; record the adoption PR
and merge/source identity before removing its native block from issue 305.
Retain the original satisfied issue-288 relationship and parent 139. The followup
implementation belongs to issue 305 and must cite the adopted amendment; merging
this design provides no successful native receipt or runtime authorization.

Adoption permits only the three fixture-limit changes and their six added
profile assertions with the described hierarchy and retained tests. No product
code, persistent format, CLI/API, service, dependency, paid runner, Actions workflow
or live disk change. Runtime writes remain disabled. Controlled initial seeds
remain fixture setup, not worker shutdown provenance. Fault/mixed-history,
credential/lineage/concurrency/census/publication/context/capacity families,
configured startup/workers/backend/global launch and broader browser/client/
hardware/performance/release acceptance remain separate and open.

Design issue 318 is completed after that reviewed adoption. Its native dependency
remains attached to issue 305 as a satisfied prerequisite; implementation and
qualification do not inherit a native pass from the design decision. Historical
proposal wording above retains the decision and its original review boundaries.
