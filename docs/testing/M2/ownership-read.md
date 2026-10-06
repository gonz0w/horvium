# M2 bounded native ownership record reads

Date: 2026-10-05. Status: implementing; native qualification and review pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Contract

`StateDirectory.ReadOwnership(poolGUID)` is an internal FreeBSD metadata reader
under the [held private directory lease](state-directory.md). The caller must
still be kernel root; an inherited descriptor does not exempt another UID. A
canonical nonzero uint64 decimal GUID of at most twenty bytes selects only
`ownership-<GUID>.json` within that directory. There is no caller pathname or
directory creation/repair fallback. Other platforms return unavailable without
file operations. Nil, zero and closed handles fail. Reads and `Close` serialize
on the lease mutex; returned policies have independent slices.

Before reading, recheck the opened directory's root UID/type/exact 0700 mode and
the held empty 0600 writer inode against its anchored name. A changed private
mode, missing/replaced lock or invalid handle denies access. The original opened
directory remains the anchor after a root rename; a replacement at its former
pathname cannot redirect the reader. Other root actors and nontrivial ACLs remain
outside this qualified boundary; these checks do not fence root or grant storage
mutation authority.

Inspect the record with `fstatat(AT_SYMLINK_NOFOLLOW)` before opening, refusing
known nonregular/unsafe objects. Open relative to the held descriptor with
`O_RDONLY|O_NOFOLLOW|O_NONBLOCK|O_CLOEXEC`, then independently check the actual
opened file. Require root UID, regular type, exact 0600 mode, one link and a
nonnegative size at most 64 KiB. Never follow a symlink, truncate a file, repair
owner/mode, create a missing record or treat denial as empty enrollment.

Read at most 64 KiB plus one detection byte. The byte count must equal the
observed size. Compare the pre-open name, opened file, post-read file and final
anchored name for device/inode, owner/group, mode, links, size, modification/change/
birth timestamps, flags and generation. Access time is excluded because reads
may advance it. Close the record descriptor before returning a policy; a read,
metadata or close failure returns none. Strictly decode through the
[accepted ownership codec](ownership-records.md), then require the decoded pool
GUID to match the selected filename identity. Unknown versions, checksum damage,
bad scope and mismatched identity never become enrollment.

FreeBSD errors: `ErrInvalid` for a bad selector; `ErrStateDirectory` for a missing/
unsafe file, UID/lease or read/close failure; `ErrLimit` for byte/budget exhaustion;
`ErrStateRecord` for invalid codec content; `ErrDrift` for observed file/name/
identity changes. All errors return `Policy{}`. A valid policy still needs trusted
ACL-qualified root file provenance, actual enrollment and fresh native identity/
eligibility/recovery reconciliation before writes. The runtime does not call the
reader and remains read-only.

## Native procedure and proof scope

The existing `state` suite requires kernel root and both disposable VM opt-ins.
Create a new private `/root` anchor and owned directory/record; preserve a separate
foreign sentinel. Read the valid record twice, verify original bytes/inode and
independent returned slices. Refuse missing/invalid selectors and regular mode/
owner/hardlink, symlink, FIFO, directory, empty, oversized, damaged, newer and
wrong-pool controls without changing object bytes or replacing them.

Copy only the built test executable to an owned traversable temporary directory.
Pass the live root directory/writer descriptors to a child with kernel UID/GID
65534 and cleared supplementary credentials. Independently verify its effective
identity, absence of other groups and inherited root descriptors; require reader
denial. FreeBSD's primary group can appear in `getgroups`; it is not mistaken for
a retained privileged supplementary group. Wait/reap that owned child and remove
the copied executable fixture. No helper request or product executor is created.

Rename the owned state directory and place unrelated bytes at its former path;
require the held descriptor still reads the original record. Change directory
mode, lock mode and lock inode between calls and require denial. Close the lease,
refuse further access, verify foreign bytes and independently verify both owned
fixture directories were removed. Only default-ACL disposable UFS/ZFS system
storage fixtures define this scope.

Tests failed first because the reader was absent; the expanded evidence checker
first failed when it accepted incomplete reader/codec evidence. The checker now
requires sixty-three named assertions (thirty-five lease and twenty-eight read),
two executed native tests, the five codec tests with all fifty-seven distinct
subtests, and the metadata-comparison unit test. Missing, duplicate, unknown,
failed, skipped or oversized evidence is refused. The metadata unit test mutates
each compared field independently and permits atime change; it is not a
synchronized concurrent file-replacement experiment. That race/failure
qualification remains open before a broader runtime/store claim.

```sh
python3.14 tools/local-qualification.py state 15.1 --run
python3.14 tools/local-qualification.py state 14.5 --run
make lab-status
```

Both releases must complete the expanded suite, base/runtime preservation,
image/key/variable removal and supervised process teardown. No extra raw disk,
host service or persistent runner is introduced. The twenty-one portable fuzz
targets remain unchanged; untrusted record bytes use the already fuzzed codec.
Earlier manifests retain their original tested sources and are not relabeled.

## Primary sources and remaining gates

Both pinned release revisions' descriptor/VFS/credential sources and stat/read
manuals were inspected and hashed. These explain descriptor-relative status,
read/access-time and primary-group semantics; native execution remains the proof
for the declared fixture. Sources are not installed binary build provenance.

- [FreeBSD stat/fstat/fstatat](https://man.freebsd.org/cgi/man.cgi?query=stat&sektion=2&format=html)
- [FreeBSD read](https://man.freebsd.org/cgi/man.cgi?query=read&sektion=2&format=html)

Partial M2-OWN-001/M2-STATE-001 only: ACL-qualified provisioning, full directory
enumeration/store budgets, atomic publication, job/outcome/history schemas,
admission/reservations/tombstones, synchronized read/write failures, disk-full/
read-only/sync/rename faults, power-loss/restart recovery and metadata export/
restore remain ahead. Read byte bounds do not establish a filesystem I/O deadline.
There is no runtime enrollment or record-writing path. M2 acceptance and M1's
maintainer-deferred Safari/manual accessibility gate remain pending. GitHub Actions
stay disabled; no physical/controller/commercial qualification follows.
