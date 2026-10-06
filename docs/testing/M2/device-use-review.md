# M2 combined device-use review

Date: 2026-10-05. Status: all fifty-one assertions, four executed native tests
and complete preservation/teardown passed on both targets; accepted in PR 36.
[Approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [issue 19](https://github.com/gonz0w/horvium/issues/19).

## Observation and comparison contract

`storage.ReviewDeviceUse` combines the accepted [disk correspondence](disk-correlation.md),
[system use](system-device-use.md), [imported-pool use](imported-pool-use.md) and
[filesystem signature](filesystem-signatures.md) contracts. All parsers receive the
same complete GEOM capture. Independent diskinfo/serial captures select a direct
uniquely identified disk. The supplied catalogue/status captures cover exactly
all imported pools; the mounted-pool subset is selected internally from parsed
mount rows. Missing/extra/contradictory/unsupported sources return a zero review.
No new native grammar, executable or storage mutation is introduced.

Retain only mounts, swaps and imported pools backed by the selected disk, with
their full backing disk sets and native pool identity/health. A supplied probe
must recognize qualified positive UFS/ZFS metadata on that exact disk. Nil probe
evidence sets `SignatureAvailable=false`; it cannot imply absent metadata. Empty
positive-use sets cannot imply an unused disk. Unknown/failed/unsupported probe
output never becomes a successful absent/blank state. Recognition is not checksum
integrity, all-label coverage or exported-pool identity. Multiple positive metadata
facts are retained rather than assuming that a recognized type excludes remnants
of other formats.

The caller must establish full successful execution, fresh capture, exact target,
provenance and exclusive input ownership during the call. Syntax cannot prove
that rows were removed or reads were atomic. Inputs are internal trusted future
helper observations, not a client request, transport or persisted schema.

The private `DeviceUseReview` stores an immutable baseline; `Observation` returns
independent nested arrays. `Validate` reparses a full fresh capture set and requires
new review when selected address, stable identifier/provenance, actual geometry,
mount/swap facts, pool GUID/health/full backing membership or signature type/
availability differ. Retain original GEOM IDs for observation, but exclude their
transient values from comparisons; normalize identifier ordering and use the
existing parsers' canonical row/backing ordering. Valid changes confined to other
devices do not alter the selected facts; unavailable global captures still refuse.
An uninitialized baseline cannot validate. Failed validation exposes no new or
partial review. Swap values remain KiB/native observations, without inferred
physical geometry or capacity policy.

Combined bytes include GEOM, mount, swap, catalogue, diskinfo, serial, every pool
name/GUID/paired status and optional probe address/output. Check the aggregate cap
before parsing. Divide `MaxRecords` conservatively into disjoint equal quotas for
mount preparse, disk correspondence, system use, imported-pool use, and an optional
signature stage. Reserve two scalar records in the disk quota. Divide
`MaxXMLElements` among the graph-reading stages. This counts repeated parsing;
each stage refuses its quota rather than borrowing another stage's remainder.
Field/XML-depth/inherited status bounds remain enforced. These are conservative
capabilities, not precise allocator/RSS or commercial scale qualification.

This is an observation baseline, not an operation plan: no session/expiry, policy,
enrollment, reservation, durable single-use submission or launch occurs. Unchanged
facts are not write clearance or a freshness/exclusive-writer proof. Full partition/
signature absence, exported-pool membership, raw-open use, shared-host/locality
fencing, reservations and last-moment execution remain separate. Mount options and complete vdev
GUID/scan/import-incarnation/property drift needs its other qualified projections;
this record does not replace them. Runtime/helper/CLI/HTTP/planner wiring and all
storage mutation authority remain absent.

## Tests and disposable native procedure

Tests preceded the API; the initial compile failed because the types/functions
were absent. Behavioral assertions cover selected mount/swap/pool/signature joins,
root/whole-pool backing, empty/unavailable facts, same-path identity/geometry reuse,
address changes, new use, pool GUID/other-member drift, source order/transient IDs,
unrelated swap changes, missing/unsupported captures, combined bounds and input/
nested-output isolation. `FuzzDeviceUseReview` is the eighteenth local review target;
it mutates every raw capture/address and probe presence, and requires zero-on-error
or a stable independently identified reparse. A separate failing-first evidence
regression refuses missing/duplicate/unknown/failed native join/drift cases.

Reuse the [five owned snapshot disks](devices.md) and existing guarded GPT, pool,
UFS and swap fixtures. No additional fixture writes are introduced. All existing
identity/geometry/root exclusion, foreign preservation, probe hashes and cleanup
controls still run. Only read-only collection and pure comparison controls are added.

`TestNativeImportedPoolUse` builds a no-probe baseline with no selected positive
use facts and explicitly unavailable signatures. After the existing owned-pool
creation, require selected unmounted-pool/ZFS joins and new-use drift. Independently
recollect actual pool/GEOM/probe/system captures and validate the unchanged baseline.
After guarded owned-pool destruction, require residual ZFS recognition with no
imported membership and pool-removal drift. Omit only the owned pool's status in
a copy of actual captures and require a zero combined review. Hash the closed
modified fixture across the probes/comparisons, retaining the prior whole-byte
preservation proof. Emit only after all existing preservation and cleanup controls:

- `DEVICE_USE_POOL_RESULT`: `imported_join`, `new_pool_drift`, `unchanged`,
  `residual_join`, `pool_removal_drift`, `missing_capture_denial`, `no_clearance`.

`TestNativeSystemDeviceUse` independently captures the complete imported catalogue
and statuses for each combined view. Require the existing actual read-only UFS
mount/signature and active swap to join their exact owned disks. Both new uses
invalidate their pre-write baselines. Change only a returned nested array and
verify retained facts remain intact. After owned mount/swap cleanup, require
the no-probe positive-use baselines to match and active baselines to refuse their
removed uses. Restored empty positive facts are not signature clearance; modified
guest metadata remains. Emit only after all preservation and cleanup controls:

- `DEVICE_USE_SYSTEM_RESULT`: `mount_join`, `swap_join`, `mount_drift`, `swap_drift`,
  `copy_isolation`, `use_cleanup`, `mount_removal_drift`, `swap_removal_drift`.

Four actually executed tests, all fifty-one assertions and full owned host cleanup
are required. A guest-only pass is insufficient. Prior manifests retain their
original source/case sets; no new controller-death or physical support is claimed.
Run sequentially from clean signed source:

```sh
python3.14 tools/local-qualification.py devices 15.1 --run
python3.14 tools/local-qualification.py devices 14.5 --run
make lab-status
```

Discard each guest after success/failure, preserve all five original host backings,
base image/runtime, remove disks/image/key/variables and stop the owned group.
Capture the existing twelve utility hashes and actual versions. Export only
sanitized hashes/assertions, never raw inventories or credentials. GitHub Actions
remain disabled, with no persistent runner or host service.

## Executed evidence

Signed source `9dd27d946cb726da1b0a5215b6392b1524aaff76` passed all fifty-one
assertions and four native tests on FreeBSD 15.1-RELEASE-p3 and 14.5-RELEASE
through Mac arm64 / QEMU 11.1.1 TCG (two CPUs, 6144 MiB). Actual roots were ZFS
and UFS, respectively. Independently recollected unchanged imported-pool facts
revalidated; new and removed owned pool/mount/swap use invalidated their baselines.
Residual ZFS metadata remained positive after owned-pool removal. Selected UFS
mount/signature and swap joined exact independently identified disks; no-probe
facts remained unavailable, missing status refused and returned nested arrays
could not alter retained observations. Existing read/foreign/root/catalogue/
geometry preservation controls and complete teardown passed on both releases.

All five original host backings, base images and runtime inputs were preserved;
owned disks/image/key/variables were removed and groups stopped. The
[sanitized execution manifest](device-use-review-manifest.json) binds source/input
hashes, actual versions, twelve utility hashes per guest, reused primary-source
manifest hashes and outcomes. Functional-source review passed 65 tooling tests,
Go unit/race checks and eighteen bounded fuzz targets. Final PR-head review is
reported separately; any recorded input change requires fresh qualification.
All prior manifests and controller-death evidence retain their original sources
and case sets. This is partial M2 observation/drift evidence, not device clearance.

## Provenance and acceptance limits

The unchanged accepted disk/system/imported/signature manifests retain inspected
producer/manual source hashes for their exact grammars. This composition adds no
new native source contract. Record their file hashes as reused provenance alongside
new exact functional/test/tool inputs, executed utility hashes and native outcomes.
Source inspection does not prove installed binary build provenance.

The native manifest records actual outcomes and exact tested input hashes. This
is partial
M2-DEVICE-001/M2-DRIFT-001/M2-PLAN-001/M2-NATIVE-001 evidence only. Complete device
eligibility, slice 1, later mutation/recovery slices and M2 acceptance remain open.
The maintainer-deferred M1 Safari/manual accessibility gate remains pending.
