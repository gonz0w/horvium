# Adopted original lifecycle caller fixture

Status: implementation candidate; native qualification pending.
Date: 2026-10-11. Owner: Cam (`@gonz0w`).
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305),
[parent 139](https://github.com/gonz0w/horvium/issues/139) and adopted
[decision 331](https://github.com/gonz0w/horvium/issues/331).
Authority: [PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[retained-policy contract](../../milestones/M2/retained-policy-startup.md),
[adopted caller amendment](../../research/M2/policy-lifecycle-caller.md) and
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
Requirements: M2-RUNTIME-002, M2-STATE-001, M2-RECOVERY-001, M2-OWN-001,
M2-DRIFT-001, M2-PRIV-001 and M2-NATIVE-001.

## Fixture behavior and acceptance

PR 332, merge `d5bdf24d4ef5df480ae960b985365de8610cc274`, adopts one absolute
180-second startup caller across the original factory's review and activation,
390-second first/restart children, a 1200-second original parent work context and
44-minute lifecycle phase. Every product operation remains 90 seconds; other
caller, fact/review, native collector, cleanup, output, remote and supervisor
limits remain unchanged. The 120/150 trace and separate 180/210 diagnostic remain
byte-identical. No producer wrapping, cache, reset, reseed, guard deletion, extra
CPU or product/runtime change is introduced.

A new actual startup caller assertion raises each child to 22 controls and each
lifecycle parent to 58 root plus 21 fact controls. The complete gate requires
five parents, **293 controls, thirteen normally waited children and forty ordered
scalar observations per release**. Each child retains its ten original helper
calls in order, with monotonic elapsed nanoseconds, selected allowance, actual
unchanged deadline and separate classified operation/caller errors before deferred
cancellation. Startup alone records returned-owner presence. Unknown raw errors
become `other`; no raw paths, arguments, stacks or credentials are published.

Successful child rows are validated and printed by the parent. Missing, duplicate,
extra, reordered, cross-mode/profile, expired, malformed or inconsistent successful
rows reject the gate. Failed children retain only actual partial rows through the
existing bounded transcript; absent later stages stay absent. Nested operation
expiry does not imply caller expiry or final publication state. Existing original
successor, policy/census, lease, lineage, publication/uncertainty and retirement
checks remain required. Scalar helpers are test-only; tagged native fixture files
stay outside normal product builds. Interrupted supervisor cleanup now removes
the lifecycle executable as well as the existing owned devices/image/key.

## Failing-first and execution plan

Signed red source `cbadda7b31d233413ee42b37266a77c107acc736` specifies the new
profile/consumer tests and compiled implementation-absent scalar helpers. The
profile/scalar Go tests, nine strict Python consumer tests and interrupted cleanup
regression fail against the absent/old implementation, then pass after the change.
Additional regression coverage distinguishes nested timeout from actual original
caller expiry, preserves scalar snapshots after cancellation/variable mutation,
and rejects failed partial observations. These are portable fixture tests, not
fabricated failures of unchanged product behavior or native acceptance.

Run actual-title `make review` from a signed clean candidate. Shared tooling
selects conservative full portable race/fuzz coverage. Then run the unchanged
`policy-startup-lifecycle` selector on disposable 15.1 followed by 14.5 with the
new adopted profile and one checksum-bound binary. Stop after the first failure;
no automatic retry, second target, budget change or alternative CPU profile.
Both complete release vectors and independent guest/host teardown are required.
All nineteen preceding policy receipts remain immutable. Final evidence-only
changes need a fresh baseline with verified exact-input deep reuse where valid.

## Qualification boundary

Native results will be recorded against their actual executed source, compiler,
artifact, lab profile and log/result digests. At this implementation stage no new
native result is claimed. Historical failures and skipped/unverified cleanup
remain preserved; later clean teardown cannot retroactively qualify them.

This bounded fixture does not close issues 305/139. Mixed/fault history,
credential/lineage/concurrency/census/publication/context/capacity families,
configured startup/workers/backend/global launch, manual browser/client,
hardware/performance and release acceptance remain separate. Controlled closed
seeds are fixture setup, not worker shutdown provenance. Runtime writes remain
disabled; no service, deployment or external infrastructure is enabled.
