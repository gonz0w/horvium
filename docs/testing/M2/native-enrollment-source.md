# M2 native enrollment source

Date: 2026-10-07. Status: both native targets and complete portable review passed; maintainer review pending.
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

## Executed evidence

Signed behavioral source `d13f4ba489cd659b87f81f2746830aaa3a94bfef`, against accepted
base `52195cf5135f04d09422dd85c337e84329bf6cf9`, passed all 84 exact assertions on
both disposable targets with the same cross-built binary
`cdb1dab097fb4f8834df2c6a835740a38b1f9602c49e1d2cb2c9e534b5570474`.
The [sanitized manifest](native-enrollment-source-manifest.json) binds source,
inputs, tools, actual release/backing results, logs and teardown evidence.

| Fixture | Native test | Guest qualification | Observed source result |
| --- | --- | --- | --- |
| FreeBSD 15.1, root ZFS | 80.40 s | 112.522 s | Successor review and independent native/metadata revalidation passed; unrelated-record drift refused |
| FreeBSD 14.5, root softupdates UFS | 64.33 s | 96.166 s | Zero-review `ErrStateDirectory` refusal and zero-binding `ErrInvalid` revalidation refusal; records preserved |

The full portable review took 187.219 seconds: `make check lint test secrets`,
112 tooling tests, all 11 race packages and all 35 compiler-listed fuzz targets
passed. Initial missing-source compile failure also included a fixture helper typo;
it was corrected before green. The explicit evidence inventory failed before
checker implementation, then passed. Their original log hashes remain recorded.
All 44 prior manifests and every existing native storage metadata source remained
byte-identical to the accepted base. No historical state-suite results are relabeled.

The evidence checkpoint verified all 371 then-owned sessions stopped with complete
teardown. Guest fixtures, snapshot disks, image/firmware overlays, keys, binaries
and host process groups were removed; the standalone cross-compile binary was
removed separately. No host service, persistent runner or Actions job was enabled.
The documentation/evidence followup still requires a clean final-head baseline
review and exact behavioral-input match; it may reuse only this executed deep run.
Separate system mounts/non-softupdates UFS success, first enrollment through the
new source, production hardware/scale and runtime publication remain unqualified.
