# Portable dashboard resource request driver

Owner: [issue 358](https://github.com/gonz0w/horvium/issues/358), under
[M2 gate 131](https://github.com/gonz0w/horvium/issues/131).
Requirements: DASH-REQ-006 / DASH-PERF-001.
Authority: [adopted resource Test Design](../../design/dashboards/resource-qualification.md),
[PRD](../../design/dashboards/PRD.md), [EDD](../../design/dashboards/EDD.md),
[Test Design](../../design/dashboards/TDD.md).

## Implemented boundary

[Controller helpers](../../../cmd/horvium/dashboard_workload_helpers_test.go)
are compiled only into tests. They add no runtime flag, route, schema, package,
service, dependency or native operation. The existing native performance fixture
and legacy evidence validator remain unchanged. This slice cannot run or qualify
the adopted five-case native resource procedure by itself.

The prepared plan schedules 1,000 requests across ten workers, each owning 100
fixed indices and 20 requests of each case. Case is index modulo five: Overview,
pools default/maximum, datasets default/maximum. Default limit is omitted;
maximum is 500, name order is explicit, pool offset is zero, and dataset offset
is floor(index/5) times limit modulo the declared count. Overview has no query.
Invalid counts, changed plans or non-loopback HTTP origins fail before requests.
The declared count comes from a fixture, never a hidden request in the timed loop.

Every scheduled request gets an index, case, HTTP status, outcome and monotonic
request/read/close duration. Complete bodies up to the existing 32 MiB console
ceiling are read and closed; limit+1, read/close/transport failures and redirects
fail. No early stop or retry changes the distribution. A missing response predicate
fails before HTTP. The mandatory predicate runs after timing stops and must validate
exact original inventory, full tables/comparisons, freshness and generation links,
or safe expiry without successful projections. Its controller costs are outside
the paired PIDs, but its pacing affects the observed workload. A permissive callback
used in an offline scheduling test is not an HTML validator or service qualification.

Initial observations require all 1,000 successes. Only replacement-phase Overview
may record a predicate-verified 409 expiry. Every replacement case still requires
successes, and all 1,000 outcomes must be accounted for. Unclassified denials,
failed predicates, duplicate/missing observations, wrong indices/cases/statuses,
unknown phases/outcomes and nonpositive durations cannot produce a valid summary.
Raw observations are not reordered or changed during percentile calculation.

Success and expiry durations have separate nearest-rank p50/p95/p99 summaries,
aggregate and per case. Initial large-workload p95 must be below 250 ms for every
case and aggregate; a faster aggregate cannot hide a slow Overview case. No new
small-workload latency threshold is introduced. These helpers do not measure RSS,
idle CPU, warm-up, replacements, original native commands, source-model limits,
crash recovery, provenance or teardown; those existing requirements remain.

## Behavioral verification

[Portable regressions](../../../cmd/horvium/dashboard_workload_test.go) exercise:

- Exact five-case plans at 1/100/137/10,000 datasets, per-worker distribution,
  omitted default limit, maximum limit, rotating offsets and invalid count denial.
- Complete zero/small/32 MiB bodies, ceiling+1 and failed read/close denial.
- Aggregate/per-case nearest-rank values, immutable raw observations, separately
  accounted expiry, malformed records, all-expiry denial and a masked slow case.
- Actual loopback HTTP requests with all-success, mixed expiry, unsafe-predicate,
  initial/non-Overview denial, redirect and transport-failure schedules. The
  concurrency barrier observes exactly ten active readers and no login follow.
- Missing predicates, changed routes/counts/queries/indices/cases and invalid
  origins rejected before HTTP.

Run `make test-focus PKG=./cmd/horvium TEST='^TestDashboard'` for development.
Final signed review runs `make check lint test secrets` and affected-package
race/fuzz checks via `make review`; a FreeBSD/amd64 test cross-build verifies
compilation only. The source-bound receipt is added after actual review execution.
No VM, browser, helper process, management service or operator storage is needed.
Owned test sessions still require verified teardown.

## Remaining work and rollback

Next review original-inventory HTML predicates and their mutation regressions,
then the complete candidate/transcript/release receipt validator and integration
with all four native modes. Only after fixture review run a clean signed assembled
candidate on both pinned releases, preserving warm-up/idle, replacements, native
crash/cap-recovery, paired RSS/CPU and disposable teardown requirements. Historical
measurements retain their original scope; this slice supplies no fresh measurements.

Overview/capacity delivery gates #289/#290/#131, native lifecycle #305/#139,
manual/browser/accessibility and file-daemon continuity remain independent.
Runtime writes remain disabled. This adds no hardware/commercial support claim.
Rollback removes the two test files and this documentation; product/native fixture
behavior and old receipt validation are unchanged.
