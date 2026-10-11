# Original root lifecycle with two retained native policies

Status: bounded inventory declared before execution; qualification pending.
Date: 2026-10-11. Owner: Cam (`@gonz0w`).
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305) and
[parent M2-RUNTIME-002](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted design](../../milestones/M2/retained-policy-startup.md),
[PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
Requirements: M2-RUNTIME-002, M2-STATE-001, M2-OWN-001, M2-DRIFT-001,
M2-PRIV-001 and M2-NATIVE-001.

## Bounded outcome and seed authority

Extend the [original two-policy review fixture](retained-policy-native-review.md)
with two actual fixed-factory lifecycle parents, on UFS without softupdates and
ZFS with distinct journal/ledger datasets. Reuse the declared seven-device
profile, original excluded metadata spare, independent actual mirrors and exact
member/scope baselines. Extract only the shared disposable metadata setup/cleanup
harness; preserve every existing review and fact control. Qualify all affected
parents with one new checksum-bound artifact on both pinned releases.

The existing tagged `SeedExecutionPolicyFixture` creates one original closed
policy-only seed per lifecycle parent through original commissioning, pending
journaled publication of both policies, confirmed census, strict closed successor
and original release. No JSON editing, receipt reconstruction, registry reset,
synthetic positive native facts or guard relaxation. The initial closed seed is
controlled fixture setup, not worker shutdown or admitted execution provenance.

A normally waited child calls the actual existing
`activateNativeClosedExecutionOwner`, with its original process registry, actual
root credentials and fixed backing/system/boot/policy producers. No alternative
factory or wrapped producer. Require exact active generation+1/new owner/current
boot and unchanged ledger/other journal files; independently observe, refuse
copied/decoded/nil/cancelled authority, then publish the exact unused generation+2
closed successor. Entered finish retires observation/finish; release retains
idempotence, releases both actual locks and cannot reset registration.

A second normally waited child starts from that same first child's original
closed successor, without reseeding or replacing either namespace. It must perform
the same full cycle with a new owner and two additional generations. The parent
independently checks each complete successor, original ledger bytes/names and
released locks. This is controlled unused metadata retirement, not worker draining.

## Initial inventory and limits declared before native execution

The new focused `policy-startup-lifecycle` runner uses one tagged host binary in
three serial, normally completed eighteen-minute phases:

1. Facts: `^TestNativeMultiplePolicyFacts$`.
2. Review: `^TestNativeMultiplePolicyReview(UFS|ZFS)$`.
3. Lifecycle: `^TestNativeMultiplePolicyLifecycle(UFS|ZFS)$`.

Every parent retains eight minutes, every child 150 seconds and GOMAXPROCS=1.
Construction uses the accepted two-minute caller budget across review/activation;
observation and finish each use an independent 90-second caller budget. Every
product operation retains its own existing 90-second bound, shared across all
policies/collectors/publication checks. No per-policy renewal or retry. The
whole-child cumulative timeout remains 150 seconds; no deadline is enlarged.
Separate denial probes use fresh unexpired contexts. Output remains bounded to
16 KiB; cleanup retains an independent 30-second budget. The existing two-hour
host/controller bound remains unchanged. Run releases serially, separate from
deep portable checks. A timeout remains an unqualified candidate.

| Required evidence | Controls |
|---|---|
| Retained standalone fact and UFS/ZFS review parents | Same three parents, 135 controls and seven children |
| Each lifecycle child, `first` and `restart` | registered, active_successor, active_private_record, current_boot, leases_held, nonjournal_unchanged, original_observation, observation_preserves_history, copied_denied, decoded_denied, cancelled_no_effect, closed_successor, ledger_unchanged, close_retains_leases, retired_observation, retired_finish, release_once, locks_released, registration_consumed (19 each) |
| Lifecycle parent | seed_closed (1); for each child: child_waited, child_inventory, exact_closed, ledger_preserved, leases_released (10); domain_teardown and mounts_restored (2) |
| Each lifecycle parent's retained fact/data fixture | Same 12 child/nine parent controls (21) |

Require five completed parents, **279 controls and thirteen normally waited
children per release**: each lifecycle parent has 51 root controls plus 21
fact/data controls. All original phases, controls and independent cleanup remain
required; no stale standalone receipt substitutes for an affected parent. Missing,
extra, duplicate, skipped, failed or misplaced phases/parents/control vectors or
incomplete host cleanup reject the whole gate. Require metadata backing/directory
teardown, exact data sentinels, original-GUID pool removal, input removal, restored
mounts/swap and device/guest/key/artifact/supervisor cleanup, preserving base/runtime.

## Review and remaining acceptance

Write failing-first strict evidence-consumer tests. These new native tests qualify
existing product behavior; no new product implementation or fabricated behavioral
red is claimed. Run signed clean-source `make review` for required
`make check lint test secrets` and the conservative full portable fallback for
shared qualification tooling. Record actual source/base/title, input fingerprint,
compiler/artifact/log/record hashes and any failed/skipped attempt. Preserve all
eight earlier retained-policy manifests byte-for-byte. Normal builds exclude
these fixture files and the existing seed bridge.

This covers valid original fixed-root two-policy activation, observation and unused
retirement, plus one subsequent clean process restart. The retained review phase
keeps its own last-member refusal scope. Remaining multi-policy drift/credential/
lineage/concurrency/census/publication/context/capacity families, mixed-history
fixed-root composition, configured startup, workers/backend/global launch and
browser/client/hardware/performance/release gates remain separate. No commercial
capacity, physical crash, worker-shutdown or anti-rollback claim. Product behavior
and runtime write denial remain unchanged; issues 305 and 139 remain open.

## Failed initial run and bounded affinity candidate

Signed source `c4591db14c44a61c33eb5e3de0f9624cf332b988` failed its FreeBSD
15.1 UFS lifecycle child during construction: `context deadline exceeded` at
120.02 seconds, before returning an owner. Its UFS lifecycle parent failed at
211.28 seconds. Standalone facts and both retained reviews completed at 25.16,
255.88 and 333.14 seconds respectively, but the whole gate remains unqualified.
No ZFS lifecycle or 14.5 attempt completed; do not infer their result. Host/device/
artifact/supervisor teardown and base/runtime preservation passed. No incomplete
in-guest parent teardown is accepted. Portable review was not run for this failed
native candidate; earlier receipts remain unchanged.

Declare a fixture scheduling experiment before its execution: use FreeBSD base
[cpuset(1)](https://man.freebsd.org/cgi/man.cgi?query=cpuset&sektion=1) to restrict
only the new lifecycle phase's test processes and inherited native command
children to emulated CPU 0. The two-vCPU QEMU profile, retained facts/review phases,
GOMAXPROCS=1 root children, original fixed factory/producers and every existing
time/byte/record bound remain unchanged. Do not modify system-wide CPU sets or
product scheduling. Each lifecycle parent and each normally waited root child
queries its own actual kernel affinity; any mask other than exactly CPU 0 denies.
This does not establish causality for the failed run, production performance,
parallel execution or commercial capacity. Qualification requires complete actual
passes of the corrected gate, not an assumption that affinity improves timing.

The corrected lifecycle child inventory adds `cpu_affinity` to all original
nineteen controls (20 per child); each lifecycle parent retains all old controls
and adds the two child affinity assertions (53 root controls plus 21 fact/data
controls = 74). Corrected whole-gate inventory: **five parents, 283 controls and
thirteen normally waited children per release**. Require exactly one affinity
phase marker inside the normally completed lifecycle phase, checksum-bound
`/usr/bin/cpuset` provenance, actual kernel affinity and independent teardown.
Every existing case, assertion and bound stays required. The failed initial
source and skipped second-release/ZFS lifecycle remain in the final receipt.
