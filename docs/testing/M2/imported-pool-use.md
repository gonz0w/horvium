# M2 imported-pool use observations

Date: 2026-10-05. Status: all twenty-nine assertions, four executed tests and
complete preservation/teardown passed on both targets; accepted in
[PR 34](https://github.com/gonz0w/horvium/pull/34), merged source
`50d6399b58f201ea26ec007879fe56792993c312`.
[Approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Positive membership contract

A disk can belong to an imported pool even when no filesystem is mounted. The
accepted [system-use projection](system-device-use.md) maps mounted pools only.
`storage.ParseImportedPoolUse` independently maps the complete imported-pool
catalogue to positive direct-disk membership. It performs no I/O and is not wired
into the collector, helper, planner, CLI or HTTP interface.

Require complete fresh successful `zpool list -H -p -o
name,guid,size,alloc,free,health` with no selected pool names, a complete GEOM
capture, and independent GUID-value plus paired GUID/path status captures for
exactly every listed pool. Use the existing strict six-column inventory parser:
duplicate names/GUIDs, partial rows, wrong columns/units and unavailable values
refuse. Pool addresses use the existing reviewed name subset with no dataset,
snapshot or bookmark suffix. Listed GUID and health must match the independent
GUID/status observations; unknown/missing/extra/repeated/contradictory captures
return fixed diagnostics and no result rows.

The shared resolver retains all terminal native pool members, following exact
GEOM provider addresses and backing consumers to direct DISK roots. Broken,
cyclic, ambiguous, logical/memory/file-backed leaves refuse. Missing/duplicate disk
identifiers and adverse but supported health remain positive use observations;
they cannot hide an in-use disk. Overlapping disk membership is retained rather
than declared exclusive. Sort pools by literal GUID spelling and disks by name.

All raw captures and pool addresses share a byte budget. GEOM classes/geoms/
providers/consumers, catalogue rows, captures, topology nodes, backing visits and
expanded references share one record budget. Count catalogue rows before parsing
and retain the existing depth, field and status-format bounds. Resource refusal
returns no partial success. The mounted-pool parser uses the same backing helpers;
its behavioral and actual native cases must pass again on this source.

A successful empty catalogue is an observation supplied by the caller, not proof
that a device is unused. Syntax cannot establish completeness, atomicity or
capture authenticity. Exported/destroyed pools, on-disk labels/signatures, raw open
use, physical locality/shared-host fencing, enrollment, reservations and fresh
launch clearance remain open. No product pool creation/import/destruction/wipe,
eligibility flag or runtime authority follows. This is partial M2-DEVICE-001 and
M2-NATIVE-001 evidence, not completed M2 device safety or milestone acceptance.

## Tests and disposable native procedure

Behavioral tests failed before implementation. They cover positive membership
without any mount input, successful empty catalogues, exact name/GUID/status
binding, missing/extra/duplicate/contradictory captures, unsupported leaves, adverse
health, overlapping disks, missing/ambiguous identity and cumulative resource
limits. `FuzzImportedPoolUse` is the sixteenth bounded local review target and
mutates GEOM, catalogue, pool name, GUID and both status captures together.
The evidence gate has a failing-first regression for missing/duplicate/unknown/
failed/skipped result cases and executed tests.

The native test runs after the accepted read-only/attachment tests and before
UFS/swap setup, on the [five declared snapshot disks](devices.md). Independently
verify all serial/count/sector cases, sole boot-image disk, actual imported pools,
root/mount/swap backing and exact diskinfo/attachment correspondence. Only the
unique `HVM-UNIQUE-4K` fixture may receive pool writes; require it absent from
initial imported/mounted/swap backing. Hash whole bytes of the other four disks.
No arbitrary operator disk or boot-image disk is a write target.

Declare a new random fixture pool name and register cleanup before creation.
Create one single-leaf test pool without force, with `cachefile=none`,
`compatibility=legacy`, `mountpoint=none` and `canmount=off`. These options isolate
a positive-membership fixture; they do not qualify Horvium's redundant creation
operation or common feature profile. The existing 64 MiB backing meets the
inspected native minimum; actual success still requires native execution.

Require the actual full catalogue and independent GUID/status/GEOM captures to
identify exactly that pool and disk, while actual mount/swap observations do not
use it. Independently compare the catalogue row GUID; omit its actual status to
prove refusal without partial rows. Cleanup recollects positive membership and
checks the random name, recorded GUID when available and exact sole declared disk
before destroying only that owned pool, without force. Cleanup runs on failure
with a fresh bounded context; an uncertain/unsupported cleanup is failure.

Recollect after removal: imported-pool projections and actual root/mount/swap
facts must match baseline, all five fixture identities/geometries must remain
unchanged, and whole bytes of the other four disks must match. Destroying a test
pool does not establish that labels/signatures were cleared. Discard the guest;
snapshot mode must preserve all five original host files, base images and runtime.
Remove owned disks/image/key/variables and stop the owned process group before
qualification. No host service or persistent runner is installed.

```sh
python3.14 tools/local-qualification.py devices 15.1 --run
python3.14 tools/local-qualification.py devices 14.5 --run
make lab-status
```

Run sequentially on clean signed source. Four actually executed tests, all
twenty-nine assertions and complete host preservation/teardown are required.
The six new cases are `unmounted_pool`, `catalogue_binding`, `missing_denial`,
`fixture_teardown`, `foreign_preservation` and `root_preservation`. Previous
manifests retain their original source and case sets; no new controller-death
trial is claimed. GitHub Actions remain disabled and all checks run locally.

## Executed evidence

Signed source `5b0a94128aa081fbb3f5d8a79ed8ab560d1f4a09` passed all twenty-nine
assertions and four native tests on FreeBSD 15.1-RELEASE-p3 and 14.5-RELEASE
through Mac arm64 / QEMU 11.1.1 TCG (two CPUs, 6144 MiB). Actual roots were ZFS
and UFS, respectively. Both unmounted imported pools mapped to their declared
disk, missing-status observations refused, only owned pools were removed without
force, foreign fixture bytes/catalogues and root/mount/swap facts were preserved,
and all fixture identities/geometries remained unchanged. The subsequent actual
UFS/swap test passed again on both releases.

All five original host disk backings, base images and runtime inputs were
preserved; owned disks/image/key/variables were removed and process groups
stopped. The [sanitized execution manifest](imported-pool-use-manifest.json) binds
source/input hashes, actual versions, eleven utility hashes per guest, pinned
primary sources and outcomes. The functional-source review passed 63 tooling
tests, Go unit/race checks and sixteen bounded fuzz targets. Final PR-head review
is reported separately; changing recorded inputs requires fresh qualification.
Previous manifests and controller-death evidence retain their original revisions
and case sets. This is partial M2-DEVICE-001/M2-NATIVE-001 evidence only.

## Primary source provenance

Release-pinned FreeBSD/OpenZFS sources were inspected before native execution.
The list manual and producer confirm all imported pools are selected when no
names are supplied, `-H` uses tab-separated headerless fields and `-p` uses exact
values. Unsupported properties emit unavailable values which the parser refuses.
The creation manual documents the fixture's no-mount and legacy compatibility
options. The pinned kernel checks raw device size against `SPA_MINDEVSIZE` before
subtracting labels; the header defines that minimum as 64 MiB. This is a fixture
allocation, not a product hardware or pool-size assumption.

The actual executed zpool/sysctl/mount/diskinfo binaries must be hashed with the
other existing device-suite utilities. Inspected source hashes identify reviewed
contracts; they do not establish installed binary build provenance. Export only
sanitized assertion/version/hash records, never raw host inventories or fixture
credentials. The native manifest records exact tested source/input hashes, all
retrieved primary source hashes and actual outcomes. Binary build provenance
remains unproven.
