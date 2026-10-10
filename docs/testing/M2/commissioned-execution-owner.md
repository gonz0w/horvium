# Registered fresh commissioning owner

Status: test-first private root commissioning gate; qualification pending; runtime denied.
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
and observation/finish/release exclusion. These seams are private portable
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
