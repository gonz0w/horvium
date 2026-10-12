# Current-dashboard native resource fixture bindings

Date: 2026-10-12. Status: test-only implementation awaiting maintainer review.
Owner: [issue 366](https://github.com/gonz0w/horvium/issues/366), under
[M2 gate 131](https://github.com/gonz0w/horvium/issues/131).
Authority: DASH-REQ-006 / DASH-PERF-001 and the approved
[resource Test Design](../../design/dashboards/resource-qualification.md).
Foundation: [prepared adapter](workload-receipt-adapter.md),
[original-inventory predicates](resource-html-predicates.md) and
[strict offline validator](resource-receipt-validator.md).

## Four existing fixture modes

The [synthetic fixture](../../../cmd/horvium/performance_freebsd_test.go)
and [built native-reader fixture](../../../cmd/horvium/reader_resources_freebsd_test.go)
now call the prepared five-case dashboard workload rather than dataset-only
prefix checks. Each phase schedules 1,000 complete HTML requests with ten readers,
one authenticated session and 200 requests per case. The controller derives
legacy aggregate latency fields from the same initial success quantiles. Large
initial aggregate and each case must remain below the unchanged 250 ms p95 budget.
Replacement expiry responses retain their separate summaries; no retries or
workload pacing are added to hide an early completion or failed predicate.

Synthetic 100/10,000 retain the original deterministic typed generator. For both
native-reader modes, the controller uses the actual FreeBSD source to collect
original host, GEOM, pool and dataset facts after the full warm-up/idle, outside
measured requests. Native-small separately runs fixed `zfs list -H -p -t filesystem
-o name,guid,used,available,referenced -r` against its owned disposable pool. The
bounded parser requires exactly 100 distinct owned rows and their names, GUIDs and
integer values to agree with the original snapshot. The earlier independent
count/sorted-name and built-helper owned data GUID checks remain. Native commands
run only inside explicitly opted-in disposable guest fixtures, never on the Mac.

Projection retains original native host/GEOM/pool facts and replaces only the
controller's owned dataset array with the declared command-emitter 10,000 rows.
It owns zero real datasets. The real helper still uses the existing built
command-boundary projection; no runtime collector or native observation is frozen.
The typed clone owns original inputs, and its digest contains no observation
clocks. No inventory dump, names/GUID list, cookie, CSRF token or response body is
published. Original values remain constant throughout both phases: drift fails,
including boot-pool accounting changes. Expectations never rebaseline from HTML
or the management API. This assumption still needs actual native execution.

## Replacement completion and retained observations

The [controller](../../../cmd/horvium/dashboard_native_helpers_test.go) completes
initial load before starting replacement load. It requires that load to remain
active before and after each of exactly three completed refreshes; early completion
fails explicitly. The bounded replacement control returns denied/transport/schema/
timeout errors instead of aborting before the coordinator can retain raw request
outcomes. Existing 409 refresh contention polling and 503 readiness polling remain
controller work, separate from measured HTML requests. Each control has a 60-second
deadline. No failed HTML request is retried or turned into successful work.

`DASHBOARD_WORKLOAD` is a partial, test-only observation containing mode, fixture
count/origin/digest, initial/replacement receipts, sanitized raw outcomes,
completed replacement count and workload result. Failed phase arrays survive even
when no passing phase receipt exists. It is deliberately distinct from the full
`DASHBOARD_RESOURCE` contract and cannot qualify resources. Future reviewed wiring
must combine real resource/procedure/control observations and independent host
bindings; it must not manufacture successful full records from partial output.
A failure in earlier fixture setup or an external interruption remains an incomplete
failed transcript, without invented request samples.

Existing 60-second warm-up, complete 300-second small idle observation, scheduled
30-second collection, 100 ms RSS sampling/count/max-gap, paired-process high-water
RSS and unchanged budgets remain. The separate model-record/byte-exhaustion
recovery and built native command/helper crash, subtree exit, stale retention,
recovery, original boot-state and socket cleanup gates remain in place. Extra
original collection, witness parsing and HTML predicates are controller work
outside the two measured PIDs; their workload pacing effects remain a limitation.
No new evidence of these native controls is claimed by cross-compilation.

## Execution gate and historical evidence

The existing local `performance` runner and guest shell entry point fail before
VM preparation/build/storage execution until independently bound full dashboard
resource/control/provenance/teardown receipts are reviewed. Dry-run description
remains available. Runner result validation rejects all partial or legacy logs,
even with an exit-zero/PASS marker. This temporary explicit gate avoids promoting
the old dataset-only validator to current-dashboard acceptance. Its removal belongs
to the reviewed complete host binding, with behavioral coverage for the strict
candidate/both-release contract; no new switch or suite bypass is supplied.

The unchanged historical validator remains available for historical records.
Original manifests and their failed/skipped limitations remain unchanged. Existing
M1 resource measurements still describe their original sources/workloads. No native
VM was executed for this component, and no root service or operator storage was
used. Rollback restores the two fixture files and runner/shell guard changes without
changing product behavior or removing historical receipts.

## Portable review and remaining acceptance

Behavioral regressions cover independent 100-row witnesses, malformed/duplicate/
changed accounting, immutable original inputs, projection scope, three overlapping
replacements, late/control/initial failures with retained raw outcomes, bounded
replacement completion/errors and fail-closed runner/shell execution. Test command
and JSON samples are synthetic controller inputs, not native observations.
FreeBSD/amd64 cross-compilation verifies wiring compilation only. Required signed
clean review runs `make check lint test secrets`; shared runner changes require the
conservative full portable race/fuzz plan. Owned teardown is verified afterward.

Full independent resource measurements, release/artifact/compiler provenance,
original suite completion/control counts, fixture cleanup and host supervisor/
base/runtime bindings remain next, followed by fresh sequential both-release
execution on the reviewed assembled candidate. [Overview 289](https://github.com/gonz0w/horvium/issues/289),
[Capacity 290](https://github.com/gonz0w/horvium/issues/290), lifecycle #305/#139,
historical startup uncertainty, browser/manual accessibility, continuity,
hardware/commercial and release gates remain open. Runtime writes stay disabled.
