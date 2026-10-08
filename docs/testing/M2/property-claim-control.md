# M2 original-claim native property-control binding

Date: 2026-10-08. Status: qualification pending; runtime writes disabled.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0006](../../adr/0006-storage-ownership-and-plans.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-PROPERTY-001, M2-PRIV-001, M2-DRIFT-001 and M2-NATIVE-001 work.

## Exact applying lineage and fresh control facts

The private `newPropertyClaimControl` connects the existing fixed one-attempt
native control to the same original opaque root state claim and claim-bound
native observer. It has no public constructor or runtime consumer. The complete
original backend must supply a mandatory live-clearance callback for its original
plan/authenticated session/civil clock, recovered-state and global/pool launch
reservations, and own exactly one control object. This binding does not implement
that callback or prevent multiple independently constructed controls. Only declared
disposable fixtures install test-only clearance. Neither a callback returning nil
nor this fixture is complete native execution authority.

`VerifyPropertyApplyingAcknowledgement` is a separate root-only metadata check.
Under the original lease mutex, require the active claim, its successful private
post-sync applying hash/revision, an exact two-event applying record and a complete
strict mixed census including selected ownership and all other job holds. Reject
raw/visible-but-unacknowledged records, foreign/copied state handles, closed/reopened
leases, changed expected/durable records and any later or consumed successor attempt.
Claim copies share provenance; repeated verification consumes nothing. Orphan
uncertainty latches the existing publication block. This verifies canonical metadata
lineage only, without launch authority, native effect attribution or hold release.

The private factory owns the expected applying record, checks actual kernel UID
and metadata acknowledgement, and installs only methods from the original claim,
fixed native source/runner and actual clock. Authorization checks the entire typed
intent and acknowledged predecessor before fresh independent property/provider/
system/backing collection, matches the full returned record and baseline, then
repeats exact metadata/live/context checks after owning the reply. Read-only
observations must retain false eligibility. Missing adapters/clearance, cancellation,
drift or uncertainty return zero projections and perform no control invocation.

The existing control still consumes one attempt per object and rechecks root,
expiry, civil-clock order, applying timestamp, context, allowed arguments and
baseline before one guarded `/sbin/zfs set`. Its ten-second/4 KiB bound and descendant
quiescence semantics remain unchanged. Sequential observations/advisory locking
cannot fence a competing OS-root actor. No store schema, transport, constructor in
the runtime, scheduler, success publisher, replay, reservation release or recovery
resolution is introduced.

## Test-first and native qualification

Portable tests first require the absent acknowledgement/binding contracts. They
cover eleven private lineage cases and eighteen binding failures, exact expected
record/intent, fresh baseline, false eligibility, repeated live/metadata checks,
cancellation and zero outputs. Existing original-claim/control tests remain required.
An initial unexecuted FreeBSD cross-build caught a fixture referring to an absent
publication-fault type; it was corrected to the existing private phase hooks.
Unexecuted development binaries are removed and supply no native evidence.

The existing disposable state submission parent adds nineteen
`claim-acknowledgement` cases: exact/copy/repeated reads, queued/raw/lost applying,
foreign/copied/closed/reopened denials, expected/durable/ownership/hold/orphan drift,
observing/success/reconciliation and a failed ordinary successor attempt. Require
unchanged bytes/private lineage, stable lock inode, retained claims and complete
owned teardown. Actual non-root credentials must refuse the new method.

The independent native claim fixture retains its original eight direct children
and raw-applying observation. Its applying child additionally runs six nested
`control-binding` cases: raw acknowledgement denial, clearance denial, one exact
quota control/repeat denial, independent postcondition, retained applying/policy/
lock/lineage and changed-ledger denial. Use a separate private state subdirectory on
the already declared independent metadata ZFS pool, the enrolled data-pool GUID
and actual member/system/backing checks. Arm cleanup before effect; require initial
default quota, restore exact default value/source and verify the original full raw
projection/sentinel/mount/backing and owned teardown. Test-only intent capture and
live clearance remain explicit. The actual dropped-UID native control child must
also refuse the factory before input validation.

The first 15.1 pool attempt at signed source
`277afb1ef7e139d61ae930a720f064e92dc5ed86` passed the exact native control,
independent postcondition and applying preservation, but failed quota restoration
and the enclosing full data-projection preservation gate. Ordinary `zfs inherit`
cannot reset this non-inheritable property; the fixture now uses the existing
`inherit -S` restoration form, with GUID and exact value/source checks. See
[OpenZFS's documented default restoration](https://openzfs.github.io/openzfs-docs/man/v2.2/8/zfs-inherit.8.html).
The failed run remains unqualified and source-bound; owned guest/fixture/process
teardown and base/runtime preservation completed. No product control, reader,
publisher, assertion, deadline or shared tool changed. Both state suites passed
at that source; only the changed pool fixture needs fresh native execution.

Retain all previous state/backing/codec cases, live/applying/successor groups, pool/
control/guard cases, original selectors/checkers/supervisors and every deadline.
Audit the new nineteen/six children exactly once in addition to unchanged shared
checkers. Run both complete affected native suites serially on each pinned release,
then focused portable review; do not overlap guest qualification and deep review.

```sh
make test-focus PKG=./internal/storage TEST='^TestPropertyClaimApplyingAcknowledgement'
make test-focus PKG=./internal/host/freebsd TEST='^TestPropertyClaimControl'
.tools/venv/bin/python3 tools/local-qualification.py state 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py state 14.5 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
PR_TITLE='feat(freebsd): bind property controls to original applying claims' make review
make lab-status
```

The complete original backend/live authority, actual effect interruption/death/
reboot/recovery, serialized admission, operator workflows and assembled M2 acceptance
remain separate. Historical manifests retain their original executed sources.
