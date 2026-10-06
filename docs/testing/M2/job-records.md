# M2 job lifecycle metadata and strict codec

Date: 2026-10-06. Status: accepted in PR 48.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Current boundary

`Job`, `TransitionJob`, `EncodeJobRecord` and `DecodeJobRecord` implement bounded
lifecycle metadata for the existing quota/refquota/compression intent subset.
They operate only on caller-owned in-memory data. No directory/file is opened,
job admitted, submission claimed, reservation acquired, acknowledgement made,
helper/HTTP/CLI contract exposed or native operation launched. The application
remains read-only. The existing ownership-only catalog still refuses job files.

This is an independent partial slice 2 prerequisite. The intent preserves plan/
submission identifiers, policy revision, pool GUID and selected property change;
it does **not** capture the full reviewed native baseline, process/boot identity,
creation nonce or independently reconstructible recovery postconditions. An
observation digest is a reference, not the observation or proof that it exists.
Before durable admission, the assembled store must retain all required baseline/
intent/attribution data in a reviewed versioned format, enforce mixed budgets and
trusted provenance, and qualify actual recovery. This development lifecycle
format must not be mistaken for a complete durable intent/recovery anchor.

## Lifecycle and restart projection

The first event is revision 1, queued, reason `admitted`, with no observation.
Each subsequent event has the next exact revision and a nondecreasing positive
UTC Unix-millisecond timestamp, bounded through year 9999. Timestamps preserve
recorded data; they establish neither clock trust nor review freshness/deadlines.
All history is retained in order. No repeated-state polling event, arbitrary
state jump, terminal replay or history truncation is accepted. Eight events is
the conservative metadata ceiling; the current graph needs at most four.

| From | Permitted next states | Requirements |
|---|---|---|
| queued | applying, cancelled, reconcile-required | applying: `launch-intent`; cancelled: `operator-cancelled`; reconciliation: `review-expired`; no observation |
| applying | observing, reconcile-required | observing: `native-observation` with lowercase 64-hex observation digest; reconciliation retains the empty previous digest |
| observing | succeeded, failed, reconcile-required | success: `verified`; failure: `verified-failure`; both require an observation digest; reconciliation retains the exact previous digest |
| succeeded, failed, cancelled | none | terminal; no replay |
| reconcile-required | none in this prerequisite | nonterminal; native attribution/quiescence and explicit resolution remain unimplemented |

After applying/observing, reconciliation reasons are restricted to
`ambiguous-effect`, `storage-unavailable`, `executor-unverified` and
`state-io-failure`. `storage-unavailable` is accepted only as a reconciliation reason. No
post-launch generic cancellation is represented. Reconciliation cannot erase
the previous observation reference or become terminal to make room. `Terminal()`
returns false for invalid records; classification never authorizes eviction or
reservation release. Lifecycle validation checks internal consistency; a root
actor can forge labels/digests, so future independently verified native evidence
remains necessary even for a well-formed success history.

The package-private restart projection returns reconcile-required/review-expired
for queued work and reconcile-required/executor-unverified for applying/observing.
Existing terminal or reconciliation state/reason is retained. Projection changes
no record, discards no original history/observation, acknowledges no recovery,
frees no reservation and enables no replay. It is not startup/recovery wiring.

## Version 1 lifecycle format

Every field is required, nonnullable and case-sensitive. Unknown/missing/duplicate
keys at every level fail; escaped key spellings cannot conceal duplicates. Integer
fields require canonical unsigned decimal integers, rejecting strings, exponents,
fractions, negatives and overflow. Unknown/newer versions or record kinds fail
without repair, reset or guessed defaults. Errors return zero metadata/nil bytes.

| Object | Fields in canonical order |
|---|---|
| Envelope | `version`: 1; `kind`: `job`; `job`: body; `checksum`: lowercase SHA-256 hex |
| Body | `id`: opaque lowercase 64-hex; `intent`: intent; `history`: nonempty ordered event array |
| Intent | `plan_id`, `submission_id`: opaque lowercase 64-hex; `policy_revision`: nonzero uint64; `pool_guid`: canonical positive decimal GUID; `change`: existing property intent |
| Change | `filesystem_guid`: canonical positive decimal GUID; `property`: quota/refquota/compression; `value`: existing reviewed subset (none/positive canonical literal bytes for quota/refquota; off/lz4 for compression) |
| Event | `revision`: sequence from 1; `at`: positive UTC Unix milliseconds; `state`, `reason`: exact lifecycle values above; `observation_digest`: empty or lowercase 64-hex as required above |

