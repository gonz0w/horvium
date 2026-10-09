# Private property executor supervision

Status: Native prerequisite qualified; awaiting maintainer review; runtime writes disabled. Date: 2026-10-08.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
[Accepted mechanism](../../milestones/M2/property-recovery-mechanism.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).

PR 107 accepted the mechanism and Test Design at merged source
`02a995e5681f38b4a40141402dd2d0a9f0244933`. This first integrated prerequisite
adds the raw kernel boot owner and a dedicated Go property supervisor. The existing
private property control consumes it; the root listener still denies every runtime
operation. No journal, startup clearance, shared global permit, provisioning,
operator recovery or service installation is supplied by this component.

A fixed-size syscall reads exactly sixteen raw `kern.boot_id` bytes without a
string conversion or kernel-sized allocation. Only the original boot owner may
repeat checks; a changed/failed read permanently retires it, and copied owners
cannot use or retire the original. Observed identity does not grant storage
clearance or attest restoration. FreeBSD/amd64 is the initial native ABI; other
platforms/architectures are unavailable pending actual qualification.

The private control reexecutes the same executable with two root-owned inherited
anonymous pipes. Native pipes have zero links/permission bits and the creator
UID; both endpoints are bidirectional; the channel checks root creation and native access, never chmod or
a pathname FIFO. Its internal entrypoint is not a management command or API. The
request is a bounded canonical property intent, random exchange token and original
boot identity. The child rejects non-root users, unsafe descriptors, malformed or
noncanonical intent and foreign boot/phase/token. Only fixed validated `/sbin/zfs`
property arguments can execute. Inherited channels are close-on-exec before the
native guard starts. A ready response is not launch authorization: the original
parent repeats boot/context/expiry checks, sends a one-time handshake and closes
the request. Missing/extra bytes or peer loss before authorization denies launch.

The isolated supervisor acquires native reaper ownership before spawning the base
guard. Its own kernel status must identify it as owner. It waits the actual guard,
kills/reaps residual descendants through its own kernel subtree and requires zero
children/descendants. Unexpected residual descendants are killed/reaped but still
withhold normal completion, even if the utility itself reported success.
It repeats boot identity before reporting completion through
its original channel. The parent requires that exact report, actual supervisor
wait and terminal EOF. An interrupted, expired, unavailable or failed exchange
never returns success or establishes quiescence. No saved PID or global process
list is used by production cleanup. The ordinary native guard/output cap remain.

The native effect budget remains ten seconds. Supervisor descendant cleanup is
bounded at two seconds; failed parent exchange allows bounded join/kill/join
cleanup. Exhaustion remains unverified, not clean completion. Without the later
independent journal/global owner, this path remains confined to the private
backend fixtures. Supervisor death can abandon descendants: a future replacement
helper must retain pending state and deny same-boot takeover under ADR 0009.

The focused disposable suite extends the actual recovery fixture: all seventeen
prior recovery cases, ten original boot/supervisor cases and a post-reboot kernel
identity test plus twelve retained property-control cases/two guard children
are required by the checker. Cases use native header layout checks,
real guest clock adjustment/restoration, root pipe exchange, pre-launch peer loss,
foreign handshake, escaped/nested base-guard containment, guard/supervisor/helper
death and actual private backend quota effects. Blocking probes are containment
fixtures, not active kernel property interruptions. Actual reboot and complete
host/device/supervisor teardown remain required. Portable rejection tests precede
native acceptance; all historical manifests retain their executed source and bytes.

The initial native run exposed an invalid assumption that anonymous pipes could
be chmodded and report mode 0600. Pinned FreeBSD [pipe implementation](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/kern/sys_pipe.c)
reports FIFO type without permissions, zero links and the creator credential UID;
chmod applies only to named pipes. Receipt SHA-256
`322acd04a21eef65530c052cf9c9d5a75091507a31cd611d0b967e83fcbf137e`.
The failing native cases remain unqualified; regression coverage accepts only
actual anonymous pipe metadata and rejects named/foreign/unsupported channels.

Source inspection during a subsequent failed attempt found that inherited blocking
descriptors also need explicit nonblocking setup before `os.NewFile` can support deadlines.
The pinned Go toolchain's `File.Fd` switches inherited pipe ends to blocking mode.
The child now restores nonblocking mode before registering its channels with the
poller; the existing actual ready/peer-loss cases cover this boundary. The failed
attempt remains unqualified and all fixture teardown is retained.

The next focused attempt exposed FreeBSD's bidirectional anonymous endpoints:
both are created with `FREAD | FWRITE` and report `O_RDWR`. Descriptor positions
and the original phase protocol define request/response roles. The same pinned
pipe source establishes this native access mode; the boot/ABI fixture now checks
both actual endpoints before starting an exchange. The failed attempt created no
storage fixtures and remains unqualified.

## Executed evidence and limits

Signed source `71f8a1197a0c6540287425ace4a143b0d901b12f` passed the full focused
fixture on 15.1-RELEASE-p3 and 14.5-RELEASE using the same prebuilt Go executable.
Each run contains 41 child results plus the standalone post-reboot boot check:
42 cases across five parent tests. Actual native headers confirmed the 80-byte
status/kill layouts and fixed self-reaper commands. Both endpoints' native pipe
metadata, ten supervisor cases, all seventeen prior recovery cases and fourteen
retained control/guard children passed. Real guest reboot changed the raw kernel
boot identity; unchanged durable history, holds and fixture contents survived.
Every guest, backing disk, inherited channel fixture, test binary and host process
was removed/stopped with verified base/runtime preservation and teardown.

The [sanitized manifest](property-supervisor-manifest.json) records actual versions,
source/binary/log hashes, native layouts, four unqualified attempts and fresh full
portable review (144 tooling tests, eleven race packages, all 49 bounded fuzz
targets). Five new portable behavioral parents cover boot-owner lifetime, bounded
canonical requests, phase/identity binding and native pipe provenance. The original
working-tree boot rejection tests failed before implementation. All 81 historical
M2 manifests remain byte-identical. Documentation-only final review repeats the
cheap baseline and reuses successful deep evidence only for exact non-prose inputs.
Actions remain disabled; qualification installs no host service or background job.

These are disposable amd64 fixture results, not a production support/resource or
file-client claim. Seeded enrollment and fixture-only clearance remain test inputs.
The independent journal/census, original startup/provisioning leases, restore
provenance and shared global owner still need implementation and actual native
cut/restore tests. Supervisor-first death remains uncertain and requires explicit
owned-fixture cleanup; a replacement runtime helper cannot infer quiescence.
Active property-command/kernel interruption is still untested. The read-only root
listener, retained pool holds and pending M1 manual accessibility gate are unchanged.
