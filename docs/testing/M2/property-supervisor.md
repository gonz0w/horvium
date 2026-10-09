# Private property executor supervision

Status: Implemented prerequisite; native evidence pending. Date: 2026-10-08.
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
private pipes. Its internal entrypoint is not a management command or API. The
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
children/descendants. It repeats boot identity before reporting completion through
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
prior recovery cases, nine original boot/supervisor cases and a post-reboot kernel
identity test are required by the checker. Cases use native header layout checks,
real guest clock adjustment/restoration, root pipe exchange, pre-launch peer loss,
foreign handshake, escaped/nested base-guard containment, guard/supervisor/helper
death and actual private backend quota effects. Blocking probes are containment
fixtures, not active kernel property interruptions. Actual reboot and complete
host/device/supervisor teardown remain required. Portable rejection tests precede
native acceptance; all historical manifests retain their executed source and bytes.
