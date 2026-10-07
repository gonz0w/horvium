# M2 read-only property store readiness

Date: 2026-10-07. Status: both native targets and complete portable review passed; maintainer review pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

`StateDirectory.CheckPropertyState` checks complete private metadata, actual
system-backing capability and retained recovery holds under the borrowed root
writer lease. `PropertyStateSource` inserts that check after native member/system
safety and before its final canonical policy reread/context check. Its success
still keeps `Eligible=false`. The existing `PropertySafetySource` retains its
accepted read-only observation scope. The composed source retains the 135-second context and checks cancellation after
state inspection. A context cannot preempt the lease mutex or a blocking filesystem
syscall; future helper containment remains required. Neither path implements a runtime backend,
provisioner, executor, transport or operator workflow.

## Contract

Require effective kernel UID 0, the held validated directory/lock, valid bounded
member-bearing root policy and initialized complete native system-use observations.
Under the lease mutex, deny previously latched publication uncertainty. Inspect
the entire bounded ownership/job namespace with existing descriptor/ACL/identity,
strict codec, count/byte, duplicate submission/plan and final namespace checks.
Bind the selected canonical ownership policy, including members, against the
caller policy. Missing policy denies; changed policy is drift. Other legacy
ownership records remain readable; their presence does not grant enrollment.
No partial catalog or observation escapes failure.

Kernel descriptor `fstatfs` captures bounded filesystem/source/mountpoint names,
filesystem ID and mount flags before and after a successful check. Ignore volatile
free-block counters. Require writable ZFS or UFS without soft updates. Match the
actual backing exactly to the fresh complete system mount/provider view, with
nonempty physical backing. Initial accepted mountpoint policy covers `/`, `/boot`, `/etc`,
`/usr`, `/var`, `/root`, and descendants of the non-root system namespaces.
A directory on the root filesystem can live elsewhere; its actual mountpoint is
still `/`. Separate arbitrary data mounts refuse this prerequisite. Any enrolled
imported pool or physical overlap with an enrolled imported pool also refuses.
These are capability/namespace limits, not controller or hardware assumptions.
No mount settings, filesystem flags or provisioned paths are changed.

Every retained nonterminal job globally suspends the check until independent
executor/recovery validation exists. Every retained job on the selected pool,
including succeeded/failed/cancelled labels, keeps that pool held. Terminal jobs
on other pools remain retained and do not alone deny this metadata prerequisite;
that does not prove executor quiescence or release any reservation. This check
claims no job, releases no hold and performs no repair, pruning, replay or write.
Recognized orphan temporaries return uncertainty and latch the lease's existing
publication block. Removing one externally cannot clear that block. Read-only
inspection remains available; reopening alone does not validate restored metadata.
Non-FreeBSD returns unavailable without filesystem work. Private reduced-limit
and after-read seams are disposable test fixtures, never request/runtime controls.

Sequential metadata/native views and local advisory locks cannot fence another
root actor. Actual restored-state/rollback validation, executor boot/process
quiescence, live atomic admission/capacity/durability and authorization remain
required before write eligibility. An empty valid store and checksum are not
recovery authority. Enrollment provisioning/adoption remains pending.

## Qualification

Failing-first portable tests cover unsupported hosts, observed backing/capability
and source/target mismatch, system namespace restrictions, owned/shared backing,
all selected-pool lifecycle holds and global pending work. The composed regression
requires its trusted check, complete bound inputs and final policy/context checks;
all failures return zero. The historical state suite's exact test inventory stays
unchanged; enrollment/store-specific tests run in ordinary portable review.

The focused native parent retains 46 cases and adds ten: `state_backing_gate`,
`state_composed_gate`, `state_policy_drift_denied`, `state_unknown_denied`,
`state_job_hold_denied`, `state_terminal_hold_denied`, `state_orphan_denied`,
`state_uncertainty_latched`, `state_preserved`, `state_missing_policy_denied`.
Use only its previously declared owned mirror, root lease and private records.
Require actual ZFS success or softupdates UFS refusal according to kernel backing,
with a sanitized backing/result line. Independently change selected-policy input,
remove/recreate the owned policy, introduce a preserved unknown file, record queued
and cancelled jobs, and add/remove an owned orphan. Require exact refusal reasons,
unchanged records and the composed source's latched uncertainty. These are test
setup changes, not a product enrollment/recovery path. Existing GUID-checked
pool/swap/md/private cleanup and host/guest/device/image/key/binary/process teardown
remain mandatory. No extra disk or host service is introduced.

```sh
make test-focus PKG=./internal/storage TEST='^TestPropertyStore'
make test-focus PKG=./internal/host/freebsd TEST='^TestProperty(Store|Safety)'
PR_TITLE='feat(storage): check native property store readiness' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

All 56 exact cases, actual executed PASS and complete teardown must pass on both
releases. The shared checker requires one complete portable race/fuzz review.
Unchanged metadata file engines/crash/ACL/capacity evidence retain their original
sources; this focused run does not requalify the historical 777-case state suite.
GitHub Actions stay disabled. No production, hardware, populated commercial scale,
throughput, full recovery or M2 acceptance follows.

## Executed qualification

Signed source `231b8f0f2c250fcbfc900d95d17ed94de6a24cbb`, based on PR 65 merge
`e31bbfbff94c083756ba8cf7974dd3b66c2bbf2a`, passed all 56 exact focused assertions
on FreeBSD 15.1-RELEASE-p3 and 14.5-RELEASE with the same checksum-bound executable.
Kernel statfs observed actual root-backed ZFS store-check success on 15.1 and
root-backed softupdates UFS refusal on 14.5. Separate system mountpoints and
non-softupdates UFS success are policy targets with no new native pass here. Both verified the selected-policy/namespace/retained-job/orphan
refusals, uncertainty latch and unchanged metadata. Guest workloads including
setup/teardown took 81.846/79.402 seconds; actual tests took 49.88/47.01 seconds.
These are Mac/QEMU fixture measurements, not NAS throughput or hardware qualification.
See the [sanitized execution manifest](property-store-readiness-manifest.json).

Complete portable review passed `make check lint test secrets`, 112 tooling tests,
all 11 race packages and all 35 compiler-listed bounded fuzz targets in 184.021
seconds. All 41 historical M2 manifests and pre-existing native metadata file
engines remain byte-identical to the base. The earlier 777-case state-suite proof
retains its original source and was not rerun or relabeled. New state checks are
qualified by the focused fixture; existing durability/ACL/capacity behavior is
unchanged.

Every owned pool/mount/swap/md/private record and guest/image/key/variables/
executable/device fixture was removed. Snapshot/base/runtime inputs were preserved;
`make lab-status` verified all 343 owned sessions stopped at this checkpoint.
No Actions, runner or host service was enabled. Documentation followups rerun the
complete cheap baseline and may reuse only exact successful deep inputs.
Provisioning/adoption, full restored-state/executor validation, atomic admission,
authorized transport/execution, operator workflows and M2 acceptance remain open.
