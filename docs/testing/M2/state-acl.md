# M2 descriptor-based state ACL validation

Date: 2026-10-05. Status: implementing; native qualification and review pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Boundary

The existing [state lease](state-directory.md) and [ownership reader](ownership-read.md)
now inspect permissions through actual opened descriptors before trusting mode
bits. This adds no public API, record schema, configuration or runtime wiring.
All existing UID, mode, type, link, lock and bounded-codec checks still apply.
Product code never modifies an ACL, repairs permissions or provisions a directory.

Query FreeBSD `fpathconf` for `_PC_ACL_EXTENDED` and `_PC_ACL_NFS4` on each opened
ancestor/private directory, writer lock and ownership record. Only two confirmed
zero answers establish mode-only access. An error, unknown/negative answer or
conflicting enabled models denies; unsupported ACL calls never become a fallback
to mode bits. On an enabled model, read its ACL through `__acl_get_fd` using the
existing x/sys syscall adapter and a fixed 4,088-byte kernel buffer. Require the
documented 254-entry capacity, valid bounded count and zero reserved fields.
This introduces neither cgo nor a command executor in product code.

POSIX access ACLs must contain exactly the three owner/group/other entries with
permissions matching the mode, undefined special-entry identifiers and no
extension fields. A directory's POSIX default ACL must be empty. Named entries,
masks and any inheritable default are refused, including currently masked grants.

NFSv4 ACLs may contain only owner/group/everyone entries, known allow/deny types,
known rights, undefined special-entry identifiers and no flags. Every special
principal must have an allow entry. Root-owner entries may use known rights.
Group/everyone allow entries can grant data reads or search only when the mode
permits them, plus FreeBSD's normal ancillary read-named-attributes, status,
read-ACL and synchronize rights. They cannot grant data/attribute/ACL/owner writes,
append, delete or delete-child. Named principals and all inheritance/audit flags
deny. Additional deny entries can restrict access; this conservative allowlist
does not implement general ACL equivalence or normalize operator ACLs. In
particular, ancillary attribute access is not a confidentiality claim for xattrs;
Horvium's metadata records remain regular file bytes.

Open validates ancestors before creating a lock and validates the actual lock
before claiming the lease. Each ownership read rechecks the held directory and
writer and validates the actual record. An ACL refusal returns the existing
`ErrStateDirectory` with no policy/handle. Read metadata comparisons still detect
ordinary observed changes; none of these checks fences another root actor or
qualifies a synchronized ACL change during a read.

## Native procedure

Require kernel root and the two disposable `state` VM opt-ins. For each ACL model,
allocate one new owned 32 MiB memory-backed `md` device, format only that device as
UFS and mount it under an owned private `/root` anchor with `acls` or `nfsv4acls`.
Validate the returned device name, filesystem and actual descriptor capabilities.
Create only owned private directory/lock/record fixtures. Preserve their original
ACLs for test-only restoration. The ordinary suite also re-executes its lease/read
controls on each guest's root filesystem: ZFS on 15.1 and UFS on 14.5.

Require baseline lease/read success. Install named ACL controls on the directory,
ancestor, lock and record; require refusals, no missing-lock creation and retained
mode/UID/inode/ACL/bytes. Install a POSIX default or NFSv4 inheritable control and
require refusal without repair. Restore only owned controls and require reading
works again. Kernel ACL setters/deletion exist only in this test fixture.

For NFSv4, grant UID 65534 directory search and record read while retaining root
UID and exact 0700/0600 modes. Copy the built test into an owned traversable
temporary directory and re-exec with actual UID/GID 65534, no privileged groups
and the held directory/writer descriptors. The child independently opens and
reads the record through the named grant, proving mode bits do not describe that
access, while the product reader still refuses its actual UID. Wait/reap it and
remove the copied executable fixture.

Close leases and all fixture descriptors, unmount each owned filesystem, detach
only its recorded memory device and remove the anchor. Cleanup also runs on
failure; the enclosing disposable VM supervisor independently destroys the owned
VM and verifies process/image/key/variable teardown. No operator disk, live state,
host mount or persistent service is used.

The suite requires all 108 named assertions: 35 lease, 28 reader and 45 ACL cases;
three native parent tests, seven unit parent tests and 57 codec subtests. The
portable policy test checks ABI layout, mode-equivalent examples, unsafe headers,
principals, rights, flags, defaults and additional restrictions. The evidence
checker rejects missing, duplicate, unknown, failed, skipped or oversized results.
Its tests reject incomplete ACL evidence before the checker change. Existing
21 bounded fuzz targets remain; this kernel ABI adds no untrusted byte parser.

```sh
python3.14 tools/local-qualification.py state 15.1 --run
python3.14 tools/local-qualification.py state 14.5 --run
make lab-status
```

## Primary evidence and remaining gates

Both pinned FreeBSD release revisions' ACL/type/capability headers, descriptor ACL
syscalls, UFS capability/ACL implementations, POSIX/NFSv4 permission logic and
libc retrieval code were inspected and hashed. Interface declarations establish
the fixed-width ABI and descriptor semantics; sources are not installed-binary
provenance or a substitute for actual native execution.

- [FreeBSD ACL interface](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/sys/acl.h)
- [Descriptor ACL retrieval](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/kern/vfs_acl.c)
- [UFS capability queries](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/ufs/ufs/ufs_vnops.c)

Partial M2-STATE-001/M2-OWN-001 only. Complete state provisioning, store/enumeration
budgets, atomic publication, job schemas/admission/reservations/history,
coordinated races and syscall failures, read-only/full-store/sync/rename faults,
acknowledgement/reboot/power-loss/restore recovery remain ahead. There is no
publisher, enrollment or runtime storage writer. No external client/share ACL,
hardware/controller/commercial support claim follows. Prior manifests retain
their original sources and scopes. M2 acceptance and maintainer-deferred M1
Safari/manual accessibility remain pending. GitHub Actions stay disabled.
