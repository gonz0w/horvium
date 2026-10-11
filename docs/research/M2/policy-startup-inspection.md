# Original startup inspection review and proposed caller experiment

Status: adopted in [PR 327](https://github.com/gonz0w/horvium/pull/327); implementation/diagnostic evidence remains separate. Owner: Cam (`@gonz0w`).
Date: 2026-10-11. Time box: one development session, offline research/design.
Owning [issue 326](https://github.com/gonz0w/horvium/issues/326), under
[parent 139](https://github.com/gonz0w/horvium/issues/139); broader
[implementation/acceptance issue 305](https://github.com/gonz0w/horvium/issues/305).
Authority: approved [M2 PRD](../../milestones/M2/PRD.md),
[EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[retained-policy contract](../../milestones/M2/retained-policy-startup.md),
[recovery mechanism](../../milestones/M2/property-recovery-mechanism.md) and
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
The [source-bound review manifest](policy-startup-inspection-manifest.json)
separates observations, source deductions and the unadopted experiment.

## Question, provenance and scope

How do original repeated inspections compose, what authority survives failure
following publication, and does the existing caller constrain activation's own
allowance? Review the completed [diagnostic](policy-startup-stack.md) rather than
repeat it. Propose one separate fixture-only caller experiment if warranted.
There is no implementation, new native execution or runtime change in this review.

PR 325 merged at `72f1058e8ab4968470fe43db258fea29518d6c85`, the inspection base.
Its [qualification receipt](policy-startup-stack-qualification-manifest.json)
binds executed native source `2d78311dbf7c71678b02f48f2a3488a4c8a737fa` (PR 324
merge, exact tree of reviewed signed head `026aa9cb682516cfa951af8831fa2744437d969a`).
Replay matches that receipt to the private original result/log, artifact metadata,
pinned compiler and 97 source-relative frames (64 Horvium / 33 Go). Product and
fixture source inspected here is byte-identical between executed source and base.
All fifteen preceding policy receipts remain byte-identical, including the first
unqualified seed failure. Later success cannot accept its unverified guest cleanup.

Replay records 39 source assertions across 21 source files, line numbers, hashes,
source ordering and the compiler-matched context implementation. One intermediate
private assertion incorrectly expected three calls on receiver `j`; the source
contains one on `j` and distinct calls on `s`. Correcting that assertion produced
the passing replay; its rejected-log hash/reason is preserved in the manifest.
This was an analysis assertion error, not a new product or native test failure.
Raw guest stacks were previously deleted by the adopted diagnostic; there is no
independent re-extraction of raw data. No host inventory or raw arguments enter Git.

## Original inspection composition

`internal/host/freebsd/execution_restart_freebsd.go` reserves the original root
registration, invokes original review, then original activation with the same
caller context. It returns an owner only after activation succeeds. A complete
successful nonempty-policy path performs four full inspections:

| Inspection | Original source | Safety role |
| --- | --- | --- |
| Initial review | `internal/storage/execution_startup_freebsd.go:59` | Original closed journal, native objects, boot, binding, complete census and each policy are validated under original journal-first leases. |
| Activation entry | `internal/storage/execution_restart_freebsd.go:50` | Revalidate the exact retained review before constructing the successor. |
| Before publication | `internal/storage/execution_restart_freebsd.go:75` | Entropy collection and successor construction cannot absorb drift. |
| After publication | `internal/storage/execution_restart_freebsd.go:100` | Validate the acknowledged active successor and native facts before creating the activation object at line 103. |

Each full inspection checks original object identities/metadata, journal bytes,
record/binding/census, boot and original native backing/system facts. With retained
policy digests, it observes the complete pair before and after per-policy validation
and requires equality. Each pair observation itself observes journal and ledger
backing before/after, compares fresh system facts twice and retains complete census
and journal comparisons. Each policy validates canonical digests and fresh native
pool/provider/system facts with the borrowed original ledger, without reopening it.
These repeated checks protect distinct mutation intervals; this review does not
establish that any check is redundant or authorize removing/caching one.

For **N nonempty retained policies**, one complete full inspection conditionally
contains **8 + 2N backing callbacks** and **4 + N domain/policy system collector
callbacks**. Four complete inspections with N=2 imply 48 backing callbacks and 24
system collector callbacks; N=1 implies 40 and 20. Fixed backing and system
collectors themselves perform two passes. Additional boot, object, property,
provider and command/syscall work is outside this model. These are static
complete-success callback counts, not measured invocations or native command
counts in the failed run; its final inspection did not finish. Primitive diagnostic
costs cannot be multiplied into an observed full factory duration.

## What the point snapshots prove about the caller

The unique original factory goroutine was still in initial review at some instant
within 30.008717–30.012838 seconds after caller creation. It was in activation's
first inspection at some instant within 75.008148–75.009936 seconds, and in its
post-publication inspection within 110.000572–110.002112 seconds. Original startup
returned at 120.010624 seconds, with caller/startup errors classified deadline and
no owner. These are source-mapped point observations, not phase boundaries.

Let B be activation's original 90-second context creation time after caller
creation. Source ordering gives the conservative interval:

```text
30.008717s < B <= 75.009936s
120.008717s < B + 90s <= 165.009936s
original caller deadline = 120.000000s
```

The compiler-matched Go context implementation preserves an earlier parent
deadline. Therefore the 120-second caller **did clamp activation's own 90-second
allowance in this observed call**. This deduction does not identify the earliest
nested timeout, exact failing probe, command duration, deadlock or an emulation,
capacity or production performance cause. A different run may enter activation
at a different time. Sampling overhead is unmeasured and can affect scheduling.

## Post-publication failure and release

Independent final journal bytes changed from closed generation 6 to active
generation 7; ledger/nonjournal bytes were preserved and child leases released.
The final snapshot is after the original publication at activation line 84.
An error/nil owner consequently does not imply absence of publication or reusable
closed history.

Activation failure marks the original pair blocked, retires the journal store and
blocks ledger publication. `execution_journal_store_freebsd.go:154` retirement
sets failure/publication-denial flags; it does not publish a closed successor.
Original factory error handling joins the activation error with the original
review's explicit close. Pair close releases original ledger/journal leases;
release is not durable clean retirement. The original process registration is
consumed before construction and remains consumed after failure/close. These are
source-backed failure semantics, not an independently executed future retry gate.

Preserve sticky denial and active/uncertain history. Do not reset registration,
retry that seed, synthesize a closed record, call a substitute idle retirement or
restore an old acknowledgement. Explicit recovery of active history is a separate
accepted-contract requirement. The observed active bytes do not grant launch
permission, finish authority or native lifecycle/release acceptance.

## Proposed separate fixture-only envelope

Propose one separately identified diagnostic with a strict separate evidence
consumer. Implementation identifiers belong to its subsequently reviewed slice;
this document defines no product CLI flags, routes, schemas or package names.
Maintainer adoption of this proposal authorizes only this fixture experiment.
Product context limits and original profiles remain unchanged.

| Bound/profile | Existing diagnostic | Proposed separate diagnostic |
| --- | --- | --- |
| Absolute original factory caller | 120s | 180s, created once, never renewed |
| Whole startup child | 150s | 210s |
| Parent work / process phase | 480s / 18min | Unchanged 480s / 18min |
| Product review / activation | 90s / 90s | Unchanged 90s / 90s |
| Native collector / individual command | Existing 45s / 10s where declared | Unchanged |
| Snapshot offsets / buffer / stop-join | 30/75/110s; 64 KiB; 5s | Unchanged |
| Cleanup / remote / supervisor / grace | 30s / 7200s / 9000s / 15s | Unchanged |
| Child output ceiling | 16 KiB | Unchanged |

Preserve the original `policy-startup-trace` 120/150 profile and its consumer.
Preserve the full lifecycle caller 120 / child 330 / parent 1080 / phase 40min profile and
all 289 controls. Do not silently reinterpret either prior candidate's receipts.
The new profile must be bound to actual caller/child/parent/phase controls, source,
artifact and consumer; reject cross-profile evidence in either direction.

Use one fresh disposable two-policy closed UFS seed, exactly one original fixed
factory call, one original facts child, one parent, two normally waited children
and all 42 existing diagnostic controls, including all 21 fact/data controls.
First and only initial target is pinned FreeBSD 15.1, unchanged qemu64/TCG,
one CPU, 6 GiB RAM and seven owned devices. No automatic 14.5 run, retry or CPU
comparison. The Mac/QEMU profile is a lab observation, not a hardware requirement.

Three 0600 bounded snapshot files remain in the owned 0700 guest directory only.
Retain exclusive creation, full/truncation rejection, exact source/compiler frame
binding, scalar timing/error/owner observations and stop/join checks. Early return
must stop sampling and preserve skipped status. Capture final journal bytes/state,
ledger/nonjournal equality and actual released child locks independently. Any
returned owner is released only through its original `Close`; no substitute
retirement/clean history publication. Require all original guest cleanup plus
independent host/image/device/key/artifact/supervisor teardown and base/runtime
preservation. Missing controls, abnormal child waits, incomplete sanitation/raw
removal or unverified teardown invalidate diagnostic completeness.

180 seconds is a **hypothesis envelope**, not a guaranteed success budget: opening,
transfers, cleanup, scheduling and earlier nested/product deadlines may still fail.
It need not provide two complete consecutive 90-second phases. Do not renew a
caller between review and activation or widen product/native limits to force a
pass. Separate operation result, diagnostic completeness and lifecycle acceptance;
**native lifecycle acceptance is always false** for this diagnostic, even on success.
The full original five-parent/289-control/thirteen-child gates on both releases,
second clean cycle and broader native/recovery/browser/client/hardware/performance/
release gates remain open. Runtime writes remain disabled.

## Proposed failing-first Test Design

These criteria are proposed, not implemented tests or accepted native evidence.
After explicit adoption, implement the portable failures before native execution.
Reuse unchanged existing controls rather than rerun a milestone suite here.

| ID | Observable criterion |
| --- | --- |
| M2-POLICY-STARTUP-ENVELOPE-001 | Portable schedule/context tests reject caller renewal, wrong absolute 180 / child 210 bounds and inconsistent actual profile controls; preserve all original profiles. Test exact parent clamping without relying on slow wall-clock sleeps. |
| M2-POLICY-STARTUP-ENVELOPE-002 | Separate strict evidence tests reject old 120 / 150 evidence under new profile, new 180 / 210 evidence under old consumer, source/artifact mismatch, missing/duplicate controls and false native acceptance. |
| M2-POLICY-STARTUP-ENVELOPE-003 | Portable regressions retain fixed samples, bounded private files, early return/skips, stop/join, sanitization/source binding and uncertainty; failed operation remains distinct from completeness. |
| M2-POLICY-STARTUP-ENVELOPE-004 | One native 15.1 fresh two-policy UFS parent executes the original factory once and original facts child under unchanged product/native/CPU limits; both normal waits and all 42 controls are required. |
| M2-POLICY-STARTUP-ENVELOPE-005 | Independently record final journal/state, ledger/nonjournal preservation and lock release; release only actual returned owner; do not retry, erase active history, reset registration or substitute retirement. |
| M2-POLICY-STARTUP-ENVELOPE-006 | Signed/DCO candidate review, exact source/base/title/artifact/compiler profile and immutable fifteen prior receipts; original guest and independent host/device/key/binary/supervisor cleanup, verified lab teardown and no new runtime/service/dependency/Actions authority. |

## Review decision and remaining work

Issue 326 owns the offline findings and explicit adoption/rejection through its
review PR. Merge must record that decision before fixture implementation/execution.
GitHub owns live status and dependencies; this proposal contains no duplicate
progress checklist. A rejection leaves the original profiles/evidence authoritative.
Adoption permits a separately bounded test slice and candidate-specific receipt,
not a product timeout change, launch clearance or acceptance of issues 305 / 139.

## Recorded adoption

Maintainer merged PR 327 at `38b69d77c1f5c4fcbb8c8a79e074ff14f76815dc`,
adopting only the separate fixture experiment and M2-POLICY-STARTUP-ENVELOPE-001–006.
The proposal language above records the original reviewed decision boundary.
[Issue 328](https://github.com/gonz0w/horvium/issues/328) owns the
[implementation and candidate evidence](policy-startup-envelope.md). No original
profile, product limit, launch authority or broader acceptance is amended.
