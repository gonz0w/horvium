# M1 combined-process performance qualification

Date: 2026-10-04. Status: synthetic qualification accepted by maintainer merge of
[PR 10](https://github.com/gonz0w/horvium/pull/10), source
`65750aa81aeca0054a371690fbef5a7e7d9fffcc`. M1 remains unaccepted.
Work: [issue 1](https://github.com/gonz0w/horvium/issues/1).
Authority: approved [PRD budgets](../../milestones/M1/PRD.md) and
[Test Design M1-PERF-001](../../milestones/M1/TDD.md).

## Fixture and measurement

Current execution uses the [local runner](../local-qualification.md), selecting
`performance` explicitly for each release. GitHub Actions are disabled. Historical
hosted run inputs and measurements below retain their original provenance. The
original [workflow source](https://github.com/gonz0w/horvium/blob/65750aa81aeca0054a371690fbef5a7e7d9fffcc/.github/workflows/bsd-performance.yml)
verifies compressed builder 2.2.8 FreeBSD 15.1/14.5 images and AnyVM 0.7.2 against
[SHA-256 pins](../../../tools/performance-vm-pins.json). It captures the decompressed
base-image hash and rechecks base/runtime hashes after the snapshot-mode run.
Each VM explicitly uses qemu64, two vCPUs and 6144 MiB. Ubuntu 24.04 hosted runner,
actual QEMU, guest kernel/userland, ZFS, Go and core/memory observations are logged.
Firmware and hosted runner scheduling are not immutable; this is a dated VM
application qualification, not a physical disk benchmark.

`TestNativePerformanceBudgets` requires FreeBSD, root and both disposable native
and performance opt-ins. Each size starts fresh separate processes: the actual
cgo-free management executable under the configured non-root UID and a root
reexecuted test-only helper using the actual authenticated Unix-socket protocol.
No production fake-source mode exists. Generated typed inventory has one pool,
four providers and 100 or 10,000 datasets, with deterministic integer sequence
0..n-1, fixed names/GUIDs/numbers, no random seed and no native storage mutations.
Real native reader fixtures run separately before this measurement.

Both sizes warm for 60 seconds. The small fixture then has five minutes without
client requests; scheduled 30-second collection remains enabled. CPU uses the
sum of accumulated user/system time for the two owned PIDs, divided by actual
elapsed wall time as a percentage of one core. FreeBSD ps's decaying percent-CPU
field is not used. Measurement excludes the test controller, OS/ARC and independent
file services. Sampling overhead can affect VM scheduling and is outside the two
measured processes.

Ten concurrent HTTP readers share one authenticated operator session (session
capacity remains eight). Each executes 100 complete HTML dataset pages, for
1,000 requests, page size 100. Timings include network/body reads and assert the
expected synthetic row. Report nearest-rank p50/p95/p99. A second 1,000-request
workload runs while three explicit generation replacements complete. RSS sampling
covers startup, warm-up, idle, reads and replacement at a nominal 100 ms interval;
report sample count and actual maximum gap. After orderly process exit, sum each
process's getrusage peak RSS, converting FreeBSD KiB to bytes. Summing separate peaks can overestimate simultaneous
usage. Native runs also showed ps samples slightly above the kernel high-water
sum; the two accounting observations are retained and the larger value is used
for the budget. Kernel high-water accounting supplements sampling, but neither
proves an exact instantaneous upper bound. Budgets compare against that maximum: small <=128 MiB and idle <1% of one core;
large <=256 MiB and p95 <250 ms. Shutdown verifies owned socket removal.

A third separate process pair tests 100,000 aggregate model records (99,990 dataset
rows plus class/four geoms/four providers/pool), then one-record overflow and
20,000 rows with 4,000-byte name padding exhausting the 64 MiB model budget.
Transport's fixture record cap is raised to 200,000. The boundary pair alone raises
the transport byte cap to 128 MiB to isolate rejection by the 64 MiB aggregate
model budget; other configured limits remain enforced. Each failed attempt must
finish with an explicit limit state, retain the previous generation stale, and
recover after restoring 100 rows. Boundary-case RSS is reported separately and
is outside the small/large fixture budgets.

## Execution record and limits

Portable behavioral tests first failed before the usage parser, one-core CPU
calculation and aggregate-record fixture were implemented. Tool regressions reject
checksum mismatch and preserve an existing fixture. FreeBSD cross-compilation is
build evidence only. A runner-side [evidence validator](../../../tools/check-performance-evidence.py)
requires both full measurement records, passing budgets using the larger RSS
observation, and the explicit cap/recovery record; a green VM boot or skipped test is rejected.

The first [run 37217655025](https://github.com/gonz0w/horvium/actions/runs/37217655025)
passed existing native tests but its performance jobs only booted VMs: detach mode
skips AnyVM's final guest command. It is not performance evidence. The corrected
[run 37217893498](https://github.com/gonz0w/horvium/actions/runs/37217893498) executed
all native performance tests successfully, including cap rejection/recovery, but
the runner validator failed because sampled RSS was slightly larger than the
getrusage sum. A regression now accepts either accounting observation as larger
and rejects any budget violation in either. Qualification now runs a separate
SSH command after detached boot, with no guest copyback, and compares the greater
RSS value. These repairs keep the approved budgets unchanged.

The helper fixture includes the test binary and generated typed models, but omits
native command/XML allocations. This qualifies the management executable, cache,
authentication, rendering, HTTP and real helper framing with synthetic observations;
it does not prove worst-case native collector RSS or physical throughput. The
16 GB MicroServer and larger commercial environments remain unmeasured. No
installable port/image, full M1 acceptance or production support follows.

## Final measured execution

Functional source: `d85d5d1b4e9a7b1656085eef21ff21def0f50dc9`.
[Run 37218753397](https://github.com/gonz0w/horvium/actions/runs/37218753397)
passed all seven jobs, including both measured workloads, explicit cap/recovery
records, runner evidence validation and unchanged base/runtime hashes. All prior
native reader/helper/cache/auth/CLI/runtime/SFTP fixtures passed too.

| Guest | Datasets | p50 ms | p95 ms | p99 ms | Budget RSS MiB | Small idle CPU, one core |
|---|---:|---:|---:|---:|---:|---:|
| 15.1 | 100 | 12.985 | 44.829 | 117.023 | 33.23 | 0.0200% |
| 15.1 | 10,000 | 15.615 | 65.891 | 152.251 | 43.25 | not measured |
| 14.5 | 100 | 17.336 | 60.665 | 107.420 | 35.33 | 0.0233% |
| 14.5 | 10,000 | 21.402 | 90.322 | 273.227 | 44.39 | not measured |

All approved small/large measured thresholds pass. The p95 bound is not a p99
bound: 14.5's large-fixture p99 reached 273.227 ms. Sampling gaps reached 1,044.28 ms
on the hosted runner despite the nominal 100 ms interval; kernel process peaks
supplement those samples. Both separate boundary tests passed rejection, stale
retention and recovery, with approximately 357 MiB (14.5) / 362 MiB (15.1) combined
sampled RSS. Those raised-transport-cap fixtures are outside the normal-size
budgets and do not imply every cap-bound workload stays below 256 MiB.

| Observed guest | ZFS userland/module suffix | CPU / actual physical bytes |
|---|---|---|
| 15.1-RELEASE-p3 amd64 | 2.4.2-1-FreeBSD_g6330a45b0 | two qemu64 vCPUs / 6,387,654,656 |
| 14.5-RELEASE amd64 | 2.2.9-FreeBSD_g079ba86d7 | two qemu64 vCPUs / 6,393,864,192 |

Both used Go 1.27.1, x/sys v0.48.0, locally embedded HTMX 2.0.11, guest development
Python 3.14.7, host Python 3.12.3 and QEMU 8.2.2
(`1:8.2.2+ds-0ubuntu1.18`). The configured guest memory was 6144 MiB; the
reported physical total was lower. The image/runtime
SHA-256 values, complete RSS/sample/latency records, synthetic seed and source
file hashes are in the [source manifest](../../research/M1/source-manifest.json).

Local macOS 27.0 arm64 `make check lint test secrets build go-check` passed,
including behavioral tooling tests, race detection and all five bounded fuzz
targets. The final evidence revision changes documentation only. No operator-host
service was enabled. Full native-reader resource, browser/accessibility, abuse,
SMB/NFS continuity and release/package gates remain open.
