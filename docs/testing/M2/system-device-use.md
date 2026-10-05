# M2 system device-use observations

Date: 2026-10-05. Status: behavioral tests and FreeBSD cross-build pass;
native qualification pending. [Approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Positive use contract

`storage.ParseSystemDeviceUse` correlates complete successful GEOM, `mount -p`,
`swapinfo -k` and paired GUID/path status plus GUID-value captures for exactly the
mounted ZFS pools. It performs no I/O. All raw captures and pool addresses share
one byte budget. Graph, rows, topology nodes, backing visits and expanded output
references share one record budget; XML/backing depth, field and inherited status
format bounds apply. Rows are counted before splitting, and field allocation is
bounded before parsing. Errors expose fixed diagnostics and a zero projection.

Exactly one root mount is required, with UFS or ZFS backing. Mount rows have six
columns, canonical uint32 dump/pass fields, unique absolute canonical targets and
exactly one `ro` or `rw` option. The native producer prints raw names; ambiguous
whitespace, backslash/escape guesses and unsupported source formats refuse the
projection. Potentially truncated on/from names at the native 1023-byte payload
limit refuse even when their spelling matches another provider. Sources under `/dev/` resolve exactly through GEOM provider names and
backing consumers to direct DISK roots. Broken/ambiguous/cyclic relationships,
unknown leaves, logical or memory-only roots refuse. Identifiers are retained as
observations; missing/ambiguous root identifiers do not hide a disk's positive use.

Every ZFS mount binds to a supplied canonical positive GUID and the existing
[paired complete topology parser](topology.md). All terminal pool members must
resolve to GEOM-backed `/dev/` leaves; every disk in the mounted pool is marked
as backing that mount. The mount dataset does not imply a single selected vdev.
Missing, repeated, unrelated or contradictory pool captures refuse. Separate
native reads cannot prove atomicity or independent GUID-capture authenticity.

Only non-root `devfs`, `tmpfs`, `procfs`, `fdescfs` with their literal source names
and NFS source forms containing a remote separator have recognized non-block
mount observations. Unsupported nullfs/unionfs/file-backed mounts refuse this
partial projection. These observations grant no namespace or mount authority.

Swap requires the exact `1K-blocks` header and canonical numeric rows, unique
resolved device addresses below the native 31-byte name payload limit, positive size, used/available arithmetic and a native
integer percentage in 0..100. Preserve KiB values and the printed rounded
percentage; do not invent byte/geometry policy. Multiple devices require exactly
one terminal total matching their numeric sums; a single/empty swap set has no
total. Missing rows, wrong units, overflow, unresolved/NFS/file swap or contradictory
totals refuse. The caller must supply the full fresh successful captures; syntax
cannot establish that rows were not removed.

The result contains positive mount/swap-to-disk facts and no unused/eligible flag.
It does not prove absent signatures, unmounted imported/exported pool membership,
raw open-device use, locality/shared-host fencing, durable enrollment, reservations
or fresh launch clearance. No helper, HTTP, CLI, collector, policy or planner wiring
is introduced. Full M2-DEVICE-001 and mutation gates remain open.

## Tests and disposable native procedure

Failing behavioral tests preceded implementation. Coverage includes UFS alias
chains, multiple ZFS pool members/datasets, missing root, shadowed mount targets,
missing/extra/contradictory pools, cyclic/dangling/ambiguous backing, unsupported
sources, swap unit/number/arithmetic/total failures and aggregate/field/record
bounds, potentially truncated names that match a different provider, and row/field
floods. `FuzzSystemDeviceUse` is the fifteenth bounded local review target.

The [five-disk snapshot fixture](devices.md) first runs its eight original
read-only, four correlation and five attachment assertions. The third test,
`TestNativeSystemDeviceUse`, independently validates the five declared 64 MiB
serial/count/sector cases and the sole positive-capacity boot-image disk. Empty
unbacked optical providers are excluded from this test-only boot-image selector;
an additional positive-capacity disk refuses it. The actual UFS/ZFS
root must map to that image, outside all fixture disks. Every fixture must be
absent from observed mount/swap backing. Independent diskinfo identity/geometry
and the accepted attachment classifier must agree before any fixture writes.
Only `HVM-UNIQUE-512` and `HVM-UNIQUE-4K` are writable; hash the other three disks.

Create a UFS filesystem on the first owned disk and mount it read-only at a new
private test directory. Activate the second owned disk as swap. Require actual
mount and swap rows to map back to the exact expected providers, then refuse an
intentionally unsupported unit header derived from the actual capture. Remove
only owned swap and mount, without force; cleanup is registered before mutation
and runs on failure. Recollect native use, verify owned mount/swap absence and
preserve root source/type/GUID/backing, all fixture identity/geometry and whole
bytes of the other three disks. The fixture has one boot-image disk; multiple
boot members and other graph forms have portable coverage only.

Filesystem creation can leave signatures on the modified guest fixture. No guest
byte restoration or signature clearance is claimed. Discard the guest even after
success. Global snapshot mode must preserve all five original host raw files,
base image and QEMU/firmware. Stop/reap the owned group and remove disks/image/key/
variables before qualification. This test privilege adds no product newfs/mount/
swap/partition/wipe operation and installs no persistent host services.

```sh
python3.14 tools/local-qualification.py devices 15.1 --run
python3.14 tools/local-qualification.py devices 14.5 --run
make lab-status
```

Use clean signed source and run guests sequentially on the Mac. The evidence
checker requires three actually executed tests, all twenty-three assertions and
complete host preservation/teardown. The six new cases are `root_backing`,
`mounted_disk`, `active_swap`, `unsupported_denial`, `fixture_teardown` and
`foreign_preservation`. Skipped/missing/duplicate/failed cases, a guest-only pass
or unverified cleanup cannot qualify. Prior manifests retain their original source
and case sets. No complete device safety or physical/controller support follows.

The initial 15.1 run at source `42cab8524d3690ac6f9ff19fae28cb6c57397392`
passed the original seventeen assertions but failed the new test before writes
at its boot-image selector (0.15 seconds). QEMU also presents an empty DVD drive.
The selector now excludes zero-capacity unbacked devices and still refuses extra
positive-capacity disks, with a failing behavioral regression. Original backing/
base/runtime preservation and complete teardown passed; this initial run remains
unqualified. Fresh native qualification of the repaired source is pending.

## Primary source provenance

The release-pinned producers were inspected before implementation. `putfsent`
prints raw mount fields; `swapinfo -k` selects KiB and prints one total only for
multiple active devices. The release-pinned `mount.h` and `kvm.h` were also
inspected: mount on/from buffers are 1024 bytes and swap device names 32 bytes.
Their possible full-buffer truncations refuse rather than bind to another disk.
Source hashes are retained with the native execution
record, alongside actual installed utility hashes; source inspection cannot prove
installed binary build provenance.

| Release revision | Source | SHA-256 |
|---|---|---|
| `231eee334c098a8398bc959202415c48058f67f0` (14.5) | [mount.c](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sbin/mount/mount.c) | `5843c0b258546796caf1e2caf197f2a8cd9a4f2310dedb4a349eec915fbe45df` |
| Same 14.5 revision | [pstat.c / swapinfo](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/usr.sbin/pstat/pstat.c) | `8c72b9897eeebdb5b2e170bee175f2d90bb1f5e400e4ad204d2a64fd8d36081f` |
| `e38a7085f3a8ecd317c947591fe42b5ca9ab317b` (15.1) | [mount.c](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sbin/mount/mount.c) | `755d3571712d1e852594ae5c1c187a99e1cce590670e284cf5890f330450c41a` |
| Same 15.1 revision | [pstat.c / swapinfo](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/usr.sbin/pstat/pstat.c) | `66551b72d36c410f23839eba2db8b3322774b42ef77bc4c8396b31e8a641bd75` |

Base manuals were reviewed for [mount](https://man.freebsd.org/cgi/man.cgi?query=mount&sektion=8&format=html),
[swapinfo](https://man.freebsd.org/cgi/man.cgi?query=swapinfo&sektion=8&format=html),
[newfs](https://man.freebsd.org/cgi/man.cgi?query=newfs&sektion=8&format=html) and
[swapon/swapoff](https://man.freebsd.org/cgi/man.cgi?query=swapon&sektion=8&format=html).
Record eleven executed utility hashes, actual guest/toolchain/image/runtime versions
and exact functional/test inputs with the native outcomes. Export only sanitized
assertion/version/hash records, never raw system inventory or credentials.
GitHub Actions remain disabled; all qualification runs locally.

Additional inspected name-buffer headers:

| Release | Source | SHA-256 |
|---|---|---|
| 14.5 | [mount.h](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/sys/mount.h) | `c34d3ddd015eca39f29b6b989f24a707b9398808feb5c60b7fc96a30ed53b3d3` |
| 14.5 | [kvm.h](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/lib/libkvm/kvm.h) | `0d7fb9f22aafef522ef0d636617097d2003db2e97cdadca62a9e9306140e696d` |
| 15.1 | [mount.h](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/sys/mount.h) | `2bc2017d63389c39dfee52b3041970450ef1eebaa149c8fe5e84a871c0f0b738` |
| 15.1 | [kvm.h](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/lib/libkvm/kvm.h) | `5c3aad8e62f58923c16f4729c7882209ad991a1f2e06d75f4fdae655b5ccc7ec` |
