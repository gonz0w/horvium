# M2 property job binding and record

Date: 2026-10-06. Status: accepted in PR 53; portable qualification passed.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Binding and successors

`BindPropertyJob(job, intent)` combines the accepted [job lifecycle](job-records.md)
and [reviewed property baseline](property-intent.md) in an immutable `PropertyJob`.
It requires independently valid inputs with exactly matching plan ID, policy
revision, pool GUID and complete typed property selection. It retains the entire
review, full policy digest and plan-scoped session digest. Binding checks internal
consistency; it claims no volatile plan, reserves no resource, authenticates no
caller and publishes no admission acknowledgement.

The initial queued event's UTC millisecond time must be at or after the captured
nanosecond time and strictly before review expiry. A capture with a fractional
millisecond therefore cannot bind to the earlier truncated millisecond. It must
wait until the next representable millisecond. Exact expiry is refused. Later
events may occur after expiry, because accepted work/history can outlive a review;
this rule never grants launch under an expired plan. The trusted future admission
controller must check actual clocks, sessions, policy and independent live native
observations rather than caller-supplied event labels.

`Job()` and `Intent()` return owned copies of history and every review row.
`TransitionPropertyJob` uses the existing lifecycle rules, preserving the complete
baseline. `ValidatePropertyJobSuccessor` requires a single append with unchanged
job identity, selection, all prior events and **every baseline field**, including
capture time and policy/session digests. Canonical row ordering does not create
false drift. The future publisher must obtain its predecessor independently from
the trusted current durable file. Neither lifecycle labels nor matching checksums
prove native effects, eligibility, authorization or root provenance.

Inputs remain caller-owned during calls. Every failed constructor, transition or
decode returns zero metadata; failed encoding returns nil bytes. No executor,
helper, HTTP/CLI handler or startup path consumes this type.

## Version 1 composite format

| Object | Required fields in canonical order |
|---|---|
| Envelope | `version`: 1; `kind`: `property-job`; `job`; `intent`; `checksum` |
| `job` | The exact version-one job body: `id`, `intent`, `history`; selection and events use the existing explicit wire fields |
| `intent` | The exact property-intent body: `captured_at`, `policy_digest`, `session_digest`, `review`; all existing explicit nested review fields |
| `checksum` | Lowercase SHA-256 hex over the domain and canonical content below |

The single checksum covers `horvium:property-job:v1`, newline, then compact
canonical JSON with `version`, `kind`, `job`, `intent`, excluding `checksum`.
There are no independent nested envelopes/checksums that could substitute an
unbound baseline. JSON field ordering/whitespace are presentation only; review
rows canonicalize through the accepted normalizer. History retains original order.
Checksums detect accidental damage; a root actor can rewrite consistent metadata.

All fields are required, nonnullable and case-sensitive. Unknown, duplicate,
missing, wrongly typed or case-altered keys at every level fail. Exact uint64
numeric fields reject exponent/fraction/string/negative/overflow values; baseline
timestamps retain canonical UTC RFC3339Nano precision. Fixed-shape token traversal
rejects arbitrary wrong nesting before following it. UTF-8 damage, trailing JSON,
unknown kinds/versions and incorrect checksums fail without repair or defaults.

The **entire composite record**, including history/baseline/checksum, has one
64 KiB input/output ceiling. Existing pre-allocation field/row limits, conservative
baseline budget and eight-event history bound still apply. Growing history can
exhaust the combined record even when the standalone baseline fits; it is never
truncated. Numeric capacity/property values remain literal strings where required.

This new kind is deliberately distinct from `job` v1 and `property-intent` v1.
At this portable prerequisite, native readers/catalogs/publishers rejected it as unsupported,
preserving unknown files and blocking unsafe namespaces. No existing file is
rewritten, upgraded or dropped. No product schema migration or native file support
is claimed. Future native integration must qualify safe files, mixed budgets,
immutable predecessors and sync/rename failure paths on both FreeBSD releases.

## Behavioral qualification

Failing-first tests cover baseline/history binding, ownership isolation, initial
time precision, all selection mismatches, exact expiry, immutable successors,
valid lifecycle/uncertainty/cancellation, canonical checksums and row order, every
required nested field, independently rechecksummed semantic refusals, full uint64
precision, exact byte ceilings and hostile/trailing/nested input. `FuzzPropertyJob`
requires bounded independent stable round trips or zero metadata on any denial.
Existing job/ownership/property-intent tests remain required.

Traceability: partial M2-JOB-001, M2-PLAN-001 and M2-STATE-001 at T1. No native
facility or syscall changes. Prior native evidence retains its original source.
Run the focused development loop, then signed-head review:

```sh
make test-focus PKG=./internal/storage TEST='^TestPropertyJob'
PR_TITLE='feat(storage): bind reviewed property baselines to jobs' make review
make lab-status
```

Portable qualification below establishes metadata consistency. FreeBSD
cross-compilation establishes portability only. Native persistence, complete executable intent/eligibility, actual admission
and reservations, caller/executor attribution, postconditions, restart/restore/
acknowledgement cuts and M2 acceptance remain open. The runtime remains read-only;
the deferred M1 manual accessibility gate remains pending.


## Executed qualification

Signed functional/test source `ddec32450bfc533eca98c2e39d6a68ce09951404` passed six parent tests
and 859 distinct subtests, including 830 required-shape cases, without skips or
failures. The [sanitized manifest](property-jobs-manifest.json) records exact
source/inputs/toolchain/cases and log hashes, including the missing-API failure
before implementation.

Local `make review` passed all required baseline checks (92 tooling regressions),
storage-only race checks and all 18 compiler-listed storage fuzz targets in
94.039 seconds. No complete milestone deep suite or unrelated native suite ran.
A supervised cgo-free FreeBSD/amd64 test binary compiled at the same source,
was removed and was never executed. All owned process groups stopped with verified
teardown; no VM, mount, fixture or host service started. All 28 prior M2 manifests
remain byte-identical with their actual original sources. Final prose/evidence
review reruns baseline/signature/DCO/title/source checks and may reuse this
exact-input executed deep result.

This qualifies bounded metadata binding/encoding/immutable successors. Native
persistence, full executable eligibility/intent, admission/reservations, attribution/
postconditions, recovery/acknowledgement cuts, M2 acceptance and the deferred M1
manual accessibility gate remain open.

Accepted in [PR 53](https://github.com/gonz0w/horvium/pull/53), merged source `417afe1c965cd13ccde53934babaf67875f07635`. The separate [native store extension](property-job-storage.md) qualifies descriptor access, publication and mixed census for this format.
