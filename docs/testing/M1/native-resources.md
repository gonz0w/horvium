# M1 native reader resource and integrated crash qualification

Status: native qualification passed; under maintainer review. Owner: Cam (`@gonz0w`).
Date: 2026-10-04. Requirements: M1-REQ-001,002,006,007. Tests:
M1-NATIVE-001, M1-PERF-001, M1-FAIL-001, M1-RESTART-001, M1-NOCHANGE-001.
This supplements [synthetic performance](performance.md) and
[executor crash containment](crash-containment.md); neither prior result measures
these additional procedures.

## Disposable procedures

The checksum-pinned performance workflow executes both existing synthetic
workloads and `TestNativeReaderResourceBudgets` on FreeBSD 15.1/14.5. The same
qemu64 CPU, two vCPUs, 6144 MiB, immutable image hashes and snapshot isolation apply.
No service is enabled on an operator host. Native execution, including XML capture,
base timeout/reaper and TSV decoding, occurs in the disposable guest only.

`native-small` creates a fresh file-backed md pool and 100 unmounted owned
filesystems. The guest's boot filesystems remain observed, so total dataset count
can exceed 100. Independent fixed ZFS reads check count, name ordering and owned
GUID before the actual built root helper and unprivileged management executable
are measured. Host, GEOM, pool and dataset observations all use native projections.
Actual provider/pool counts follow the guest; this supplements rather than replaces
the exactly four-provider/one-pool synthetic budget fixture.

`projection-10000` uses a separately compiled test helper with actual native
Host/GEOM/pool reads. Only dataset projection is replaced by a test subprocess
emitting 10,000 deterministic `fixture/dataset-NNNNNN` TSV rows through the same
native executor, reaper, output cap, decoder, framing and cache. There is no
alternate source in the product executable. The helper test binary adds overhead;
10,000 actual ZFS filesystems and worst-case GEOM topology remain unmeasured.

Both pairs warm for 60 seconds and serve 1,000 complete HTML pages through ten
concurrent authenticated readers; another 1,000 requests overlap three generation
replacements. The small pair measures five minutes of idle Go process CPU with
scheduled native collection. Paired PID RSS samples and the conservative sum of
individual kernel high-water marks are retained; the greater observation is used.
Small RSS <=128 MiB and idle CPU <1% of one core; projection RSS <=256 MiB and
p95 <250 ms. Sampling includes startup and transient reader/parser/cache work.
CPU/RSS budgets cover the two Go processes, excluding native child processes,
OS/ZFS ARC, fixture controller and independent file daemons. Native execution time
still affects collection and HTTP concurrency. These bounds are not whole-NAS
memory or physical throughput claims.

After the small workload, kernel fork/exec tracking identifies an actual fixed
native command under the built helper's timeout child, stops that command and confirms parentage
and stopped state before killing the helper. It requires both captured guard and
command to terminate, the previous generation to remain stale, and a restarted
helper to publish fresh state. Only the captured test-owned socket inode is removed
after SIGKILL. Config and token digest must remain byte-for-byte unchanged; orderly
shutdown must remove the new socket. Owned pool properties and pool/md teardown
are checked by the storage fixture. No production pause/failure injection is added.

The runner validator rejects absent/duplicate measurements, incomplete procedures,
budget violations, missing integrated crash/boot-state results or a skipped suite.
Passing synthetic results alone cannot satisfy this workflow.

## Execution record

Portable validator regression reproduced acceptance of synthetic-only results
before the new gate and passes after it. FreeBSD cross-compilation is build
verification only. The full pinned native resource workflow passed all seven
executed jobs in [run 37233391796](https://github.com/gonz0w/horvium/actions/runs/37233391796),
source `cebc8973f5e37d3ca0bbd5da87dd21dff39703d8`. Both measured suite parents,
the integrated crash/stale/recovery/boot-state assertions and runner validators
passed. Kernel/userland are 15.1-RELEASE-p3 and 14.5-RELEASE amd64, Go 1.27.1,
ZFS 2.4.2-1-FreeBSD_g6330a45b0 / 2.2.9-FreeBSD_g079ba86d7 with matching modules.
QEMU 8.2.2, qemu64/two vCPUs/6144 MiB snapshot guests, builder 2.2.8 and AnyVM
0.7.2; compressed/runtime image hashes were reverified after execution. Exact
pins, counts, sampling/high-water records and file hashes are in the
[source manifest](../../research/M1/source-manifest.json).

| Guest | Native procedure / actual count | Budget RSS MiB | Small idle CPU, one core | HTTP p50 / p95 / p99 ms |
|---|---|---|---|---|
| 15.1 | native-small (115 rows) | 32.81 | 0.0567% | 8.611 / 30.198 / 59.370 |
| 15.1 | projection-10000 (10000 rows) | 44.75 | not an idle case | 15.418 / 43.939 / 112.824 |
| 14.5 | native-small (100 rows) | 36.53 | 0.0867% | 19.403 / 67.597 / 128.056 |
| 14.5 | projection-10000 (10000 rows) | 40.67 | not an idle case | 22.057 / 82.115 / 397.301 |

Both small cases contain 100 owned filesystems; 15.1 additionally observes 15
boot filesystems. RSS uses the greater sampled pair/individual high-water sum.
Large projection p95 is below 250 ms and all memory/idle budgets pass. The 14.5
projection p99 is 397.301 ms; the declared budget targets p95, not all requests.
Nominal 100ms sampling had maximum gaps up to 1210.175ms; conservative kernel
high-water accounting accompanies it, rather than claiming every transient was
sampled. Original synthetic 100/10,000 workloads and record/byte-cap/recovery
cases passed in the same run. Local full check/lint/test/secrets/build/race/fuzz
passed; native fixture-only repairs passed required checks and actual execution.
Hardware, commercial scale, full M1 acceptance and package readiness remain open.

Early capture fixtures failed verification: FreeBSD ps treats `=` as the start
of the final column header, so comma-joined empty-header fields did not return
the intended columns. Separate `-o` arguments correct this fixture parsing.
The attempted larger crash fixture and helper pause were removed; the measured
and crash fixtures again use the same 100 owned filesystems. No budget is relaxed.


The short actual-built-helper capture gate passed on both selected releases in
[run 37233158523](https://github.com/gonz0w/horvium/actions/runs/37233158523), source
`cebc897` (5.51s on 15.1; 4.91s on 14.5). It confirms live stopped native-command
parentage and both command/guard termination after helper SIGKILL. All five jobs
passed. The full workflow above separately qualifies measured crash/stale/recovery;
the short test alone does not qualify budgets. A later combined assembled-source
run (37234167265) verifies subsequent HTTP/browser changes alongside these gates.
