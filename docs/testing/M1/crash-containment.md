# M1 helper death during a native probe

Date: 2026-10-04. Scope: fixed native probe execution, not full M1 acceptance.
Traceability: M1-RESTART-001, M1-FAIL-001, M1-SCALE-001, M1-NOCHANGE-001.

## Candidate behavior and procedure

The original executor kills its process group while it is alive. SIGKILL of the
helper prevents that cleanup. A direct child parent-death signal alone does not
cover grandchildren: FreeBSD clears that setting across fork.

The candidate FreeBSD reader wraps each of its seven fixed projections in base
`/usr/bin/timeout -s KILL <bounded-duration> <fixed-command> <fixed-arguments>`.
Duration is at most ten seconds, shortened to the collection command's remaining
deadline. The root helper starts a separate group and uses Go's FreeBSD
`SysProcAttr.Pdeathsig=SIGALRM`. Go installs the signal before exec and checks
whether the parent already exited. Once running, timeout uses its own FreeBSD
reaper to signal all its descendants. Helper death or cancellation sends SIGALRM;
the reaper then sends SIGKILL to its probe subtree, even across process groups.
No request controls this duration, program, signal, environment or argument.

The Go byte budget includes timeout's stderr. Failed streams remain unpublished.
Output overflow and caller cancellation retain precedence; timeout status 124
means timeout, 126 permission, 127 unavailable, other nonzero status failed.
There is one base guard per in-flight batch probe, not one process per resource.
No custom watchdog, shell, new CLI command/config field or external module is added.
The additional process/allocations must be included in native resource measurement.

`TestNativeProbeOwnerDeath` runs only in explicitly opted-in disposable FreeBSD
guests. It starts a separate test owner using the actual guarded executor, waits
for generated PID files proving a live probe and grandchild, verifies their group
relationship, then SIGKILLs the owner. Both same-group and separate-session child
cases require probe/descendant/guard termination. Cleanup signals only fixture
processes. This is a deterministic test-binary stall, not an injected ZFS stall
or proof of every possible kernel failure. Existing built-process restart tests
cover authenticated stale context/recovery, volatile state invalidation, and
preserved boot config/digest independently.

`TestNativeGuardedProbeBounds` verifies actual wrapped success, failed exit,
stdout/stderr limits, command deadline and missing executable behavior. All
existing native reader/helper/CLI/console tests also exercise the wrapped fixed
projections. A skipped native regression is not qualification.

The guard must itself be able to run: deliberate privileged destruction/freezing
of the guard and kernel-uninterruptible processes are outside these assertions.
No storage, independent service or configuration mutation is introduced.

## Sources and execution

Reviewed the Go 1.27.1 bundled `syscall/exec_freebsd.go` and FreeBSD's
[process-control manual](https://man.freebsd.org/cgi/man.cgi?query=procctl&sektion=2&manpath=FreeBSD+15.0-RELEASE),
[timeout manual](https://man.freebsd.org/timeout%281%29), and
[immutable stable/14 timeout implementation](https://github.com/freebsd/freebsd-src/blob/514bd4465001c5d57a9dda605d00f934f8ca4e91/bin/timeout/timeout.c).
The selected guest's actual utility behavior and binary hashes must be recorded;
this source review alone does not qualify those binaries.

Native failing regression source: `eab6a0237d0940cbf273cfdb5063fc63f3e8572f`.
[Run 37227304136](https://github.com/gonz0w/horvium/actions/runs/37227304136)
failed both product jobs: both owner-death cases left the probe and grandchild
active. Foundation/research jobs passed; the optional performance job was skipped.

Functional fix: `0bda60d844ec6e496775f4390134a0274e0df07f`, under review in
[PR 13](https://github.com/gonz0w/horvium/pull/13).
[Run 37227688891](https://github.com/gonz0w/horvium/actions/runs/37227688891)
passed all five executed jobs. Both owner-death cases and every guarded bounds
case actually executed and passed on both guests. Each owner-death suite took
0.04 seconds. Existing native discovery, kernel peer authorization, cache,
administration, built runtime/CLI/console/restart and independent SFTP tests passed.

| Actual kernel/userland | SDK / architecture | OpenZFS userland/module suffix | `/usr/bin/timeout` SHA-256 |
|---|---|---|---|
| 15.1-RELEASE-p3 | Go 1.27.1 / amd64 | 2.4.2-1-FreeBSD_g6330a45b0 | `099b29836a7efe1270842ffe093c971c7883e646293b8906b38652901ef1673d` |
| 14.5-RELEASE | Go 1.27.1 / amd64 | 2.2.9-FreeBSD_g079ba86d7 | `1dadc6aaf707f8e4f7a08ea28bfa0d53bf6890ca9ea7873a72e4a051e5f9aa10` |

Environment: ubuntu-24.04 hosted runner, VM Action
`a2f9a41fa97f6848b8c3b791087dfcdaa5b473ff`, builder 2.2.8, 6144 MiB,
default CPU/image selection. Neither an immutable performance image nor physical
hardware is qualified by this run. Optional performance was intentionally skipped;
native resource measurements remain separate. Local
`make check lint test secrets build go-check` and FreeBSD test-binary
cross-compilation passed; product remains cgo-free. The final evidence commit is
documentation-only. Exact functional/suite hashes are in the
[source manifest](../../research/M1/source-manifest.json).

This establishes owner-death containment at the actual executor boundary. The
existing built restart fixture freezes the helper before native execution; an
additional built-helper kill during a captured, stalled native command remains
part of integrated qualification. No complete M1, NAS protocol, resource,
browser/accessibility or package support is claimed here.
