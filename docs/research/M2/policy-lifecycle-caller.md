# Original lifecycle caller composition review

Status: proposed fixture-only amendment; adoption pending, no implementation or new native run.
Date: 2026-10-11. Owner: Cam (`@gonz0w`). Time box: one offline development session.
Owning [issue 331](https://github.com/gonz0w/horvium/issues/331), under
[parent 139](https://github.com/gonz0w/horvium/issues/139), coordinating
[acceptance issue 305](https://github.com/gonz0w/horvium/issues/305).
Authority: approved [PRD](../../milestones/M2/PRD.md),
[EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[retained-policy contract](../../milestones/M2/retained-policy-startup.md),
[adopted cumulative fixture budgets](../../milestones/M2/retained-policy-fixture-budgets.md),
[startup inspection review](policy-startup-inspection.md) and
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
Requirements: M2-RUNTIME-002, M2-STATE-001, M2-RECOVERY-001, M2-OWN-001,
M2-DRIFT-001, M2-PRIV-001 and M2-NATIVE-001.

## Question and evidence replay

What distinguishes the second-cycle startup failure from a successful first
cycle, and what caller envelope can the next fixture test without weakening
original safety checks? Review [PR 330's candidate](../../testing/M2/retained-policy-lifecycle-qualification.md)
before another experiment. Its merge is
`c017ccbd0f706c8475b5df89ad73bc69663326d2`; actual native source remains
`c9b01533a8c3e343c2cc9dd176ce35f12ad9aed4`.
The [offline replay receipt](policy-lifecycle-caller-manifest.json) binds 36 source
assertions across eight files, original source/artifact/log/result hashes, two
earlier diagnostic replays and all eighteen preceding immutable policy receipts.
Inspected source is byte-identical between the native candidate and review base.
These assertions are source checks, not new behavioral tests or native execution.

The original 15.1 facts/review prefix completed three parents, 135 controls and
seven normal children. UFS lifecycle failed at 372.68 seconds, specifically
`restart_child_waited`. The second child's startup logged
`context deadline exceeded` at 120.02 seconds; child failure followed at 120.05
seconds. The 330-second child and 40-minute process limits did not fire. The
parent's 1,080-second original work allowance exceeds its reported elapsed time;
the log does not separately record the live remaining parent deadline.

Both children execute the same test function and original fixed factory with
identical caller/product/profile selections and the same journal/ledger paths.
Each is a fresh normally waited process with its own original process registry;
there is no special restart factory, reset or alternate native producer. Only
the first uses the initial controlled seed. The second starts from the first's
independently validated closed generation+2/new owner/current boot successor,
unchanged ledger and released original leases. That source ordering supports
the first-cycle observation, not a full parent pass or an independently retained
successful child transcript. Native facts and scheduling are observed anew;
identical source does not prove equal duration or state-independent behavior.

## Deadline and logging limits

The root factory reserves registration, constructs fixed sources, opens/reviews
original leases, then calls activation with the **same caller context**. Review
inspection creates a 90-second child; activation creates its own 90-second child
after review. Domain opening precedes the review inspection's child context.
Activation's three full inspections share one original operation deadline.
The four total successful-path inspections and their drift barriers remain needed.

Let C be the fixture caller allowance and B the activation context creation time
after caller creation. Its effective deadline is `min(C, B + 90s)`:

| Fixture caller | Parent constrains activation when | Interpretation |
| --- | --- | --- |
| Current 120s | B > 30s | Review/opening work can consume part of activation's nominal allowance. |
| Proposed 180s | B > 90s | Allows the nominal 90+90 envelope, but opening/other overhead can still constrain activation. |

The old source-mapped 120/150 diagnostic proved clamping **in its own call**.
Its point observations do not establish B for this later failed second cycle.
The separate 180/210 diagnostic succeeded at 84.436750 seconds, before even the
old ceiling; it does not prove a causal wider-caller fix or stable performance.
Neither observation is a controlled paired comparison with this lifecycle run.

The current lifecycle helper logs returned error text and rounded elapsed time;
it does not separately log the caller's `Err()` before deferred cancellation.
The root owner joins underlying errors with caller errors. Consequently the
120.02-second error is consistent with reaching the caller boundary, but these
logs cannot identify the earliest expired nested context. PR 330's description
of caller expiry is a timing/source interpretation, not a separately measured
caller-error field. Preserve its receipt; do not retrospectively manufacture one.
The second cycle also lacks independent final metadata readback. Its publication
state, precise failing probe and any causal emulation/deadlock/capacity explanation
remain unknown.

## Proposed fixed fixture hierarchy

Propose extending only the tagged original full lifecycle's startup caller to
the already separately tested **180-second hypothesis envelope**, with coherent
aggregate fixture allowances. This changes effective test budgets and requires
explicit adoption. It is not a product timeout fix or a guarantee of success.

| Boundary | Current | Proposed |
| --- | ---: | ---: |
| One absolute lifecycle startup caller, across review and activation | 120s | **180s** |
| Each first/restart lifecycle child Go test | 330s | **390s** |
| Each UFS/ZFS lifecycle parent work context | 1,080s | **1,200s (20m)** |
| Combined lifecycle process phase | 40m | **44m** |
| Facts/review process phases | 18m each | Unchanged |
| Other fact/review parents and children | 480s / 150s | Unchanged |
| Product review, activation, observation and finish operations | 90s each | Unchanged |
| Observe/finish/denial fixture callers | 90s each | Unchanged |
| Native collector/command, each cleanup, output, remote/supervisor/grace | Existing limits | Unchanged |

Child planning: 180 + 90 + 90 positive-call seconds + the same 30-second fixture
reserve = **390s**. Two children plus the retained 150-second facts child use
930s; the same 270-second setup/assertion reserve gives **1,200s** parent work.
Conservatively reserve the existing three independent 30-second cleanup
invocations per parent, with no additional retries. Two parents use
2 × (1,200 + 90) = 2,580s; the same 60-second process/affinity allowance gives
2,640s = **44m**. Three serial phases total **80m**, within the unchanged 120m
remote/150m supervisor allowances. These reserves are planning margins, not
measured work, renewed stage deadlines or exact syscall wall-time guarantees.

Opening, collector/nested deadlines, assertions, denial probes and scheduling can
still fail. Every child remains subject to its parent's original absolute deadline.
Keep the original producer/factory/registry, leases, complete policy/census checks,
four inspection barriers, actual CPU/affinity, qemu64/TCG/6 GiB/seven-device profile,
publication/acknowledgement and sticky uncertainty/retirement protocol. No product
source, runtime flag/schema, public API, cache, guard deletion or CPU change.

Only after adoption may the new profile supersede these four rows for subsequent
full-lifecycle candidates. Preserve original 120/150 trace and separate 180/210
diagnostic profiles/consumers exactly. Historical 120/330/1080/40 candidates remain
failed or partial under their own actual profiles, never reclassified as new passes.

## Proposed observation and acceptance inventory

Retain every original control and normal phase/parent/child order. Add one
`startup_caller_budget` assertion per lifecycle child: a single fixed 180-second
caller is created once, retains its actual original deadline across the factory
call and is not renewed between review, activation or policies. Bind before/after
creation time bounds to the actual live deadline, not a reported constant alone.
Existing child `budget_profile` and parent `parent_budget` assertions must validate
their new actual 390/1200 allowances and the strict 44-minute phase marker.

Child inventory becomes **22** controls. Each lifecycle parent carries its two
new child assertions: **58 root + 21 fact/data = 79** controls. Retained facts and
both reviews contribute 135; total **135 + 2 × 79 = 293** controls, preserving
all 289 original controls with their applicable new fixture profile. Require the
same five parents and thirteen normally waited children per release. Profile
provenance must include caller 180, child 390, parent work 1200 and phase 44;
missing, duplicate, unknown or cross-profile evidence denies the whole gate.

Also retain bounded, sanitized scalar observations for each of the child's ten
existing helper calls: startup, observe, copied, copied-finish, decoded,
decoded-finish, finish, retired-observe, retired-finish and registration-consumed.
Record mode, stage, actual monotonic elapsed time, selected caller allowance,
unchanged actual deadline and separate classified operation/caller errors **before
deferred cancellation**. Startup also records returned-owner presence, consistently
with the operation result. Parent extraction must preserve exact stage order and
inventory for both cycles, including successful children. No raw error arguments,
stacks, paths, credentials or host inventory in committed scalar observations.
Four successful lifecycle children require forty ordered stage observations per
release; these are additional evidence requirements, not forty extra controls.

A failed child may expose only its actual completed stage records through the
existing bounded failure transcript. Missing later stages are honestly absent;
the strict release consumer still rejects the failed parent. Do not fabricate
rows, retry, turn observation completeness into native acceptance or infer final
publication state from an error/owner result. Outer-call duration is not individual
collector/command timing or attribution of the earliest nested timeout.

## Proposed failing-first Test Design and decision

These planned IDs refine M2-NATIVE-001/M2-RUNTIME-002 and are not executed tests.

| ID | Required observable behavior after adoption |
| --- | --- |
| M2-POLICY-LIFECYCLE-CALLER-001 | Failing-first portable context tests prove fixed scoped 180/390/1200 choices, actual deadlines, cancellation propagation and no renewal; unrelated 120/150 trace, 180/210 diagnostic and fact/review profiles stay exact. |
| M2-POLICY-LIFECYCLE-CALLER-002 | Strict evidence regressions first reject the proposed 293-control/44m profile, then accept only its exact markers/vectors; old 289-control evidence, diagnostic evidence, missing controls, skips/failures, artifact/source mismatch and false acceptance remain refused. |
| M2-POLICY-LIFECYCLE-CALLER-003 | Failing-first scalar tests require all ten ordered calls per successful child, original mode/deadline, pre-cancel caller and operation classifications, owner/result consistency and bounded monotonic timing; unknown/duplicate/reordered/raw/synthetic rows deny. Honest failed-stage partial rows never qualify a release. |
| M2-POLICY-LIFECYCLE-CALLER-004 | Each actual child and parent proves the live new caller/child/parent profile and original deadline retention; every existing factory/producer/lease/policy/boot/census/publication/copy/retirement/release/registry control remains required, including the same original successor between cycles. |
| M2-POLICY-LIFECYCLE-CALLER-005 | Both pinned disposable 15.1 then 14.5 complete five parents/293 controls/thirteen normal children and forty ordered scalar stage observations with one checksum-bound artifact. Stop after the first failed target; no automatic retry, increased limit, reseed or second target. Preserve failed/skipped/unverified evidence and independently verify original guest and host/device/key/binary/supervisor teardown and base/runtime preservation. |
| M2-POLICY-LIFECYCLE-CALLER-006 | Signed/DCO source/base/title/artifact/compiler-bound candidate, failing-first portable evidence, mandatory review and conservative full fallback for shared tooling; normal build exclusion, all eighteen prior receipt bytes and final owned-session teardown verified. Evidence-only follow-ups use the documentation review policy. |

Alternatives are to keep the original profile and improve its scalar diagnostics
alone, or qualify an accelerated amd64 lab under a separately reviewed preservation/
containment profile. Neither is silently selected if this proposal is rejected.
No new home-lab infrastructure is required for the proposed Mac fixture experiment.

Issue 331's reviewed PR must record explicit adoption or rejection before a changed
fixture is implemented or executed. Implementation/qualification remains under
issue 305; the research issue remains open until its decision criterion is met.
Both full release gates, remaining fault/mixed-history/credential/lineage/concurrency/
census/publication/context/capacity families, configured startup/workers/backend/
global launch and browser/client/hardware/performance/release acceptance stay open.
Controlled initial seeds are not worker shutdown provenance. Runtime writes remain
disabled. This proposal performs no native run or deployment and grants no retry,
clean retirement or launch authority.
