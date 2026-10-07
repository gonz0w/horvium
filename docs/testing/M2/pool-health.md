# M2 read-only pool health and scan observations

Date: 2026-10-07. Status: implementation; native qualification pending.
Authority: [approved PRD](../../milestones/M2/PRD.md),
[EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-HEALTH-001, M2-SCRUB-001, M2-DRIFT-001 and M2-NATIVE-001 evidence.

## Observation contract

`PoolHealthSource.Collect` validates a canonical pool GUID/name before native
access. It runs six fixed read-only probes through the accepted native runner:
release, pool state, GUID status, path status, pool state, release. The pool state
projection includes permanent GUID, import GUID and all nine existing properties.
Paired complete status views must agree on membership, health, counters,
diagnostics and exact scan text. Require identical before/after pool state and
release, supported unmixed 14.5/15.1 kernel/running/userland format, expected pool
GUID and matching property/status health. Complete qualified legacy output may
omit the scan section: retain false `ScanReported`, empty raw scan text and typed
`unreported` state. This cannot attest idle/quiescence or an outcome.

Keep one shared 4 MiB consumed stdout/stderr budget, existing per-parser field/
line/topology bounds, a ten-second child timeout and sixty-second total deadline.
The accepted runner kills and reaps native children. Failure, cancellation,
unknown format, unsupported release, budget exhaustion and ordinary drift return
zero. No retry, cached substitution or partial healthy result is supplied.
Sequential identity checks are not an atomic snapshot or execution fence.

Return owned pool state, topology and a typed `PoolScan`. Preserve degraded,
offline and nonzero root/leaf/data-error facts; do not apply mutation eligibility
checks to an observation. Existing property eligibility checks stay unchanged.
The result contains no eligibility, policy, job, ledger or executor authority.
There is no new CLI, HTTP handler, service or write path.

## Legacy scan format

Accept complete C-locale scrub and ordinary resilver records: none, running,
paused scrub, cancelled and completed. Preserve reported hundredths of percent
only for running/paused scans. Completed scans carry the native error count;
unreported/none/cancelled/running/paused do not invent a final error count. Completed/cancelled
scans do not invent progress. Native sizes and rates remain rounded display
strings, not exact byte counters. Dates retain canonical native labels without
inventing timezone, boot identity, process identity or executor attribution.
Estimates are native labels only; paused scans cannot report rates/estimates.
Error scrubs, sequential rebuilds, unsupported status sections, malformed dates,
nonfinite/over-100 percentages and incomplete/localized/contradictory text fail
unavailable. Full JSON/error-scrub/rebuild adapters remain open.

Adopt the common formatter identified in the pinned primary sources:

- [FreeBSD 14.5 zpool formatter](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/cmd/zpool/zpool_main.c):
  `print_scan_scrub_resilver_status` and `secs_to_dhms`.
- [FreeBSD 15.1 zpool formatter](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/cmd/zpool/zpool_main.c):
  the same legacy grammar, separate error-scrub/rebuild sections.
- [Bundled size formatter](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/lib/libzutil/zutil_nicenum.c):
  native human size rendering; scan buffers contain at most six display bytes.

No terminal scan observation acknowledges a job, attributes execution, resolves
recovery or releases holds. Actual scrub start/stop, external scan ambiguity,
checksum fault injection, native progress/pause/cancellation and durable attributed
lifecycle remain separate qualification gates.

## Test and native fixture

Failing-first portable tests cover the reported states, exact counters, malformed/
unknown/truncated text, deterministic fuzz decoding, unsupported host, selection
before I/O, each failed probe, consumed budget mismatch, cancellation/deadline,
identity/release/status drift, qualified scan omission, owned results and unhealthy facts.

Reuse the existing disposable `job-status` suite without modifying its shared
runner/checker/selector or time limits. Preserve all 21 original status assertions
and fifteen assessment cases. After assessment preservation, the already-owned
unmounted mirror adds `native-health` with six mandatory children: `fresh-pool`,
`wrong-guid`, `missing-pool`, `scrub-completed`, `degraded`,
`restored-and-preserved`. Native-only fixture commands scrub its selected owned
pool and offline/online its owned vnode leaf. Require actual completed scan facts,
degraded/offline observations and full restored raw property projection.
Progress may finish before collection; never infer unobserved running/paused facts.
A thirty-second child context bounds each test convergence wait, inside the
unchanged two-minute parent deadline. Require every child/group once with no
FAIL/SKIP, GUID/backing verification before destruction, absence after detach,
fixture removal, base/runtime preservation and complete host/process teardown.

```sh
make test-focus PKG=./internal/storage TEST='^TestPoolScan'
make test-focus PKG=./internal/host/freebsd TEST='^TestPoolHealth'
PR_TITLE='feat(storage): observe native pool health and scan state' make review
.tools/venv/bin/python3 tools/local-qualification.py job-status 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py job-status 14.5 --run
make lab-status
```

Preserve all 52 prior M2 manifests, existing storage/native-source code/tests and
shared tools. Historical 819-state and 88-pool evidence retains its original scope.
These targeted observations do not qualify a native writer, attributed scrub job,
recovery coordinator, complete M2 or hardware/client/commercial scale support.
