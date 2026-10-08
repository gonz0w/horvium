# M2 separate native claim qualification fixture

Date: 2026-10-08. Status: complete fixture qualification; maintainer review pending.
Authority: [approved Test Design](../../milestones/M2/TDD.md),
[EDD](../../milestones/M2/EDD.md), [issue 19](https://github.com/gonz0w/horvium/issues/19).
This is fixture/checker maintenance for the existing read-only observer contract;
no product behavior or execution authority changes.

## Problem and bounded fixture layout

PR 84 merged the original claim observer with full pool gates outstanding. Its
new eight native children ran inside the historical pool collector's five-minute
context. Two original attempts timed out during old tests. A retry at merged
source `41140a8061267a000bd321b5c5f4f536954cfb25` with no unrelated QEMU guests observed
at start also exceeded that unchanged deadline, during the old control group.
The new claim group passed in 20.84 seconds, and all postcondition cases passed.
That attempt remains unqualified. Host contention cannot be assumed as the sole
cause; fixture accumulation consumes the existing deadline budget.

Keep `TestNativePoolPropertyCollection`, all 88 named assertions and every old
subgroup; explicitly amend its aggregate context to eight minutes as recorded
below. Remove only the new claim call from its system fixture. `TestNativePropertyClaimCollection` has its own 135-second
context and a fresh root-excluded enrolled mirror supplied by the same declared
five-disk setup. It returns from setup before the old enrollment/system scenarios.
The historical setup path retains its original operations and assertions.
Both actual native parents are selected by the existing checksum-bound prebuilt
suite and run serially, without parallel fixture use or new opt-in flags.

The claim fixture retains all eight actual backing/queued/applying/foreign/closed/
reopened/preservation/teardown children unchanged. After it destroys the independent
metadata pool, the outer GUID-bound data mirror and private lease/directory must
also be removed. Actual mount and swap presentations must match their original
values. The new parent emits exactly one `CLAIM_PROPERTY_COLLECTION_RESULT
fixture_teardown=pass` only after this check. Guest root backing settings remain
unchanged. Every storage device is one of the existing disposable snapshots.

## Complete qualification and test-first checks

The host checker still requires the original once-only collector parent and all
88 exact assertions. It now also requires the new once-only claim parent, group,
all eight exact child names, and outer teardown result. Missing, unknown, repeated,
failed or skipped children and missing/failed/repeated teardown reject. Nested
failure/skip lines are denied. Host backing/base/runtime/group cleanup and
checksum-bound artifacts remain separate mandatory checks.

Two failing behavioral tooling tests showed that the old checker accepted legacy
output without claim execution and ignored absent/repeated/failed/skipped new
children. After implementation, the checker and local runner regressions pass.
These synthetic logs test the validator; they are not native evidence. The runner
tests use a complete synthetic fixture record and still require host device
teardown. The selector test binds both actual native parents and the retained
eight-minute/new 135-second contexts.

Require both pinned disposable FreeBSD releases, run serially without overlapping
portable deep review. Because shared qualification tools change, the existing review
planner requires its full portable mode; this is a verification-input change,
not an expansion of product scope. Preserve all old manifests and product bytes.
No historical evidence is relabeled, and this change does not accept M2 or a
runtime backend. Earlier state-reader/race/fuzz results retain their own sources.

```sh
PR_TITLE='test(storage): isolate native claim qualification fixtures' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

Actual native results, artifact/report hashes, the unqualified five-minute retries
and complete owned teardown are recorded in the
[manifest](property-claim-fixtures-manifest.json). Root backend/eligibility, executor/recovery clearance, interruptions, reservation release
and authenticated operator workflows remain open.

## Aggregate fixture budget amendment

The first separated-fixture attempt at source
`2e62e58e08e21a5b749e814464ff9668a9610711` passed the independent claim parent in
27.05 seconds. The historical collector still hit its five-minute deadline during
the control group and remains unqualified. Isolation alone does not resolve the
accumulated scenario budget. Increase that aggregate lab-fixture context to eight
minutes, explicitly changing the old aggregate limit. Retain every old assertion,
subgroup and failure/cleanup check. The new parent still has 135 seconds; production
native observations still have 135 seconds and property commands ten seconds.
There is no change to runtime safety timeouts or a NAS performance/support claim.
Both full parent results are mandatory on each pinned release, with serial guests
and fresh source-bound artifacts. Record the failed five-minute retry separately.

## Source-bound qualification

Signed source `03b7dce3f70ea13d972febba8702536785c78fb0` passed mandatory `make check lint test secrets`
with 124 tooling tests, all 11 portable race packages and
all 44 bounded fuzz targets. Shared qualification tooling required
a freshly executed full portable review; subsequent eligible prose/evidence uses
its exact verification inputs under the existing reuse policy.

Both serial pool runs passed every original 88 assertion, all old postcondition,
assessment, job-source and control/descendant-guard cases, backing-appropriate
enrollment-publication checks, and all eight actual claim children plus outer teardown. The old aggregate parents
completed in 316.89/289.67 seconds on 15.1/14.5, within the explicitly amended
480-second fixture limit; independent claim parents took
27.21/23.75 seconds within their 135-second limits. Identical
checksum-bound artifacts ran on both pinned guests. Root metadata publication
retains ZFS success on 15.1 and softupdates UFS refusal on 14.5; the six positive
publication subcases apply to ZFS only. Independent claim/control metadata uses
its explicitly owned ZFS backing on both guests. These are observed fixture
timings, not a NAS performance or hardware support claim.

All guest pools/private state, snapshot devices, keys/images/firmware, test
executables and owned process groups were removed or verified stopped; base and
runtime hashes were preserved. Actions remain disabled, with no installed host
service. Product Go bytes, dependencies and all 60 historical M2 manifests remain
byte-identical. Earlier state-reader evidence keeps its original source; unchanged
state fixtures were not rerun. The quiet merged-source failure and first separated
five-minute failure remain unqualified, with complete teardown and actual sources.
See the [manifest](property-claim-fixtures-manifest.json).

This closes the previously pending pool gates for the read-only original-claim
observer subset. It does not accept a full backend, M2 or a release. Runtime
eligibility/writes, executor/recovery clearance, interruption/reboot tests, terminal
resolution/reservation release and authenticated operator workflows remain open.
