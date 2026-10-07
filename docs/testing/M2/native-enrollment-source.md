# M2 native enrollment source

Date: 2026-10-07. Status: implementation; qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0006](../../adr/0006-storage-ownership-and-plans.md),
[ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

`PropertyEnrollmentSource` joins the accepted fixed native collectors, immutable
proposal review and root-leased state binding. It borrows an already-held private
state lease; it opens no store, enables no service and publishes no record. The
proposal and exact predecessor are explicit typed inputs from a future trusted
OS-root workflow. No HTTP handler, CLI command, transport schema or runtime
consumer is added. A root-context string binds the review but authenticates no
actor; actual effective UID 0 remains enforced by native metadata inspection.

## Contract

Before native I/O, reject a nil/cancelled context, malformed context token,
invalid or incomplete proposal, missing member/scope selections and invalid
first/successor revisions. Only an entirely zero predecessor explicitly requests
first enrollment. Read failures never infer absence. Reuse the bounded ownership
codec to own canonical copies of both selections before collection; caller mutation
cannot change them during native reads. Legacy predecessors remain explicit;
there is no automatic adoption or scope/member discovery.

Collect complete pool properties/topology/features by the root-selected GUID/name,
then independent provider identity/geometry, then full imported-pool/mount/swap
system use. Retain each accepted collector's fixed commands, bounded output,
timeouts and two-pass ordinary drift checks. The composed native window is capped
at 135 seconds. Build the bounded five-minute review and bind it to exact predecessor,
complete private metadata, projected capacity, supported actual system backing and
retained job holds through the borrowed lease. Check cancellation and actual clock
again after binding, repeat sealed state validation and check clock/cancellation
again after metadata inspection. Every failure returns zero binding; success keeps
false eligibility.

`Validate` rejects a zero binding before native I/O, checks selection/cancellation
and expiry, independently repeats all three native collections and then validates
the binding's original root lease and full metadata baseline. It checks actual
clock/cancellation after inspection. Review copies cannot hydrate a binding or
transfer it to another lease. The runtime clock is `time.Now`; private fixture
seams are not operator configuration.

These sequential snapshots detect ordinary drift; they are not atomic with a
future publication. Context checks surround metadata work but cannot preempt its
mutex or kernel calls. Final fresh native/context/clock, kernel caller authority,
full restored-state/executor reconciliation and atomic durable publication remain
required before any operator workflow can change storage. No reservation is
claimed/released and no latched uncertainty is reset.

## Behavioral and native checks

Portable tests exercise selection denial before native I/O, canonical ownership
of both selections, required binding, provider/system/native failure, zero partial
results, cancellation and expired/backwards clocks after binding. Unsupported
hosts must refuse the factory. The source cannot mint or hydrate a fake successful
binding in portable tests; actual root-backed success needs native proof.

Retain all 76 focused assertions and add eight: `enrollment_source_review`,
`enrollment_source_revalidate`, `enrollment_source_policy_denied`,
`enrollment_source_member_denied`, `enrollment_source_cancelled`,
`enrollment_source_context_denied`, `enrollment_source_store_uncertain` and
`enrollment_source_preserved`. Use the existing owned mirror and separate private
root-state fixture; no extra disk/service or filesystem option changes. Actual
root-backed ZFS must permit successor review and fresh revalidation, and deny
same-revision predecessor/member drift and unrelated-record drift. Softupdates UFS
must return `ErrStateDirectory` with zero review and unchanged records; its zero
binding must fail revalidation with `ErrInvalid`, not qualify successful refresh.
Both backings must retain orphan uncertainty and preserve records/content/native
state with complete guest/device/process/host teardown.

```sh
make test-focus PKG=./internal/host/freebsd TEST='^TestPropertyEnrollmentSource'
PR_TITLE='feat(storage): collect native enrollment reviews' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

Require all 84 exact assertions and actual PASS on both releases. Shared evidence
checker changes require complete portable review. The historical 777-case state
suite is not rerun: existing native metadata engines and historical manifests
must remain byte-identical to the accepted base. No hardware/client/scale, reboot,
power-cut, restored-state or milestone acceptance follows.
