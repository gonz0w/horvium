# M2 bounded ownership catalog

Date: 2026-10-06. Status: accepted in PR 46.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Contract and scope

`StateDirectory.InspectOwnership()` observes the complete current ownership-only
namespace through an existing private [lease](state-directory.md). It returns
independent, lexically GUID-sorted record summaries: pool GUID, policy revision,
actual record bytes and the total byte count. Summaries describe metadata; they
are neither trusted enrollment nor fresh native identity or storage authority.
No runtime, helper, HTTP or CLI caller consumes this primitive yet.

Only `writer.lock`, canonical `ownership-<GUID>.json` records and the accepted
publisher's exact `.ownership-<32 lowercase hex>.tmp` syntax are recognized.
Unknown names, future job files, noncanonical selectors and duplicate/missing
lock entries cannot be silently skipped. A recognized temporary returns
`ErrStateUncertain` with no usable catalog; inspection preserves it and never
assumes its contents are disposable. Root recovery must later reconcile it.
Unknown/malformed names return `ErrStateRecord`. Unsafe descriptors, closed leases
and non-root kernel UIDs return `ErrStateDirectory`; ordinary observed drift
returns `ErrDrift`. Every error returns the zero catalog, never a valid prefix.
Other platforms return `ErrUnavailable` without filesystem work.

Revalidate the held directory and stable lock and inspect through a new
`openat(directory, ".", O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC)` descriptor.
Its independent enumeration offset prevents a repeated scan from accidentally
appearing empty. Read names in batches of 64 and refuse more than the existing
model ceiling of 100,000 records plus the lock. Classify the complete name set
before opening records. Each record goes through the [accepted native reader](ownership-read.md),
including descriptor-relative type/UID/mode/link/ACL/size/JSON/version/checksum and
filename/pool-GUID checks. Per-record bytes remain bounded at 64 KiB; total
published ownership bytes must fit 64 MiB. Refuse before reading a record that
would exceed the remaining budget. No truncation, eviction or input repair occurs.

The helper mutex excludes cooperating publication/close during collection.
Record device/inode/mode/owner/link/size/timestamps/flags/generation and the
anchored directory namespace metadata are rechecked after collection; atime alone
is ignored. These observations detect ordinary drift. They are not an atomic
cross-record transaction, a fence against another root actor, or a native I/O
time deadline. Reads do not clear the existing lease publication-uncertainty latch.

At the qualified catalog source below, these were **inspection bounds**, not
write-admission/store-allocation limits. The subsequent [capacity guard](ownership-capacity.md)
connects this inspection to ownership publication; its evidence is separate.
The accepted publisher remains a per-record primitive; future assembled store
admission must require a valid catalog, reserve byte/count capacity including
transition temporaries, reject unsafe state and reconcile restored metadata.
This slice does not enforce total bytes on `PublishOwnership`, recover/delete
orphans, handle job/history/tombstone schemas, export metadata, enroll a pool or
launch/replay a storage operation. A catalog success alone permits none of those.
The count ceiling is inherited from the bounded model, not hardware sizing;
commercial scale and the M2 large-workload/resource gates remain unqualified.

## Behavioral and native gates

Failing-first portable tests cover deterministic selectors, immutable results,
unknown/malformed/duplicate/missing names, exact orphan syntax, bounds and
unsupported-runtime refusal. A bounded fuzz target exercises hostile name sets;
all errors must return no selectors and success must preserve exact sorted scope.
Evidence regressions reject old 161-case logs and each missing, duplicated,
failed or skipped catalog assertion.

The complete state suite now requires **202 assertions**: the accepted 161,
38 catalog controls and three catalog ACL controls in the existing owned UFS
publication fixture. Execute six native and eleven unit parents (seventeen
total), plus all 57 codec subtests. The inherited-descriptor UID-65534 child must
also deny catalog inspection before any read.

Actual native cases require an empty validated namespace, complete/repeated
scans, independently sorted record revisions/bytes, independent returned values,
anchoring after directory rename, byte/inode preservation and exact reduced
count/byte limits. Lower package-private fixture limits exercise the same scan/
accounting code on actual files; they do not qualify a populated 64 MiB or
100,000-record scale workload. A private deterministic post-read seam changes
real records/names to prove final revalidation; there is no operator/runtime
failpoint and no adversarial root-race claim.

Owned controls include unknown files, malformed selectors, preserved orphan
bytes, record symlink/hardlink/FIFO/directory/type/owner/mode/empty/oversize/damage/
newer-version/wrong-pool refusal, record/namespace/directory/lock drift, retained
publication uncertainty, closed/inert handles and foreign sentinel preservation.
The already owned POSIX UFS ACL fixture additionally proves actual catalog ACL
refusal/preservation/restored access. Close all leases before removing directories;
existing mount/md/child cleanup and host base/runtime/group teardown remain required.

```sh
make test-focus PKG=./internal/storage TEST='^TestOwnershipCatalog'
# After signed commits, review the actual PR source/base/title:
PR_TITLE='feat(storage): inspect bounded ownership metadata catalogs' make review
python3.14 tools/local-qualification.py state 15.1 --run
python3.14 tools/local-qualification.py state 14.5 --run
make lab-status
```

The expanded native evidence checker and guest selectors are shared tooling, so
this PR conservatively triggers full portable race/fuzz execution once. Subsequent
prose-only qualification updates can reuse matching executed deep evidence while
rerunning cheap baseline/identity checks. No unrelated native suites are required
by this read-only state prerequisite. The executed records below retain the actual source; cross-compilation alone
proves none of these cases.


## Executed qualification

Signed functional/suite source `5ea125fdc815fd7c6c54aba36824e50ea876b9f6` passed all
202 assertions, seventeen parent tests and 57 codec subtests on both declared
releases, without skips or failures. See the [sanitized manifest](ownership-catalog-manifest.json)
for complete source/input/SDK/module/image/runtime/utility/log provenance. The
portable review passed 86 tooling tests, all ordinary Go tests, race checks across
all eleven packages and all **22 compiler-listed bounded fuzz targets**, including
the new catalog-name parser. Shared native evidence tooling correctly selected
full portable mode; its deep checks executed once at the functional source.

Both actual root filesystems supported repeated independent scans and preserved
bytes/inodes. All unknown/orphan/unsafe/damaged record controls, final real-file/
namespace drift, inherited UID denial and owned UFS POSIX ACL controls passed.
Package-private reduced bounds did not relax any production ceiling. Inspection
preserved orphan bytes and the existing publication-uncertainty latch; it neither
removed state nor claimed recovery. There were no failed native qualification runs.

| Target | Root | Catalog parent | Full owned guest | Result |
|---|---|---|---|---|
| 15.1-RELEASE-p3 | zfs | 0.1 s | 32.315 s | all 202; complete teardown |
| 14.5-RELEASE | ufs | 0.11 s | 32.312 s | all 202; complete teardown |

Both verified the same cross-built executable before execution. The supervisor
verified every owned group stopped, unchanged base/runtime hashes and removal of
image/key/variables/executable. All 21 prior M2 manifests remain byte-identical.
The final documentation followup records its current source/title/baseline and
original deep source in the PR; it does not relabel native execution as the new head.
Complete store allocation/admission/recovery, populated scale/resource workloads,
M2 acceptance and deferred M1 manual accessibility remain pending.
