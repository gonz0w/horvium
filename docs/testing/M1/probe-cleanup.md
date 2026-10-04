# M1 native probe process cleanup

Date: 2026-10-04. Scope: M1 reader execution boundary, not milestone acceptance.
Design: [EDD](../../milestones/M1/EDD.md),
[Test Design](../../milestones/M1/TDD.md).

## Problem and behavior

A fixed native probe can exit while descendants retain its output pipes. Go's
250 ms `WaitDelay` bounds the pipe wait and closes descriptors; it does not
terminate those descendants after the parent has already exited. A descendant
with closed output pipes can also outlive an otherwise successful probe.

The Unix executor now finishes each started probe by sending SIGKILL to its
isolated process group, after `Run` joins its cancellation watcher and I/O
copiers. An absent group is normal. Other cleanup failures prevent successful
output publication. Deadline/cancellation and combined stdout/stderr limits
retain their existing error precedence and bounds. No probe can intentionally
leave background work behind. The seven fixed native projections, sanitized
environment and management/helper contracts are unchanged.

This cleanup runs on completion or cancellation while the helper is alive.
SIGKILL of the helper prevents its Go cleanup from executing. This slice does
not establish automatic descendant cleanup after abrupt helper death, or handle
a process that deliberately escapes the isolated group. Those limits must not
be inferred away from this result. There is no added HTTP command, operator path
argument, daemon, runtime dependency or storage mutation.

## Behavioral regression procedure

`TestProbeProcessGroupCleanup` calls the actual private executor with a
reexecuted test binary. Only this fixture accepts generated temporary paths;
the product reader continues to supply fixed native commands. A readiness and
release handshake confirms parent/child group membership before failure injection.

Cases: deadline, explicit cancellation, cancellation through `Source.Collect`,
stdout overflow, discarded stderr overflow, successful/nonzero parent exit
with inherited pipes, and successful parent exit with child pipes closed.
Each case checks bounded return, expected safe failure class, no failed output
publication, termination of both owned processes, a separate live fixture group
left unaffected, and a subsequent healthy probe. Collection cancellation performs
only one native probe and leaves every section explicitly unavailable/cancelled.

The parent is reaped by the executor. An orphan descendant must either disappear
or have zombie status (terminated, awaiting init's reap); a live/sleeping descendant
fails. The test does not claim control over host reaper timing. Cleanup kills only
test-owned processes; no live disks, pool, service or credentials are used.

Before the fix, synchronized local regression passed cancellation/overflow cases
and failed all three parent-exit cases because the child remained active. After
the fix, repeated local cases passed. Local race instrumentation is development
only; the product remains cgo-free and FreeBSD-only.

```sh
make check lint test secrets build go-check
# In the explicitly disposable FreeBSD QEMU guest, as the native job already does:
env HORVIUM_NATIVE_VM=1 sh tools/bsd-native-discovery.sh
```

## Execution status

Native qualification on both selected FreeBSD releases is pending. Record exact
functional source, run, guest/Go/OpenZFS versions and file hashes after execution;
cross-compilation and macOS fixture execution do not substitute for it.

Traceability: M1-FAIL-001, M1-SCALE-001 and the probe portion of M1-RESTART-001.
Full helper/HTTP abuse review, abrupt-helper-death process policy, independent
SMB/NFS continuity, native-reader resource budgets and required-browser/manual
accessibility acceptance remain open. No physical NAS, commercial scale,
file-client, package or full M1 acceptance is claimed.
