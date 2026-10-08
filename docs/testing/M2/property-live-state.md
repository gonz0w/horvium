# M2 fresh property submission state claim

Date: 2026-10-07. Status: implementation awaiting source-bound qualification.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md)
and [issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-JOB-001, M2-STATE-001, M2-DRIFT-001 and M2-RECOVERY-001 evidence.

## Original lease receipt

The existing state-readiness prerequisite refuses every pending job. After a new
submission, a backend needs to distinguish its own hold from restored/foreign
work without granting that exception merely because a disk record says queued.

Only a successful new initial property submission publication, after the existing
exact inode/bytes check and final directory sync, adds a volatile canonical-record
SHA-256 receipt to the original root lease. Failed/uncertain publications,
duplicate readback, raw property-job publication and disk decoding add nothing.
Receipts never replace an earlier binding or survive lease close/reopening.
The fixed bound is 64 receipts per lease; exhaustion preserves successful durable
metadata acknowledgements but denies readiness claims for uncaptured submissions.
It changes neither the durable capacity nor the configured job limit.

FreeBSD's identifier-only `StateDirectory.ClaimPropertyState` validates actual root
EUID, the original descriptor/lock, publication certainty and a strict complete
mixed namespace. Exactly matching queued metadata may receive one opaque claim.
There is no public claim constructor or persisted form. One successful claimant
is retained for this lease, including when never used. Copies share its identity;
repeat/different claims are denied. Failure before grant does not consume it.
No automatic release, reassignment or replacement-lease authority is added.

## Read-only readiness

`PropertyStateClaim.CheckPropertyState` retains the complete original selected
policy, private namespace, uncertainty, backing capability and independent system
backing checks under the same root mutex. It permits only its own exact original
queued job or its single applying successor. The full expected record must match
strict independent readback, and its initial queued predecessor must hash to the
original post-sync receipt. Capture, intent and first-event drift deny.
Observing/terminal/reconciliation stages cannot use this exception.

All other nonterminal jobs anywhere and all other retained jobs on the selected
pool still deny, including terminal labels. Other-pool terminal metadata passes
only this prerequisite. Ordinary `StateDirectory.CheckPropertyState` keeps its
unchanged blanket denial; no reservations, record bytes or backing settings change.
The existing final descriptor/backing identity checks and orphan write latch remain.

This claim is metadata provenance and a read-only prerequisite. It is not the
coordinator's live session/plan, expiry/clock authority, recovered-state clearance,
native eligibility, executor quiescence, launch scheduler or a storage fence.
Policy and system facts must still come from trusted fresh native collection.
The future original backend must independently join these gates with the accepted
[handoff](property-handoff.md) and [private control](property-control.md).
No runtime consumer, command, helper/HTTP/CLI route or mutation is enabled.

## Test-first and native qualification

Five portable behavioral parents cover receipt provenance, complete record and
stage binding, independent retained holds, fixed bounds/global exclusivity,
nonreplacement and close invalidation. Bounded action-sequence fuzzing requires
at most one grant. The initial absent-contract compile failure is retained;
fixture field/reason corrections are not native/product failures.

The unchanged state selector runs the existing submission parent plus its new
live-state group: queued-applying, copy, reopen, raw-publication, duplicate-reopen,
concurrent, late-claim, current-drift, receipt-drift, uncertain-sync, other-pending,
same-terminal, unsafe-mode, orphan, policy-drift, missing-system, closed and
wrong-uid. Require every child once, no skips/failures, and 32 actual concurrent
claim callers. The copied UID child runs with actual UID/GID 65534 and empty
supplementary groups; it is waited and its owned executable directory removed.

Kernel root identity, lease, publication, strict namespace, descriptor/statfs and
metadata backing are actual. Policy/member/physical system-use inputs are explicitly
synthetic. Positive readiness using those model inputs does not qualify actual
GEOM backing, native enrollment, root eligibility or a runtime executor.
The existing native pool-source evidence remains historical and is not relabeled.

Run both pinned disposable releases serially with the existing state runner.
Preserve every historical state/codec/backing assertion and all original limits;
no shared runner, selector, checker, supervisor or old manifest changes.
Require mandatory baseline, affected-package race/fuzz review, checksum-bound
native artifacts and complete backing/process/guest/base/runtime teardown.

```sh
make test-focus PKG=./internal/storage TEST='^TestPropertyLiveState'
PR_TITLE='feat(storage): bind property readiness to fresh root admissions' make review
.tools/venv/bin/python3 tools/local-qualification.py state 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py state 14.5 --run
make lab-status
```

Full native backend/eligibility, authenticated transport, effect interruption/
reboot/recovery, terminal resolution/hold release and assembled M2 remain open.
