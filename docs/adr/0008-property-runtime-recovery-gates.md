# ADR 0008: property runtime recovery and launch gates

Status: Proposed. Date: 2026-10-08. Owner: Cam (`@gonz0w`).
Authority: pending maintainer review; this document does not amend the approved
M2 design until accepted. [Issue 19](https://github.com/gonz0w/horvium/issues/19),
[approved EDD](../milestones/M2/EDD.md),
[runtime contract and test sequence](../milestones/M2/property-runtime.md).

## Context

The accepted prerequisites through PR 104 supply the private native property backend, application
authority, typed transport and original kernel-peer request binding. They do not
assemble a runtime writer. The backend's mandatory external clearance currently
comes only from tests. Supplying a successful callback at runtime would bypass
the remaining recovery and global scheduling obligations.

The approved EDD already requires original admission, one native launch by default,
independent worker lifetime and recorded boot/process containment evidence on
restart. The remaining implementation choice is how those obligations compose.
Existing state-directory qualification establishes a singleton metadata writer;
it does not establish executor absence. Existing job assessment compares fresh
properties without attributing an interrupted effect or releasing reservations.

## Proposed decision

Keep startup observation-only until a root-owned recovery review establishes
complete state provenance and absence of outstanding prior execution. Startup,
recovery and launch permission are distinct from possessing the metadata lock.
Every helper incarnation must obtain its own clearance; serialized records,
restored metadata, a previous clean-exit assertion or a client context cannot
recreate live execution authority. An empty ledger alone is insufficient because
an older restored ledger may omit interrupted work.

Separate the global native launch permit from retained pool/resource holds.
An original helper owner reserves the global slot before its applying transition
and keeps it until its actual control/descendants have finished and the exact
durable outcome has been acknowledged. Unknown quiescence or publication
uncertainty blocks further launches for that owner. Closing a socket, cancelling
an HTTP request or expiring an application session cannot release the slot.
Retained pool holds remain conservative even after the global slot is released.
Recovery resolution and pool-hold release need their own reviewed implementation.

Root peer/session checks cover review and admission. After admission, a helper
worker uses an independent lifetime context and the original admission authority;
it repeats fresh policy, native identity, deadline, recovery and reservation checks.
It must not keep asking a disconnected management socket for launch permission.
Logout denies new requests; it does not silently cancel accepted work. An expired
launch window still denies a new effect, while a completed effect retains its
observation and durability duties.

Before choosing durable executor identity or a FreeBSD process-inspection adapter,
run the bounded native recovery experiment in the runtime contract. Unsupported
or ambiguous evidence keeps startup closed. Record proposed durable-format and
native-mechanism details in a followup design review after actual observations;
no new schema, command, endpoint or service is specified here.

## Alternatives

The state lock alone cannot exclude prior native or kernel work. A PID or process
name alone cannot attribute an executor. An in-memory mutex cannot survive helper
death. Automatically replaying a retained applying job risks a second effect.
Keeping the client connection alive throughout execution violates the accepted
disconnect contract. These choices are insufficient for runtime clearance.

The conservative startup gate can make writes unavailable after an interruption.
This is preferable to automatically inventing an outcome or discarding a hold;
read-only management and independent file services remain available. No container,
external queue, database or distributed fencing service is introduced.

## Consequences and acceptance

Use the dependency-ordered delivery and acceptance table in the
[runtime contract](../milestones/M2/property-runtime.md). A native experiment is
evidence, not permission to deploy its mechanism. Runtime exposure follows real
assembled denial, success, disconnect, process-death/reboot, durability and
teardown gates on both pinned FreeBSD targets. Portable tests and old native
manifests do not satisfy those gates.

Revisit after the recovery experiment, when adding non-property operations or
raising launch limits. Future kernel scans need separate episode/maintenance
tracking: property-control quiescence must not be generalized to scrub or resilver.
