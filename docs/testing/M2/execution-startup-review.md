# Original closed-incarnation startup review

Status: failing-first private read-only gate; qualification pending; runtime denied.
Date: 2026-10-10. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted recovery mechanism](../../milestones/M2/property-recovery-mechanism.md),
[registered commissioning owner](commissioned-execution-owner.md).

## Contract and Test Design

M2-RUNTIME-002/M2-RECOVERY-001 require fresh original startup inspection before a
new incarnation can be published. This component reviews an existing closed
incarnation under both original journal-first leases, using fixed actual native
backing/system and exact raw boot producers. It retains the exact journal file
receipt, binding and complete census for repeated inspection. Closed bytes alone,
ordinary metadata observation, matching hashes or a caller declaration cannot
reconstruct this original review.

Require a valid closed journal with no metadata/native pending or restore barrier,
matching actual binding and complete validated ledger census, bounded generation,
independent UFS/ZFS restore scope and full actual system/owned backing checks.
Repeated current kernel boot reads must agree; the prior clean closed owner's boot
is historical metadata. Active history always denies this review, even with a
changed boot: unclean recovery requires its own explicit qualified decision.
Missing halves, unsafe objects, native errors, drift or incomplete history deny.
Never create or repair a namespace, alter records, clear holds or replay work.

The original read-only review serializes observation/release and rejects copies
or decoded values before touching copied mutexes. An entered inspection error
permanently retires it; pre-entry cancellation consumes no producer. Retain both
leases until original idempotent release. The fixed root factory uses the same
single-use helper registration as fresh commissioning, reserving before opening
any domain. Different paths, failure and release cannot reset that registration.
Finishing an unused fresh activation is unavailable on a startup review.

T1/T2 require four portable parents: candidate consistency and fourteen unsafe
variants, original lifecycle/copy/serialization/context checks, read/cancellation/
release failures, and release exclusion while observation is active. Ordinary
journal successors must continue to deny owner replacement. No new mutable
journal format, publication receipt or private persistent protocol is introduced.

T3 uses a dedicated `execution-startup` suite, retaining the existing runner's
snapshot/device ownership and supervised cleanup. Two independently owned UFS/ZFS
parents use actual initial activation/unused close to seed closed history and a
separate unclean active history. Eight exact-binary children per parent use fresh
production process registries, never fixture resets: clean, active, missing,
ledger drift, actual UID drop with inherited root descriptors, and boot/system/
late-directory fault transfers around actual native producers. Parent checks wait
results, both released locks, exact retained history, disjoint declared metadata
backing, original data sentinel and complete filesystem/device/mount teardown.
Require all 82 named controls per profile and reject incomplete/old host evidence.
Existing native suites, selectors and historical manifests remain separate.

This qualifies read-only closed startup review only. Durable new-incarnation
publication, unclean changed-boot/root recovery, journaled mutation consumption,
global native launch, worker/supervisor/backend assembly and operator configuration
remain required. Retained nonempty job/history startup, actual reboot recovery,
active kernel interruption, whole-host rollback, power loss, throughput and
commercial scale are not qualified by empty warm closed fixtures. Runtime writes
remain denied; no host service or GitHub Actions execution is enabled.
