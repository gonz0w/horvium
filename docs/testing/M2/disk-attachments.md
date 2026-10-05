# M2 disk attachments and partition denial

Date: 2026-10-05. Status: all seventeen assertions, both executed native tests
and complete preservation/teardown passed on both FreeBSD targets; accepted in
[PR 32](https://github.com/gonz0w/horvium/pull/32) at merged source
`f53688f29d6ba85e47b7062bbada73a857f02378`. [Approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Read-only contract

`inventory.ParseDiskAttachments` examines a complete successful GEOM capture and
one single-component provider address. It performs no I/O. The address and graph
share the byte budget; existing field, record, depth and UTF-8 limits apply.
Errors return a zero projection and fixed diagnostics.

The address must resolve exactly once to a direct DISK provider with one output,
no backing parents and a present collection-unique consistent serial. Every
reachable incoming attachment must be a terminal DEV node with the exact parent
name, or one direct whole-disk LABEL provider named exactly `diskid/DISK-<serial>`
with equal byte capacity/logical sector size and its own terminal DEV. Aliases
remain logical providers. Absent aliases are permitted; claimed alias names with
broken/detached backing fail. Duplicate names/DEV nodes, chained labels, differing
geometry/serials, missing DEV, PART (including an empty table), multipath, ELI,
MIRROR and all other attachment classes refuse this narrow correspondence.
Encoded/truncated alias guesses are never substituted for an exact match.

The caller must supply the full successful fresh graph; valid XML cannot prove
capture completeness or atomicity. This is a partial attachment projection, not
an unused/eligible flag. Raw signatures, imported/exported pools, boot/swap/mount
use, locality/shared-host fencing, durable serial enrollment, reservations and
fresh drift still require separate evidence. No HTTP, helper, CLI, collector or
storage planner wiring is introduced.

## Tests and disposable native procedure

Failing behavioral tests preceded implementation. They cover normal DEV/LABEL
correspondence, unsupported/empty PART shapes, contradictory aliases/geometry,
missing/duplicate serials, missing DEV, dangling/cyclic backing, zero-on-error and
aggregate/field/record limits. `FuzzDiskAttachments` is the fourteenth bounded
local review target.

The [owned device fixture](devices.md) first executes its original read-only test:
eight fixture assertions and four [disk correlation](disk-correlation.md)
assertions. The subsequent `TestNativeDiskAttachments` independently validates
all five declared 64 MiB DISK providers, configured serial/count/sector cases,
direct backing, the selected fixture's actual diskinfo identity/geometry, and the
actual mounted UFS/ZFS root's backing exclusion before any partition command.
Only the unique `HVM-UNIQUE-512` fixture is writable. An actual exact disk-ident
alias is required for the native positive case. Hash the other four fixture disks;
never hash boot content or accept arbitrary operator paths.

Create an owned GPT table; independently observe its PART consumer with no
outputs and require parser refusal. Add exactly one owned 16 MiB `freebsd-zfs`
partition and require refusal with independently observed PART geometry/backing.
Delete only its declared index and destroy the empty table without force. Cleanup
is registered before mutation and runs on failure; failure still discards the VM
and cannot qualify. Verify restored attachment correspondence, all fixture
identity/geometry, whole-byte preservation of the four other disks and stable
root mount/backing plus ZFS root GUID when applicable. GPT cleanup need not restore
the modified guest disk's original bytes. The host's original five raw backings,
base image and QEMU/firmware must all remain unchanged under global snapshot mode.
Stop/reap the owned group and remove disks/image/key/variables before qualification.

```sh
python3.14 tools/local-qualification.py devices 15.1 --run
python3.14 tools/local-qualification.py devices 14.5 --run
make lab-status
```

Run clean signed source and guests sequentially on the development Mac. The
checker requires both executed tests and exactly seventeen assertions: eight
fixture, four correlation and five attachment (`native_alias`,
`blank_table_denial`, `partition_denial`, `fixture_teardown`,
`foreign_preservation`). Guest success alone, missing/duplicate/failed/skipped
cases or missing host cleanup cannot qualify. This test-only privilege adds no
product partition/wipe operation. No controller, physical performance, full
M2-DEVICE-001 or production claim follows.

## Executed evidence

Signed source `0c9882f45c3abc1a81c693af01720ece9e50a9e6` passed all seventeen
assertions on FreeBSD 15.1-RELEASE-p3 and 14.5-RELEASE through Mac arm64 / QEMU
11.1.1 TCG (two CPUs, 6144 MiB). The actual guest roots were ZFS and UFS,
respectively. Both runs restored the fixture attachment graph, preserved the
four other guest disks and root correspondence, preserved all original host
backings/base/runtime inputs, removed disks/image/key/variables and verified
the owned group stopped. The [sanitized execution manifest](disk-attachments-manifest.json)
records source/input hashes, actual versions, six utility hashes and outcomes.

An initial run on source `286c51da2d5a6ff59fbe99b8eef815238df1db0e` was
stopped through its supervisor before native tests after source inspection found
an incorrect test expectation (`Logical` instead of the existing `Partition`
kind). Host preservation and teardown passed; the interrupted run remains
unqualified and is retained separately in the manifest. The corrected signed
source was fully rerun. Its complete portable review passed 61 tooling tests,
Go unit/race checks and fourteen bounded fuzz targets; final PR-head review is
reported separately in the PR. Input changes require fresh native qualification.
No new controller-death trial is claimed; the accepted original fixture
containment evidence retains its earlier source.

## Provenance

The release-specific DEV/LABEL/disk-ident sources were inspected and their SHA-256
values verified against the unchanged [original source record](devices-manifest.json):
[14.5 DEV](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/geom/geom_dev.c),
[14.5 LABEL](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/geom/label/g_label.c),
[14.5 disk-ident](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/geom/label/g_label_disk_ident.c),
[15.1 DEV](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/geom/geom_dev.c),
[15.1 LABEL](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/geom/label/g_label.c),
[15.1 disk-ident](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/geom/label/g_label_disk_ident.c).
The [base gpart manual](https://man.freebsd.org/cgi/man.cgi?query=gpart&sektion=8&format=html)
was reviewed for create/add/delete and ordinary empty-table destroy semantics.
Source inspection cannot establish installed binary build provenance. Record actual
versions and utility hashes, including `/sbin/gpart`, with native results.
Historical eight/twelve-assertion manifests retain their original source/case sets
and cannot qualify this extension. Export only sanitized assertion/version/hash
records, never raw GEOM/boot inventory or credentials. GitHub Actions remain disabled.
