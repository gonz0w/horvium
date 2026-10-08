# M2 original coordinator execution authority

Date: 2026-10-08. Status: portable subset qualified; maintainer review pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-PLAN-001, M2-JOB-001, M2-DRIFT-001 and M2-RECOVERY-001 evidence.

## Original backend binding

The original property coordinator's handoff now requires its admission/execution
backend to implement `PropertyExecutionAuthorityBinder`. There is no replacement
backend argument or returned adapter. The binder receives an opaque
`PropertyExecutionAuthority` for that same original admission. There is no public
constructor, activation, reset, serialization or recovery method.

Consume the original single handoff under the coordinator mutex before binding.
Call the bounded synchronous binder outside that mutex; it may retain the handle
and read an independent copy of the whole canonical queued acknowledgement via
`AdmittedJob`. Binding must perform no publication or native effect. The pending
handle refuses `CheckLive`, including through copies and concurrent calls. Only
successful binding followed by final original plan/record/clock/context checks
activates it. Every binding failure remains consumed and cannot be retried.

The exact queued acknowledgement remains admission evidence after retirement,
never current job status or a new durable acknowledgement. Each issued identity
owns its plan snapshot and canonical record within existing plan/record bounds;
the coordinator still has its 64-entry ceiling. The future bounded root backend
must retain only its own bounded active work, rather than accumulating old handles.

## Live checks and retirement

`CheckLive` accepts only the independent worker context. It checks the original
coordinator's uncertainty latch, active private identity, exact retained entry,
fresh-admission/single-handoff flags, whole plan including its session context and
policy hash, exact submission and canonical acknowledgement. A replaced entry
with the same public IDs cannot substitute for the original private identity.
Require the original civil-clock lineage, creation/expiry window and final
clock/context checks after bounded encoding. Rollback latches that coordinator.

The handoff envelope uses this same capability before/after preparation and
refresh and before the control. The backend can use `CheckLive` inside its final
native prelaunch callback, without accepting caller-supplied live flags or clocks.
Retire launch authority immediately when the control call returns, including an
error or unverified result. The issued worker also retires it on every consumed
attempt exit, including preparation/clock/record denials and publication cuts.
Copies share retirement. Invalid inputs rejected before attempt consumption keep
the existing worker contract; they do not create an additional attempt.

Retirement releases no reservation, deletes no metadata and resets no uncertainty.
It cannot erase an actual verified late effect. Independent observation and exact
durable reconciliation/observing/success publication continue through the original
backend under the existing sequence rules, even after forward expiry/review
reclamation. The original handoff request's cancellation is not the independent
worker's lifetime context.

## Required proof and limits

Nine new behavioral parents require owned admission copies, inactive binding,
missing-binder refusal, ten binding cuts, seventeen live denials, three final
clock/context cuts, eleven attempt retirement cases, late exact success, independent
worker context and 32 concurrent callers during and after binding. Retain all ten
existing handoff/uncertainty parents and extend the existing bounded handoff fuzz
target with copied authority checks and failed binding actions. No second
admission, handoff or effect may occur.

Two failing-first behavioral regressions demonstrate missing binding and a
silently accepted backend lacking the mandatory binder. Complete mandatory
baseline and affected-package race/all selected bounded fuzz checks are required.
FreeBSD cross-compilation is compilation evidence only. This change touches no
native facility adapter, metadata transaction, fixture, shared runner/checker or
historical manifest; unchanged native evidence keeps its actual source. No native
suite rerun is claimed or substituted for combined backend qualification.

This capability proves retained session **context**, without authenticating an
application session or kernel peer. It supplies no root lease, native eligibility,
executor/recovery clearance, process containment or global/pool launch fence.
The combined original root backend must independently prove those gates, own one
fixed control and bind its original state claim. The private native factory is
unchanged and has no runtime consumer. Transport/operator writes, native
interruption/reboot/recovery, hold release and complete M2 acceptance remain open.

## Executed qualification

Signed source `68a6cf6e1794d258be31c42a7b7913574fec7311`, based on PR 89 merge
`025e496a3d10f935071fae59ba606e0a246e9d62`, passed all nine new parents and
41 children, all ten retained handoff/uncertainty parents, mandatory baseline with
125 tooling tests, nine affected race packages and every one of 38 selected fuzz
targets. Focused review took 212.450 seconds; both supervisors completed cleanly.
FreeBSD/amd64 compilation passed without execution and its artifact was removed.
All 65 historical M2 manifests and 364 protected native/metadata/shared-tool paths
are byte-identical to the base. Lab status found no active or unverified sessions;
Actions remain disabled. See the [source-bound record](property-execution-authority-manifest.json).

The first baseline stopped on six existing process-fixture failures because the
sandbox denied their Unix-socket bind. The unchanged source passed through the
permitted owned review gate. Preserve that initial run as unqualified; no test,
process-containment or product restriction was relaxed. Full native/backend and
M2 acceptance gates remain open.
