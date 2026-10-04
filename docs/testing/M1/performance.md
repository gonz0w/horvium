# M1 combined-process performance qualification

Date: 2026-10-04. Status: procedure implemented; native execution pending.
Work: [issue 1](https://github.com/gonz0w/horvium/issues/1).
Authority: approved [PRD budgets](../../milestones/M1/PRD.md) and
[Test Design M1-PERF-001](../../milestones/M1/TDD.md).

## Fixture and measurement

The separate manual [workflow](../../../.github/workflows/bsd-performance.yml)
is also callable from the existing BSD workflow with `performance=true`. It
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
observation, and the explicit cap/recovery
record; a green VM boot or skipped test is rejected.

Native source/run IDs and measured results will be added
only after actual execution; pending is not pass.

The helper fixture includes the test binary and generated typed models, but omits
native command/XML allocations. This qualifies the management executable, cache,
authentication, rendering, HTTP and real helper framing with synthetic observations;
it does not prove worst-case native collector RSS or physical throughput. The
16 GB MicroServer and larger commercial environments remain unmeasured. No
installable port/image, full M1 acceptance or production support follows.
