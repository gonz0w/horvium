# M2 native job reader and immutable successors

Date: 2026-10-06. Status: qualified on both declared native targets; maintainer review pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Reader contract

`StateDirectory.ReadJob(id)` reads one `job-<ID>.json` record through an existing
root-controlled [writer lease](state-directory.md). The selector is exactly 64
lowercase hexadecimal characters; callers cannot select arbitrary paths. The
kernel effective UID must be root, even when an inherited directory descriptor or
a native ACL grant gives another UID access. Other platforms return zero job and
`ErrUnavailable` without filesystem work. Invalid root selectors return
`ErrInvalid`; closed/inert handles and missing/unsafe files return
`ErrStateDirectory`. No missing job becomes an empty queue or guessed state.

Ownership and job reads now share the same descriptor-relative byte reader.
Revalidate the held root-0700 directory, safe ACL and stable root-0600 single-link
empty lock before use. Inspect the selected name without following symlinks;
require a root-owned regular 0600 single-link file and refuse more than 64 KiB.
Open with `openat`/no-follow/nonblocking/close-on-exec and validate the actual
opened object and ACL before reading. Read at most 64 KiB plus one, compare exact
size and before/after/named device/inode/ownership/mode/link/timestamps/flags/
generation, and require successful close. Atime alone is ignored. Directory
rename preserves descriptor anchoring; pathname replacement cannot redirect reads.
These are ordinary drift observations, not a fence against another root actor.

Decode through the accepted [strict lifecycle codec](job-records.md), then require
the record's job ID to match its filename selector. Mismatch returns `ErrDrift`.
Damaged/incompatible/invalid lifecycle bytes return no job, retaining codec errors;
oversize returns `ErrLimit`. Returned history is independent of input/other reads.
Reading changes no state/ACL, resets no uncertainty, enrolls no resource and
acknowledges no admission/recovery. The application remains read-only; no helper,
HTTP, CLI or startup caller consumes this primitive yet.

## Successor contract

`ValidateJobSuccessor(previous, next)` is a pure guard for future publication.
Both jobs must independently pass complete lifecycle validation. Job ID and every
intent field (plan/submission identifiers, policy revision, pool and filesystem
GUID, property and value) must remain equal. Exactly one event may be appended;
all prior events, including timestamps and observation references, must be
exactly equal as typed values. Stale, skipped, truncated or rewritten
updates return `ErrDrift`; malformed lifecycle/budget errors retain their category.
Inputs remain unchanged. A valid transition cannot replay a terminal job or
resolve reconciliation without the later qualified resolution path.

Future publication must obtain the predecessor from the current safe durable
record under the same writer lock, repeat native/revision checks and durably
publish before acknowledgement. This guard does none of those steps. It does not
compare live resource identity, acquire a reservation or supply root authority.
The job lifecycle intent still lacks complete reviewed native baseline/attribution
and recovery postconditions; it is not admissible executable intent.

## Native qualification procedure

Require **275 named state assertions**, twenty-eight parents (eight native and
twenty unit) and all **176 codec/schema subtests** on both declared targets.
The added reader has thirty actual root-filesystem controls: absent/invalid
selectors, valid independent reads, byte/inode preservation, exact 64 KiB input,
mode/owner/type/symlink/hardlink/FIFO/directory denial, empty/oversize/damaged/newer/
wrong-ID/wrong-kind/invalid-history records, control preservation, anchored rename,
directory/lock drift and lock replacement, closed/inert/nil handles, foreign
preservation, lease close and fixture teardown.

The existing owned UFS POSIX/NFSv4 ACL fixtures add eleven named job assertions:
unsafe directory/lock/record ACL refusal, no record repair and restored read on
each ACL model, plus a real UID-65534 inherited-descriptor child. That child must
prove an actual effective NFSv4 grant can read private job bytes while `ReadJob`
still refuses the non-root kernel UID. Existing ownership/lease/catalog/publication/
capacity, actual ENOSPC/read-only, ACL and process-teardown controls remain required
because the native byte reader is shared. All seven codec parents/119 schema
subtests and the append-only successor test execute on each guest as well.

Tests first fail because reader/successor APIs are absent. Evidence regression
first demonstrates that old 234-case logs incorrectly qualify, then refuses every
missing/duplicate/failed/skipped job assertion and both job schema subtest families.
Native controls use newly owned private fixture files only; no operator jobs,
datasets, disk contents or backup history are consumed.

```sh
make test-focus PKG=./internal/storage TEST='^TestJob(Successor|ReadUnsupported)$'
PR_TITLE='feat(storage): read private job records and guard successors' make review
python3.14 tools/local-qualification.py state 15.1 --run
python3.14 tools/local-qualification.py state 14.5 --run
make lab-status
```

Shared guest selectors/evidence validation conservatively require one complete
portable review; prose-only evidence followups may reuse matching executed deep
checks while rerunning baseline/source/title verification. Executed qualification below retains the actual source; cross-compilation alone
proves none of the native reader cases.
Traceability: partial M2-STATE-001/M2-JOB-001/M2-PRIV-001 at T1/T3.

## Remaining boundaries

There is no job publication, mixed catalog/budget, durable admission/deduplication,
reservation release, tombstone/retention, executor quiescence, startup recovery or
restoration. The ownership-only catalog/publisher continue to refuse all job files,
including valid lifecycle files; the next assembled namespace must validate and
account for both kinds before admitting writes. Reading a job in a disposable
fixture therefore does not qualify a functioning job store or recovery anchor.
No physical/commercial/client/scale or power-loss claim follows. Existing native
manifests retain their original executed sources; new evidence does not relabel
those historical records. M2 acceptance and deferred M1 accessibility remain open.


## Executed qualification

Signed functional/suite source `5241ae7f0bccde0423613af1dfcde9aed951ac34` passed all 275 state
assertions, twenty-eight parents and 176 codec/schema subtests on both declared
releases with no skips/failures. See the [sanitized manifest](job-read-manifest.json)
for exact source/input/SDK/module/image/runtime/utility/log provenance. Shared
verification changes conservatively selected one full portable review: all baseline
checks, 89 tooling regressions, race checks across eleven packages and all
23 compiler-listed bounded fuzz targets passed in 113.36 seconds.

Actual job reads passed every private-file/size/codec/ID-binding denial, preserved
bytes/inodes and remained anchored after directory rename. Both owned UFS ACL
models proved job directory/lock/record denial, no repair and restored read. The
UID-65534 child proved effective native NFSv4 access to private job bytes while the
product refused the inherited-descriptor non-root call. Existing shared-reader
ownership and all state/publication/capacity controls still passed. All codec and
immutable successor tests also executed on both guests. No native qualification
failures occurred.

| Target | Root | Job reader parent | Full owned guest | Result |
|---|---|---|---|---|
| 15.1-RELEASE-p3 | zfs | 0.1 s | 37.161 s | all 275; complete teardown |
| 14.5-RELEASE | ufs | 0.1 s | 32.829 s | all 275; complete teardown |

Both guests verified the same cross-built executable before running it. All owned
leases/files/mounts/memory devices/children closed or were removed; supervisor
groups stopped; base/runtime hashes were preserved and snapshots/keys/variables/
test executables removed. All 24 prior M2 manifests remain byte-identical. The
final evidence-only review reruns baseline checks and retains the original deep
source when matching execution is reused; native execution is not relabeled as
that documentation head. Job publication, complete intent/mixed budgets, admission,
actual recovery, M2 acceptance and deferred M1 accessibility remain pending.
