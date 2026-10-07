# M2 independently collected enrollment publication

Date: 2026-10-07. Status: native and focused portable qualification passed; accepted in PR 71.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

`PropertyEnrollmentSource.Publish` connects the accepted native source to the
accepted [one-attempt metadata publisher](enrollment-publication.md). The caller
supplies a sealed root-state review and its context, with no observation, proposal,
predecessor or pathname arguments. The source derives selection from owned review
copies and independently repeats all fixed pool/property, provider/geometry and
system/imported-pool collectors. This is an internal root integration boundary;
no operator command, HTTP route, transport message or service is introduced.

## Contract

Reject a zero binding, nil/cancelled context, invalid selection, unavailable source
or expired/backwards clock before collecting. Apply one 135-second parent bound
to collection and publication, retaining each collector's bounded output, deadlines
and independent two-pass reads. Check clock/context again after collection.

Only the actual sealed binding can finish publication. Its original root lease,
actual effective UID, actual clock, context, complete native baseline, explicit
predecessor, whole private census, job holds, system backing and projected capacity
remain required. The source creates no replacement plan or claim and never opens,
closes or transfers a lease. Changed native facts refuse before the binding's
native publication attempt. Native metadata failure consumes the existing claim
as defined by the accepted publisher; uncertainty still suspends that lease.

Return the actual publication acknowledgement or failure unchanged. Do not turn a
completed durable publication into cancellation/expiry after it returns; neither
an HTTP disconnect nor a timeout proves that publication had no effect. No durable
retry lookup or replay is added. Wrong valid context tokens are refused by the
sealed binding after independent reads; the context string is not authentication.

No schema, dependency, metadata transaction, root-state recovery rule or ZFS
operation changes. Observations remain ineligible. Restored-state/executor
reconciliation, authorized runtime provisioning, mutation transport/backend and
operator storage workflows remain separate M2 gates.

## Qualification

Portable orchestration checks prove complete fresh inputs, a bounded context,
pre-attempt denials, exact acknowledgement/failure preservation and zero-binding
refusal before I/O. Private test callbacks qualify orchestration only; they cannot
hydrate a successful root binding. Actual publication requires native evidence.

Use the existing disposable mirror/private-root fixture. All 88 exact pool-suite
assertions remain required on both releases, with its four publication markers
now going through `PropertyEnrollmentSource.Publish`. On supported root ZFS,
first/successor publication and consumed-review refusal must succeed. Six additional
mandatory native subtests reuse the same first review: cancellation, expiry,
backwards clock, collector failure, wrong context and property drift. Each checks
its expected error and ownership-record absence; valid first publication afterward
proves the denials did not consume the review. Property drift alters the read
projection in a private test wrapper; it performs no native storage change.

Actual root softupdates UFS must refuse review with zero binding, and source
publication of that binding must return `ErrInvalid`. First denial leaves no
record; a separately provisioned fixture predecessor permits unchanged-record
successor denial checks. This does not qualify publication on that backing.

```sh
make test-focus PKG=./internal/host/freebsd TEST='^TestPropertyEnrollmentSource'
PR_TITLE='feat(storage): publish enrollment from fresh native collection' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

Run the guests sequentially and require complete guest/device/process/host
teardown. The metadata package, engine, state selectors/checker and all historical
manifests must remain byte-identical to accepted PR 70. Its 819-case state/backing
suite is retained at its actual sources rather than rerun or relabeled. This
focused host integration does not qualify physical hardware, resources, clients,
power loss, restoration or M2 acceptance.

## Executed evidence

Corrected signed source `d6fc9eb9dfee6cda75de7ed727db11bcaf00cb8d`, against
accepted PR 70 source `1562334b09f3492b81a540459368efc2911a1309`, passed both
88-assertion suites with the same cross-built executable
`2c8af31a8fd17b2e2a4b7a20bf81e31f648ac5f818baefce48ca1c7443bc4464`.
The [sanitized manifest](native-enrollment-publication-manifest.json) binds
actual sources, native/portable logs, compiler/images/tools, preservation and
complete supervisor teardown.

| Fixture | Native test | Guest qualification | Observed result |
| --- | --- | --- | --- |
| FreeBSD 15.1, root ZFS | 127.21 s | 169.077 s | First/successor publication, consumed-review denial and all six pre-attempt refusal/absence subtests passed |
| FreeBSD 14.5, root softupdates UFS | 68.19 s | 93.759 s | Zero-review/source-publication denial, independently provisioned predecessor unchanged and complete teardown passed |

The focused portable review took 71.243 seconds: `make check lint test secrets`,
114 tooling tests, eight affected/transitive race packages and all seven selected
compiler-listed fuzz targets passed. The five new portable publication parents
cover orchestration and valid projection drift. The fixture correction changes
no production code; its portable regression failed before correction and passed
afterward.

The initial 15.1 run at signed source
`28dbba14e1bdfa76007296fc3fa643b144106c73` remains unqualified. Its drift fixture
changed a default quota value without changing the native source, so the parser
correctly rejected it before sealed validation. Correcting the fixture to a valid
local quota and adding the failing-first portable regression resolved that error.
All failed-run fixtures and process groups were also torn down.

All 46 historical M2 manifests, the entire storage package and metadata/native
selectors/checkers remained byte-identical to PR 70; 190 retained paths were
verified. The prior 819-case state/backing evidence keeps its actual source and
was not rerun or relabeled. Owned guests, snapshot devices, temporary images,
firmware variables, fixture keys, binaries and process groups were removed; the
standalone cross-build was removed separately. No Actions job, host service or
persistent runner was enabled.

The final prose/evidence followup still requires a clean final-head baseline
review; only exact-input deep evidence may be reused. Operator provisioning,
restored-state/executor recovery, native mutation authority, hardware/client/scale
and full milestone acceptance remain pending.
