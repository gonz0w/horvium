# M2 property planning model: contracts and partial evidence

Date: 2026-10-04. Status: D1 model accepted in [PR 21](https://github.com/gonz0w/horvium/pull/21),
source `f334094eae3d3b6d8ca740465ac09cb5be08ba20`.
Design: [PR 20](https://github.com/gonz0w/horvium/pull/20), approved source
`753f601960529cd17596d0ffc628afe38d041c82`.
[PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [slice sequence](../../milestones/M2/implementation.md).

## Implemented boundary

`internal/storage` is a standard-library-only model for property review. It is
not connected to the helper, HTTP, CLI or M1 inventory. The existing runtime
remains read-only. There is no root policy file schema, enrollment command,
plan lookup/submission, reservation, job store or native mutation executor.
This is the first part of slice 1, not completed slice 1 or M2 acceptance.

Trusted prospective helper inputs are a per-pool local policy and a fresh,
complete filesystem/property observation, independent dataset identity/type table
and complete filesystem mount/readonly/crypto state table, plus a matching
pool identity/import/policy observation. The policy explicitly selects pool and
filesystem GUIDs, baseline names/parents and each allowed property. Observation,
names, property sources and caller claims cannot create scopes. The observation's
eligibility flag is a prospective native-adapter obligation, not proof from a
request: no implemented adapter can currently establish it. Do not convert an M1
snapshot, paginated inventory or client-supplied object into this authority.

The policy and observation must be exclusively owned during a call. A plan
retains private scalar state and owned review slices; every review accessor
returns independent slices. Plans are volatile and have no serialization or
restore path. Internal JSON is used only for deterministic policy hashing and
review sizing, not an adopted wire or persistent schema.

## Property reviews and refusal behavior

- Quota and reference quota accept positive canonical decimal uint64 bytes or
  the explicit `none` choice. Reject zero, signs, whitespace, suffixes, empty
  values and overflow. The model keeps literal strings, including values above
  browser integer precision; it never converts through floating point.
- Compression requests allow `off` or `lz4`. Observed effective values and
  default/local/received/inherited sources must be internally consistent. This
  is model validation; the bounded native property parser has separate evidence.
- Parent quota requires quota permission on every descendant, regardless of a
  child's local quota. Reference quota requires only the target's permission.
  Compression requires permission on every inheriting descendant until a local
  or received override protects a branch. Changing a default to a local value
  still has descendant impact. No affected foreign child is implicitly enrolled.
- Review context contains target, ancestors and the entire target subtree,
  including protected branches. It shows baseline GUIDs/names/parents, effective
  values and sources so ancestor constraints and overrides remain reviewable.
  The [property projection](properties.md) adds retained received values to
  context and drift checks. [Dataset identities](identities.md) bind exact filesystem
  coverage and snapshot/bookmark membership; volumes deny this planner. Reviews
  retain relevant versions with local owner GUIDs and [filesystem constraints](dataset-state.md)
  preserve mount/readonly/crypto state. [Pool observations](pool-state.md)
  bind permanent/import identity and policy metadata. It does not yet show usage, topology,
  devices, feature evidence or operation recovery details.
- Malformed or duplicate GUIDs/names/scopes, missing/false parents, inconsistent
  sources, unknown properties, incomplete observation or failed eligibility deny
  planning. GUIDs are positive canonical decimal uint64 strings. Management
  names use a deliberately narrow subset: at most 32 components/1,024 bytes,
  each at most 64 ASCII bytes, starting with a letter, then letters/digits/`_`/`-`.
  Pool prefixes `mirror`, `raidz`, `draid`, `spare`, `log` are refused. Unsupported
  existing names remain observable through M1 but unmanageable here.
- Preflight caps policy scopes, filesystem rows, identity records and state rows at 100,000 each, individual
  fields at 4,096 bytes, and combined input accounting at 64 MiB before building
  maps or sorting. The review has a 64 KiB ceiling with conservative overhead;
  large impacts fail rather than truncate. These are safety caps, not scale or
  performance qualification, and no four-disk or hardware profile is hardcoded.

Each plan gets an independent cryptographic 256-bit identifier, session binding,
policy digest/revision and five-minute expiry. Revalidation rejects expiry at the
exact boundary, backward time, another session, changed policy, missing/reused
identity, rename, changed context/source/value or new subtree members. Input
ordering is normalized. An unchanged GUID after rename remains the resource
identity, but a new explicitly approved baseline is required. Unrelated sibling
property/version changes do not alter the target's context.

Revalidation requires a genuinely fresh helper observation; this pure library
cannot prove collection freshness, exclusive writer status or native eligibility.
It does not fence external root mutations. Validation can be repeated and never
claims a plan: durable duplicate admission/single-use submission, final native
checks, topology/device drift and restart recovery remain later gates. Never use
successful model validation as permission to launch a native command.

## Behavioral tests and remaining gates

The tests were written first and observed failing against deny-only model stubs
before implementation. Fixed synthetic GUIDs/names/property values are fixtures;
no operator inventory or disk is used.

| Test Design ID | Model assertions covered | Remaining acceptance |
|---|---|---|
| M2-OWN-001 | `TestObservationNeverEnrollsResources`, `TestNewBaselinePreservesGUIDAfterRename` | Native adoption without effects, root enrollment, persistence/origin and kernel privilege |
| M2-DRIFT-001 | `TestPlansRequireFreshReviewOnDrift`, `TestAncestorConstraintsAndUnrelatedSibling` | Real collection, topology/device/use drift and execution boundary |
| M2-PLAN-001 | `TestReviewScopeAndImpact`, `TestPlanContextAndExpiry`, `TestPlanOwnsReviewAndInputCopies`, ordering tests | Helper/UI/CLI shared review, restart, single-use durable duplicate admission |
| M2-PROPERTY-001 | Literal-limit, override/default impact, reference-quota and malformed-source tests | Actual native properties, complete source/usage projections, partial-write outcomes |
| M2-SCALE-001 | `TestObservationAndPlanBudgets`, `FuzzPropertyPlan` | Native/store/job/capacity/concurrency/resource workloads |

Local gates passed: `make check lint test secrets` and `make go-check` (vet, complete unit
suite, race detector and bounded fuzzing, including the new property planner).
The FreeBSD amd64 test-binary cross-build also passed; it checks compilation only. Native property-source results are tracked separately in
[the projection record](properties.md), [identity binding evidence](identities.md)
and [filesystem state evidence](dataset-state.md). There are no M2 mutation, state-failure,
browser or performance execution results. Every complete
Test Design row remains pending. The retained M1 native results qualify M1 only;
Safari/manual accessibility remains pending by maintainer choice.
