# Original root review of two retained native policies

Status: bounded candidate qualified on both pinned releases; maintainer review pending.
Date: 2026-10-11. Owner: Cam (`@gonz0w`).
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305) and
[parent M2-RUNTIME-002](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted design](../../milestones/M2/retained-policy-startup.md),
[PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
Requirements: M2-RUNTIME-002, M2-STATE-001, M2-OWN-001, M2-DRIFT-001,
M2-PRIV-001 and M2-NATIVE-001.

## Bounded outcome and authoritative seeds

Extend the [proven two-mirror fixture](retained-policy-native-facts.md) with an
optional callback before its existing child and unchanged 21 fact/data/cleanup
controls. Historical standalone behavior remains required on both releases.
Select the same scoped seven-device profile: four unique disks form two actual
mirrors; one independently verified, root/mount/swap/pool-excluded historical
spare supplies disposable metadata backing. Duplicate identity is not data-pool
enrollment; it is permitted only for this explicitly owned metadata fixture.

Use actual UFS without softupdates and ZFS metadata domains. ZFS has distinct
journal/ledger datasets; UFS has distinct protected namespaces. The existing
fixture-only `SeedExecutionPolicyFixture` creates fresh domains using original
commissioning, publishes both canonical policies through the original pending
journal/ledger/confirmed-census protocol, then publishes the strict acknowledged
closed successor and confirms it under both original leases before release.
No raw JSON edits, reconstructed acknowledgements, factory reset or product guard
relaxation. This controlled seed is fixture setup, not worker shutdown or proven
native execution provenance. Normal product binaries exclude the seed bridge.

Two seeds per filesystem: `clean` with both actual member/scope baselines, and
`last-member` with only the lexically last ownership policy's copied member serial
changed. Sort by actual ownership filename, matching the complete reader order.
Never alter native observations. A normally waited, exact-binary root child uses
the original process registry and `commissionExecutionOwner`, with actual EUID
and a private fixture composition of `ReviewExecutionPolicyStartup`, original
Observe/Close and a denied finish. Fixed backing, system, boot and policy
collectors supply every positive fact. A trace-only wrapper verifies actual kernel
locks, visitation, zero eligibility and the shared caller deadline; it returns
native results unchanged. This fixture composition is not a new product root API.

The valid child must review and independently observe both policies, deny copied/
decoded owners and finish, preserve every journal/ledger filename and byte, release
both actual locks and leave process registration consumed. The last-member child
must visit the first policy before the invalid last policy, return no owner,
preserve exact history, release both locks and retain consumed registration.
No activation, idle retirement publication, worker or launch capability is exposed.

## Inventory and bounds declared before native execution

The `policy-startup-review` runner uses one checksum-bound tagged host binary in
two normally completed, serial phases:

1. Facts: `^TestNativeMultiplePolicyFacts$`, eighteen-minute test process.
2. Review: `^TestNativeMultiplePolicyReview(UFS|ZFS)$`, eighteen-minute process.

Each parent retains eight minutes; every root/fact child retains 150 seconds and
GOMAXPROCS=1. Each review or observation call has one 90-second caller context
across the complete two-policy/native collection. No per-policy renewal. Separate
calls receive separate contexts, within the unchanged cumulative child timeout.
All child output remains bounded to 16 KiB. Both pinned FreeBSD/amd64 15.1 and
14.5 releases must pass serially. No product or existing suite deadline changes.

| Evidence | Required inventory |
|---|---|
| Standalone fact parent | All existing 21 controls |
| Each review parent's retained fact/data fixture | Same 12 child and nine parent controls (21) |
| Clean root child | registered, original_observation, both_policies, both_leases, shared_deadline, zero_eligibility, copied_denied, decoded_denied, finish_denied, finish_terminal, unchanged_journal, unchanged_ledger, release_once, locks_released, registration_consumed (15) |
| Last-member root child | denied, last_policy, both_leases, shared_deadline, zero_eligibility, unchanged_journal, unchanged_ledger, locks_released, registration_consumed (9) |
| Root parent | For each mode: seed_closed, child_waited, child_inventory, exact_history, leases_released (10); domain_teardown, mounts_restored (2) |

Require three distinct completed parents, 135 controls and seven normally waited
children per release. The standalone fact phase cannot be substituted for either
root parent. Missing/extra/duplicate/failed/skipped parents, vectors, phases or
host device cleanup reject the whole run. Independently require metadata backing/
directory teardown, exact data sentinels, original-GUID pool cleanup, input removal,
restored mounts/swap, device/guest/key/artifact/supervisor cleanup and base/runtime
preservation. Failed runs remain unqualified even when host teardown succeeds.

## Review and limits

Use failing-first strict evidence-consumer regressions and signed clean-source
`make review` for `make check lint test secrets`; shared qualification tooling
requires its conservative full portable fallback. Native runs remain serial and
separate from deep portable testing. Record source/base/title, input fingerprint,
compiler/artifact and original logs/records, preserving all seven earlier retained-
policy manifests byte-for-byte. Normal builds exclude these tagged fixtures.

This qualifies only the private original policy-aware root **review/observation**
composition and one last-member rejection. Original multi-policy activation and
unused retirement, mixed history, other drift/lineage/credential/concurrency/census/
publication/context/capacity families, configured startup, workers/backend/global
launch, browser/client/hardware/performance and release gates remain separate.
There is no anti-rollback or commercial scale claim. Existing single-policy root
and synthetic storage receipts retain their original scope. Product behavior is
unchanged; issue 305 and parent 139 stay open and runtime writes stay disabled.

## Executed candidate

Signed source `88b837f026332e369fd983cd5c35dc98960d99eb` passed both pinned
FreeBSD/amd64 15.1 and 14.5 gates with the identical checksum-bound artifact.
Each release completed all three parents, 135 controls and seven normally waited
children, including independent metadata and fact/data teardown. The
[candidate receipt](retained-policy-native-review-manifest.json) records the
executed source/base, compiler/artifact identity, parent times, original log/record
hashes and control inventories. All seven prior retained-policy receipts remain
byte-identical; their historical failed/skipped limits remain intact. No native
attempt in this bounded gate failed.

Required signed clean-source `make review` passed 200 tooling tests, ordinary Go
tests, all 11 race packages and 54 fuzz targets with supervised teardown. The
receipt-only followup requires fresh baseline/signature/DCO/title checks; it may
reuse deep evidence only with identical verification inputs. The preceding limits
remain binding; these candidate results do not close issue 305 or parent 139.
