# Guarded persistent backing discovery

Status: Native discovery prerequisite qualified; awaiting maintainer review; runtime denied.
Date: 2026-10-09. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted mechanism](../../milestones/M2/property-recovery-mechanism.md),
[binding prerequisite](execution-bindings.md).

PR 112 accepted the binding representation/object observer at merged source
`276d7f79b0edfca3dc2856e53e3bcaac2b409020`. This private FreeBSD collector discovers
native persistent filesystem facts from an opened directory descriptor. It has no
public API, request selector, config path, service or runtime consumer. It does not
yet borrow a registered original state lease or assemble the two-domain binding.
That bridge, complete native system/owned-pool exclusion, independent restore
scope, explicit commissioning and startup/global ownership remain required.

## Fixed native projections and correspondence

Actual effective UID 0, a root-owned 0700 directory and writable local UFS/ZFS
without softupdates are required. Full descriptor metadata is retained throughout
collection, excluding only atime; directory content/change timestamps still detect
in-call changes. Before and after each probe pass, the current kernel-reported
mount root is opened as a no-follow directory and its device and complete live
mount view must match the original descriptor. Source, target, filesystem kind,
mount ID and flags must remain unchanged across both passes. Mount/device values
establish live correspondence and never enter the persistent identity returned.
The private collector does not replace the original state lease's full ancestor,
ACL, lock, namespace and publication checks. The assembled owner must hold that
lease and perform those checks around the observation; opening a directory alone
is not commissioning provenance.

UFS uses only base `/sbin/dumpfs -l <observed source>`. Parse exactly one
`/dev/ufsid/` line with sixteen lowercase hexadecimal digits and a final newline;
both 32-bit on-disk components must be nonzero. The actual root-owned character
provider's device number must equal the directory's native device, and its full
metadata must remain stable. The on-disk pair must also match the current UFS
kernel mount ID. Remapped zero/colliding mount IDs deny rather than falling back
to volatile identifiers. This is a conservative supported subset; no raw
superblock offsets or new syscall ABI are introduced.

ZFS uses fixed `zfs get -H -p -o name,property,value
guid,type,mounted,mountpoint <observed dataset>` and `zpool get -H -p -o value
guid <derived pool>`. Exactly four distinct dataset property rows must select the
same observed name, nonzero canonical decimal GUID, filesystem type, mounted=yes
and the actual target or legacy mountpoint. The pool GUID is one nonzero canonical
decimal line. Dataset names come only from the kernel mount source and pass a
bounded ASCII component grammar, with no snapshot/bookmark, options, whitespace
or controls. Provider names are canonical absolute `/dev/` paths with the same
component restrictions. Unsupported names and newer/unknown release formats deny.
No recursive dataset inventory or metadata mutation is executed.

Both complete passes must return identical persistent facts, mount observations
and exact kernel/userland version projection, restricted to the already reviewed
14.5/15.1 release families. Combined stdout/stderr consumes one 64 KiB collection
budget. Every fixed probe uses the accepted native base timeout/reaper guard, a
ten-second deadline and fixed environment; collection has a 45-second context.
Bounded strict parsing, errors, cancellation, partial output, unsupported inputs
and any observed drift return zero facts. These are probe limits, not a new
qualification of all possible kernel/filesystem stalls or production resources.

## Tests and qualification

Failing-first portable controls cover two positive filesystem cases and 33 denial
cases, including malformed/duplicate/foreign output, noncanonical/zero/overflow
identities, release drift, mount drift, accounting, deadlines and invalid selectors.
A bounded fuzz target validates the compact UFS/ZFS projections and zero partial
results. Parsing supplied bytes remains separate from native observation.

The focused disposable native suite selects two parents and requires 17 UFS and
thirteen ZFS assertions plus six filesystem/device/directory teardown controls.
Actual UFS observations survive record changes and an unmount/remount; read-only,
unsafe mode, invalid/closed descriptors, cancellation and credential-dropped
inherited descriptors deny. ZFS observes actual pool/dataset GUIDs, survives rename
and unmount/remount, distinguishes a second dataset and detects a new identity
after destroying/recreating the same dataset name. Both use owned memory devices
and are removed before reporting success. No operator data or live storage is
used. The ZFS fixture closes its directory before rename's automatic remount and
opens the resulting mount to compare identity. These are identity observations, not power-loss, reboot, rollback, snapshot
restore or integrated writer qualification.

Run `tools/local-qualification.py persistent-backing 15.1 --run` and then 14.5
through the existing pinned Mac/QEMU runner. The host builds once per run; the
guest verifies the exact executable and base utility hashes. Both complete
identity/teardown results are required, and skipped/duplicate/incomplete evidence
denies. Existing state/pool historical manifests are preserved. Shared tooling
maintenance requires one fresh portable review; later prose/evidence reuses its
exact deep inputs while rerunning baseline checks.

## Primary sources

Reviewed FreeBSD source pins are 14.5
`231eee334c098a8398bc959202415c48058f67f0` and 15.1
`e38a7085f3a8ecd317c947591fe42b5ca9ab317b`. Base `dumpfs -l` prints the actual
superblock ID through its libufs reader; see
[14.5 dumpfs](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sbin/dumpfs/dumpfs.c),
[15.1 dumpfs](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sbin/dumpfs/dumpfs.c)
and [15.1 libufs](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/lib/libufs/sblock.c).
The existing [object evidence](execution-bindings.md) records why UFS can remap
mount IDs. ZFS's live fsid GUID can also change on collision and is distinct from
its persistent dataset GUID; see
[14.5 dataset source](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/module/zfs/dsl_dataset.c)
and [15.1 dataset source](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/module/zfs/dsl_dataset.c).
The [OpenZFS property manual](https://openzfs.github.io/openzfs-docs/man/master/7/zfsprops.7.html)
describes GUID, type and mounted properties. Source reasoning does not replace
the required actual two-target qualification.


The [manifest](persistent-backing-manifest.json) records final signed source
`7954e5b4db923e79f5ea862f45a054fb85ff4f1e`, both actual guest results and
identical native binary SHA-256
`63a69ae5a11e45e7f00d0be49539be0dd7e31470f7cba197f002560aeef36251`.
Both native parents and all 30 identity/safety plus six teardown controls passed
on each target. The final runner automatically removed its compiled executable
as well as owned processes, devices, filesystems, keys and images. Generic prebuilt
cleanup has a failing-first regression proving all registered filenames are removed
without touching a foreign sentinel. Failed discovery/rename attempts, the stopped
superseded review and earlier positive runs with incomplete compiled-artifact
cleanup remain recorded with their actual sources; no original record was rewritten.
The retained binaries from those earlier runs were hash-verified and removed.
Fresh local review passed all baseline checks, 153 tooling tests, eleven race
packages and 53 bounded fuzz targets. All 86 historical M2 manifests and 141
existing host/helper/command inputs retained their bytes. All owned sessions
stopped; GitHub Actions remained disabled. Production resources, original
dual-lease integration, system exclusion, independent restore proof and
commissioning/startup/global ownership remain required.
