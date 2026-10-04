# Research: M2 storage contracts and durable-state risks

Date: 2026-10-04. Status: source/documentation evidence only; no M2 native tests.
Work: [issue 19](https://github.com/gonz0w/horvium/issues/19).
[Source manifest](source-manifest.json), [PRD](../../milestones/M2/PRD.md),
[EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md).
The designs/ADRs were subsequently adopted by maintainer merge of
[PR 20](https://github.com/gonz0w/horvium/pull/20). These findings remain source
evidence only. The immutable manifest preserves its pre-approval capture status;
current approval and implementation evidence live in the linked milestone records.

## Sources and observed baseline

Refreshed primary FreeBSD support/ZFS documentation, Go downloads and HTMX docs.
Captured 34 release-specific FreeBSD source files with immutable revision and
SHA-256 records: OpenZFS META/manuals, the common compatibility profile, and libc
sync/rename/lock manuals. Source hashes describe inspected files; they do not
prove which source built an existing guest image or what its installed files contain.

| Component | Dated source/prior evidence | M2 qualification |
|---|---|---|
| FreeBSD | 15.1/14.5 supported upstream; prior M1 guests 15.1-RELEASE-p3 / 14.5-RELEASE amd64 | No M2 native operation executed |
| Bundled OpenZFS | Prior guest strings 2.4.2-1-FreeBSD_g6330a45b0 / 2.2.9-FreeBSD_g079ba86d7; refreshed branch revisions match the M1 source pins | M2 health/mutation contracts unqualified |
| Go / x/sys | Product pins 1.27.1 / v0.48.0; exact M1 source/checksums retained | No new dependency proposed |
| HTMX | Embedded 2.0.11; upstream docs still describe that version | M2 forms/job interaction not implemented |
| Common profile | `openzfs-2.2`, 584 bytes, SHA-256 `8b5a635bf7609a0f521b08fe0e389c651dd6a266d33bd53a2e253b99089c3bf7`; identical in both inspected source branches | Installed file and cross-release import still untested |

[FreeBSD security information](https://www.freebsd.org/security/) lists planned
release EOL dates of 2027-03-31 for 15.1 and 2027-06-30 for 14.5. Refresh these at
native execution/release; upstream support dates are not Horvium commitments.
[M1's qualification](../../testing/M1/qualification-status.md) supplies real
inventory/security/resource/file-daemon/browser evidence, not M2 write safety.
The inherited manifest records the combined run and preserves pending manual UI
acceptance. No physical storage/controller or larger commercial lab is available.

## Findings from primary sources

1. Pool create preview computes a prospective layout but does not reserve devices
   or guarantee later privilege/use checks. Its force mode can override apparent
   use. Propose fresh identity/use validation and no force fallback, backed by
   [the pinned create manual](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/man/man8/zpool-create.8).
2. Import can select a numeric pool identity and avoid mounts. Force import can
   override apparent activity; rewind/recovery modes can discard data. This is not
   a safe generic retry. Propose explicit registered-created-pool recovery and
   preserve adopted OS import policy. See [the pinned import manual](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/man/man8/zpool-import.8).
3. Replacement involves reconstruction before the old member is removed; the
   replacement size must satisfy the native mirror/RAIDZ constraint. Scan work
   is not equivalent to the launch utility's lifetime. The manuals describe scan
   progress/error reporting and interactions between scrub and replacement.
   Propose independently observed completion and no general resilver cancel.
   [Replace](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/man/man8/zpool-replace.8),
   [scrub](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/man/man8/zpool-scrub.8),
   [status](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/man/man8/zpool-status.8).
4. The newer pinned status manual includes JSON options; the older one does not.
   Both describe vdev GUID output. Propose bounded version-qualified adapters,
   not a JSON-only assumption or permissive parsing of arbitrary status text.
   [15.1 status source](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/man/man8/zpool-status.8).
5. Quota accounts for a filesystem and descendants, including snapshots; reference
   quota excludes descendants. Inheritance and mount properties have separate
   effects. Propose explicit local/effective/source projections and a small property
   allowlist, with a native gate for encoding strings safely.
   [Filesystem properties](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/man/man7/zfsprops.7).
6. Compatibility profiles constrain which features may be enabled. The inspected
   profile bytes are identical across releases, supporting a candidate common
   creation policy. This does not establish installed availability, clean native
   import, replication compatibility or compatibility with every older receiver.
   [Pool properties](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/man/man7/zpoolprops.7),
   [common profile](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/cmd/zpool/compatibility.d/openzfs-2.2).
7. FreeBSD documents file synchronization, atomic namespace replacement and advisory
   file locks separately. None makes a ZFS command atomic with a metadata record.
   Propose ordered file/directory sync and explicit acknowledgement-gap recovery;
   test the exact directory filesystem and failures natively. A lock does not fence
   external ZFS/root actors or another host.
   [Sync](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/lib/libsys/fsync.2),
   [rename](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/lib/libsys/rename.2),
   [lock](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/lib/libsys/flock.2).

These are source statements and engineering inferences, not runtime observations.
The metadata/source files were fetched into a private temporary research directory;
only digests/links and this summary enter the repo. No upstream implementation or
new executable/asset is vendored, and no storage mutation was attempted.

## Proposed experiments after design approval

Before each native slice, use the [Test Design](../../milestones/M2/TDD.md) to prove
virtual block identity/use checks, common installed profiles, status/property
encodings, directory-sync durability, actual launch/scan acknowledgement gaps,
mount collision behavior and cross-release clean reimport. Pin a bounded question
and procedure with its implementation PR. A capability failure blocks that operation.

The designs propose preserving observation-only support for everything outside the
qualified mutation scope. Do not promote existing fixture teardown commands, root
privileges, source examples or M1 reaper behavior into a generic storage operation.
New CLI/API/on-disk fields are deliberately left to approved contract implementation.
