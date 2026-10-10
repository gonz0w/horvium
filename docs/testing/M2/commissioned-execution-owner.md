# Registered fresh commissioning owner

Status: private root commissioning prerequisite accepted in PR 125; runtime denied.
Date: 2026-10-10. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted recovery mechanism](../../milestones/M2/property-recovery-mechanism.md),
[original unused close](initial-activation-retirement.md).

## Contract and Test Design

M2-RUNTIME-002/M2-STATE-001 require one registered original helper/ledger owner.
The fixed native root factory claims a process-wide, original, single-use
commissioning registry before it invokes the accepted fresh activation producer.
Concurrent or later attempts, including different directory pairs, fail before
provisioning. Nil/cancelled contexts, missing private seams, foreign registries
and nonroot callers before entry consume nothing. Once a provisioning attempt
enters, even failure or orderly unused close cannot reset that registry. A new
process still needs separately qualified configured startup/recovery review;
this registry is not cross-process execution fencing or restart provenance.

The sealed owner retains the actual activation and fixed root check, serializes
observation/unused finish/release and rejects copies or reconstructed values.
Observation errors, lost root and every entered finish retire observation/finish
permanently. Close always attempts original release exactly once and retains its
result; lost root is reported as denial even when descriptor release succeeds.
A partially returned source is released before factory failure is returned, while
registration stays consumed. No successful close can commission another ledger
within this helper lifetime. Release alone leaves the prior active journal.

T1/T2 tests require positive lifecycle, source count before/after every invalid
entry, original/copy/decoded registry/owner rejection, nine failure/revocation
modes, 32 concurrent competing attempts while the original factory is blocked,
cancellation while waiting for registration, and observation/finish/release exclusion. These seams are private portable
transfers; production fixes the one process registry, actual UID and original
native activation. No caller boolean or serialized value supplies authority.

T3 exact-binary children exercise the production registry independently on actual
UFS/ZFS domains: clean original commissioning and durable unused close; unknown
ledger-entry drift and preserved active history; failed constructor and no retry
creation; actual UID drop with inherited root descriptors. Each child uses a new
process rather than resetting production registration. The parent waits every
child, independently probes both locks and checks active/closed history. Retain
all previous 296 fixed-host controls and unchanged eight-minute parent deadline;
reject previous evidence without the new commissioning controls. Run both pinned
native releases, local review and complete fixture/VM/supervisor teardown.

This closes fresh root commissioning registration only. No configured startup,
existing-state restart, enrollment/publication consumer, global native permit,
worker drain, supervisor/backend assembly or operator route/configuration/service
is enabled. Existing metadata/native launch paths remain denied by the runtime.
The single-use registry cannot detect whole-host rollback, prevent another trusted
root process selecting another configuration, or qualify commercial scale.

## Executed qualification

The four initial portable behavioral parents failed against denial stubs. A
separate regression then reproduced cancellation while waiting for registration: the
initial implementation consumed that reservation before checking cancellation
under the registry lock. It passed after adding the under-lock context/root checks.
The signed native denial-stub run failed exactly the two affected system parents,
while all four retained original/joint parents passed; its owned fixtures and VM
were torn down. The old-evidence rejection regression failed against the previous
consumer before requiring the new controls.

Both pinned FreeBSD/amd64 targets passed six native host parents and 388 controls
each, preserving all 296 earlier controls and adding 46 commissioning controls
per filesystem. Eight new exact-binary children per target use separate production
process registries, complete all declared assertions and are waited normally.
Parents independently probe original lock release and active/closed journal history;
UFS/ZFS backing and fixture teardown assertions remain required. The existing
eight-minute parent deadline is unchanged. The same compiled source artifact
ran on both releases.

Five portable behavioral parents, nine failure cases, 32 competing attempts,
32 cancelled reservation waiters and 31 focused evidence-consumer tests passed.
Local review passed check/lint/test/secrets, all 176 tooling tests, eleven race
packages and 54 bounded portable fuzz targets. Final prose review reruns baseline
checks and can reuse that executed deep record only for exact unchanged
non-prose inputs/base/environment. All 434 unaffected original Go inputs and 102
historical manifests preserve bytes. Storage implementation, state fixtures,
checker and selected runner retain exact bytes; PR 124's accepted state evidence
remains historical and bound to its executed source, with no new state rerun.

The [sanitized manifest](commissioned-execution-owner-manifest.json) records actual
source, native binary/image/tool, test-first log, review and teardown hashes.
Compiled fixture artifacts were removed; all owned supervisors, VMs, devices,
filesystems and fixtures are stopped, removed or restored as applicable. No host
service or GitHub Actions execution was enabled. Configured startup/restart,
journaled publication consumption, global native launch and runtime assembly
remain required before write enablement.

The subsequent [original closed-incarnation startup review](execution-startup-review.md)
uses the same consumed process registration for read-only existing-state review.
Its separately source-bound qualification does not change this historical execution
manifest or supply new-incarnation publication, recovery or native launch authority.
