# M2 root policy and native property binding

Date: 2026-10-06. Status: implementing; qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [issue 19](https://github.com/gonz0w/horvium/issues/19).

`freebsd.PropertyPolicySource` accepts only a pool GUID. It derives the pool name
and explicit scopes from `StateDirectory.ReadOwnership` through an already-held
root-owned lease, performs the existing bounded two-pass native collection, then
reads the policy again. Complete canonical policy equality includes revision and
every scope/property, while harmless ordering differences remain equivalent.
The source borrows the lease; its owner must close it. Closed, unsafe, missing or
damaged metadata is refused by the existing descriptor/ACL-aware reader.

`storage.ValidatePropertyPolicyBinding` checks the complete collected graph,
pool incarnation/policy, identities, filesystem states, features, topology and
every explicit enrolled filesystem GUID/name/parent/property scope. Missing or
renamed scopes, inconsistent unselected descendants and empty scope deny binding.
This consistency check accepts ineligible observations without setting eligibility;
the planner's original eligibility gate remains mandatory and unchanged.

Every error exposes zero policy and observation. Successful policy output is
an independent canonical copy and observation eligibility always remains false.
There is no new config schema, record version, HTTP/helper operation, CLI flag,
native mutation, enrollment command or runtime consumer. Policy metadata alone
does not establish root enrollment. System/device exclusions, reconciliation and
the full trusted coordinator backend still precede executable admission.

Both policy reads are bounded by the existing 64 KiB record reader/codec; native
collection retains its shared 64 MiB/100,000-record budgets, ten-second command
and 45-second collection deadlines. Context is checked before policy I/O and
after each collection/read boundary. Synchronous policy reads use the existing
lease mutex/file reader, without a new cancellable filesystem API. Equal repeated
reads detect ordinary drift, not an atomic policy/native/publication transaction,
intervening root changes or a ZFS fence. Publication must still bind fresh policy
and eligibility under its future admission boundary.

Failing-first portable tests cover full graph/scope denials, both failed policy
reads, revision/property drift, foreign pool, native failure, cancellation,
invalid selection before I/O, canonical ordering, independent results and empty
scope. Bounded fuzz exercises hostile policy input and the second read.

The focused native fixture retains the original 16 collector assertions and adds
six mandatory assertions: actual root policy binding without authority, absent
policy refusal, an actual fixture-owned revision change during collection,
foreign filesystem scope, GUID-preserving external rename and closed lease
refusal. Metadata files are provisioned only by test setup in an owned private
directory; this is no enrollment/publication/durability claim. Both FreeBSD targets
must pass all 22 assertions, the actual test and executable checksum, with verified
pool/provider/lease/guest/artifact teardown. Prior native state evidence is retained
at its original source; the state publisher/reader code does not change here.

```sh
make test-focus PKG=./internal/storage TEST='^TestPropertyPolicyBinding'
make test-focus PKG=./internal/host/freebsd TEST='^TestPropertyPolicy'
PR_TITLE='feat(storage): bind native observations to root policy' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

Changing the shared native evidence checker requires one complete portable deep
review. Daily behavior checks remain focused; this slice adds no Actions jobs or
home-lab requirement beyond the selected Mac/QEMU fixtures.
