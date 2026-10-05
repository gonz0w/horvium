# M1 helper transport evidence

Status: implementation under review; actual native tests passed on both releases. This is a library
slice under the approved M1 design, not an installed management service.

## Contract and safety limits

`internal/helper` permits one version-1 `inventory` request per connection. The
client sends a request ID and closes its write half; extra bytes or any other
operation/field fail before native collection. The server authenticates the
kernel peer UID before reading the request; the client authenticates helper UID 0
before sending it. FreeBSD `LOCAL_PEERCRED` through x/sys is the authority; an
unknown credential ABI or lookup error denies the peer. There is no group-only
or root-client exemption.

Configuration is a library value supplied by a trusted caller. It requires a
nonzero management UID and socket GID, an absolute clean socket path, positive
validated budgets, 1–64 admitted clients, and a positive deadline up to five
minutes. No executable/config-file schema or runtime CLI is introduced here.
The future service sets these values from validated root-owned configuration.

The existing directory must be root-owned, group-restricted 0750 with the exact
configured GID. All ancestors must be root-owned without non-root writes; a
root-owned sticky ancestor such as `/tmp` is permitted because it protects the
root-owned child. Symlink components fail. The helper creates a new root-owned
0660 socket with the configured GID. Existing files/symlinks/sockets fail without
adoption or unlinking. Cleanup removes only the listener's own inode; it never
removes a replacement. The client rechecks path permissions before each dial.
It additionally checks kernel listener credentials even if file metadata passes.

Messages are four-byte big-endian lengths followed by strict JSON. A frame is
at most 64 KiB; the header is checked before allocating its body, including before
integer conversion on 32-bit targets. Defaults bound each exchange to 64 MiB of
JSON payload and 100,000 records, including begin/end and each graph/ZFS item.
Fields default to 4,096 bytes and nesting to 32. These are transport budgets,
not an exact Go heap/RSS bound; allocation/whole-application budgets remain an M1
performance gate. Framing overhead is four bytes per admitted record.

Metadata, individual classes/geoms/consumers/providers/pools/datasets, and end
are distinct typed records. Consumers are split so a large geom cannot create an
unbounded array in one record. GUIDs remain decimal strings. Unknown/duplicate/
case-variant fields, missing required fields, invalid UTF-8/control strings,
wrong IDs/versions/types, inconsistent counts/completeness, invalid graph links,
invalid section status and truncated/missing/trailing end data fail the exchange.
A failed receive returns a zero snapshot, never partially publishable state.
Source partial/unavailable sections retain their timestamps/problems.

Socket deadlines and context cancellation cover I/O and collection. Admitted
handlers are bounded and only one collection runs; busy connections fail rather
than queue another root operation. The existing native reader provides fixed
commands, byte/record caps and child-process cancellation. No command/path/device
parameter reaches it. Idle peers expire. Shutdown cancels workers before waiting
and cleaning the socket. The caller/collector must honor its context; the supplied
native reader does. No file service is started, stopped or reconfigured.

## Procedure and traceability

Local: `make check lint test secrets go-check`, including meaningful denial,
malformed-stream, count/size/depth, cancellation/concurrency, unsupported-host,
race and bounded request/response fuzz tests. macOS sandbox execution needs local
Unix-socket binding permission; no FreeBSD peer behavior is inferred from these
injected-credential component tests.

Native: use the [local runner](../local-qualification.md) with `discovery` for
each reviewed release. The guest executes `HORVIUM_NATIVE_VM=1 sh tools/bsd-native-discovery.sh`
in disposable 15.1/14.5 amd64 QEMU guests, including actual Go helper tests.
The test parent remains root; reexecuted children use `syscall.Credential` to drop
UID/GID before connecting. `nobody` is the allowed fixture UID; `daemon` uses the
same allowed socket group but a different UID. No user account is created.
Only generated temporary Unix sockets/test executable copies and the separate
reader's owned md/pool fixture are used. Each guest is discarded.

