# M2 explicit property enrollment review

Date: 2026-10-07. Status: both native targets and complete portable review passed; maintainer review pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0006](../../adr/0006-storage-ownership-and-plans.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

`BuildPropertyEnrollment` creates an immutable, expiring read-only review of an
explicit root proposal. The proposed policy must already select its exact pool,
individual filesystem/property scopes and every member's identity/provenance and
native geometry. Observation cannot supply missing selections. Selecting a child
alone does not select its parent, descendants or other properties. The complete
observed filesystem/version/property/state/feature/topology context remains
visible, including unselected resources. No ZFS or metadata write occurs.

## Contract

Inputs belong to a future trusted OS-root workflow during collection/build.
The context string only binds a volatile review; it cannot authenticate root or
establish authorization. There is no runtime consumer, transport, CLI provisioner,
UI route or durable enrollment schema. Existing version 1/2 ownership formats
remain unchanged. A policy read failure never means that an empty store or first
enrollment is safe.

Require complete ineligible pool observations, explicit valid scopes/members,
fresh direct-DISK member/geometry proof, complete system exclusions and independent
provider correspondence against the later system-use graph. Share that last check
with the existing native safety collector, retaining its behavior. Graph-local IDs
are excluded from independent provider comparisons; exact identity/provenance and
geometry are retained. Success leaves `Eligible=false` and cannot build an
executable property plan.

The private plan owns its proposal and full observation/system/imported-pool
review. Input and accessor mutations cannot change it. Stable sorting retains
native topology ordinals and all provider identity facts. Normalized full system
use and imported topologies bind the private baseline, including other pools and
positive mount/swap facts. Fresh validation requires the same explicit proposal
and complete independent native inputs; policy revision alone is insufficient.
Properties, scopes, members, incarnation, full dataset graph or system-use drift
requires a new review. Invalid or missing evidence also requires new review.
Sequential reads cannot fence a competing root actor or establish locality.

A random 256-bit identifier and five-minute lifetime bind the root context.
Validation before creation or at/after expiry denies. Repeated validation grants
no claim, reservation, publication or recovery authority. Plans are volatile and
cannot be hydrated from review copies. Input/structure costs are checked before
maps, copies, sorting or encoding, using a conservative 384 KiB working budget
including all native inputs. The complete generated review must fit an exact
64 KiB serialized ceiling. Over-budget contexts fail entirely, without paging,
truncation or granting a partial scope. These are safety budgets, not hardware
limits or evidence of populated commercial scale.

Root authorization and an actual provisioning workflow, current stored
predecessor/revision/capacity/backing checks, restored metadata/executor quiescence,
atomic single-claim durability and independent final native checks remain required
before publication. Existing metadata primitives are not enrollment permission.
The prior store check may remain uncertain even while this observational review
succeeds; the review neither clears uncertainty nor releases retained holds.

## Qualification

Failing-first behavioral tests cover explicit child-only selection, retained
unselected context, random IDs, expiry/context, zero results on failure,
incomplete/authoritative inputs, missing scopes/members, native/system mismatch,
system mount refusal, bounds, input/accessor isolation, canonical ordering and
fresh policy/member/property/incarnation/graph/system drift. Existing safety/store
regressions must retain their zero-result and final policy/context checks.
The existing bounded codec/collector fuzz targets remain applicable; the new
model uses fixed typed inputs and explicit hostile size/count tests.

The focused native fixture retains all 56 prior cases and adds eight:
`enrollment_review_bound`, `enrollment_review_expired`,
`enrollment_review_context_denied`, `enrollment_review_policy_drift`,
`enrollment_review_native_drift`, `enrollment_review_missing_scope`,
`enrollment_review_immutable`, `enrollment_review_preserved`.
Use only the declared owned mirror and private metadata/sentinel. Independently
recollect native properties, providers and full system use twice. Require fresh
validation, exact refusal classes, preserved complete native observations and
unchanged policy bytes/content. No new device or host service is introduced.
Existing GUID-checked pool/mount/swap/md cleanup and full guest/device/image/key/
variables/binary/process teardown remain mandatory.

```sh
make test-focus PKG=./internal/storage TEST='^TestPropertyEnrollment'
make test-focus PKG=./internal/host/freebsd TEST='^TestProperty(Safety|Store)'
PR_TITLE='feat(storage): review explicit property enrollment' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

Require all 64 exact assertions and actual executed PASS on both releases.
The shared evidence inventory change requires complete portable review. Prior
metadata file engines and historical manifests remain unchanged; the original
777-case state-suite qualification is not rerun or relabeled by this fixture.
Actions remain disabled. No production, hardware, throughput, full recovery,
M1 manual accessibility or complete M2 acceptance claim follows.


## Executed qualification

Signed source `8f6f0cef7f4321b56f37ae99f05c50696c8cc319`, based on PR 66 merge
`f2c2608d723874aab59415c9c30ae9619c7d3748`, passed all 64 exact focused assertions
on FreeBSD 15.1-RELEASE-p3 and 14.5-RELEASE with the same checksum-bound executable.
Both independently collected the complete pool/provider/system baseline twice,
validated the explicit proposal, refused expiry/context/policy/native/missing-scope
cases, retained immutable copies and preserved policy bytes, content and complete
native observations. The composed state gate still passed on actual ZFS root
backing and refused actual softupdates UFS root backing; review success does not
clear its retained uncertainty. Guest workloads including setup/teardown took
94.419/84.806 seconds; actual tests took 62.17/53.26 seconds. These are Mac/QEMU
fixture measurements, not NAS throughput or hardware qualification. See the
[sanitized execution manifest](property-enrollment-review-manifest.json).

Complete portable review passed `make check lint test secrets`, 112 tooling tests,
all 11 race packages and all 35 compiler-listed bounded fuzz targets in 185.317
seconds. All 42 historical M2 manifests and pre-existing native metadata file
engines remain byte-identical to the base. Version 1/2 ownership codecs and their
bytes are unchanged. The historical 777-case state suite retains its original
qualification and was not rerun or relabeled. Documentation followups rerun the
complete cheap baseline and may reuse only exact successful deep inputs.

All owned pools/mounts/swap/md/private records and guest/image/key/variables/
executable/device fixtures were removed, with snapshot/base/runtime inputs
preserved. `make lab-status` verified all 354 owned sessions stopped at this
checkpoint. No Actions, runner or host service was enabled. Runtime provisioning/
publication, full restored-state/executor validation, atomic admission, authorized
transport/execution, operator workflows and complete milestone acceptance remain
open.
