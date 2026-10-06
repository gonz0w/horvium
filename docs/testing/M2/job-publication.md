# M2 durable job metadata and shared publication budgets

Date: 2026-10-06. Status: implementing; native qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Publication contract

`StateDirectory.PublishJob(job, expectedRevision)` writes strict lifecycle metadata
through the existing root-controlled writer lease. Zero expected revision requires
an absent record and a single initial queued event. A replacement requires the
exact current record/revision and `ValidateJobSuccessor`: unchanged ID, complete
intent and all historical events, with exactly one valid new event. Stale,
duplicate, skipped, rewritten or missing predecessors are refused. Inputs are
caller-owned during the call. Other platforms return `ErrUnavailable` without
filesystem work; every FreeBSD call checks the kernel effective UID is root.

A nil result confirms this metadata publication sequence, not admission or a
native effect. No runtime/helper/HTTP/CLI/startup caller uses this primitive.
The lifecycle schema still lacks the complete reviewed native baseline,
executor/boot attribution and recovery postconditions required for executable
intent. Its recorded state/reason/digest cannot prove a storage operation occurred.

Ownership and job publication share one native descriptor-relative transaction.
Under the held mutex/lease, require writable local UFS/ZFS and a complete safe
[mixed census](state-catalog.md). Validate current predecessor bytes and identity.
Exclusively create a private same-directory `.job-<32-lowercase-hex>.tmp` (ownership
uses its existing prefix), validate opened-object ACL, write, file-sync, close,
and revalidate temporary/destination/lease. Repeat complete census and capacity
checks before rename, excluding only this exact already inode-validated temporary.
Rename to `job-<ID>.json`, then sync the directory. An old open descriptor retains
its old bytes. No cross-record atomic transaction is claimed.

Recognized orphan job and ownership temporaries return `ErrStateUncertain` and
latch further publication on that lease. Unknown, unsafe, incompatible or damaged
entries block both publishers; no file is repaired or removed. A known pre-rename
failure closes and unlinks only the verified owned temporary and syncs the
cleanup. Failed cleanup or post-rename directory sync returns `ErrStateUncertain`
and blocks both publishers; a visible record is never acknowledged as durable
success and is never rolled back. Reading/inspection remain available, preserving
uncertainty. Closing/reopening alone is not a qualified recovery procedure.

## Capacity and identity

Both publishers now account for all ownership/job files against 64 MiB, reserving
the full new record alongside every old published record before temp creation and
again before rename. Do not subtract the destination until rename. Keep the 64 KiB
record limit and 100,000 ownership enumeration ceiling. Job creation requires a
free nonterminal slot; replacements occupy the existing slot. A transition into
succeeded/failed/cancelled requires a free terminal slot among the 1,000 retained
terminal records. Reconcile-required stays within the 64 nonterminal slots.
These counts cannot release a reservation or authorize pruning.

A submission ID cannot occur in another job record. Updating its current job
retains the same ID; duplicate creation returns `ErrDrift`. This consistency guard
is not the future request-admission API that returns an existing durable job.
Admission still needs complete executable intent, plan/session/review validation,
replay windows/tombstones and reservation semantics. Nothing is evicted to fit.

`PublishOwnership` now uses mixed census/shared bytes; valid job records no longer
block it. Exact revision checks, rooted selectors, private ACLs and durability
semantics remain. `InspectOwnership` stays the earlier strict ownership-only
observer; use `InspectState` for the mixed namespace. Lower count/byte bounds and
phase wrappers are package-private disposable fixture seams, not operator settings.

## Qualification procedure

Require **427 named state assertions**, thirty-five parents (ten native and
twenty-five unit), and all **176 codec/schema subtests** on both declared targets.
The seventy-four native job controls cover actual creation/replacement/canonical
private bytes, predecessor/history/intent refusal, terminal outcomes/replay denial,
mixed ownership creation/replacement, exact shared byte and class budgets, no
partials/eviction, anchored rename, stable lock/old descriptors, unsafe/damaged
predecessors, unrelated bad state, both orphan prefixes, each pre-rename failure,
failed cleanup/final sync and shared uncertainty, late census/capacity/collision
checks, serialized duplicates/last slot, closed/inert handles and full teardown.

Thirteen additional assertions extend owned UFS POSIX/NFSv4 ACL controls: job
publication refuses unsafe directory/lock/ownership/job/inherited ACLs, restored
publication works, and a real UID-65534 inherited-descriptor child refuses both
publishers despite effective native read access. Four filesystem assertions use
owned memory-backed UFS: real job publication, independently observed ENOSPC
refusal, native read-only refusal and preserved job bytes. Existing ownership
phase/failure/catalog/ACL/capacity controls remain mandatory because the native
transaction is shared. Reduced limits prove accounting/refusal, not resource scale.

Portable tests first fail for missing APIs and cover exact/overflow transient
bytes, class transitions/exhaustion, submission collisions, stale/invalid revisions
and unchanged inputs. The evidence regression first accepts old 336-case logs,
then refuses them after expansion; every missing/duplicate/failed/skipped assertion
and required parent/schema subtest remains denied. Existing 24 compiler-listed
fuzz targets execute; no extra fuzz target mirrors the transaction.

```sh
make test-focus PKG=./internal/storage TEST='^Test(JobPublication|MixedOwnershipPublication|StateCatalog)'
PR_TITLE='feat(storage): publish durable jobs with shared capacity guards' make review
python3.14 tools/local-qualification.py state 15.1 --run
python3.14 tools/local-qualification.py state 14.5 --run
make lab-status
```

Shared verification changes select one complete portable review. Evidence-only
followups may reuse matching executed deep evidence and rerun required baseline/
source/title checks. Cross-compilation proves no native behavior.
Traceability: partial M2-STATE-001/M2-JOB-001/M2-PRIV-001 at T1/T3.

## Remaining gates

Complete executable intent, request admission/deduplication, reservations,
retention/tombstones, startup reconciliation/old-executor quiescence, metadata
export/restore, acknowledgement/process-death/reboot/power-cut experiments and
assembled native storage mutation remain open. No running NAS/release,
commercial resource/scale, physical controller durability or client compatibility
claim follows. Preserve prior manifests and tested sources. Deferred M1 manual
accessibility and M2 acceptance remain pending.
