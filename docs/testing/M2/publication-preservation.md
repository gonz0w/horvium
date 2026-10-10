# Publication-cut snapshot preservation

Status: Observed test failure corrected; native gates passed; awaiting maintainer review.
Date: 2026-10-10. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted property cuts](property-publication-cuts.md),
[recovery mechanism](../../milestones/M2/property-recovery-mechanism.md).

PR 116 accepted system backing observation and retained two unqualified
publication-preservation attempts. A focused repetition on its merged source
`3119446165a346ac0b58a4600cb25b6485d4b3e3` reproduced a failure after 18 passes:
`after_file_sync_retry_safe`. The field diagnostic identified only the record's
`Blocks` allocation count; the exact bytes and every other compared stat field
were unchanged. The two earlier attempts lacked that diagnostic and cannot
retroactively be assigned a field difference.

## Comparison contract

Snapshot preservation compares exact namespace, regular-file identity, link count,
ownership, mode, size, change/modification/birth times, flags, generation and exact
bytes. Access time changes caused by fixture reads remain excluded. Allocated
512-byte block count is physical space accounting rather than object identity or
logical contents. ZFS computes it from current physical dnode accounting and
updates that accounting during transaction synchronization. A durability barrier
for published metadata does not freeze this allocation statistic.

Primary pinned FreeBSD source: [ZFS getattr](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/module/os/freebsd/zfs/zfs_vnops_os.c#L2129),
[system attribute size](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/module/zfs/sa.c#L2179),
[DMU physical size](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/module/zfs/dmu.c#L2873),
[sync-context accounting](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/module/zfs/dnode.c#L2471).
This explains the observed allocation-only difference; it does not prove a guest
power-loss guarantee or establish the unobserved fields in earlier failures.

Only the test snapshot projection changes: exclude `Blocks` alongside access time.
Every other native stat field and exact bytes remain compared. Production readers,
publishers, durability calls and acknowledgement ordering are unchanged. The
existing production metadata comparator already omits physical allocation count;
its checks are not relaxed. No sleeps, sync-before-snapshot workaround, retries,
record rewriting or cleanup exception are introduced.

## Regression and gates

Before the correction, a native regression must fail when only access/allocation
accounting differs. It must also reject a mutation of each remaining stat field.
The full cut parent must report `snapshot_metadata_contract=pass` in addition to
all 66 historical controls. Failing-first evidence tests reject old logs without
that new assertion. Full affected state qualification requires 77 parents, 1,076
native controls and 1,930 codec children on both pinned disposable targets, plus
existing actual backing and teardown gates. Focused repetitions are diagnostic
only and must never become complete-corpus qualification.

Run the normal signed-source portable review; shared verifier changes select the
complete portable inventory. Run guests serially. Preserve every historical
manifest and unchanged production Go input. The new manifest will retain actual
reproduction/regression sources and failed records separately from passing gates.
Runtime writes, independent restore proof and startup/global assembly remain
pending; this correction grants no storage or recovery authority.

## Executed regression and qualification

The [source-bound manifest](publication-preservation-manifest.json) retains the
allocation-only reproduction on merged PR 116 separately from qualification.
Its bounded 1,000-repeat diagnostic stopped after 18 passes when the exact
comparison reported only `Blocks`. That run remains unqualified; no contents or
other stat-field drift was reported. The two earlier uninstrumented failures keep
their original unknown field differences and are not relabeled as passing results.

The new native regression then failed on signed source
`35364276055ae266beaa6a77a3771c617498b48f` before normalization changed. Corrected
source `50ee93c136f5e1a9ced4ff2cf6b6b0cf2f5b4a08` passed both complete pinned
FreeBSD/amd64 state suites: 77 parents, 1,076 required native controls and 1,930
codec children each. The 67 cut controls retain all 66 original controls and add
the snapshot contract. The regression accepts access/allocation changes alone and
rejects mutations of each of the 17 remaining fields of the actual native stat
structure. Existing exact namespace/content and backing/teardown assertions stay
required. Both guests executed the same checksum-bound binary; the manifest
records its compiler, image, utility, source, log and record provenance.

Full portable review passed check, lint, test and secrets, 161 tooling tests,
eleven race packages and all 53 bounded fuzz targets. Failing-first evidence
coverage rejects old logs without the new snapshot contract. Evidence-only
followups rerun baseline and may reuse only these exact passing deep inputs.

All 90 historical M2 manifests and 358 other original Go inputs retain their
bytes. Only the original cut test changed among those Go inputs; every production
Go file and durability/acknowledgement path is unchanged. All owned sessions
stopped, guest filesystems/directories and compiled artifacts were removed, keys
and images were removed, and supervised process-group teardown was verified.
Inert base/runtime caches were preserved. GitHub Actions remain disabled and no
persistent service was installed.

This resolves the reproduced test allocation-accounting failure mode. It does not
supply missing diagnostic fields for earlier failures, prove physical power-loss
durability or qualify commissioning, independent restoration or runtime writes.
