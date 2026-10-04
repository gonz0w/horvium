# M1 collector, cache and query evidence

Status: implementation under review; native integration passed on both releases.
Approved [EDD](../../milestones/M1/EDD.md) and
[Test Design](../../milestones/M1/TDD.md). No executable, HTTP interface or
installed service is introduced by this library slice.

## Collection and publication

`internal/observation.Manager` accepts the authenticated helper client's typed
`Collect(context, requestID)` contract. One Run worker collects on startup and
every 30 seconds by default, with a 45-second total deadline. Manual refresh
reserves one immediate operation; overlapping refreshes return busy, and ticks
that arrived during work are discarded. There is no queued duplicate work or
immediate retry loop. Sources must honor context cancellation and keep returned
arrays stable while the caller copies them; the helper client returns owned data.

A successful transport can carry partial/unavailable native sections. Publish
that observation atomically with its timestamps/status/problems. A transport,
validation, byte/record-budget or deadline failure preserves existing generations
and exposes only a safe error class. Error text/native output is never exported.
Current attempt metadata and last completed attempt metadata are separate, so a
retry does not erase the previous error or make stale data fresh before success.

Cache retains at most two generations: latest plus last complete while latest
is partial; otherwise latest plus previous latest. Consecutive partial results
cannot evict the last complete context. Without any complete result, latest plus
previous latest are retained. No section arrays are copied between generations.
The complete ID identifies explicit historical context, not an automatically
substituted fresh section. Transport failures do not consume generation slots.

Each manager creates a random 128-bit namespace plus a monotonically increasing
attempt number, suitable for the helper request ID. Successful publication uses
that ID. A cursor contains the exact generation; eviction/restart returns a
reload-required error rather than selecting newer rows. Shutdown cancels work,
clears inventory and prevents restarting the same instance. A new instance has a
new namespace and empty volatile state.

Freshness is per section. LastSuccessfulAt advances only on that section's OK
result; unrelated section success cannot reset it. At 90 seconds without such
success it is stale. Failed/partial sections, explicitly historical generations
and prior transport failures are stale immediately. Native status/problems remain
visible; successful empty lists differ from unavailable lists. Initial absence
of a generation is unavailable, not invented zero metrics. The overview's counts
must always be interpreted with their section status.

## Ownership, limits and queries

Inventory validation is shared by the helper and cache; its existing semantics
are unchanged. `inventory.CloneBounded` bounds records, strings and estimated
model storage before validation maps/cloning, then owns all arrays, pointers and
string backing storage. This also prevents small string slices from retaining a
large source buffer. Nested GEOM relationships/provenance and host memory are
copied. Times are normalized to UTC; no source location object is retained.

Default per-generation bounds are 64 MiB of model/string storage, 100,000 graph/
ZFS items and 4,096 bytes per field. Graph consumers count as records. Helper
framing additionally counts begin/end against its own exchange cap. These are
logical budgets, not exact allocator/RSS limits; allocator/map overhead, incoming
collection/cloning, query indices and GC require whole-application measurement.
Limits are validated trusted configuration, never per-request options.

Host overview and typed provider/pool/dataset pages expose detached data, never
a cached snapshot pointer. Default pages have 100 rows, maximum 500. Filters are
literal case-sensitive name substrings; sorts use an allowlist for each resource.
GUID ordering compares canonical decimal strings numerically without browser/
float precision loss. Numeric capacity sorts use uint64; name/ID ties give stable
ordering inside the generation. Negative/oversized/unknown/control-byte queries
fail; huge offsets return an empty valid page without arithmetic overflow.

Paging/filtering/sorting invokes no helper or native command. At most 16 page
queries are admitted by default (validated 1–64); excess returns busy. Each query
uses bounded indices into the selected cached generation and copies only its
page. It holds a read lock through materialization, preventing evicted snapshots
from accumulating in slow readers. Publication waits for bounded active queries;
no root/native operation runs under that lock. Query context is checked before
admission, during filtering and after sorting; sorting a bounded admitted set is
not interrupted in its comparator. Caller-retained detached pages are outside
the cache's two-generation ownership limit.

## Tests and traceability

Local `make check lint test secrets go-check` covers startup, single flight,
manual/periodic deduplication, cancellation/deadline, invalid budgets, ownership,
partial/failure publication, retry visibility, freshness boundary, two-generation
retention/expiry, restart namespaces, paging/filter/sort, query admission, and
cached reads that never collect. Test-first snapshot-copy/collector cases failed
before implementation. A failing regression caught retry clearing stale failure
state and now protects last-completed-attempt semantics.

