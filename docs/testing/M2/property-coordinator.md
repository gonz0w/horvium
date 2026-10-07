# M2 property review and submission coordinator

Date: 2026-10-06. Status: implementation; qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Contract

`PropertyCoordinator` joins the accepted property plan, intent capture, job
binding and submission publisher contracts. It holds at most 64 volatile plans
and retained admission acknowledgements behind one mutex. It owns no goroutine,
timer, file descriptor, service or native command. Expired entries can be discarded
from memory without deleting durable records or releasing pool holds.

The trusted `PropertyBackend` supplies fresh helper-side policy/observations and
root-lease-bound metadata publication. Requests provide only a typed property
intent for review, then plan/submission identifiers and an authenticated session
context for submission. They cannot supply policy, eligibility, review rows,
timestamps, jobs or an execution command. There is no full native backend or
runtime consumer yet. A fixture's eligibility flags grant no product authority.
Kernel UID authorization, application authentication and session validity remain
obligations of the future helper/application adapters.

Review independently collects current policy/observations and retains an immutable
plan. Submission collects them again and compares the entire approved baseline,
including descendant impact. It checks session, context cancellation and trusted
clock before collection and publication. Civil clock rollback blocks this
coordinator until reconstruction; it does not silently extend a plan's lifetime.
Creation/capture use millisecond timestamps so the existing job clock never
rounds admission before intent capture. Plans still expire conservatively after
five minutes. A final clock/context check follows capture/encoding.

One plan is claimed for one submission identifier immediately before invoking
publication. Successful new acknowledgements must exactly match the bound
candidate; existing duplicate acknowledgements must pass the accepted complete
review/policy/session/selection check and retain current history. All output owns
its slices. Matching concurrent confirmations return the same retained admission
acknowledgement without a second publication or fresh durability acknowledgement.
That retained response is not a current job-status query or permission to launch.
If publication finishes successfully after expiry/cancellation, report its actual
acknowledgement; do not fabricate a failed admission that encourages replay.

Every publication error or invalid acknowledgement returns zero/false and blocks
further submissions on this coordinator, including attempts through another plan.
This deliberately conservative slice does not distinguish retryable pre-write
denials from uncertain I/O. Review remains available. Reconstruction is not proof
of safe recovery: a native adapter must validate the store and retained holds
before publication. Expiry/restart removes volatile retry responses; durable retry
lookup/reconciliation remains a later gate. No automatic replay or orphan cleanup
is added.

The native publisher remains responsible for atomic complete-store reservation,
capacity checks, root ownership/backing validation and durable file/directory
publication. The coordinator mutex is not a filesystem lock, cross-process claim,
ZFS fence or native eligibility proof. Backend inputs must remain independently
owned/stable during each call. Backend operations must be bounded and obey their
native cancellation contracts; the coordinator adds no asynchronous cancellation
or latency guarantee for a blocked backend.

## Focused qualification

Failing-first behavioral tests require the absent coordinator before implementation.
They cover review/input/output isolation, fresh policy/property/eligibility denials,
32 concurrent confirms, session/submission conflicts, expiry during observation,
cancellation, clock rollback, 64-plan exhaustion/reclamation, restart invalidation,
uncertain/foreign/malformed acknowledgements, current duplicate history and late
durable acknowledgement. A bounded action-sequence fuzz target exercises these
transitions and refuses partial results or a second publication for one plan.

```sh
make test-focus PKG=./internal/storage TEST='^TestPropertyCoordinator'
PR_TITLE='feat(storage): coordinate property review and submission' make review
make lab-status
```

Run the mandatory baseline and affected storage race/all compiler-listed portable
fuzz targets, plus a FreeBSD test cross-build. This slice changes portable
orchestration only; no native filesystem pipeline, syscall, transport, runtime
admission or fixture/selector changes. Prior native evidence stays bound to its
actual executed sources. Portable mocks and an unexecuted cross-build do not
qualify a native review-to-admission workflow. Full native policy/eligibility,
kernel-authorized transport, persistent retry lookup, executor attribution,
postconditions/recovery/release and assembled M2 workflow qualification remain open.

## Executed qualification

Pending signed-source review and sanitized execution record.
