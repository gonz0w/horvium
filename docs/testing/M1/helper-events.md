# M1 helper denial events and transport abuse

Date: 2026-10-04. Status: implementation and local regressions; native execution pending.
Work: [issue 1](https://github.com/gonz0w/horvium/issues/1).
Authority: approved [EDD](../../milestones/M1/EDD.md) observability boundary and
[Test Design](../../milestones/M1/TDD.md) M1-PRIV/FAIL/SCALE/NOCHANGE tests.
M1 remains implementing and unaccepted.

## Behavior

The foreground helper passes its existing process logger into the transport server.
Library callers can still construct a server without logging. Requests rejected by
kernel UID/credential checks emit `helper` / `authentication` / `rejected` /
`permission`. Malformed/unknown/extra requests and oversized frames emit coarse
`protocol` or `limit` classes. Saturated client admission and concurrent collection
emit `capacity`. Read/write/collection deadlines emit `timeout`, cancellation emits
`cancelled`, and failed response validation/serialization produces a failed exchange.
Successful transport exchanges produce no additional event; existing native source
collection events retain their successful/partial observation semantics.

All events use a newly generated helper-local ID, fixed section/outcome/error class,
UTC timestamp and total elapsed duration. The caller's request ID, claimed UID,
actual kernel UID/group, payload, path, raw error, native output and inventory never
enter the exported event. Socket closure and collection-lock release precede log
recording. Credential denial still happens before request reads or collection.
Socket expiry can precede context expiry; classification checks the owned deadline
rather than exposing a framing error as the cause of an expired read.

The same single writer, 64-record queue and 60-record-per-minute process budget
cover startup, native collection and denials together. Logger calls do not wait
for sink I/O. Floods can suppress any category and queued events can be lost on
exit: sampled operational events are not a durable security audit ledger or proof
that no denial occurred. No new config, network listener, telemetry destination,
file-service command, storage mutation or persistence mechanism is introduced.

## Procedure and evidence

A behavioral UID-denial regression failed with a missing event before the logging
implementation. Portable Unix-socket tests cover wrong UID/credential error with
no payload, forged operations/fields, oversized frame headers, idle deadlines,
busy collection, cancellation, over-limit failed response and lock release.
They assert zero unauthorized collections and absence of raw caller/error data.
A blocked sink with a hundred rejected connections must permit a subsequent
healthy inventory exchange; existing logger regressions bound volume/queue growth.

The guarded `TestNativeHelperDenialEvents` uses the actual FreeBSD credential
wrapper, root-owned socket and separately dropped nobody/daemon UIDs sharing a
fixture group. It checks another UID and root denial, forged field rejection,
oversized header and idle expiry, counts fixed-class events and verifies a valid
caller can collect afterward. Only its disposable socket and copied test executable
are created; shutdown asserts owned socket removal.

`TestNativeBuiltHelperDenialLogging` launches the actual built product helper with
root-protected config/state inside the opted-in guest, captures stderr in memory,
and verifies a root client sending no payload is rejected immediately and produces
the safe permission event through foreground assembly. It verifies orderly process
exit and socket cleanup. Logs are inspected inside the guest; no raw inventory or
credentials are exported. Existing native reader/runtime/SFTP tests run alongside it.

Native tests require FreeBSD/root and `HORVIUM_NATIVE_VM=1`; otherwise they explicitly
skip. Cross-compilation alone is not native evidence. Exact source/run/version
records will be added after native execution. Required browser/accessibility,
native-probe process-kill, SMB/NFS continuity, full abuse review, native-reader
resource limits and package/release gates remain separate.
