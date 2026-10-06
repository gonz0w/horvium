# M2 private ownership record publication

Date: 2026-10-06. Status: implementing; native qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Contract and boundary

`StateDirectory.PublishOwnership(policy, expectedRevision)` publishes bounded
metadata only through an already held [private state lease](state-directory.md).
It does not enroll a pool, reconcile native state, admit a job or authorize a ZFS
write. No HTTP/CLI/runtime calls it. The root caller must separately establish
trusted enrollment and fresh native identity before any future use as authority.
Other platforms return unavailable without filesystem operations.

The policy goes through the [accepted codec](ownership-records.md). A canonical
pool GUID selects only `ownership-<GUID>.json`; no caller pathname is accepted.
Expected revision zero requires an absent record and initial revision one.
Replacement requires a valid, identity-bound [existing record](ownership-read.md)
with the exact expected revision and its successor. Zero, skipped/stale revisions
and uint64 rollover are refused. A missing record for an existing revision, or
an already present record for initial publication, never becomes repair/adoption.
A revision is local optimistic concurrency, not evidence against restored older
metadata or another root actor.

The actual kernel UID must be root, including with inherited descriptors.
Publish/read/close calls serialize through the held lease mutex. Revalidate the
opened directory, actual lock/name and ACLs, then the writable local UFS/ZFS
filesystem. Reject unsafe or damaged existing records rather than replacing them.
There is no implicit chmod/chown/ACL repair, record deletion or lock replacement.

Create one random, exclusive same-directory `.ownership-<128-bit-hex>.tmp` through
`openat(O_WRONLY|O_CREAT|O_EXCL|O_NOFOLLOW|O_NONBLOCK|O_CLOEXEC, 0600)`.
A collision fails; no existing object is truncated/reused. Inspect the actual
created inode for root UID, regular type, exact 0600 mode, one link, empty bytes
and [safe descriptor ACLs](state-acl.md). The process umask may restrict creation;
refuse rather than repairing the result. Write the entire canonical record,
validate the actual written inode/size/ACL, sync the file and close it. Recheck
the held directory/lock, temporary name/metadata and original destination state
before descriptor-relative same-directory rename, then sync the directory.
Only the complete ordered sequence returns success. Readers holding the old file
continue to address that inode; fresh readers address the replacement. No
cross-record transaction or ZFS effect is part of this method.

## Errors and uncertainty

| Result | Meaning |
| --- | --- |
| `nil` | File and directory synchronization both returned success for this publication; not an enrollment/storage/job acknowledgement |
| `ErrDrift` | Stale/skipped revision, unexpected existence or detected destination/temporary metadata drift; review actual state |
| `ErrLimit` / codec error | Bounded policy/record validation refused; no publication |
| `ErrStateDirectory` | UID/lease/ACL/filesystem or prepublication operation refused; no successful acknowledgement |
| `ErrStateUncertain` | Final directory sync or prepublication cleanup could not confirm durability; reconcile actual state before retry |
| `ErrUnavailable` | Runtime is not FreeBSD; no filesystem work |

Before rename, cleanup closes the temporary descriptor, verifies its actual
created device/inode against the anchored name, unlinks only that name and syncs
the directory. A cleanup error cannot be hidden by the original failure. An
unidentifiable created inode is retained for reconciliation. After successful
rename, failed directory sync never unlinks or rolls back the published record:
it may already be visible. Uncertainty permanently blocks further publication
through that lease, while bounded reads remain available. Closing/reopening the
primitive is **not** permission to retry; persistent recovery/reconciliation and
stale temporary enumeration are later gates. No native I/O deadline is claimed.

## Tests and procedure

Portable behavioral tests first failed because publication/order/revision types
were absent. They require exact write/file-sync/close/revalidation/rename/
directory-sync order, stop on every failure, distinguish visible uncertainty,
retain cleanup failures and refuse revision rollover. Off-FreeBSD refusal and
strict native-evidence negative tests also run. Existing codec fuzz coverage
remains twenty-one total targets; no new byte parser is introduced.

Run the complete source-bound local review on a clean signed checkout, then the
expanded state suite sequentially on the Mac:

```sh
PR_TITLE='feat(storage): publish private ownership records atomically' make review
python3.14 tools/local-qualification.py state 15.1 --run
python3.14 tools/local-qualification.py state 14.5 --run
make lab-status
```

Each guest must pass all **161 assertions**: 35 lease, 28 reader, 45 ACL,
43 publication and 10 filesystem controls. Execute five native and nine unit
parent tests (fourteen total), plus all 57 distinct codec subtests without skips.
The native reader's actual UID/GID-65534 reexec now also denies publication with
inherited root descriptors and no privileged groups. Publication refuses unsafe
record owner/mode/link/symlink/FIFO/directory, corrupt/newer/foreign-pool bytes,
invalid/bounded policies and duplicate/stale revisions. Real cooperating
concurrent calls commit a revision once. Verify canonical private bytes, new
inode and unchanged held old file/lock/foreign sentinels, directory rename
anchoring, directory/lock/destination drift and removal of owned temporaries.

A separate owned 32 MiB memory-backed UFS fixture installs only a test-owned
masked POSIX named grant, preserving exact ACL and bytes across refusal. Fill
that filesystem until the actual write returns ENOSPC; publication must fail
without replacing the old record or leaking a temporary. Close writable handles,
remount it read-only and prove per-call refusal through test-constructed real
read-only descriptors. Unmount, detach only the recorded owned md unit and remove
the fixture. No host mount, operator disk, persistent service or runner is used.

Package-private transaction wrappers inject write/file-sync/close/verify/rename/
directory-sync/cleanup failures around actual native fixtures. They establish
error accounting and ordering, including visible unsynced replacement and blocked
future writes. They **do not** establish actual kernel/device EIO, crash/reboot/
power-cut durability or a successful durable job acknowledgement. Test-only seams
are neither environment failpoints nor operator-facing configuration.

Native qualification and its sanitized manifest are pending. Source inputs and
previous accepted manifests must retain exact provenance; full host supervisor
teardown, image/key/variable removal and base/runtime preservation are required.

## Primary evidence and remaining gates

Pinned FreeBSD release manual/kernel sources define exclusive descriptor-relative
creation, same-filesystem rename and full descriptor synchronization:

- [15.1 open/openat](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/lib/libsys/open.2)
- [15.1 rename/renameat](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/lib/libsys/rename.2)
- [15.1 fsync](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/lib/libsys/fsync.2)
- [14.5 syscall implementation](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/kern/vfs_syscalls.c)

These are interface evidence, not installed-binary provenance or physical cache
honesty. Complete store provisioning/enumeration/byte budgets, orphan temporary
handling, bounded jobs/history/tombstones, durable admission/reservations,
metadata export/restore and acknowledged restart/power-cut recovery remain open.
There is no assembled helper store, enrollment route or storage mutation.
Malicious root, external synchronized ACL races, physical/commercial support and
external client/share behavior remain outside this subset. M2 acceptance and
maintainer-deferred M1 Safari/manual accessibility remain pending. GitHub Actions
stay disabled; the ordinary supervisor must verify all owned teardown.
