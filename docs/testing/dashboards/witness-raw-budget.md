# Single-original witness limit through the broker

Status: inactive test-only control; candidate results pending.
Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[accepted EDD](../../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-007 / DASH-WIT-NATIVE-005.

This control verifies the unchanged 67,508,864-byte single-original allowance
through the real broker and authenticated product manager. First a normal product
helper supplies an independently confirmed native-small publication. Then a
**separate test executable** replaces the source helper for a negative control:
it sends exactly 67,508,865 zero bytes after the actual original request. These
malformed synthetic bytes are neither trusted normal helper traffic nor native
accounting, valid replies, publication or resource workload samples. The normal
product helper's bounded transport cannot ordinarily exceed the derived raw cap.

## Boundaries and observations

The optional helper starter is private to the disposable test harness. Both
cases preserve original actor/executable/socket checks, privilege-dropping
connector/handoff/wait/EOF, the actual management process and its authenticated
HTTP session. The negative source records the actual kernel peer UID/PID, exact
request bytes/ID, source actor/socket and acknowledged byte/hash/error/timing.
The peer PID is the connector, reaped before the observer sends the request;
compare it with the original observer identity/wait receipt, not the manager PID.
Accepted source writes alone are not proof that the receiver retained those bytes.

The broker must independently seal exactly the raw cap with `raw-limit`, no
syscall capture error, completion, forward, original index or publication. The
archive allowance must still exceed the raw cap; its prior amount is only the
actual preceding normal originals. No pressure file, arbitrary seeded counter,
smaller mount, raised/reduced cap or changed scheduling/deadline supplies failure.
An actual authenticated refresh (or genuine busy join) stays denied without a
second connector or changed first boundary/original. Retain actual manager status
and unavailable response bytes. All original bytes are exported/reopened by
object and SHA-256; synthetic zeroes have no reconstructed frame/numerical oracle.

The new failing native test was written before its test producer/fixture starter;
the initial cross-build failed on the missing fixture types/functions. This is
compile-stage evidence only. Behavioral coverage is the native two-case test,
which cannot run portably because it observes real credentials, process waits,
tmpfs, sockets and owned ZFS. Existing portable wire/custody/forward tests remain
part of focused review; no portable mirror of FreeBSD behavior is introduced.

## Reproduction and teardown

After clean signed-source review and original image/compiler/QEMU/firmware/build
and ABI preflight, run the fixed selection on both pinned disposable releases:

```sh
cd /root/work/cmd/horvium
env HORVIUM_NATIVE_VM=1 HORVIUM_PERF_VM=1 /root/work/.tools/dashboard-native.test -test.v -test.count=1 -test.timeout=15m -test.failfast -test.run='^TestNativeDashboardWitnessRawBudget$'
```

The outer diagnostic and archive-transfer allowances do not alter the original
45-second collection deadline or unchanged 320 MiB tmpfs / 256 MiB archive /
32-attempt / 64 KiB frame bounds. The negative source has a bounded socket deadline
and accepts one request only, then remains alive for identity readback until its
owned SIGTERM/wait. No installed service or persistent runner is needed.
Independently reopen source and broker receipts/originals, normal versus synthetic
artifact roles, actual request/connector identity, hashes and permanent denial.
Verify original configuration/token, processes/sockets/tmpfs/ZFS/md and host
image/key/listener/firmware teardown, then lab-stop/status. Preserve failure
observations and missing/skipped facts; no same-source until-pass retries.

This establishes one caller/broker raw-limit negative control with an explicitly
malformed test producer. It establishes no ordinary producer-scale or numerical
accounting claim, projection-specific exhaustion, integrated admission ordering,
full cancellation/recovery, cadence/overhead, resource/activation, browser/manual,
hardware or release acceptance. Historical failures remain unchanged. Canonical
resource instrumentation stays inactive and runtime storage writes stay denied.
No runtime interface, schema, dependency, service or Action changes.
