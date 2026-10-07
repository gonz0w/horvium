# M2 reserved property-submission publication

Date: 2026-10-06. Status: implementing; native qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Contract

`StateDirectory.PublishPropertySubmission` combines complete retained
plan/submission/pool checks with initial queued property-record publication under
one root writer lease and mutex. Separate inspection/publication calls could let
two cooperating callers pass the same empty reservation view. Here exactly one
new same-pool or same-submission record can publish; distinct pools proceed
through the bounded metadata transaction. No native storage command is launched.

Require a complete strictly valid initial queued property job, effective kernel
UID 0, validated directory/lock identity and safe opened files/ACLs. Inspect the
full mixed namespace, including ownership bytes and both accepted job formats.
Retain all terminal/uncertain pool holds. Exact retry identity follows the
[reservation contract](job-reservations.md): full selection/review/policy/session
binding, allowing a newly generated job identifier and capture time only. Legacy
jobs cannot establish that full binding. Reused plans, contradictory submissions,
occupied candidate identifiers and held pools deny new records. Unknown, orphan,
unsafe and damaged state is preserved.

A matched retry returns independent existing complete metadata/current history,
with `duplicate=true`, before candidate capacity checks. It creates no file,
changes no inode/history and does not re-sync existing state. This is metadata
readback, not a fresh durability acknowledgement, restored-state validation or
authority to execute after restart. Current-lease publication uncertainty denies
even matched retries; existing read/inspection APIs remain available.

New records use exclusive temporary creation, file sync/close, rename, exact
published-inode/content verification and directory sync. Return the bound record
with `duplicate=false` only following final directory sync. Repeat the full
reservation/capacity census before staging and before rename, excluding only the
transaction's independently verified temporary. A newly appearing holder/retry is
drift, never overwrite permission. Mixed presentation bytes plus the temporary
must fit fixed limits. Each complete record view and catalog share one census;
this is neither an atomic snapshot nor fencing against another OS-root actor.

Every error returns zero/false. Known pre-rename failures clean only the verified
owned temporary. Failed cleanup or any post-rename verification/final-sync failure
latches `ErrStateUncertain`, preserving a possibly visible destination without
acknowledging success. No rollback, repair, pruning or terminal release is added.
Existing revision-based metadata publishers keep their accepted contracts; they
do not become executable admission APIs. Non-FreeBSD returns `ErrUnavailable`
without filesystem work. Reduced limits/phase injection remain private fixtures.

No helper/HTTP/CLI/startup consumer is added. Caller/root policy, fresh native
eligibility/session/clock checks, volatile plan claims, executor/boot attribution,
recovery/release and an executable admission coordinator remain separate gates.
Stored model flags/checksums grant none of that authority. Runtime remains read-only.

## State filesystem capability restriction

All native metadata publishers now refuse UFS with `MNT_SOFTDEP` enabled, before
staging; submission publication also refuses matched retries on that backing.
Use ZFS or UFS without soft updates for this prerequisite. No product code changes
mount options, tunes a filesystem, disables soft updates or reformats storage.
Read/inspection contracts remain available, including their existing strict drift
refusals. This is a state-directory restriction, not a restriction on NAS data pools.
Runtime write eligibility must carry this capability gate before admission.

The initial 14.5 softupdates-root runs safely refused metadata drift under
contention. Diagnostics confirmed changing ctime and unchanged mtime; extra
inode sync/readback did not eliminate later changes. The pinned FreeBSD source
explains why: rename removal work can later decrement on-disk link accounting and
set `IN_CHANGE`, and `ufs_itimes_locked` updates ctime from that flag. Ordinary
fsync flushes do not establish that all deferred removals have finished. Keep
strict retained-record timestamp/identity checks; do not guess that a ctime-only
change is harmless or retry a failed census. The failed/superseded runs retain
their original source and teardown records.

Primary inspected source: FreeBSD 14.5 revision
`231eee334c098a8398bc959202415c48058f67f0`,
[ffs_softdep.c](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/ufs/ffs/ffs_softdep.c)
(`handle_workitem_remove`, `softdep_sync_metadata`, `softdep_fsync`) and
[ufs_vnops.c](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/ufs/ufs/ufs_vnops.c)
(`ufs_itimes_locked`). The same pinned
[newfs manual](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sbin/newfs/newfs.8)
defines explicit `-u`/`-U`; omitting both enables soft updates for UFS2. This finding adopts a conservative tested capability
restriction through the EDD/compatibility amendment in this PR, not a new storage
architecture or a claim to fence other root writers.

## Qualification procedure

Require all 710 state/backing assertions, 60 parents and 1,837 distinct codec/model
children without skips/failures on FreeBSD 15.1/14.5 amd64. Twelve exact backing
markers require actual filesystem/softupdates capability and complete backing
teardown. Core metadata tests use owned root ZFS directories or independently
allocated UFS memory fixtures with soft updates explicitly disabled by
base `newfs -u`; the negative fixture uses `newfs -U`. A separate owned
softupdates UFS fixture on each target must prove publication/retry refusals,
read visibility and unchanged files. Existing POSIX/NFSv4, ENOSPC and read-only
fixtures use explicit `newfs -u` and retain their ACL/failure controls and
complete teardown. Root filesystem
identity is recorded separately from metadata fixture backing.

New tests require canonical private publication, duplicate inode/byte preservation,
owned results/current history, reopened metadata readback, contradictory typed
reuse denials, terminal/legacy holds, unrelated pools, 32 concurrent callers in
same-submission/same-pool/last-slot/distinct-pool scenarios, full-store retries,
ownership-byte capacity, every phase/uncertainty block, post-rename inode/content
refusal, late pool/plan/submission/unknown/orphan/byte drift and complete teardown.
Prior 622-case inspection evidence cannot qualify this extension. Strict evidence
validation must refuse missing/failed/duplicate assertions or backing markers.

```sh
make test-focus PKG=./internal/storage TEST='^Test(PropertySubmission|Reservation|PropertyJob|OwnershipPublication|MixedOwnershipPublication|JobPublication)'
PR_TITLE='feat(storage): publish reserved property submissions' make review
.tools/venv/bin/python3 tools/local-qualification.py state 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py state 14.5 --run
make lab-status
```

Shared publication/evidence/supervisor changes require one complete portable
review. Run guests sequentially on identical clean signed source; preserve all
prior manifest bytes. Remove owned fixtures/devices/mounts, overlays, executables,
keys, firmware variables and process groups, preserving inert base/runtime caches.
Evidence-only followups may reuse exact matching deep checks with baseline/title/
signature/DCO checks rerun and the original source retained. Failed pre-worker
supervisor startup remains failed, with no invented group and prepared native
resources cleaned. Phase injection does not qualify real device EIO, abrupt
process/reboot/power-cut durability, restoration or physical controller behavior.
Native qualification is pending; complete M2 acceptance remains open.
