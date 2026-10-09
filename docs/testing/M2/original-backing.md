# Original-lease persistent backing observation

Status: Native prerequisite qualified; awaiting maintainer review; runtime denied.
Date: 2026-10-09. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted mechanism](../../milestones/M2/property-recovery-mechanism.md),
[native backing discovery](persistent-backing.md).

PR 113 accepted guarded persistent backing discovery at merged source
`1e3255da9a13f5e192da04ba5ae97b5051125aff`. The private host bridge now runs that
fixed native producer through the original state lease. No caller of the bridge
supplies a command, descriptor, backing identity or path. There is no runtime
consumer, registered dual-domain owner, commissioning, storage effect or write
clearance. Complete system/owned-pool exclusion, independent restore scope,
original journal-before-ledger ownership and startup/global integration remain
required. Observing either domain separately does not prove their simultaneous
ownership or independent restoration.

## Contract and lifetime

The internal storage transfer contract accepts bounded UFS/ZFS native facts from
a trusted read callback. That callback must not retain or close its borrowed
descriptor, re-enter the lease or change credentials. It is not a user-facing
extension point. Supplied callback values remain metadata; only the private host
bridge selects the actual guarded native collector.

Actual effective UID 0, the original self-bound lease and an unblocked publication
state are required. The same lease mutex stays held throughout object inspection,
all native probes and final checks. Close and other lease operations must wait.
The established protected-object observer is split into a mutex-owning wrapper
and a locked implementation; its historical behavior is retained. Complete
original directory/lock metadata, no-follow lock name, owners, modes, ACLs and
live mount observations surround the callback. Full mutable metadata detects
in-call changes and never enters the persistent identity.

The new method checks context cancellation before and after borrowing and adds
a 45-second child context; the fixed native producer retains its existing output,
probe and collection limits. A callback error, malformed facts, filesystem-kind
mismatch, object/mount drift, closure, copied lease or retained publication
uncertainty returns a zero observation. The original uncertainty latch is never
reset. Contexts bound the native producer's guarded work; mutex acquisition and
filesystem kernel calls do not gain new preemption guarantees.

The successful opaque observation retains persistent backing, protected object
identities and its original lease pointer. It exposes no file descriptors or
serialized authority. Reopening unchanged objects creates a different origin;
neither an old observation nor a reconstructed lease registers an owner. An
observation is a momentary fact, not a lifetime revalidation, durability receipt,
anti-rollback proof, boot identity or execution permission. Later integration must
recheck both original leases and the complete namespaces around every transition.

## Failing-first tests and native gates

Portable behavioral tests cover two valid transfers, nine invalid transfers and
unavailable off-target observation without invoking the callback. The initial
projection stub failed the positive assertion before implementation. The native
bridge denial stub failed both actual UFS/ZFS positive domain assertions before
being connected to the native producer; the disposable fixtures were removed.

The focused suite requires two native parents and 33 assertions per filesystem:
actual original descriptor correspondence, repeat identity, nil/cancelled contexts,
inert/copied leases, nil/error callbacks, malformed/mismatched facts, cancellation
within the callback, bounded deadlines, directory/lock modes, nonempty/hardlinked
locks, same-inode and namespace changes, lock replacement, restored observations,
actual root-to-nonroot credential drop, original uncertainty persistence, Close
exclusion/completion and reopening with a different origin. Both owned UFS2 and
ZFS fixtures run on each pinned guest; filesystem, memory device and directory
teardown are required separately. Test callbacks inject negative conditions only
in disposable fixtures and do not become runtime failpoints.

Run `tools/local-qualification.py original-backing 15.1 --run` and then 14.5.
The affected historical `state` corpus remains required on both releases because
the original object observer is refactored. Its inventory and assertions are
unchanged. The original `persistent-backing` suite and all historical manifests
remain unchanged. Shared runner/checker maintenance requires one fresh full
portable review; later evidence-only updates reuse its exact deep inputs while
rerunning baseline checks. No GitHub Actions or persistent host service is used.

## Qualified observations

The [manifest](original-backing-manifest.json) binds the final implementation to
signed source `a8d74d38a1a78e5ec74a15c2e71aedafcff6a753`. Both pinned
FreeBSD/amd64 releases passed all 66 original-lease assertions and six fixture
teardown controls, using the same native executable SHA-256
`2d469c08900e22ec4ddc3585fbbba78694a6f2e85775ab11b02765249c1329ad`.
The UFS/ZFS parents took 4.56/4.91 seconds on 15.1 and 4.15/5.15 seconds on 14.5;
these small identity fixtures do not qualify production resources or performance.

Both affected historical state suites passed their unchanged 76 parents, 1,054
required native assertions and 1,930 codec children. Full portable review passed
check, lint, test and secrets, 155 tooling tests, eleven race packages and all
53 bounded fuzz targets. The initial native denial-stub record remains unqualified
and source-bound; its teardown and compiled artifact removal were verified too.
All 87 historical M2 manifests and 340 other original source inputs retained their
bytes. The sole intentional original source refactor is the locked observer split.
Every owned session stopped, and each runner removed its executable, disposable
storage, process group, key and image. GitHub Actions remained disabled.

This completes the individual original-lease observation prerequisite. It does
not register a simultaneous journal-before-ledger owner, qualify system/owned-pool
exclusion or independent restore scope, commission an installation, assemble the
startup/global protocol or enable a storage writer. Those remain required before
runtime use. M1 manual Safari/VoiceOver acceptance remains pending separately.
