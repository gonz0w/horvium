# Adopted original lifecycle caller fixture

Status: fixture implementation reviewed locally; first-target native qualification failed.
Both complete release gates remain pending.
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

## Actual candidate and result

The [candidate receipt](retained-policy-lifecycle-caller-manifest.json) binds signed
implementation source `ea50080bec1a66bab614ae000e45b2b71ee08217`, adoption base
`d5bdf24d4ef5df480ae960b985365de8610cc274`, pinned compiler and native binary
`e816430ce0bbfd40e03838e345e97f656d3c4ea29a5db233c587409de7f302d4`,
QEMU/firmware/base hashes and original profile. Normal FreeBSD runtime file
selection excludes these fixture helpers; tagged compilation passed.

Actual-title `make review` passed `make check lint test secrets`, all 228 tooling
tests and ordinary Go tests, followed by the shared-tooling full fallback:
**eleven race packages and all fifty-four fuzz targets**, with two clean supervised
steps. The receipt records signed red source and private red/green log digests;
portable green includes six Go parents, nine strict consumer methods, interrupted
cleanup and all twenty-five unchanged diagnostic methods. The later evidence-only
head requires a fresh baseline; exact-input deep reuse must be verified against
this successful original execution, not assumed.

| Actual FreeBSD/amd64 15.1 parent | Result | Seconds |
| --- | --- | ---: |
| Standalone multiple-policy facts | Passed | 29.02 |
| Original UFS review | Passed | 320.26 |
| Original ZFS review | Passed | 249.78 |
| Original UFS lifecycle | Failed at `first_child_waited` | 201.58 |
| Original ZFS lifecycle | Not attempted after failure | — |

The strict facts/review prefix completed three parents, **135 controls and seven
normally waited children**. The full release consumer rejects the run. No complete
lifecycle control vector or 293-control gate is established.

The failed first child's actual startup scalar records **126.708043619 seconds**,
a selected/unchanged 180-second deadline, `operation_error=deadline`,
`caller_error=none` and `owner_present=false`, captured before deferred
cancellation. Child failure followed at 126.75 seconds. This returned operation
deadline error occurred while the original caller had not expired; this
candidate remains failed under the adopted wider caller. The row does not identify which
nested context or native probe failed, or establish performance/emulation/capacity
causality. No 390-second whole-child, 1200-second parent or 44-minute phase timeout
is shown. No final independent metadata readback establishes the failed startup's
publication state.

Only that one actual completed scalar row is retained. Later first-cycle stages,
restart and ZFS lifecycle are absent, not invented. Under the adopted stop rule,
**14.5 was not attempted**. No retry, reseed, changed limit, guard removal, alternate
producer or CPU profile followed the failure. Two bounded read-only inspections
of existing guest log progress did not alter fixture source/state or run new
native collectors; the final source-bound transcript supplies the evidence.

## Preservation, teardown and next work

All nineteen prior policy receipts and six original trace/separate diagnostic
source/consumer inputs remain byte-identical. Independent host/device/image/key/
executable and supervisor cleanup, base/runtime/device-backing preservation and
owned-artifact removal passed. The incomplete lifecycle guest vectors remain
**unverified**; independent host cleanup does not satisfy them. `make lab-stop` and
`make lab-status` verified all 105 owned sessions stopped before final evidence
review, which has its own supervisor/teardown. Later clean teardown cannot
retroactively qualify historical failed/skipped/unverified candidates.

Next bounded work is source/evidence review of this narrower operation timeout
before any new experiment, product limit, collector instrumentation or accelerated
lab profile is proposed. Both complete native release gates remain pending.

This bounded fixture does not close issues 305/139. Mixed/fault history,
credential/lineage/concurrency/census/publication/context/capacity families,
configured startup/workers/backend/global launch, manual browser/client,
hardware/performance and release acceptance remain separate. Controlled closed
seeds are fixture setup, not worker shutdown provenance. Runtime writes remain
disabled; no service, deployment or external infrastructure is enabled.
