# Retained ownership policy startup qualification

Status: implementation merged in [PR 307](https://github.com/gonz0w/horvium/pull/307); storage qualification passed on both releases; complete acceptance pending.
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

`policy-startup-owner` uses two normally waited invocations of the same
checksum-bound binary, each with the existing 34-minute process bound:

- `^TestNativePolicyStartup(Lifecycle|Revocation|Denial)Owner(UFS|ZFS)$`.
- `^TestNative(ExecutionStartup|ClosedStartupOwner)(UFS|ZFS)$`.

The root parent bound remains eight minutes. Six policy parents split clean
lifecycle, revocation and invalid-ownership fixtures on each filesystem, with
31, 23 and 39 controls respectively. All eight original modes remain required.
Every group independently checks data sentinel, domain teardown and mount
restoration. All four retained parents and their 394 controls remain required:
ten actual parents and 580 controls per release, plus independent QEMU cleanup.
The aggregate checker requires both phase boundaries, normally completed phase
results and exact parent placement. Old-only, missing, duplicate, skipped, failed
or partially cleaned receipts cannot qualify. The aggregate can take up to 68
minutes of test time; it no longer claims the previous single-process 34-minute
whole-suite limit. The existing host SSH/controller bound remains two hours.
Parent, child, operation, byte and record limits are unchanged. See
[fixed-root qualification](retained-policy-root.md).

The subsequent [child operation budget correction](retained-policy-child-budget.md)
gives independent owner calls independent bounded fixture contexts, retaining the
150-second whole-child test limit and each 90-second product operation bound.
The earlier shared 120-second lifecycle context failed during ZFS close. The
corrected context/child-profile candidate now has complete existing fixed-root
receipts on both pinned releases; expanded acceptance remains outstanding.

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

## Recorded candidate observations

The [sanitized manifest](retained-policy-startup-manifest.json) records the exact
implemented source `941d6d9302919becd3b06490eae418ec7660057b`, base
`7a62c66f2c8a8feba0c8ceec5831c63215cebef0`, compiler/binary/input hashes and
original log/record digests. Full portable review passed its baseline, eleven race
packages and 54 fuzz targets with signed/DCO/title and supervised teardown checks.

Signed red source `5a3e270fc271f420363bddf84b42a86da07349f8` reached the intended
policy-review refusal in actual FreeBSD 15.1 storage and fixed-root fixtures. Both
are expected-red, unqualified results with verified guest/base/runtime teardown;
the fixed-root device fixture was removed. Their passes are not asserted.

The implemented storage gate passed all eight actual parents and 822 controls
on each disposable FreeBSD/amd64 15.1 and 14.5 target, using the same checksum-bound
artifact. That includes 278 new controls and all 544 retained controls per release.
Actual UFS without softupdates, ZFS and process-cut fixtures completed independent
teardown; host guest/image/key/artifact/supervisor cleanup and preserved base/runtime
checks passed. Storage policy/physical-use/boot facts remain synthetic. Normal
FreeBSD build file selection excludes the tagged fixture bridge.

At that storage candidate, enabled fixed-root qualification and expanded
acceptance remained pending. The subsequent
[fixed-root receipt](retained-policy-root-qualification-manifest.json) records
corrected source `35d32e5667cfec4416884f24164c26f864714513` completing
ten parents/580 controls on each release, with identical artifacts and verified
teardown; see the [root qualification](retained-policy-root.md). Storage facts
retain their synthetic-transfer limitation, and both manifests preserve their
actual separate executed sources.

Expanded multi-policy/mixed-history/concurrency/census/bounds acceptance remains
pending. These component passes do not accept issue 305, its broader runtime gate
or M2. No private exploratory compile/stub log or earlier empty-ledger receipt is
substituted for a missing native gate.

The subsequent [portable admission coverage](retained-policy-admission.md) adds
supplied-fact drift and complete codec/catalog mixed-history refusal checks.
It changes no production behavior and qualifies no new native lifecycle family.
