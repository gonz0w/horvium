# M2 reserved property-submission publication

Date: 2026-10-06. Status: implementing; native qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Contract

`StateDirectory.PublishPropertySubmission` combines complete retained
plan/submission/pool checks with initial queued property-record publication under
one held root writer lease and mutex. Separate inspection and publication calls
could allow two cooperating callers to pass the same empty reservation view.
Here only one new same-pool or same-submission record can be published; distinct
pools proceed serially through the bounded metadata transaction. This does not
schedule or launch native commands.

Require a complete strictly valid initial queued property job, the effective
kernel UID 0, validated directory/lock identity and safe opened files/ACLs. Inspect
the full mixed namespace, including ownership bytes and both accepted job formats.
Retain all terminal/uncertain pool holds. Exact submission retry identity follows
[the reservation contract](job-reservations.md): full selection/review/policy/
session binding, allowing a newly generated job identifier and capture time only.
A legacy job cannot establish a complete property retry binding. A reused plan,
contradictory submission, occupied candidate identifier or held pool denies a new
record. Unknown, orphan, unsafe or damaged state is preserved.

A matched retry returns an independent existing complete job and its current
history, with `duplicate=true`, before candidate capacity checks. It creates no
file, changes no inode/history and does not re-sync existing state. This is a
metadata readback, not a fresh durability acknowledgement, restored-state
validation or authority to execute after restart. Publication uncertainty on the
current lease denies even matched retries; inspection/read APIs remain available.

A new record uses the existing exclusive temporary/file-sync/close/rename/
directory-sync pipeline. Return the bound record with `duplicate=false` only
following successful final directory sync. Its complete reservation census and
capacity accounting run again before staging and before rename, excluding only
the transaction's independently verified temporary. A holder or duplicate that
appears after the initial check is drift, never overwrite permission. Mixed
ownership/job presentation bytes and the new temporary must fit fixed bounds.
Each census binds its full record view and catalog together; it is not an atomic
snapshot or fencing against a competing OS-root actor.

Every error returns zero job and `duplicate=false`. Known pre-rename failures
clean only this transaction's verified temporary. Failed cleanup or final sync
latches `ErrStateUncertain`; a possibly visible record is never acknowledged as
successful. No rollback, repair, pruning or terminal release is added. The existing
revision-based metadata publishers remain available for their accepted contracts;
they do not become admission APIs. Non-FreeBSD platforms return `ErrUnavailable`
without filesystem work. Reduced limits/phase injection remain private fixtures.

This prerequisite has no helper/HTTP/CLI/startup consumer. Kernel-authenticated
caller and root policy, fresh native eligibility/session/clock checks, volatile
plan claims, executor/boot attribution, recovery/release and an executable
admission coordinator remain separate gates. Stored model eligibility flags and
checksums supply no such authority. Runtime storage writes remain disabled.

## Qualification procedure

Write failing behavioral API/evidence tests before implementation. Use owned root
UFS/ZFS and POSIX/NFSv4 UFS fixtures on the two pinned FreeBSD amd64 guests.
Require all 686 state assertions, 59 parents and 1,837 distinct codec/model
children without skips/failures. Prior 622-case inspection evidence cannot qualify
this publication extension. Extend both exact selectors and the strict evidence
checker, including independent missing/failed/duplicate assertion regressions.

New tests require canonical private publication, duplicate inode/byte preservation,
owned returned baselines, current history and reopened metadata readback, typed
and contradictory reuse denials, terminal/legacy holds, unrelated pools, concurrent
same-submission/same-pool/last-slot/distinct-pool outcomes, full-store retries and
ownership-byte capacity, each publication phase and uncertainty blocking, late
pool/plan/submission/unknown/orphan/byte drift and complete fixture teardown.
Actual ACL/inherited-UID denials and real UFS ENOSPC/read-only controls cover the
new API. Injected phases are ordering/failure-accounting proof; they do not qualify
real device EIO, abrupt helper death, power loss or physical controller durability.

```sh
make test-focus PKG=./internal/storage TEST='^Test(PropertySubmission|Reservation|PropertyJob|OwnershipPublication|MixedOwnershipPublication|JobPublication)'
# Clean signed source, actual proposed title:
PR_TITLE='feat(storage): publish reserved property submissions' make review
.tools/venv/bin/python3 tools/local-qualification.py state 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py state 14.5 --run
make lab-status
```

Shared publication/evidence/supervisor changes require one complete portable
review. Run guests sequentially on the same clean signed source and preserve
all prior manifest bytes. Remove owned fixtures, overlays, copied executables,
keys, firmware variables and process groups while preserving inert base/runtime
caches. Successful evidence-only followups may reuse exact matching deep checks,
with baseline/title/signature/DCO checks rerun and the original source retained.

The supervisor also needs to record failed pre-worker startup as failed with no
worker group, and clean any prepared native fixture. A control-socket bind error
must never fabricate a PID, report success or strand development resources.
Native qualification is pending; complete M2 acceptance remains open.
