# M1 runtime lifecycle and observability evidence

Status: slice 6 in development. M1 remains unaccepted.
Date: 2026-10-04. Work: [issue 1](https://github.com/gonz0w/horvium/issues/1).
Authority: approved [EDD](../../milestones/M1/EDD.md) and
[Test Design](../../milestones/M1/TDD.md).

## Behavior

The configured `max_readers` now also bounds authenticated inventory responses
through query, HTML/JSON encoding and the final write. Excess inventory requests
receive an immediate 503 with Retry-After; no pending queue is created. The
existing cache query limit remains in place for library callers. Authentication
runs before admission. Small collector-status, login and refresh requests remain
available during inventory saturation. The 32 MiB encoded reply cap remains a
logical output budget, not an RSS guarantee. Server write deadlines bound slow
network responses; a blocked-writer regression proves admission covers writing.

Foreground HTTP and the collector share a lifecycle. Cancellation closes the
listener, cancels collection, waits for HTTP shutdown (five-second deadline with
close fallback), clears observations and closes sessions. Unexpected collector
termination stops HTTP. Restart constructs new volatile sessions and a new
random generation namespace. Credential state/config are reloaded, never rewritten
by discovery or management startup.

Management emits fixed-schema JSON operational events to stderr: UTC timestamp,
locally generated request/collection ID, operation, section, duration in milliseconds,
outcome, coarse error class and suppressed-record count. No raw URL, query, source
address, forwarded header, request body, token, cookie, host/resource identifier
or command output enters the event schema. Native helper collection events use
helper-local IDs; management attempt IDs correlate with its cache generations.
Section duration currently means total attempt duration, not individual probe timing.

Each process admits at most 60 records in a minute window, with a bounded queue
of 64 fixed-size records and one writer. Floods/queue exhaustion increment a
saturating count reported by a later admitted event. Suppression can omit any
category; this is operational sampling, not a durable security audit ledger.
A blocked/failed sink cannot block request handling or shutdown. On termination,
queued events may be lost. No files, syslog socket or telemetry connection are
opened. An operator can route stderr through FreeBSD system logging; packaged
logging/rotation configuration is a later delivery gate. Helper authorization
and protocol-denial events are not yet integrated into this logger.

## Test procedure

Behavioral tests preceded implementation for full-response admission, fixed-schema
bounded logging, stalled log sinks, safe attempt reporting and foreground lifecycle.
Component tests use generated inventory; no product fake mode exists.

The guarded `TestNativeBuiltProcessRestart` requires root, FreeBSD and
`HORVIUM_NATIVE_VM=1`. The native script first builds the actual cgo-free product
executable. The fixture copies it out of checkout ancestors, launches root helper
and exact non-root management UID processes with root-protected config/digest on
boot storage, authenticates over real loopback HTTP, and observes a generation.
It freezes helper before refresh so the management transport is actually pending,
then kills management/helper. After removal of only the captured fixture-owned
stale socket inode, replacement processes must reject the previous session/cursor
and collect a different generation. Killing helper alone must retain stale cached
context while HTTP responds; a replacement helper plus refresh must recover fresh
state. SIGTERM then verifies orderly exit and owned socket removal. Config and
digest bytes must remain identical. No test data pool is needed for admin startup.
A helper killed during this pending transport is not a kill during a native ZFS
probe; native reader process-group cancellation has separate existing tests.

An independent standalone FreeBSD base sshd is provisioned only inside the
explicit disposable guest. It has its own root-owned temporary configuration,
fresh ephemeral host/client keys, pinned known_hosts, dynamic loopback port,
read-only internal SFTP and a fixture-only chroot. Password authentication and
forwarding are disabled. No host sshd configuration, authorized_keys, user or
rc.d state is changed. The fixture verifies daemon PID, unchanged configuration
and exact file bytes via separate real SFTP transfers before, during and after
management/helper deaths and recovery. All keys and files are deleted; output is
withheld. This proves one independent file-service path, not SMB/NFS continuity,
client interoperability, permission policy or full M1-NOCHANGE acceptance.

## Diagnostic scale measurement

`BenchmarkAuthenticatedDatasetPages` generates one pool, four providers and either
100 or 10,000 datasets with deterministic names/numbers. Ten concurrent readers
execute 1,000 authenticated full HTML pages through the real console, session
policy, cached queries and renderer. It reports p50/p95/p99, end heap and process
peak RSS using getrusage (FreeBSD/Linux KiB converted to bytes; Darwin bytes).
Run with `go test ./internal/console -run '^$' -bench '^BenchmarkAuthenticatedDatasetPages$' -benchtime=1x`.

This is a component diagnostic: no network latency, helper process, 60-second
warm-up, five-minute idle CPU window or generation replacement sampling. Peak RSS
includes the test binary and fixture; the second subtest's peak is cumulative for
that process. It cannot satisfy the combined management/helper RSS budgets in
M1-PERF-001 or establish the 16 GB hardware/commercial profile. The existing
100,000-row and byte-cap rejection tests remain functional limit evidence.

Execution results and exact source/native run will be recorded after validation.
Required browser/accessibility, complete process-abuse review, native-probe kill,
SMB/NFS continuity, integrated helper-denial logging and the full performance
procedure remain open. No installable port or production support is claimed.
