# M2 independently collected enrollment publication

Date: 2026-10-07. Status: implementing; qualification pending.
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
