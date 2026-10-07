# M2 native system-use exclusions for property review

Date: 2026-10-06. Status: accepted in PR 63 at merged source `12ecb173e48099040f4f52662d3e1fba04de5dc6`.
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
The composed path reads canonical root policy once more after the system phase;
changed or missing policy denies the whole result. Cancellation at that final
read also exposes zero. Its equality check retains the existing non-atomic limits.
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

## Executed qualification

Signed source `c0f26eeef6e09e955f1edeb8d6fe15c59481b3cb`, based on PR 62 merge
`aebae03ab68dfe49490a1fc68537bb319e5b70ef`, passed all 32 assertions on
FreeBSD 15.1-RELEASE-p3 (ZFS system backing) and 14.5-RELEASE (UFS system backing),
with the same checksum-bound executable. Complete guest workloads took
73.607/71.764 seconds including setup/verification/teardown; these are lab times,
not NAS performance results. See the [sanitized execution manifest](property-system-exclusions-manifest.json).

The complete portable review passed `make check lint test secrets`, 111 tooling
tests, all 11 race packages and all 33 compiler-listed portable fuzz targets in
178.335 seconds. Both actual fixtures destroyed their declared pools, detached md
providers, removed the owned mounts/swap, closed root leases and removed private
metadata. Guest/image/key/variables/executable cleanup and process-group teardown
passed. All five snapshot backing files remained byte-identical and were removed
after guest stop. `make lab-status` verified all 288 owned sessions stopped at
this checkpoint; no services or Actions jobs were enabled. All 38 prior M2
manifests and native storage metadata sources remain byte-identical to the base.

The first 15.1 attempt at signed source `656ca0babab4a58f96884b51f372c8432c5918f6`
failed before fixture storage mutation: its independent root capture incorrectly
assumed UFS-only mounts, omitting the guest's mounted ZFS pool. It remains
unqualified with complete guest/host/device teardown and unchanged backing files.
A test-only correction resolves the actual mounted pools. Its superseded portable
review was stopped through the owned supervisor before editing and remains
unqualified. A separate failing-first regression then closed the policy window
introduced by the system phase with a final private policy read. Original failures
retain their source IDs and log/report hashes. Documentation followups rerun the
baseline and may reuse the exact successful deep evidence; runtime eligibility
and admission remain disabled.
