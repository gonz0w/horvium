# M2 root-leased native property-job assessment

Date: 2026-10-07. Status: implemented; qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

`PropertyJobSource.Assess(ctx, jobID)` connects the
[accepted retained-job assessment](property-assessment.md) to one borrowed
`StateDirectory` root lease and independently collected native policy/property
facts. The caller selects an opaque canonical job ID only; it cannot supply a
job, policy, native observation or native address. This source has no runtime
transport, startup consumer, command or UI workflow and grants no write authority.

## Complete metadata and fresh native binding

`StateDirectory.InspectPropertyJob` acquires the existing lease mutex and validates
the complete bounded mixed namespace with the existing census and reservation
builder. Include every ownership/job record, original byte budgets, duplicate
submission/plan checks and final record/directory/lock/ACL checks. Refuse unknown
entries, orphan temporaries, unsafe/malformed/newer/oversized records, missing
selected property job or its ownership record, and selected legacy lifecycle-only
records. Every error returns zero job/digest. No file or lease uncertainty flag
is changed. Softupdates UFS remains eligible for read-only inspection; its
publication refusal and all backing/write gates remain unchanged.

Return an owned property job and domain-separated SHA-256 digest of every sorted
record name and actual presentation bytes, with unsigned 64-bit length framing.
This fingerprint excludes the separately validated empty writer lock. It is an
ordinary content-drift comparison, not a persistence format, capability, root fence
or proof against a root actor restoring identical bytes. Equivalent formatting
changes the fingerprint; copied/restored identical bytes cannot prove provenance.
Hash records as the validated census streams them; do not build a second complete
encoded ledger. Existing 64 KiB/64 MiB/class/record ceilings remain unchanged.

The source validates the ID and context before I/O, inspects the complete ledger,
checks selected identity and nonzero fingerprint, collects through the actual
`PropertyPolicySource`, compares with the retained job, and reinspects the complete
ledger. Native pool name/scopes come from current root policy; two native passes and
two canonical policy reads retain their accepted bounds. Exact selected canonical
job bytes and the full raw-record fingerprint must match at the end, and context
cancellation discards the result. The source borrows and never closes the original
lease. The existing 135-second bounded context applies; synchronous bounded state
reads check cancellation at phase boundaries, not between every record read.

Baseline/expected/both/neither facts and conservative restart labels remain as
specified in the accepted assessment. Invalid/drifting collection returns zero.
Stable well-formed policy/relevant context differences from the retained review
can produce neither match. Do not resolve/replay, publish a successor, release a
hold, restore clock/session/plan authority or label durable success from a match.
No provider/system-use safety source or write eligibility is implied by this
property-only fact collector. Actual executor attribution/containment, restored
state, durable outcomes, complete recovery and operator workflows remain open.

## Qualification gates

Failing-first portable source tests cover independent collection and two ledger
reads, invalid selection before I/O, first/final read failure, complete-ledger and
selected-job drift, wrong identity, unavailable components, cancellation, malformed
bindings and policy/native failures, with no partial output. Unsupported hosts
must return zero job/digest without native access. Existing comparison/race/fuzz
checks remain required through focused review.

Both disposable FreeBSD releases must retain all 88 original pool assertions,
eleven postcondition subtests and eight retained-assessment subtests, plus the
mandatory group beneath
`TestNativePoolPropertyCollection/property-postconditions/native-job-source`:
`baseline`, `expected`, `neither`, `unknown`, `orphan`, `malformed`, `unsafe-mode`,
`missing-job`, `missing-policy`, `legacy`, `oversized`, `duplicate-plan`,
`whole-ledger-drift`, `presentation-drift`, `wrong-uid`, `preserved`, `closed`.
Require the group itself and each child to actually pass; missing/skipped/failed
cases never qualify. The shared selector/checker keeps its existing 88-case scope;
the source-bound evidence separately checks the additional actual PASS results.

The parent owns its disposable mirror and children. This fixture owns a new private
root metadata directory with encoded applying intent, current root policy and the
actual writer lock. These are direct test provisions, not native publication or
acknowledged execution. Perform only declared fixture-local quota/refquota changes
and restore them. During real native collection change an unrelated valid ownership
record; the selected job/policy still match but complete-ledger drift must return
zero. An independent presentation change must change the actual census fingerprint.
The wrong-UID subprocess runs only a copied public test executable in a separately
owned traversable temporary directory, never private metadata or credentials;
require it to exit successfully and remove the executable/directory. Verify retained
records/lock/holds, independent restored quota/refquota raw sources and final
lease-closed denial. The parent verifies its full original raw projection.
Original pool/provider/backing/sentinel, root fixtures, child and host teardown stay
required. Previous metadata/publication/cut evidence retains its original scope.

```sh
make test-focus PKG=./internal/host/freebsd TEST='^TestPropertyJobSource'
PR_TITLE='feat(storage): bind native assessment to the root ledger' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

## Fixture collection correction

Two initial 15.1 attempts reached the unchanged five-minute aggregate fixture
limit during older postcondition cases, after all seventeen new source cases
passed. One attempt ran alongside portable review and one ran alone; neither is
acceptance evidence. Preserve both failed reports and complete host teardown.
Remove redundant native reads only where no native change occurs between capture
and comparison: use the same newly collected false-eligibility graph for capture
and retained assessment, and the same newly collected drift graph for both
comparators. Initial new-source capture uses the actual graph collected immediately
before pure policy construction. The new source still independently collects all
its native evidence; no caller graph is accepted. Its restoration control now
checks exact quota/refquota raw value/received/source columns, while the parent
retains full original projection verification. All old/new assertions and product/
fixture timeouts remain unchanged; no skipped case or longer deadline qualifies.
