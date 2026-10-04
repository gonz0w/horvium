# M1 helper denial events and transport abuse

Date: 2026-10-04. Status: native tests passed; slice 8 under review in
[PR 11](https://github.com/gonz0w/horvium/pull/11).
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
exit. Parent shutdown can also cancel logging before a cancellation event is
admitted: sampled operational events are not a durable security audit ledger or proof
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
records follow below. Required browser/accessibility,
native-probe process-kill, SMB/NFS continuity, full abuse review, native-reader
resource limits and package/release gates remain separate.

## Execution record

Functional source: `0096709e0a155701425c17ab6a2cac037a7cdaec`.
[Native run 37221158595](https://github.com/gonz0w/horvium/actions/runs/37221158595)
passed all five executed jobs. Both product targets actually passed
`TestNativeHelperDenialEvents`, the logged `TestNativeAdmittedClientsBounded`,
and `TestNativeBuiltHelperDenialLogging`, together with all portable denial,
blocked-sink recovery, cancellation and failed-exchange regressions. All existing
reader/helper/cache/auth/console/CLI/runtime and independent SFTP fixtures passed.
The optional long performance job was intentionally skipped; existing synthetic
budget evidence belongs to its accepted revision, not this native denial run.

| Guest / architecture | ZFS userland/module suffix | Go / syscall module |
|---|---|---|
| 15.1-RELEASE-p3 / amd64 | 2.4.2-1-FreeBSD_g6330a45b0 | 1.27.1 / x/sys v0.48.0 |
| 14.5-RELEASE / amd64 | 2.2.9-FreeBSD_g079ba86d7 | 1.27.1 / x/sys v0.48.0 |

The native action source was `a2f9a41fa97f6848b8c3b791087dfcdaa5b473ff`,
builder 2.2.8, Ubuntu 24.04 hosted runner, 6144 MiB guest memory, default CPU/image
selection. This is native functional evidence; it does not extend the pinned
performance qualification. Test procedure, statuses and functional-source hashes
are recorded in the [source manifest](../../research/M1/source-manifest.json).

Local macOS 27.0 arm64 `make check lint test secrets build go-check` passed,
including race detection and five bounded fuzz targets. Additional protocol
cases passed the required checks and native suite. FreeBSD amd64 helper and
command test binaries cross-compiled. The final evidence commit changes docs only;
no operator-host service was enabled. M1 remains unaccepted.
