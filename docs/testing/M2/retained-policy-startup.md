# Retained ownership policy startup qualification

Status: implementation candidate; native acceptance pending.
Date: 2026-10-10. Owner: Cam (`@gonz0w`).
Owning [issue 305](https://github.com/gonz0w/horvium/issues/305);
Refs [M2-RUNTIME-002](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted design](../../milestones/M2/retained-policy-startup.md),
[PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).

## Behavior and boundaries

A separate private startup review accepts only a complete closed journal and
policy-only ledger. Every policy must have enrolled members and pass the fixed
native pool/filesystem, provider/member and system exclusion checks. The locked
reader supplies a separately decoded policy to the producer, retaining canonical
digests and validating the original policy against the returned facts. It never
reopens the ledger or enters `ReadOwnership` while holding its mutex.

Original leases, file receipts, directory/lock/mount identity, complete census,
raw boot and process registration remain required. Policy/census checks surround
activation, observation and unused finish; deadlines cover the entire operation.
Every job in either format or any state refuses this path. No holds are pruned.
The ordinary metadata-only review retains its empty-only activation restriction.
The original active acknowledgement may publish one unused closed successor;
entered failures retire authority and explicit close releases both leases.

This candidate enables no configured startup, worker, backend, launch permit,
public route, CLI operation or service. It adds no persistent format, migration,
dependency or runtime claim. Ordinary binaries exclude the fixture bridge.

## Fixture construction recorded before native execution

Storage positives use actual disposable UFS without softupdates or ZFS domains,
original fresh commissioning, original journaled ownership publication, then the
strict acknowledged journal successor publisher for fixture-only closed metadata.
Native filesystem/locks/sync are real; physical-use, policy and boot transfers in
these storage fixtures are explicitly synthetic.

Root fixtures use an independently owned enrolled data mirror and disjoint
metadata disk. They collect facts through actual fixed native producers. A bridge
compiled only with `freebsd && horvium_native_fixture` invokes the same original
commissioning and journaled publication pipeline, publishes the controlled closed
successor, confirms complete census, then releases the original leases. Both root
and disposable lab opt-ins are mandatory. There is no hand-written JSON,
reconstructed acknowledgement, admitted worker or registry reset. The bridge is
not a qualified producer for real worker shutdown. Each child consumes those
published domains through its own process registration and the fixed factory.

Denial-only retained jobs use the actual strict job publishers for every history
prefix before original journal initialization and closed publication. They assert
only retained-history refusal, not clean shutdown or executor provenance. Missing
pool seeds may fail their final system census after the real closed publication;
the parent independently requires readable preserved policy/journal history.

## Exact planned selectors and controls

`policy-startup-publication` runs
`^TestNative(ClosedStartup(Publication|Retirement)(UFS|ZFS|Cuts)|PolicyStartup(UFS|ZFS))$`.
The process bound remains eighteen minutes. The six retained parents and all 544
controls are checked by their unchanged historical consumer. Two new parents add
139 controls per filesystem: eleven lifecycle/ordinary-review assertions; eight
observation failures with denial, sticky retirement, preservation and release;
ten activation/finish publication faults with denial, retirement, retry preservation
and release; fourteen format/state job variants with denial, preservation, holds
and release. The new inventory is independently declared in
`tools/check-policy-startup-evidence.py` before execution.

`policy-startup-owner` runs
`^TestNative((ExecutionStartup|ClosedStartupOwner)(UFS|ZFS)|PolicyStartupOwner(UFS|ZFS))$`.
The root parent bound remains eight minutes and process bound 34 minutes. The four
retained parents and all 394 controls remain required. Two new parents add the
87 controls per filesystem with exact inventories for clean, clean-again, legacy, member, scope, missing-pool,
post-publication drift and actual UID-loss children. Each independently checks
normally waited child completion, exact child inventory, both released leases,
visible parent journal history and exact policy bytes; data sentinel, domain
teardown and mount restoration complete each parent. Independent owned data and
QEMU device cleanup are mandatory. The checker rejects old-only, missing,
duplicate, skipped, failed or partially cleaned evidence.

## Required review and remaining acceptance

Run `make check lint test secrets`, signed/DCO/title review, and the existing full
portable policy because shared runner/checker code changes. Run both gates serially
on pinned FreeBSD/amd64 15.1 and 14.5 with checksum-bound binaries. Record executed
source, binary/compiler hashes, exact logs, failed candidates and owned teardown.
No skipped or historical result qualifies this candidate.

The design also requires complete mixed/same/other-pool job refusal, multiple
policy lifecycle and last-policy failure, concurrency/borrowed-lock behavior,
policy-aware census/object and context boundaries. Until these and both native
release gates have candidate-specific evidence, issue 305 remains open. Broader
runtime integration, retained-job recovery, configured startup, browser, hardware,
performance, client continuity and release gates remain independently open.
