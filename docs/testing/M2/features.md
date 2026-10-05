# M2 pool feature observations and common profile checks

Date: 2026-10-04. Status: behavioral and pinned native subset passed; maintainer review pending.
Design: [approved EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[slice sequence](../../milestones/M2/implementation.md). Extends the
[pool observations](pool-state.md); runtime remains read-only.

## Complete bounded projection

```text
/sbin/zpool get -H -p -o name,property,value,source all <pool>
```

`ParsePoolFeatures` requires a successful complete single-pool table, native
positive canonical uint64 GUID with source `-`, and every feature property in the
selected qualified format. The pinned registration tables contain 39 feature
names for `freebsd-14.5` and 47 for `freebsd-15.1`. Selection of the actual native
format is a trusted future adapter obligation, not an HTTP/client choice or a
version substring used to grant authority. The native fixture selects only its
pinned release. Adding a release or changing its feature table needs new evidence.

Retain `feature@name` states disabled/enabled/active and local source, plus
`unsupported@vendor:feature` inactive/readonly observations. Unknown well-formed
feature names remain visible. `VerifyPoolFeatures` refuses any unknown/unsupported
feature, even disabled or inactive, and any missing qualified name. It binds native
pool name/GUID independently to the observation; names cannot substitute for GUIDs.
Known active features are observed and preserved, never enabled, disabled or upgraded
by this model. Recognition does not prove that a particular layout, dataset,
operation or backup peer supports that feature.

Non-feature properties are shape/source checked but do not grant capabilities.
Unknown `@` namespaces, duplicate properties, malformed/empty/truncated records,
controls, invalid UTF-8, invalid feature names/states/sources and failed bounds
return a zero projection. Cap the raw table at 64 KiB and 256 newline records,
each field at 4,096 bytes, and retained features at 128. Typed inputs participate
in the existing aggregate 64 MiB preflight bound before maps or serialization.
A complete record omission is detectable for GUID and the qualified feature set;
no parser can establish that an unsupported row was not removed before capture.
The future collector must execute the fixed successful command directly and
reject unavailable/truncated capture. There is no arbitrary filtering authority.

Reviews normalize feature order and hold independent slice copies. Feature state,
source, format, pool identity or membership changes require a new review, even
when filesystem properties stay the same. Pool and feature metadata count against
the existing 64 KiB review cap with no truncation. Empty/partial evidence cannot
become permissive defaults. These projections cannot set complete/eligible flags;
no runtime collector or product mutation is connected.

## Approved profile bytes

`CommonFeatureProfile` accepts only the approved 584-byte `openzfs-2.2` file,
SHA-256 `8b5a635bf7609a0f521b08fe0e389c651dd6a266d33bd53a2e253b99089c3bf7`,
and returns an independent list of its 39 names. Missing, modified or differently
formatted bytes return unavailable, with no fallback. This pure function does
not open a path, prove installed ownership/lookup precedence, or authorize creation.
A later product file reader must independently qualify anchored descriptors,
root ownership, ancestor safety, bounded reading and change detection.

The disposable fixture checks the fixed installed
`/usr/share/zfs/compatibility.d/openzfs-2.2` path and ancestors for root ownership,
no symlinks or group/other write permission, and exact regular-file size/hash.
Only its final re-created owned pool uses that absolute profile path, avoiding
local filename shadowing. It asserts that all 39 approved features are enabled
or active and any newer format features are disabled, while retaining cachefile
none and an unmounted root. This is privileged test setup, not product pool creation
or device eligibility evidence. Freshly created owned snapshot/bookmark versions
activate the native bookmarks feature; fresh observations invalidate the prior
pure-model review. That native comparison also changes version membership;
behavioral tests independently isolate feature-only drift.

## Source and test gates

Feature names come from inspected pinned native registration tables:
[14.5](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/module/zcommon/zfeature_common.c),
[15.1](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/module/zcommon/zfeature_common.c).
The [pool formatter](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/lib/libzfs/libzfs_pool.c)
and [get callback](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/cmd/zpool/zpool_main.c)
define state/source columns and unsupported rows. The
[pinned common profile](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/cmd/zpool/compatibility.d/openzfs-2.2)
is identical across the inspected releases. Source facts alone do not qualify
installed tools or cross-release portability; no upstream implementation is vendored.

Behavioral tests first failed against stubs/old review behavior, then passed for
strict shape/identity/coverage/bounds, unknown/unsupported refusal, independent
copies, normalized ordering, feature-only drift and exact profile bytes.
`FuzzPoolFeatures` is the eleventh bounded fuzz target. Native fixture assertions
independently read each feature value/source, verify complete GUID-bound tables,
read preservation, installed profile and resulting feature set, and actual
feature/version drift. Six new markers join the earlier 36; the evidence checker
requires all 42 plus executed native test and owned-fixture teardown. Unknown
feature refusal and damaged-profile behavior are behavioral/source coverage,
not native unsupported-pool or installed-file fault qualification.

## Qualification and remaining work

Run [37249224834](https://github.com/gonz0w/horvium/actions/runs/37249224834)
passed both pinned guests at functional/suite source
`f3d7c63918fc5a6ef2d5c3ec1078f413f3179aae`. Each executed all 42 property/identity/
state/pool/feature assertions and teardown. Both evidence checks passed, and the
checksum-pinned base images and AnyVM runtime remained unchanged. No native
repairs were needed.

| Guest | Observed result | Tools |
|---|---|---|
| FreeBSD 15.1-RELEASE-p3 amd64 | All 42 assertions and teardown; 47 native feature rows; exact installed profile and resulting common set passed | ZFS userland/module 2.4.2-1-FreeBSD_g6330a45b0; Go 1.27.1 |
| FreeBSD 14.5-RELEASE amd64 | All 42 assertions and teardown; 39 native feature rows; exact installed profile and resulting common set passed | ZFS userland/module 2.2.9-FreeBSD_g079ba86d7; Go 1.27.1 |

The [sanitized manifest](features-manifest.json) records source/job links,
individual cases, inspected primary and tested input hashes, native utility
hashes, profile facts and guest/runtime pins. QEMU 8.2.2 on Ubuntu 24.04.5 used
fresh snapshot qemu64 guests with two vCPUs and 6,144 MiB, with VNC off. No operator
inventory, credentials or fixture keys are included.

Local `make check lint test secrets` and `make go-check` passed, including vet,
complete unit/race suites and eleven bounded fuzz targets. FreeBSD amd64 cross-build
passed before native execution; compilation alone is not native evidence.
Subsequent documentation/evidence edits preserve the tested functional/suite bytes.
Earlier projection records retain their original sources/cases.

This covers only partial M2-PLAN-001, M2-DRIFT-001,
M2-FEATURE-001 and M2-SCALE-001. Clean cross-release transfer/export/import,
product installed-file reading, creation/device identity and every complete M2
acceptance row remain pending. No receiver/send compatibility follows.

Slice 1 still requires provider/topology/status/system/device-use and assembled
complete-impact collection. Durable state/privilege/recovery/mutation/client/
browser/resource gates and physical/commercial qualification remain open.
The maintainer's deferred M1 manual accessibility gate remains pending.
