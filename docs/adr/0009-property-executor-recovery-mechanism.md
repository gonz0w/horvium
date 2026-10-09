# ADR 0009: property executor supervision and recovery provenance

Status: Accepted. Date: 2026-10-08.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
[ADR 0008](0008-property-runtime-recovery-gates.md),
[mechanism EDD and Test Design](../milestones/M2/property-recovery-mechanism.md).
Authority: maintainer merge of [PR 107](https://github.com/gonz0w/horvium/pull/107),
source `02a995e5681f38b4a40141402dd2d0a9f0244933`. This accepts the mechanism
and test plan, not native/runtime qualification.
PR 106 accepted research observations at merged source
`318ef8ac155f4004051ad6f4287fa8de2109a5a5`; it did not adopt this mechanism.

## Context

The [actual experiment](../research/M2/executor-recovery.md) demonstrated surviving
original descendants after helper/guard death and singleton reacquisition on both
pinned FreeBSD targets. Applying intent also survived a completed quota effect
without durable outcome publication. Neither lock acquisition nor a matching
property can establish absence of prior execution or restore launch authority.

The runtime contract needs a concrete way to supervise each property command,
retain uncertainty across helper death and detect restoration of older ledger
bytes. PID/start timestamps are useful research attribution, but do not provide
an atomic, durable fence. A full `kinfo_proc` decoder would add an architecture
dependent ABI without solving metadata rollback or races between inspection and
launch. Choose a smaller mechanism whose uncertainty policy is explicit.

## Decision

Use one dedicated root supervisor per active property operation. It acquires
FreeBSD process-reaper ownership before creating any native child and retains that
ownership until the command has been waited for, all its descendants have been
reaped and native reaper status is empty. It uses the existing fixed native guard
and literal validated arguments. It has no arbitrary command/path interface and
does not become a service or pool worker daemon. Go and the already accepted
FreeBSD syscall adapter remain sufficient; the research C observer is not adopted
as runtime code.

Add a bounded root-owned execution journal outside the ordinary ledger's restore
scope. One original helper owns both leases and the single global launch slot.
Journaled metadata transitions bind the exact complete ledger census; a durable
pending launch precedes applying publication and supervisor launch authorization.
Only actual supervisor/descendant completion, fresh native result observation and
an acknowledged durable outcome permit clearing the pending launch. Unknown
publication, supervision or provenance disables further writes.

Read `kern.boot_id` as an exact sixteen-byte kernel value. Use it for boot
separation rather than interpreting `kern.boottime`, elapsed time or process
names. These source interfaces need real native qualification on both targets;
the earlier experiment did not test this boot ID or acquire reaper ownership.

An unclean prior helper incarnation never receives automatic same-boot clearance,
even when the journal appears idle or a PID is absent. Require a changed qualified
boot ID before reviewing that interruption. Reboot excludes prior userspace
execution only; it neither invents a storage outcome nor releases retained holds.
A qualified orderly restart may proceed without reboot only through the complete
closed-incarnation protocol and exact independent journal/ledger validation.

Fresh commissioning is an explicit OS-root action, with exclusive original
leases, verified system backing, no prior binding and a complete valid empty
ledger. Existing or uncertain state cannot be relabeled fresh. Ledger-only restore
is detectable against the independent journal and requires explicit reconciliation.
Combined journal/system restore is outside automatic clearance: the restore
procedure must invalidate write eligibility before replacing either state domain,
and unknown provenance remains observation-only. There is no claim that a local
file can detect rollback of the entire host or a resumed VM memory image.

The detailed publication ordering, restore boundary and tests are in the
[mechanism amendment](../milestones/M2/property-recovery-mechanism.md). Its proposed
journal version is separate from existing ownership/job formats. No current CLI
flag, API endpoint, service unit, config key or installation path is added.

## Alternatives

| Alternative | Reason for the proposed choice |
|---|---|
| Singleton lock, PID absence or a process-group kill | PR 106 supplies a lock counterexample; escaped groups and reused identifiers do not prove descendant completion |
| Runtime C observer and repeated process-table enumeration | Does not create a fence or detect ledger rollback; broad ABI/parser work is unnecessary for the restricted first property path |
| Reaper supervisor without durable pending state | Supervisor death can abandon descendants; pending state must prevent a replacement helper from launching |
| Journal beside records in the same export/snapshot | An old snapshot can roll both back and hide interrupted work |
| Automatically clear pending work after reboot | The real effect may exist without a terminal event; retained holds and provenance still need review |
| Reboot after every orderly restart | Adds operator disruption where the qualified no-pending/closed-owner protocol can establish safe retirement |
| External database, distributed lease or hardware monotonic counter | No current requirement justifies new infrastructure; whole-host rollback remains an explicit unsupported automatic-recovery case |

## Consequences and acceptance

Helper or supervisor failure can require reboot before write recovery. Read-only
management and independent file daemons remain available. A stuck kernel operation
cannot be made safe merely by timing out; pending state and the global slot remain
suspended. Pool holds remain independent, including terminal records.

One small journal and one active supervisor bound memory/process overhead. Census
hashing streams the existing bounded store; actual costs need resource evidence
before commercial-scale claims. No new module, container, database or scheduled
agent is introduced. Provisioning and migration must establish the separate restore
boundary; an old ledger is observation-only until explicit bootstrap/recovery.

Acceptance of this ADR authorizes its test-first implementation and native gates,
not runtime launch, deployment or M2 acceptance. Both pinned disposable releases
must qualify reaper acquisition/escaped descendants, supervisor/helper death order,
native boot identity, journal durability/restore cuts, orderly restart and the
assembled global launch owner. Active property-command/kernel cuts remain a
separate unresolved gate; the blocking containment fixture cannot satisfy it.

Revisit for non-property or asynchronous operations, multiple active launches,
unsupported native ABI, unacceptable measured census cost, whole-host restoration
or a new requirement for automatic same-boot recovery after an unclean exit.
