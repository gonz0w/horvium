# M2 positive pool-label identity

Date: 2026-10-05. Status: implementation and portable behavioral tests complete;
new native export/reimport qualification pending. [Approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [issue 19](https://github.com/gonz0w/horvium/issues/19).

## Observation contract

`storage.ParsePoolLabels` binds complete successful base `zdb -l` stdout from an
exact absolute direct-disk argument to independent diskinfo/serial/GEOM identity
and geometry. It performs no I/O. The caller establishes execution success,
freshness, target, provenance and exclusive input ownership. This is an internal
observation contract, not a transport, persisted schema or write policy.

Default `-l` groups identical configurations. Each printed block must have the
exact separator/header/body/group format, a valid representative label, bounded
indented native scalar/container rows and a complete final newline. The union of
all groups must represent indices 0 through 3 once, in producer order. Refuse
reported bad representative checksums, read/unpack diagnostics, missing groups,
unknown top-level fields/version/state, controls, malformed indentation, duplicate
keys, overflow and contradictory pool/leaf/top identities or states. Retain each
configuration's pool name/GUID, top/leaf GUID, transaction group, native state,
label indices and SHA-256 of its complete body, including opaque nested metadata.
Nested rows are syntax-bounded; they do not establish complete vdev topology or
validate historical embedded device paths. Never execute an embedded path.

The qualified native pool states are active (0), exported (1) and destroyed (2),
for positive member metadata only. They describe recorded label state, not current
import status, ownership, usability or health. Empty, failed, unavailable and
unsupported captures return no observation; they never establish absence. Changed
identities/states among represented configurations refuse the entire capture.
Distinct transaction groups with consistent identity/state remain visible.

The inspected producer prints checksum status only for each configuration group's
representative. Even complete group coverage does not prove each member's checksum,
all signature absence, complete pool membership or safe reuse. There is no force,
label-clear, wipe or product import operation. Other signature/partition/raw-open/
locality/reservation/last-moment gates remain required for device operations.

`ReviewDeviceUse` accepts optional pool-label captures. Missing evidence is
explicitly unavailable. Provided labels must correlate to the same selected disk;
retain all configurations in the private baseline and return independent nested
copies. New/missing labels, pool identity/state/transaction group/configuration or
label coverage changes invalidate the baseline. Imported membership and recorded
labels remain separate facts; exported and destroyed metadata cannot disappear
merely because no pool is imported. Configuration formatting changes conservatively
require review. Review still supplies no operation/session/expiry/enrollment or
launch authority. The runtime remains read-only and this parser is not wired into
helper/HTTP/CLI/native collection.

Aggregate bytes include all raw captures and exact probe argument. Count every
label header/body/group row plus independent diskinfo/serial against the shared
record limit before graph parsing; enforce field and nesting depth bounds. Combined
review gives label parsing a disjoint conservative record/XML quota just like
its other stages. Bounds are refusal contracts, not measured scale/RSS guarantees.

## Test-first and native procedure

Behavioral tests preceded implementation and failed to compile on the absent API.
They cover active/exported/destroyed positive metadata, full coverage, distinct
transaction groups, conflicting identities, bad-checksum diagnostics, truncation,
unknown state/version, wrong argument, missing/duplicate fields, controls, malformed
nesting, aggregate limits, immutable copies and export/unavailable drift.
`FuzzPoolLabels` adds the nineteenth bounded fuzz target. Evidence regression first
failed because the checker accepted missing/failed/duplicate/invented label cases;
it now requires the complete new native marker in addition to prior assertions.

Extend only `TestNativeImportedPoolUse` and its existing independently verified
unique 4K owned snapshot disk. Create the same random unmounted fixture pool,
observe active labels and compare its independently collected catalogue GUID.
Export that exact owned pool without force, require it to disappear from the
imported catalogue while its labels retain the same GUID and exported state.
Bind those facts in combined review and require actual export to invalidate the
active baseline. Change only capture copies for wrong-address/unavailable denial.
Hash the entire closed exported disk across read-only probes/comparisons and
require identical bytes. Reimport only the already recorded GUID using `-N -d`
with that exact owned disk and no force; independently verify original membership,
GUID and active label state before the existing guarded destruction/cleanup.
Cleanup never guesses or retries an import; an interrupted exported fixture
remains unqualified and its disposable guest is discarded.

Emit `DEVICE_POOL_LABEL_RESULT` only after all prior foreign/root/device and
cleanup controls pass: `active_identity`, `exported_identity`, `imported_absence`,
`export_drift`, `probe_preservation`, `binding_denial`, `reimport_identity`,
`unavailable_denial`. Eight additional assertions extend the four-test suite from
51 to 59. Both release candidates and complete host teardown are mandatory.
Existing manifests remain immutable, bound to their actual tested source/case sets.
The fixture export/reimport privilege is not product mutation authority.

The first 15.1 attempt at source `57124972f94265fde2082e7b170b50cd9be938ab`
stopped during utility hashing because the fixture used `/sbin/zdb` instead of
the installed `/usr/sbin/zdb`. No native tests or fixture storage writes executed.
Host backings/base/runtime were preserved, all owned files removed and the group
stopped. This attempt remains unqualified; the repaired fixture is rerun fully.

Run sequentially from clean signed source through the Mac supervisor:

```sh
python3.14 tools/local-qualification.py devices 15.1 --run
python3.14 tools/local-qualification.py devices 14.5 --run
make lab-status
```

Record actual kernel/userland/ZFS/toolchain strings, thirteen utility hashes
(including zdb), functional/test/tool source hashes, native assertions and host
base/runtime/backing preservation plus disks/image/key/variables/group teardown.
GitHub Actions remain disabled; no persistent runner or host service is installed.

## Source provenance and acceptance

Pinned FreeBSD producer/manual/nvlist-printing/native-state sources are inspected
for both bundled release revisions. Hash records are recorded with executed
qualification; source inspection does not establish installed binary provenance.
This subset is partial M2-DEVICE-001/M2-DRIFT-001/M2-NATIVE-001 evidence. Complete
slice 1 eligibility, later mutation/recovery slices and M2 acceptance remain open.
The maintainer-deferred M1 Safari/manual accessibility gate remains pending.