Native cases include exact UID authorization, another UID/root denial, forged
payload fields, real reader over authenticated transport, unsafe path/mode/owner,
refusal of preexisting entries, idle admission-slot expiry/recovery, preservation
of a root-replaced socket entry during shutdown, and a non-root listener with root-repaired socket
metadata. The last case proves the client uses kernel UID in addition to path
metadata. Child tests do not dump host inventory.

| Test ID | This slice | Still required |
|---|---|---|
| M1-PRIV-001 | Typed policy/component tests and actual FreeBSD wrapper/UID/socket tests | Installed service/configuration integration |
| M1-FAIL-001 | Bounded broken/oversized/incomplete helper exchanges | Cache/UI stale error state and process recovery |
| M1-NOCHANGE-001 | Arbitrary operation/path/forged authority denied before collection | Independent file-service/restart checks |
| M1-SCALE-001 | Admitted-client and in-flight bounds; deadlines/cancellation | Collector/cache and measured whole-application resource budgets |
| M1-BUILD-001 | x/sys v0.48.0 checksum/source/license, no product cgo | Executables, assets and release packaging |

## Execution record

Final [native run 37166751993](https://github.com/gonz0w/horvium/actions/runs/37166751993)
passed all five jobs, including both actual-product jobs, on source and suite
`eb8996f92eaa161be543f4455993242565aa3ca5`. The earlier
[run 37166572843](https://github.com/gonz0w/horvium/actions/runs/37166572843)
also passed; the final run strengthens wrong-UID denial with an otherwise-valid
request and adds admission recovery/ancestor/replacement cases.

| Guest | Kernel/userland | ZFS userland/module | Product helper cases |
|---|---|---|---|
| FreeBSD 15.1 amd64 | 15.1-RELEASE-p3 for installed/running/userland | 2.4.2-1-FreeBSD_g6330a45b0 | Passed |
| FreeBSD 14.5 amd64 | 14.5-RELEASE for installed/running/userland | 2.2.9-FreeBSD_g079ba86d7 | Passed |

Go 1.27.1 freebsd/amd64 and x/sys v0.48.0; module verification reported
“all modules verified” in both guests. Python 3.14.7 is the SDK installer only.
Runner ubuntu-24.04; pinned VM Action
`a2f9a41fa97f6848b8c3b791087dfcdaa5b473ff` (v1.5.8), builder 2.2.8,
snapshot QEMU, 6,144 MiB, default CPU. Image hashes were not captured; this is
native behavioral evidence, not an immutable image or performance baseline.

Executed `TestNativeHelperKernelAuthorization`, `TestNativeHelperActualReader`,
`TestNativeHelperUnsafePathsAndOwnership`, `TestNativeClientRejectsNonRootPeer`,
`TestNativeShutdownPreservesReplacement`, and `TestNativeAdmittedClientsBounded`
on both guests. The separate existing discovery/pool fixture tests passed too.
`TestNativeHelperChild` skips its direct parent invocation by design; parents
explicitly execute it in dropped-credential subprocesses and require each client
subprocess to succeed; the adversarial listener subprocess is explicitly torn down.
Credential-version denial is a unit case; an actual unknown kernel ABI was not
fabricated. No helper/native inventory dump is retained.

Local failing tests preceded framing/server implementation. A race check found
that a test modified a live server's deadline configuration; the test now uses a
separate immutable-config server for its idle case. No product configuration is
mutated while serving. Local repository checks, eleven tooling regression tests,
Go unit/component tests, vet, race tests, four bounded parser/framing fuzz targets
and Gitleaks passed. SDK/module source and checksum pins are recorded in the
[source manifest](../../research/M1/source-manifest.json).

Full service restart, two-generation cache, browser/CLI authentication, application
RSS/latency and independent file-service gates remain open. No performance, NAS
readiness, physical disk identity or commercial qualification follows from this run.
