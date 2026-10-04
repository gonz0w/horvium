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

Native failing regression source: `eab6a02` (full revision/run recorded after
execution). Candidate qualification is pending. No native pass, hardware, complete
M1, NAS protocol, browser/accessibility or package support is claimed here.
