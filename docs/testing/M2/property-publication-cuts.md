# M2 property metadata publication process-death cuts

Date: 2026-10-06. Status: both native targets qualified; accepted in PR 59.
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

## Executed qualification

Signed qualification source `a3a33d5929f7fba32fd9ab6364320984b49d9162` passed all 777 state/backing
assertions, 66 parents and 1,876 distinct model/codec children on both native
targets without skips/failures. The [sanitized manifest](property-publication-cuts-manifest.json)
records exact source/base/input/compiler/guest/runtime/artifact/utility/log hashes,
case inventories, unchanged product Go inputs, prior evidence and full teardown.

The changed shared selectors/verifier required one complete portable review.
All mandatory baseline checks (103 tooling regressions), eleven-package race
and all 28 compiler-listed portable fuzz targets passed in
137.015 seconds. Documentation-only followups must rerun the
baseline/title/signature/DCO/source checks; any deep reuse retains this original
executed source.

| Target | OS root | Cut fixture backing | Process-cut parent | Full owned guest | Result |
|---|---|---|---|---|---|
| FreeBSD 15.1 amd64 | zfs | zfs | 0.51 s | 72.641 s | All required controls passed; full teardown |
| FreeBSD 14.5 amd64 | ufs | ufs (soft updates off) | 0.52 s | 60.253 s | All required controls passed; full teardown |

Both guests used the same verified cross-built artifact. All eight child processes
were actually killed/reaped per target, then every owned lease/backing and guest
image/key/test binary/firmware variable/process group was removed. Inert base and
runtime caches were preserved. All 34 prior M2 manifests are byte-identical to
the base. `make lab-status` verified all 236 owned sessions stopped.
Actual native executor recovery, reboot/power-cut durability and complete M2
acceptance remain open.


## Preservation investigation, 2026-10-10

During execution-domain system backing work, one full FreeBSD 15.1 state run
failed `after_directory_sync_objects_preserved`. A focused repetition later
failed `after_return_retry_safe` after 56 passes. These are failures of the
historical exact object comparisons, and their records remain unqualified.
The original source-bound PR 59 manifest is unchanged.

Sanitized field/role diagnostics now cover both snapshot comparisons without
logging record values or identities. All ordinary metadata except access time
and exact bytes remain required; no comparison or runtime behavior was relaxed.
A subsequent 300-repeat diagnostic passed, as did separately recorded full
reruns; focused repetition is not complete-corpus qualification. The cause
remains unresolved and needs resolution for release acceptance. See the
[system backing record](execution-system-backing.md) and its
[manifest](execution-system-backing-manifest.json) for actual sources, failures,
passing gates, diagnostic scope and teardown.