Identifiers are validated format only; no random generation, client trust,
submission deduplication or binding to a live `Plan` is claimed. Full uint64
policy revision precision is retained. Explicit private wire structs isolate the
format from planning Go field names. History stays in chronological sequence;
encoding never sorts it. Independent copies prevent caller/decoder/transition
aliasing. Inputs must remain owned by the caller for the duration of a call.

The checksum covers `horvium:job:v1` followed by a newline and compact canonical
JSON with fields `version`, `kind`, `job` in that order, excluding the checksum
field. Body/intent/change/event order is listed above. Whitespace and JSON object
field order do not affect the digest. It detects damage and domain confusion;
it provides no authentication, native success or authority. Matching independently
recomputed checksums still cannot make invalid lifecycle semantics valid.

Both input and output are bounded at 64 KiB, history at eight events and individual
fields at the existing 4 KiB ceiling. Format/semantic constraints further restrict
accepted field lengths. Check history/fields before copying/marshalling. Decoding
uses the accepted fixed-shape token traversal: unknown/wrong nesting fails before
recursing into arbitrary structure. UTF-8 damage, trailing values and partial
records fail; no data is truncated or silently upgraded. There is no installed
job-state schema or migration/rollback operation in this slice.

## Behavioral qualification

Tests first fail because lifecycle types and codec functions are absent. Focused
behavioral tests cover independent canonical/checksum bytes, immutable round trips,
alias isolation, whitespace at the exact 64 KiB ceiling, allowed transitions and
restart projections, no skipped observation/terminal replay/post-launch cancel,
uncertainty retention, every required field at every object level, null/type/case/
duplicate/missing denials, numeric/UTF-8/version/kind/checksum errors, deeply nested
wrong shapes, independently checksummed invalid semantics, history/field bounds,
full uint64 precision and zero outputs on denial. `FuzzJobRecord` requires every
accepted hostile input to round-trip within bounds without changing input bytes.

Traceability: partial M2-JOB-001, M2-RECOVERY-001 (projection only) and M2-STATE-001
at T1. No native filesystem/executor behavior changes. Existing immutable native
manifests retain their actual source; they are not rerun or relabeled as evidence
for durable job storage. FreeBSD/amd64 cross-compilation is only a portability
check. Job-file reader/publication, complete durable intent, admission/deduplication,
mixed budgets, tombstones/retention, root provenance, executor quiescence, actual
restart/restoration/acknowledgement cuts and M2 acceptance remain open.

```sh
make test-focus PKG=./internal/storage TEST='^TestJob'
PR_TITLE='feat(storage): validate bounded job lifecycle records' make review
make lab-status
```

The package-only change selects storage race checks and its complete compiler-listed
fuzz inventory. It does not change shared verification tooling or require a fresh
complete portable milestone run. Evidence-only prose followups may reuse matching
executed deep checks while rerunning baseline/source/title checks. Executed records below retain their actual source; no native execution is implied.


## Executed qualification

Signed functional/test source `a3791513f6423fe5860064e93f89399c21dc1445` passed all seven job
parent tests and 119 named schema subtests, with no failures/skips. See the
[sanitized manifest](job-records-manifest.json) for exact source/inputs/toolchain,
executed case inventory and log hashes. Local `make review` passed all required
baseline checks (87 tooling regressions), storage-only race checks and all
15 compiler-listed storage fuzz targets, including `FuzzJobRecord`, in
76.449 seconds. No complete portable milestone suite or unrelated native
suite ran. The conservative native-review marker is retained; source assessment
found only in-memory metadata logic using the unchanged fixed-shape token parser.

A supervised cgo-free FreeBSD/amd64 test binary compiled at the same source and
was removed; it was not executed and proves only target portability. All owned
supervisor groups stopped and teardown was verified. All 23 prior M2 manifests
remain byte-identical. Documentation-only final review reruns baseline checks
and identifies the original deep source when reusing matching executed evidence.

This qualifies the codec and lifecycle consistency only. Native job-file storage,
complete durable intent, admission/deduplication, mixed budgets/retention,
quiescence, actual restart/restoration and acknowledgment cuts remain pending.
M2 acceptance and deferred M1 manual accessibility are still open.
