# M2 filesystem signature observations

Date: 2026-10-05. Status: all thirty-six assertions, four executed native tests
and complete preservation/teardown passed on both targets; maintainer review pending.
[Approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Positive recognition contract

An empty imported-pool catalogue or GEOM consumer list cannot establish that a
disk has no filesystem or residual pool metadata. `storage.ParseFilesystemSignature`
binds a complete successful base `fstyp -u` scalar capture to independent fresh
GEOM, diskinfo and serial captures. The product parser performs no I/O and adds
no collector, helper, planner, CLI/HTTP wiring or storage mutation authority.

`FilesystemProbeCapture.Device` records the exact validated native argument.
Require that address to equal the canonical direct disk in the accepted
[disk observation](disk-correlation.md); wrong addresses, aliases, missing/duplicate
serials and conflicting geometry/backing refuse. Probe stdout contains no address.
The caller must establish actual execution success, target, complete capture,
freshness and provenance; syntax cannot prove those facts or atomicity.

Accept only exact `ufs` or `zfs` plus one LF, from `fstyp -u` without `-l` or `-s`.
No trimming, case folding, labels, escape decoding, additional rows or unsupported
formats are accepted. Empty/unrecognized/truncated/failed/unsupported captures
leave the observation unavailable; there is no blank/absent/eligible state. Other
filesystem types remain unsupported by this subset. Errors return fixed diagnostics
and a zero projection. Retain the full direct-disk identity/geometry correspondence
and positive type, without inventing filesystem health, pool GUID or ownership.

All raw captures and the recorded probe address share one byte budget. GEOM ID
entries and the three fixed diskinfo/serial/probe records share one record budget;
field, XML depth and element bounds remain enforced. Reserve the three records
before invoking the existing correspondence parser. `FuzzFilesystemSignature`
adds the seventeenth bounded local review target and mutates all five inputs.
Behavioral tests failed before implementation and cover positive types, address/
identity/geometry drift, malformed/unsupported output and aggregate bounds.

Base probe recognition has a deliberately narrow meaning. On both pinned
releases, the ZFS probe reads only the first label and decodes its pool-name field
without validating its checksum; it does not inspect all labels, pool state or
GUID membership. UFS uses a superblock read with checksum checking disabled.
A recognized type is positive metadata evidence, not authenticated label contents,
filesystem/label integrity or proof of a usable pool/filesystem. A failed probe
cannot prove absence; damaged/other-offset signatures and metadata may remain.

Full raw signature/label absence, exported-pool membership and identities, raw
open-device use, locality/shared-host fencing, enrollment, reservations, live
collector/planner integration and fresh execution clearance remain open. No pool
creation/import/destruction, partitioning, wiping or label-clear operation follows.
This is partial M2-DEVICE-001/M2-NATIVE-001 evidence, not full device safety or M2
acceptance. The runtime remains read-only.

## Disposable native procedure and required evidence

Reuse the [owned snapshot disks](devices.md) and actual imported-pool/UFS fixtures.
The extension adds only read-only probe/hash calls and pure denial controls; it
introduces no new fixture mutations. Existing serial/count/sector/boot exclusion,
independent diskinfo/attachment and system/imported-use prechecks still run before
any of those existing fixture writes.

In `TestNativeImportedPoolUse`, probe the declared unique 4K disk while its owned
unmounted pool is imported. Require actual `zfs` recognition and exact correlated
disk identity/geometry. After existing guarded owned-pool destruction, require the
full imported catalogue to match baseline; probe again for residual ZFS metadata.
Hash the closed modified fixture before and after all signature probes to prove
whole-byte preservation. Change only the recorded probe argument in a copy of
actual captures to the declared boot-image address: the pure parser must refuse
with a zero result. No signature probe or hash is executed against the boot image.
Keep the existing other-four-disk/root/catalogue preservation and teardown checks.

In `TestNativeSystemDeviceUse`, probe the declared unique 512-byte UFS fixture
while it is mounted read-only. Require actual `ufs` recognition and the exact
expected correlated disk. Whole-device hashes before/after must agree. Change
only a copy of actual stdout to an unrecognized type and require refusal with a
zero projection. The existing owned mount/swap cleanup, other-three-disk/root
preservation and identity/geometry checks still run.

Only after complete per-test preservation and teardown, emit the new marker sets:

- `DEVICE_SIGNATURE_UFS_RESULT`: `ufs_signature`, `probe_preservation`,
  `unsupported_denial`.
- `DEVICE_SIGNATURE_ZFS_RESULT`: `zfs_signature`, `removed_signature`,
  `probe_preservation`, `binding_denial`.

Four actually executed native tests, all thirty-six assertions and complete host
preservation/teardown are required. A failing-first evidence regression refuses
missing/duplicate/unknown/failed signature cases; the existing gate also refuses
skipped/failed/missing tests and absent host cleanup. A guest-only pass is insufficient.

```sh
python3.14 tools/local-qualification.py devices 15.1 --run
python3.14 tools/local-qualification.py devices 14.5 --run
make lab-status
```

Run sequentially from clean signed source. Modified guest disk bytes may retain
filesystem/pool metadata; no restoration or signature clearance is claimed.
Discard the guest even after success. Snapshot mode must preserve all five original
host backings, base images and runtime inputs; remove disks/image/key/variables
and stop the owned group before qualification. Export only sanitized assertion/
version/hash records. Prior manifests retain their original source/case sets and
controller-death evidence; no new controller-death trial is claimed. GitHub Actions
remain disabled and no persistent runner or host service is installed.

## Executed evidence

Signed source `9a363b053d9ba12d1a23da3465c66f438f72b8d9` passed all thirty-six
assertions and four native tests on FreeBSD 15.1-RELEASE-p3 and 14.5-RELEASE
through Mac arm64 / QEMU 11.1.1 TCG (two CPUs, 6144 MiB). Actual roots were ZFS
and UFS, respectively. UFS recognition bound to the owned read-only mounted disk;
ZFS recognition bound to its owned unmounted pool disk, including metadata retained
after guarded destruction. Closed fixture hashes matched before and after probes.
Pure wrong-address and unsupported-output controls refused with zero projections.
Existing catalogue/root/mount/swap/other-fixture preservation and teardown passed.
No new fixture writes, whole-label scan, checksum validation or absence assertion
was added by this extension.

All five original host disk backings, base images and runtime inputs were
preserved; owned disks/image/key/variables were removed and process groups
stopped. The [sanitized execution manifest](filesystem-signatures-manifest.json)
binds exact source/input hashes, actual versions, twelve utility hashes per guest,
eight pinned primary files and outcomes. Functional-source review passed 64 tooling
tests, Go unit/race checks and seventeen bounded fuzz targets. Final PR-head review
is reported separately; changing any recorded input requires fresh qualification.
Previous manifests and controller-death evidence retain their original revisions
and case sets. This is partial M2-DEVICE-001/M2-NATIVE-001 evidence only.

## Primary source provenance

Pinned revisions: FreeBSD 14.5 `231eee334c098a8398bc959202415c48058f67f0` and
15.1 `e38a7085f3a8ecd317c947591fe42b5ca9ab317b`. For each, inspect and retain hashes
of `usr.sbin/fstyp/fstyp.c`, `fstyp.8`, `ufs.c` and `zfs.c`. The manual and producer
establish `-u`, read-only opening, scalar stdout without `-l`, and nonzero status
for both errors and unrecognized formats. There is no reliable absence result.
The actual installed `/usr/sbin/fstyp` hash joins the existing eleven device-suite
utility hashes. Inspected source hashes identify reviewed contracts, not installed
binary build provenance. The native manifest records exact tested source/input
hashes, retrieved primary source hashes and actual native outcomes.
