# Execution binding representation and native object observation

Status: Native object and portable metadata prerequisite qualified; awaiting maintainer review; runtime denied.
Date: 2026-10-09. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted mechanism](../../milestones/M2/property-recovery-mechanism.md),
[journal persistence](execution-journal-persistence.md).

PR 111 accepted the private journal file component at merged source
`334b9dfdbed614d4f70cc6b6913304104bd6cec5`. This prerequisite defines private
persistent binding inputs and observes actual protected directory/lock objects.
It does not yet produce native persistent **backing** identities. Guarded backing
discovery and correspondence, system/owned-pool exclusion, independent restore
scope, fresh root commissioning, registered dual-lease ownership and startup/global
gates remain required before any runtime writer. Supplied identities are metadata.

## Private representation

The existing journal version/fields stay unchanged. Backing inputs are either a
nonzero on-disk UFS `fs_id` pair or nonzero ZFS pool/dataset GUIDs. Unknown types,
zero/missing components and mixed UFS/ZFS fields deny with zero output. The UFS
pair is deliberately not obtained from `st_dev` or a bare kernel `f_fsid`.
Backing discovery must correlate authoritative persistent identities with the
original descriptors and fresh complete native system observations; this producer
is not implemented here. No configured path, probe command or public API is added.

Each object input includes a nonzero inode and generation plus positive birth
seconds and nanoseconds below one billion. UFS1/missing birth time and unavailable
generation deny; there is no timestamp fallback. Mutable directory size, atime,
mtime, ctime, mount source/target and device/mount IDs do not enter persistent
digests. They cannot replace persistent backing identity. These hashes detect
ordinary replacement; they do not prove protection against root edits, restored
inodes, snapshots, joint rollback or memory resume.

SHA-256 framing has explicit `horvium:execution-backing:v1` and
`horvium:execution-objects:v1` domains, newline terminated. Integers are eight-byte
unsigned big endian; strings have an eight-byte byte-length prefix. Backing frames
contain filesystem kind, both UFS components, pool GUID and dataset GUID, including
the unused zero fields. Directory/lock frames contain their role, ordered ledger
then journal index, backing digest, inode, generation, birth seconds and nanoseconds.
The installation identity remains the existing separate 32 lowercase hex field;
valid formatting does not prove fresh randomness or provisioning provenance.
Both domains therefore contribute to the journal's directory and lock digests.

Intra-domain inode aliases deny. Same-backing cross-domain inode aliases deny even
when supplied generation/birth metadata differ. ZFS requires different dataset
identities; ancestor/recursive snapshot and restore scope still needs independent
proof. Distinct UFS directories on one filesystem can be represented, but only the
accepted qualified file-level ledger restore procedure can make them usable.
Equal UFS backing digests are not a recovery or whole-filesystem restore allowance.

## Native protected objects

A private FreeBSD observer requires actual effective UID 0 and the original open
state lease. It holds the lease mutex, validates descriptor modes, owners, ACLs,
empty single-link lock and its no-follow name correspondence, and collects actual
`fstat` inode/generation/birth values. Full metadata and the live `fstatfs` view are
checked again before returning. Writable local UFS/ZFS are required; softupdates
deny. Invalid, copied, closed, inert and non-root handles return zero observation.
The collector neither creates nor repairs objects and does not inspect the entire
record namespace; the independent census/journal namespace readers remain required.

Live mount fields are retained solely to detect in-call drift. The observer has
no factory that turns its mount IDs into persistent backing identities. Successful
observation does not clear previously latched publication uncertainty, create a
durability receipt, bind two leases, commission an installation or enable launch.
Descriptor anchoring preserves the existing lease semantics; no pathname-based
reconstruction of an original writer is introduced.

## Evidence and scope

Failing-first portable tests include independently calculated wire vectors,
thirteen component substitutions and twenty denial cases. Native gates require
25 protected-object assertions and five actual UFS remount assertions, including
reopening the original objects after publication, retained uncertainty, inode
replacement, same-inode in-call drift and credential-dropped inherited descriptors.
The remount fixture closes its original lease, unmounts/remounts its owned UFS
memory filesystem and reopens the unchanged directory/lock before comparing actual
identities. This tests UFS object persistence, not backing discovery, ZFS remount,
reboot, crash recovery, restore or anti-rollback behavior.

Run both pinned disposable FreeBSD/amd64 guests through the existing state runner.
The complete selected corpus now requires 76 parents, 1,054 native markers and
1,930 codec children. The evidence checker rejects old/incomplete evidence and
requires the remount fixture's actual UFS backing and verified teardown. A fresh
portable review is required for checker changes; subsequent evidence/prose uses
exact-input deep reuse with all baseline checks repeated. Preserve historical
manifests and all existing host/helper/command sources.

## Primary source rationale

Reviewed sources are FreeBSD 14.5 commit
`231eee334c098a8398bc959202415c48058f67f0` and 15.1 commit
`e38a7085f3a8ecd317c947591fe42b5ca9ab317b`, already pinned by native qualification.
The UFS mount code can replace zero or colliding on-disk IDs with a new kernel
mount ID; bare `f_fsid` cannot supply the required universal persistence claim.
See [14.5 UFS mount](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/ufs/ffs/ffs_vfsops.c)
and [15.1 UFS mount](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/ufs/ffs/ffs_vfsops.c).
UFS reports inode generation and UFS2 birth time from the inode, while `va_fsid`
uses the device number; see [14.5 UFS objects](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/ufs/ufs/ufs_vnops.c)
and [15.1 UFS objects](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/ufs/ufs/ufs_vnops.c).
ZFS exposes `z_gen` and creation time as generation/birth time; see
[14.5 ZFS objects](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/module/os/freebsd/zfs/zfs_vnops_os.c)
and [15.1 ZFS objects](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/module/os/freebsd/zfs/zfs_vnops_os.c).
Source reasoning does not replace disposable native observations.


The [manifest](execution-bindings-manifest.json) records final signed implementation
`a84e22d635bbb387ece77fd0d6a0c6e1ec609044`, both actual guest results and the
identical binary SHA-256 `e40c2ef9255b1ce9c74e5c1d045ad23beb006e6698b24f16c4cca771218edb23`.
All 76 selected parents, 1,054 native markers and 1,930 codec children passed on
both targets with verified fixture/process/key/image teardown. The object parent
took 0.17 s on UFS/14.5 and 0.06 s on ZFS/15.1; actual UFS remount took 0.14 s
and 0.20 s respectively. These small fixtures are not production resource or
performance qualification. Fresh local review passed all baseline checks, 150
tooling tests, eleven race packages and 52 bounded fuzz targets. Both unqualified
native attempts and the failed portable runner fixture are retained with original
sources. All 85 historical M2 manifests and 141 existing host/helper/command
inputs retained their bytes. All owned sessions stopped; GitHub Actions remained
disabled. Guarded persistent backing discovery, independent restore scope and
commissioning/startup/global ownership remain the next required prerequisites.
