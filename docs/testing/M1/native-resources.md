# M1 native reader resource and integrated crash qualification

Status: procedure implemented; native execution pending. Owner: Cam (`@gonz0w`).
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
native command
under the built helper's timeout child, stops that command and confirms parentage
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
verification only. Exact native source/run IDs, measurements and versions will
be recorded after actual execution. Until then these additional gates are pending.
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
passed. The full pinned resource/crash/stale/recovery workflow is running separately
in 37233391796; the short test alone does not qualify its measured budgets.
