# M2 root-leased enrollment state binding

Date: 2026-10-07. Status: implementation; qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0006](../../adr/0006-storage-ownership-and-plans.md),
[ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

`StateDirectory.BindPropertyEnrollment` connects the accepted volatile review to
one held root lease, an explicit predecessor and the complete private metadata
baseline. It distinguishes explicitly requested first enrollment from missing,
unsafe or drifted stored state. The returned `PropertyEnrollmentState` is sealed
read-only evidence. It cannot publish policy, claim work, reset uncertainty,
release reservations or set write eligibility. No runtime consumer or public
transport is added; ownership formats and metadata file engines are unchanged.

## Contract

Repeat the plan's native/member/system, five-minute expiry and root-context checks.
Only an entirely zero predecessor explicitly requests an absent record and revision
one. Otherwise require a bounded canonical predecessor with the same pool GUID
and exact next revision. Read the selected record independently under the lease;
compare its complete canonical policy, including scopes/members, with that
predecessor. An explicit legacy predecessor can be reviewed for version 2 enrollment;
there is no automatic migration. Same-revision scope/member changes are drift.
A read failure cannot establish absence or authorize a first enrollment.

Require actual kernel effective UID 0, a validated held directory/lock and no
latched publication uncertainty. Inspect the entire mixed ownership/job namespace
with the existing strict descriptor, ACL, codec, byte/count, duplicate and final
namespace checks. Bind exact bytes and names of every record, including unrelated
policies/jobs, in a private length-framed SHA-256 baseline. Hashes are internal
comparison data, not authentication, exported inventory or a durable schema.
Malformed, missing, unsafe, unknown or over-budget state produces no binding.
Recognized orphan temporaries latch the existing lease's publication block; removing
one externally cannot clear it.

Project complete capacity for the proposed record while retaining every old
record and its whole temporary publication footprint. No bytes are reclaimed or
reserved by the check. Retained nonterminal work anywhere denies; every retained
job on the selected pool, including terminal labels, keeps its hold. Terminal jobs
on other pools remain in the baseline and are not alone a denial or proof of an
absent old executor. All holds/history remain unchanged.

Match kernel-observed writable ZFS or non-softupdates UFS system backing to the
fresh complete native mount/provider view. Preserve the accepted system-mount
namespace restriction. Include all existing owners and the proposed pool when
checking physical backing overlap, even before its first record exists. Compare
stable backing identity/capability fields again before returning a binding.
No mount or filesystem flags are changed. Actual softupdates UFS refuses binding.

The opaque result owns its predecessor and review copies and retains the original
lease. Fresh `Validate` repeats native/expiry/context checks and reads the entire
metadata namespace again through that lease, requiring the same complete record
and backing digest. Changes in an unrelated record also require new review.
Closed leases fail; copies cannot hydrate a new binding or transfer it to a new
lease. Repeated validation never grants a single-use publication claim.
Non-FreeBSD returns unavailable without filesystem work. Reduced-bound/after-read
seams remain private disposable fixtures, never runtime controls.

An unchanged empty catalog or checksum is not restored-state validation.
Old metadata restoration, executor boot/process quiescence, actual provisioning,
atomic single-claim admission/capacity/durability, final native/clock checks and
kernel-authorized operator transport remain separate gates. Context strings do not
authenticate OS root. The mutex/advisory lock cannot fence another root actor, and
blocking filesystem calls are not bounded by a request context. This prerequisite
neither changes ZFS nor enables the management service to enroll anything.

## Qualification

Failing-first portable tests cover explicit zero/legacy predecessors, partial-zero
and incorrect pool/revision/overflow refusal, owned canonical copies, projected
record/temporary capacity, unexpected existence/absence, proposed-owner backing
overlap, retained queued/terminal holds, unsupported backing, zero bindings and
unsupported hosts. Existing plan/collector/store regressions remain required.

Retain all 64 focused native assertions and add twelve: `enrollment_state_bound`,
`enrollment_state_absent`, `enrollment_state_predecessor_denied`,
`enrollment_state_record_drift`, `enrollment_state_unrelated_drift`,
`enrollment_state_unknown_denied`, `enrollment_state_job_denied`,
`enrollment_state_orphan_denied`, `enrollment_state_expired_denied`,
`enrollment_state_context_denied`, `enrollment_state_preserved`,
`enrollment_state_fixture_teardown`.
Use a separately owned private lease beneath the parent fixture's trusted root
directory and its existing independently collected mirror/provider/system inputs.
Require first/successor binding and revalidation on actual ZFS backing, or exact
softupdates UFS refusal with zero binding. On qualified backing, mutate an unrelated
record and require baseline drift. Softupdates UFS cannot construct a valid binding;
its control requires continued refusal and preserved records rather than claiming
a successful baseline revalidation. Independently introduce same-revision selected
policy drift, unknown files, queued/terminal selected-pool jobs and an orphan that
latches after removal. Require exact reasons and preserved fixture bytes. Remove
all private state and verify absence. Existing pool/content/metadata/native state
and guest/device/image/key/variables/executable/process teardown remain mandatory.
No extra disk or host service is introduced.

```sh
make test-focus PKG=./internal/storage TEST='^TestPropertyEnrollmentState'
PR_TITLE='feat(storage): bind enrollment reviews to root state' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

Require all 76 exact assertions and actual PASS on both releases. Shared checker
changes require complete portable review. Existing native metadata file engines
and historical manifests remain unchanged; do not rerun or relabel the original
777-case state suite. No production/hardware/populated scale, restored-state,
throughput, manual accessibility or complete M2 acceptance follows.
