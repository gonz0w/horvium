# Single-original witness limit through the broker

Status: inactive test-only control; source-bound results below.
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
HTTP session. The full compiler/probe receipt remains original evidence. A separate root-only
ABI control frame supplies the exact observed layout without multiline compiler
prose, which the control grammar deliberately rejects. The negative source records
the actual kernel peer UID/PID, exact
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
env HORVIUM_NATIVE_VM=1 HORVIUM_PERF_VM=1 /root/work/.tools/dashboard-native.test -test.v -test.count=1 -test.timeout=15m -test.failfast -test.run='^TestNativeDashboardWitnessRaw(Budget|Producer)$'
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

## Preserved first candidate

Source `82394994a446df5da616906694172fcf65b2e678` failed on 15.1 in
153.16s: normal publication passed (42.22s), then the negative source did not
become socket-ready (76.07s). Cleanup also reported a signal failure when it tried
SIGTERM. Its stdout/stderr were not retained, so the actual child exit reason is
unknown; no negative source request/capture receipt exists. The original normal
files, failed observations and cleanup remain preserved. Guest/host teardown
passed; 14.5 was not started at that source. This is not raw-limit evidence.

The retained original compiler version contains newlines. The source attempted
to read the entire probe with the deliberately stricter control grammar; portable
coverage reproduces that rejection and verifies a separate ABI-only bootstrap.
This is an identified input mismatch, not an observed native child diagnostic.
The corrected source preserves full provenance separately, checks the dedicated
ABI frame and retains exclusive source stdout/stderr with early exit/wait
observation. A later passing source cannot upgrade the first candidate.

The next source `dbd3d3db3642ec8b5f500234f5baa8d1bc63084c` was
operator-stopped during 15.1 guest preparation, before any native test transcript
or original capture. Static review found that a broad test selection would try to
run the configured producer with no arguments. 14.5 was not started. The original
worker report remains interrupted/unqualified with `host_teardown=false`; owned
supervisor recovery removed its image/key/variables and stopped the process group.
Recovery observations are separate from missing normal worker postflight/teardown.

The final selection includes the producer's unconfigured entry explicitly. That
entry must skip before authorization/setup; the configured owned child still
executes its real negative source. This guards broad native selections without
weakening the configured negative. Both fresh release runs and independent host
readback are required at the corrected source.

## Source-bound component results

Signed clean reviewed source `43af5c838c88a67f41e213f535e2f5827ba8d553`, base `349513adc0bc0a726b4476d3e726545f55328cb3`,
passed both cases on both pinned FreeBSD/amd64 releases. The
[immutable manifest](witness-raw-budget-manifest.json) retains original command,
source/build/compiler/ABI/preflight, source request/peer/wait, actual original
object/hash and cleanup bindings for that exact source.

| Release | Focused native result |
| --- | --- |
| 15.1 | TestNativeDashboardWitnessRawBudget (129.83s) |
| 14.5 | TestNativeDashboardWitnessRawBudget (129.79s) |

Independent readback reopened the separate source receipt and its matching case
copy: **67,508,865 acknowledged synthetic source bytes** and **67,508,864 actual
sealed broker bytes** per negative. The original request/ID and dropped-privilege
connector UID/PID matched the actual observer identity/wait receipt. All connector
waits completed, while the source actor/socket remained available for readback.
Both source writes and half-closes recorded no error; those observations remain
separate from receiver retention or publication. The unconfigured producer entry
explicitly skipped on both releases while both configured negative children ran.

Each broker retained `raw-limit` without a capture syscall error, completion,
forward, original index or publication. A single real authenticated refresh/busy
join stayed denied without another connector or changed failure/original. Actual
manager status/unavailable bytes, normal product originals/indexes/publication,
unchanged config/token and guest/host teardown were independently verified. The
actual preceding normal bytes plus the negative stayed below the archive cap;
that bound did not supply the failure. Synthetic data remains explicitly distinct
from normal product helper replies and numerical/resource evidence.

Fresh check/lint/test/secrets and affected `cmd/horvium` race/one compiler-listed
portable fuzz target passed at the native source. Final prose/evidence followup
may reuse this executed exact-input deep evidence with a fresh baseline. All 176
historical manifests and seven operator files remained unchanged; lab-stop/status
and both owned guest/host teardowns passed. The first candidate `8239499` failed
source readiness on 15.1 after normal publication; missing child diagnostics remain unknown and 14.5 was not started.
Its observed input mismatch/portable reproduction is separate from an actual
native child exit reason. Source `dbd3d3d` was stopped before native execution for
the unconfigured-entry guard correction; its interrupted worker report still
records no completed host teardown, separately from verified supervisor recovery.
The preserved first candidate and historical failures/skips retain their original
scope. These component controls do not close wider acceptance or activate the resource runner/runtime.
