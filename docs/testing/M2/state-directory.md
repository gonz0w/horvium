# M2 private state directory and single-writer lease

Date: 2026-10-05. Status: partial boundary qualified on both declared releases;
maintainer review pending. [Approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Boundary and authority

`storage.OpenStateDirectory` opens an existing root-provisioned directory on
FreeBSD. The caller must be OS root; non-FreeBSD targets return unavailable and
perform no file operations. This is an internal directory lease, not a helper
request, configuration/persisted schema, job admission or storage executor.
The runtime does not call it or create state directories yet. The approved
slice 2 prerequisite is independent of incomplete slice 1 eligibility/status/
impact gates; no later operation may assume those missing gates passed.

The path must be canonical absolute UTF-8, contain no controls, be at most 4096
bytes and select a directory below `/`. Walk from `/` with directory descriptors,
`openat`, `O_DIRECTORY`, `O_NOFOLLOW` and `O_CLOEXEC`. Inspect every opened
ancestor for root UID, directory type and absence of writable/sticky/set-ID modes.
The final directory must have exact 0700 permissions. Writable/sticky `/tmp`
is refused; no symlink, repair, chmod/chown or directory-creation fallback exists.
The opened filesystem must identify as writable local UFS or ZFS; other types
are unavailable. Root provisioning selects system storage; this API does not
identify the boot dataset or accept an arbitrary management-supplied path.

First use may exclusively create an empty regular 0600 `writer.lock`. An existing
lock must already have root UID, exact mode, one link and zero bytes. Known
nonregular/unsafe objects refuse before opening; the actual opened descriptor
is independently validated. Existing files are never truncated, repaired,
replaced or unlinked. Obtain `flock(LOCK_EX|LOCK_NB)`; only native contention is
reported as busy. Other failures deny the lease. Recheck the anchored name's
inode/device/metadata correspondence, sync the lock file and directory, then
return the held descriptors. A restrictive umask that prevents exact 0600
creation fails safely and does not trigger a repair.

`Close` releases the owned descriptors once and retains the lock inode. A zero
or nil handle is inert. Close errors are reported. The lock protects cooperating
local writers on the same inode; it cannot fence root replacement/unlink, an
external ZFS actor or another host. File/parent sync success is an observed
prerequisite, not a complete record transaction or physical durability guarantee.

## Coverage and remaining gates

Tests failed first because the directory boundary and errors were absent.
Native assertions use actual FreeBSD descriptor/mode/ownership/locking operations;
portable tests cannot substitute for them. The non-FreeBSD stub has local coverage.
The evidence checker first failed when missing; runner integration first failed
because the suite was absent. It now requires exactly thirty-five named assertions
and one executed `TestNativeStateDirectory`, refusing missing/failed/duplicate/
unknown/skipped/oversized evidence. Existing twenty portable fuzz targets remain;
this privileged directory prerequisite adds no output parser or untrusted route.

This partial boundary does not decode/enumerate/checksum any job/ownership record,
publish/sync/rename a record, acknowledge admission, reserve a pool/device,
recover an executor or replay work. Strict schemas, bounded storage/history,
store-full/read-only/sync/rename failure, abrupt power loss, older metadata restore,
ACL-qualified provisioning, root-helper negotiation and management projections
still need implementation/qualification. Mode/UID evidence uses the declared
default-ACL fixtures; nontrivial filesystem ACLs remain unqualified before runtime
state writes. M2-STATE-001, slice 2 and assembled M2 acceptance remain incomplete.

## Disposable native procedure

Require guest root and both `HORVIUM_NATIVE_VM=1` / `HORVIUM_STATE_VM=1`. Create a
new private root-owned fixture under `/root`; never accept an operator directory.
Keep a separate known foreign file in that owned anchor. Test positive root
opening, exact lock metadata, native close-on-exec flags, duplicate writer denial,
close/reopen with the same inode, competing first opens and independent directory
sync. Unsafe path/owner/mode/ancestor/symlink and bad lock object controls must
refuse without modifying/repairing their objects.

Mount only a new owned tmpfs at a declared private fixture point, independently
verify its type/root ownership/private mode, require unsupported-filesystem
refusal before lock creation, and unmount it. This test privilege adds no product
mount operation. Copy only the built test executable into a separate owned
traversable temporary directory so a kernel-dropped UID can enter the refusal
test. The process has no supplementary groups or credentials. A separate root
reexec process acquires a real lease and signals readiness over a bounded pipe;
require actual contention, kill/reap only that child, reopen with the unchanged
lock inode, close all leases and verify foreign bytes. Remove both owned fixture
directories and independently assert absence. This is lease-process death proof,
not built product-helper/executor recovery or a power-cut trial.

```sh
python3.14 tools/local-qualification.py state 15.1 --run
python3.14 tools/local-qualification.py state 14.5 --run
make lab-status
```

Both releases must pass all thirty-five assertions and complete base/runtime
preservation, image/key/variable removal and stopped owned process groups under
the existing Mac supervisor. The suite uses guest system storage and adds no
raw disks, installable service or persistent runner. Historical device/property
manifests retain their original sources and do not qualify this boundary.

## Primary sources

Both pinned release revisions' descriptor and VFS syscall implementations plus
release-specific libc/libsys open/flock/fsync manuals were inspected and hashed. `O_CLOEXEC`,
relative descriptor opening, nonblocking flock ownership and VFS synchronization
are source contracts; actual guest execution and binary/version hashes must
establish qualification. Sources are not installed binary build provenance.

- [FreeBSD open](https://man.freebsd.org/cgi/man.cgi?query=open&sektion=2&format=html)
- [FreeBSD flock](https://man.freebsd.org/cgi/man.cgi?query=flock&sektion=2&format=html)
- [FreeBSD fsync](https://man.freebsd.org/cgi/man.cgi?query=fsync&sektion=2&format=html)

## Executed evidence

Signed/DCO functional and suite source
`123044304e75f6df27d43eee085c23435ddbcce1`, based on merged PR 38 source
`dea2efefc3c9b9a50053bc4794a5a21685ad5132`, passed local
`make check lint test secrets go-deep-check`: seventy tooling tests, Go unit/race
checks and all twenty bounded fuzz targets. The
[source-bound manifest](state-directory-manifest.json) records eighty-two unchanged
functional/test/tool inputs, native versions and utility hashes, assertions,
qualification log/result hashes, ten pinned primary-source hashes and sixteen
unchanged prior M2 manifests. Final documentation-head portable review is
separate; native evidence retains this exact functional/suite source.

| Native fixture | State filesystem | Executed test | Named assertions | Preservation and teardown |
| --- | --- | --- | --- | --- |
| FreeBSD 15.1-RELEASE-p3 / OpenZFS 2.4.2 | ZFS root | `TestNativeStateDirectory`, 0.13 s | 35/35 pass | base/runtime preserved; mounts/children/fixtures/image/key/variables removed; group stopped |
| FreeBSD 14.5-RELEASE / OpenZFS 2.2.9 | UFS root | `TestNativeStateDirectory`, 0.13 s | 35/35 pass | base/runtime preserved; mounts/children/fixtures/image/key/variables removed; group stopped |

Both fixtures used Go 1.27.1, Python 3.14.7 and owned x86-64 QEMU 11.1.1 TCG
guests with two CPUs and 6144 MiB. These are disposable test profiles, not product
resource requirements or hardware support evidence. No native attempt failed or
skipped. The historical device/property suites were not rerun or relabeled here.
The complete M2-STATE-001 and other gates listed above remain open.

GitHub Actions remain disabled. The maintainer-deferred M1 Safari/manual
accessibility gate remains pending. No physical/controller/commercial claim follows.
