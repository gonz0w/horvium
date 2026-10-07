# M2 bounded job restart projection

Date: 2026-10-06. Status: implementing; portable qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Contract

`JobReservations.RestartPage` exposes bounded, deterministic restart summaries
from one complete immutable mixed-job view. This makes recovery metadata visible
without resetting durable history or enabling replay. It includes both accepted
job formats and every retained state, including terminal records whose pool holds
remain in place. It performs no file access, publication, native observation,
clock/session restoration, executor discovery or storage command.

Each independent `JobRestartSummary` retains job/submission/pool identity and the
last durable transition, including its revision, timestamp, reason and observation
digest. Separate projected state/reason describe conservative restart presentation:

| Last durable state | Projected state | Projected reason |
|---|---|---|
| queued | reconcile-required | review-expired |
| applying or observing | reconcile-required | executor-unverified |
| reconcile-required | reconcile-required | Retained reason |
| succeeded, failed or cancelled | Retained state | Retained reason |

A stored observation digest or terminal label proves no current native state,
executor quiescence, restored-state validation or releasable reservation. The
complete property baseline and session/policy digests stay out of this summary.
Original records, duplicate retry history and all pool holders remain unchanged.
No projected event is appended or durably acknowledged.

Pages contain 1..64 summaries in lexical job-ID order. An empty cursor begins
enumeration; the last returned ID is an exclusive cursor on the same immutable
view. Nonempty cursors must be canonical IDs present in that view. Unknown,
malformed or absent cursors fail; cursors do not authenticate a transport request
or identify a newly collected snapshot. Consumers must retain one view throughout
traversal. An unavailable view or any error returns nil; a complete empty/end page
returns a nonnil empty slice. The view's existing ceilings remain 64 nonterminal,
1,000 terminal records and 64 MiB. Sorting allocates only a bounded ID index;
full encoded records and review baselines are not copied into each page.

`BuildJobReservations` accepts caller-owned records and grants no root provenance.
The accepted native inspector supplies a lease-bound view, but no startup consumer
or native recovery adapter is added here. Complete policy/store validation,
old-executor boot/process attribution and containment, native GUID/postcondition
reconciliation, durable reconciliation transitions, cancellation/release,
metadata restoration and acknowledgement/reboot/power-cut gates remain open.
Runtime writes stay disabled.

## Focused qualification

Failing-first tests require the missing projection API before implementation.
Five test parents cover thirty lifecycle/format cases, whole-view order/input/output
isolation, exclusive traversal across 1,064 retained jobs, invalid page bounds and
cursors, complete-empty versus unavailable/damaged views, unchanged retry history
and holds, and 32 concurrent readers. `FuzzJobRestartPage` exercises hostile record,
cursor and integer inputs while checking zero partial results and no replay.

```sh
make test-focus PKG=./internal/storage TEST='^Test(RestartPage|Reservation|JobLifecycleAndRestart|PropertyJob)'
PR_TITLE='feat(storage): expose bounded job restart projections' make review
make lab-status
```

Use the ordinary mandatory baseline plus affected-package race and all storage
fuzz targets. Cross-compile the package/tests for the FreeBSD runtime target.
No FreeBSD syscall, native store, publisher, test selector or fixture changes in
this slice; fresh guest runs would not exercise this unconnected portable API.
Prior native manifests retain their actual sources, and are not relabeled as
qualification of this projection or actual restart recovery. No complete-suite,
performance, physical hardware/client or M2 acceptance claim follows.
