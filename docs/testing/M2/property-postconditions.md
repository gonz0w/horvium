# M2 reviewed property postcondition comparison

Date: 2026-10-07. Status: implementing; qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

`PropertyIntent.ComparePostcondition(policy, observation)` checks one captured
quota, refquota or compression step against an independently fresh complete
collected graph and unchanged trusted policy. It performs no native command,
metadata publication, admission, transition, reservation release or recovery.
It accepts collected observations with `Eligible: false` and never changes that
flag. A match cannot authorize a plan or substitute for native/provider/system,
executor, restored-state, session or durable-outcome gates.

## Exact matching contract

Validate the immutable captured intent, preflight caller-owned inputs, and apply
the existing complete collected-graph and explicit policy validators. Reuse the
same bounded, sorted relevant review projection as precondition validation,
without relaxing `Build`, `Validate`, capture or `CompareBaseline`. Full policy
scope/member digest, pool GUID/import incarnation/policy, features, topology,
relevant filesystem identity/state and snapshot/bookmark context must match.
Unrelated sibling property/version rows remain outside the captured review, as
[the accepted intent contract](property-intent.md) specifies; no whole-pool
unchanged-resource claim follows.

The selected filesystem must have the exact desired effective value with local
source and its own GUID as supplier. Preserve its shadowed received column.
Quota/refquota change only that property's selected row; descendant capacity
impact does not imply descendant property changes. For compression, require the
reviewed inheriting descendants to have the desired value supplied by the target
GUID. A local/received override and its protected descendants remain unchanged.
Require all other relevant properties, received columns and context to remain
exact. Reject incomplete, malformed, omitted, oversized or drifting inputs;
never truncate a review or substitute a matching spelling for a different GUID.

An already-matching local value or another root actor's identical change can
match. This establishes an observed postcondition, not causal attribution or
proof that Horvium executed anything. It does not label a job successful or
clear an uncertain outcome. Stored capture times/digests cannot restore current
clock/session authority. Inputs and captured bytes remain unchanged on success
or failure; no persistence format changes.

## Portable and native gates

Failing-first portable tests cover exact literal/no-limit/local values,
compression inheritance and local/received protected branches, received-column
preservation, policy/member/resource/property/state/version/pool/feature/topology
drift, incomplete/oversized refusal, decoded intent, ordering and input ownership.
The bounded postcondition fuzz target checks arbitrary selected values against a
fixed immutable baseline, with exact success/refusal and shared-input preservation.

Both disposable releases must pass the existing 88 pool-source assertions plus
all eleven mandatory subtests under
`TestNativePoolPropertyCollection/property-postconditions`: `quota`, `refquota`,
`no-limit`, `compression`, `inherited`, `overrides`, `wrong-source`,
`context-drift`, `baseline-retained`, `received`, `restored`.
These execute actual fixture-only local sets and independently read native
value/received/source columns, inherited supplier GUIDs and protected branches.
An identical local child value must fail an expected inherited postcondition;
an unrelated property change must fail until the fixture restores its source.
The immutable baseline and native false eligibility must remain intact.

The parent declares its disposable mirror/backings. The new fixture creates
only children inside that owned pool, performs a bounded empty-snapshot send
and unmounted receive for real received-column coverage, removes its own children
and verifies the original property/state/identity projection. Parent identity,
provider/backing, sentinel and pool/device/host/process teardown stay required.
Test-only eligibility is used solely to capture a comparison baseline; this is
not a qualified runtime review, enrollment, write backend or recovery flow.
The guest selector/checker retain their existing 88-case inventory unchanged;
the source-bound manifest separately requires every actual mandatory subtest.
No shared runner/supervisor/tooling change is needed.

```sh
make test-focus PKG=./internal/storage TEST='^TestPropertyPostcondition'
PR_TITLE='feat(storage): verify reviewed property postconditions' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

Runtime provisioning, mutation eligibility/backend/transport, durable applying
and observed outcomes, executor/recovery/restoration and UI/CLI operation flows
remain pending. Matching a postcondition alone cannot count as M2 acceptance.
