# M2 native system-use exclusions for property review

Date: 2026-10-06. Status: implementing; qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [issue 19](https://github.com/gonz0w/horvium/issues/19).

`freebsd.SystemUseSource` collects complete system backing through fixed guarded
base reads. It accepts no request addresses. Each of two independent passes reads
`freebsd-version -kru`, `sysctl -b kern.geom.confxml`, `mount -p`, `swapinfo -k` and
`zpool list -H -p -o name,guid,size,alloc,free,health`. For every observed imported
pool it validates the name/GUID before issuing fixed independent GUID-value and
paired `zpool status -g -p` / `zpool status -L -P -p` reads. Unsupported/mixed
release formats, missing status, uncertain backing or failed/truncated projections
return a zero view. Names come from live native enumeration, never a client.

`storage.ParsePoolSystemUse` joins all imported pools and the exact mounted subset
to the existing GEOM/mount/swap parsers, retaining topology for comparison with the
selected property observation. Its fields are private; getters copy nested disk
identifiers. Parsed facts require native completeness/freshness, and grant no
enrollment or unused-device authority. Repeated normalized views and release lines
must match. Changes in observed root backing, pool membership/topology or swap use
refuse collection; busy swap can conservatively require a fresh review. GEOM IDs
remain local to the observed graph, never persisted device identities.

`PropertySafetySource` obtains the accepted private policy/property observation,
then obtains this system view. `CheckPropertySystemExclusions` requires the same
selected pool GUID/name/health and exact native topology. It denies a selected pool
mounted at `/`, `/boot`, `/etc`, `/usr`, `/var`, `/root`, or below those non-root
system directories; these are conservative capability exclusions, not configurable
mount namespaces. It also denies physical-disk overlap with other mounts, swap or
other imported pools. A pool's own ordinary data mounts remain permissible.
Unsupported logical/md backing denies the new safety path; existing read-only
collectors retain their accepted observation behavior without a bypass.

Every error returns zero policy/observation. Successful composition still sets
`Eligible=false`: full provider/enrollment proof, store/recovery validation, atomic
admission, authorized transport and actual mutation execution remain open. These
sources have no runtime consumer, new CLI/HTTP operation or record/config schema.
Policy reads and native views are sequential observations, not an atomic ZFS
snapshot or root fence; equal views cannot detect intervening changes or prevent
later root mutations. Eventual admission must independently revalidate all gates.

Each complete system collection shares 64 MiB combined stdout/stderr and 100,000
raw lines across both passes, with 100,000 parsed-record allowance split between
passes and their four parsing stages. Existing field/XML/per-parser bounds remain.
Commands have ten seconds and a system collection 45 seconds. The composed path
has a 90-second context, with separate bounded property/system budgets; it is not
one 64 MiB budget shared across the two collectors. Synchronous root policy I/O
retains the existing lease/reader behavior. These are bounded capabilities, not
hardware-specific assumptions or commercial scale/performance evidence.

## Qualification procedure

Failing-first tests cover root/system/swap/foreign-pool overlap, permitted own data
mounts, complete topology binding, zero proof, independent copies, aggregate
ceilings, unsafe enumerated names, failed projections, complete imported pools,
valid normalized drift and zero partial results. Two bounded fuzz targets exercise
hostile collection and joined parse inputs.

The focused `pool-properties` suite now attaches the existing five explicitly
declared snapshot-backed virtual disks. Guest setup verifies serial provenance,
geometry and disjoint root/mount/swap backing before any fixture mutation. The two
unique devices form a declared mirror; one separately declared duplicate-serial
disk supplies positive swap-use evidence only. It never qualifies creation identity.
Native reads must show actual root and pool backing, private policy binding without
authority, an ordinary data mount, denial after moving only the owned dataset to
an owned directory beneath `/root`, fresh restoration, active disjoint swap and
preserved sentinel data. The earlier md fixture must now deny complete system-use
resolution. No boot disk, operator storage, existing pool or service is modified.

Retain all 22 earlier assertions and require ten additional assertions (32 total)
in the actual checksum-bound native test on both targets. Direct pool/swap/mount/
private metadata/lease teardown happens before success evidence. The runner stops
the guest, verifies unchanged snapshot backing files and removes the five owned
files before appending its mandatory single device-cleanup marker. Missing,
duplicate or failed cleanup cannot qualify. The existing supervisor already removes
this owned device directory on interruption; no daemon/Actions runner is added.
Prior state/device/collector manifests remain at their original tested sources.

```sh
make test-focus PKG=./internal/storage TEST='^Test(PropertySystem|PoolSystemUse)'
make test-focus PKG=./internal/host/freebsd TEST='^Test(PropertySafety|SystemUseCollection)'
PR_TITLE='feat(storage): exclude native system backing from property review' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

Shared runner/evidence changes require one complete portable deep review. Daily
native qualification remains the focused collector fixture, without rerunning
the unrelated 777-assertion metadata publication suite.
