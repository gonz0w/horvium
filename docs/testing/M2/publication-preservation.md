# Publication-cut snapshot preservation

Status: Test correction in progress; native regression qualification pending.
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