A deterministic 100,000-item boundary fixture includes 99,999 datasets and one
pool; the next dataset fails without replacing prior data. Ten concurrent workers
perform 1,000 cached reads with only the startup source call. Provider identifier
slices, host memory pointers and metadata arrays cannot be mutated through queries.
This is functional scale evidence, not full application performance acceptance.

Native tests opt into disposable QEMU through the existing product job on
FreeBSD 15.1/14.5 amd64. Root creates only a protected temporary Unix socket and
copies the test binary. The child drops to `nobody` before helper connect and
runs the actual manager/helper client against the real native reader. Parent
stops the helper; a refresh preserves stale cached context. Parent restarts the
helper; a subsequent refresh publishes a new generation and retains the previous
one. Collector shutdown clears the cache. No host users, pools or services are
changed by this integration test; the separate existing native reader fixture
still creates/tears down its owned md/pool in the same disposable guest.

| Test ID | This slice | Remaining full M1 gate |
|---|---|---|
| M1-FAIL-001 | Stale context, section freshness/errors, retry/error/deadline publication | HTTP/UI error display and interrupted service processes |
| M1-CONTRACT-001 | Shared typed generation/paging contract and expired cursor result | Authenticated CLI/full-page/fragment parity |
| M1-SCALE-001 | One collector, no duplicate queue, two bounded generations, query limits and 100,000-item boundary | Whole application/reader workload budgets |
| M1-RESTART-001 | Volatile cache lifetime and real helper outage/recovery integration | Sessions, installed processes and independent file daemons |
| M1-NATIVE-001 | Actual helper-to-unprivileged-cache path | CLI/UI and complete independent projections |

## Execution record

[Native run 37168464257](https://github.com/gonz0w/horvium/actions/runs/37168464257)
passed all five jobs on implementation and suite source
`10c2555c60b0c839ecd46d07fe58479d9158017e`. Both product jobs executed
`TestNativeCollectorHelperRecovery` successfully, along with the existing actual
helper credential/lifecycle tests, native md/pool fixture tests and all cache
unit/boundary tests. The direct `TestNativeCollectorChild` invocation skips by
design; its parent reexecutes it after dropping UID/GID and requires success.

| Guest | Installed/running kernel and userland | ZFS userland/module | Collector/helper recovery |
|---|---|---|---|
| FreeBSD 15.1 amd64 | 15.1-RELEASE-p3 | 2.4.2-1-FreeBSD_g6330a45b0 | Passed |
| FreeBSD 14.5 amd64 | 14.5-RELEASE | 2.2.9-FreeBSD_g079ba86d7 | Passed |

Go 1.27.1 freebsd/amd64, x/sys v0.48.0, verified module cache; Python 3.14.7
installs the SDK for testing only. Runner ubuntu-24.04; VM Action
`a2f9a41fa97f6848b8c3b791087dfcdaa5b473ff`, builder 2.2.8, snapshot QEMU,
6,144 MiB and default CPU. Image digest was not captured; no immutable image or
performance qualification is implied. Exact pins/run metadata are in the
[source manifest](../../research/M1/source-manifest.json).

Local full repository gates, eleven tooling regressions, Go vet/unit/component/
race tests, four bounded parser/framing fuzz targets and Gitleaks passed. The
implementation PR's initial required checks also passed in
[run 37168572776](https://github.com/gonz0w/horvium/actions/runs/37168572776).
The native recovery test stops a helper with an idle collector, refreshes through
the outage and recovers. Killing an installed process mid-collection, sessions,
file daemons and full application resource measurements remain separate gates.

Initial local microbenchmark on Apple M5 Max, Go 1.27.1
Darwin arm64, `go test ./internal/observation -run '^$' -bench BenchmarkCachedDatasetPage -benchtime=200ms -benchmem`:

| Synthetic datasets | 100-row page | Allocations |
|---|---|---|
| 100 | 1.97 microseconds/op | 7,040 B/op, 2 allocs/op |
| 10,000 | 87.4 microseconds/op | 88,064 B/op, 2 allocs/op |

These are local averages from a short library microbenchmark, not RSS, p95/p99,
idle CPU, NAS throughput or the PRD's authenticated application performance gate.
Full 60-second warm-up/five-minute CPU, combined helper/management peak RSS,
10-reader latency, browsers and physical hardware remain pending.
