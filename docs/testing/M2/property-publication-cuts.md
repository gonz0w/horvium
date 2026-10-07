# M2 property metadata publication process-death cuts

Date: 2026-10-06. Status: native qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Contract and proof boundary

This slice exercises the accepted property submission transaction in a separate
reexec of the actual built FreeBSD test binary. The parent waits for an exact
pipe barrier, confirms a competing writer is refused, sends SIGKILL to that owned
child, and reaps the actual signal termination before reacquiring the same lock.
No sleeps coordinate the cut. Child commands take only test-owned paths inside
the disposable guest; no operator disks or state are fixtures.

The private transaction wrapper pauses before/after real operations. It replaces
no write, fsync, close, verification or rename with success. Product behavior and
on-disk schemas are unchanged. Eight boundaries are required:

| Cut | Independently observed state after kill |
|---|---|
| Before publication | Stable lock only; complete empty restart view; no job acknowledged |
| Temporary created, before write | Private empty orphan; inspection/publication refuse uncertainty and preserve it |
| After write | Private canonical temporary; orphan remains unresolved and preserved |
| After file sync | Synced canonical temporary; no published job or automatic promotion |
| Before rename | Verified, closed canonical temporary; no automatic cleanup or promotion |
| After rename, before directory sync | Canonical queued job visible; visibility does not prove durability |
| After directory sync, before return | Canonical queued job visible after real directory sync |
| After actual API success return | Canonical queued job and current metadata retry remain unchanged |

Each cut proves barrier reach, actual writer contention, SIGKILL/reap, lock
reacquisition, exact object/inode/content preservation, safe restart interpretation,
safe metadata retry/check behavior and lease close. Independent file snapshots
compare all ordinary native metadata except access time affected by fixture reads.
Foreign sentinel bytes and complete owned backing removal are separate controls.

Unpublished orphans yield no partial reservation/restart view. A submission retry
on those fixtures returns zero/false/uncertain without deleting or changing files.
An empty prepublication view allows only the pure candidate check; it creates no
record. Visible queued jobs project reconcile-required/review-expired while
retaining the original durable event and whole-pool hold. Exact metadata retries
return the existing record without rewriting or syncing it; particularly after
the pre-directory-sync cut, this is no fresh durability acknowledgement.

The accepted bounded restart-page tests also execute on both native targets:
both formats/all lifecycle states, complete 1,064-record traversal, invalid cursors
and page limits, independent copies and concurrent readers. They still prove
metadata presentation, not a startup consumer or recovery coordinator.

Require all 777 state/backing assertions, 66 parents and 1,876 distinct model/codec
children, without skips/failures, on both declared FreeBSD targets. This includes
66 new cut controls and a thirteenth exact backing/teardown marker. Supported
metadata backing remains ZFS or UFS with soft updates disabled; actual UFS
soft-updates refusal controls remain required. Record OS root and actual metadata
fixture backing separately. Preserve every prior manifest's original source.

## Procedure and limits

Failing-first evidence tests prove the old 710-case log cannot qualify these cuts,
and refuse any missing/failed/duplicate cut control, parent or backing marker.
Use focused tooling/behavior checks during development. The changed shared native
selectors/evidence verifier require one complete portable review. Execute both
guest suites sequentially on the same clean signed source:

```sh
PR_TITLE='test(storage): qualify property publication process-death cuts' make review
.tools/venv/bin/python3 tools/local-qualification.py state 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py state 14.5 --run
make lab-status
```

These are actual process-death metadata cuts while the guest kernel stays running.
They do not qualify guest reboot/power loss, controller/cache durability, native
ZFS mutation/old-executor quiescence, the product helper or HTTP acknowledgement
gaps, restored metadata, durable reconciliation, orphan repair, cancellation,
reservation release, replay-window tombstones or complete M2 recovery/acceptance.
No runtime consumer, service, deployment or storage write authority is enabled.
All owned child processes, leases, mounts/devices, guest images/keys/binaries and
firmware variables must be removed, preserving inert base/runtime caches.
