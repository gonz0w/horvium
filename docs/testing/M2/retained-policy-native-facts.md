# Native facts for two independent retained-policy pools

Status: fixture inventory declared before execution; qualification pending.
Date: 2026-10-11. Owner: Cam (`@gonz0w`).
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305) and
[parent M2-RUNTIME-002](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted startup design](../../milestones/M2/retained-policy-startup.md),
[PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
Requirements: M2-RUNTIME-002, M2-OWN-001, M2-DRIFT-001, M2-NATIVE-001.

## Bounded prerequisite and construction

Prove the existing fixed policy fact producer and validators against **two actual
independent data pools**, before extending the original root lifecycle. Use a scoped seven-device disposable QEMU profile on pinned FreeBSD/amd64
15.1 and 14.5; the existing five-device profile remains unchanged. Resolve actual root, mount, swap and imported-pool use before mutation.
Require all seven declared 64 MiB disks and their exact serial/geometry classes.
Retain the five historical disks, adding only `HVM-FACTS-512` (512 logical /
4096 physical) and `HVM-FACTS-4K` (4096 / 4096). Pair each new disk with its
matching historical unique disk in an independent two-leaf mirror. Both mirrors
meet the existing accepted topology restriction. Duplicate/missing-serial fixture
disks remain unused. This topology is a fixture, not
an operator layout recommendation or commercial capacity claim.

Independently capture each new pool/filesystem GUID, actual leaf GUID, provider,
serial provenance and geometry. Encode each baseline with the existing ownership
codec into private immutable fixture inputs **outside any ledger or journal**.
An independently executed, normally waited root child reads those baselines and
calls `newExecutionStartupPolicySource` for each pool. Every positive observation
comes from actual fixed pool/property, provider and system-use collectors. Use
existing member/scope and system exclusion validators. Change only a copied
last policy's serial or scope for denial probes; never forge native observations.
Recheck original policies, record bytes and separate data sentinels afterward.
These baselines grant no authority and are not enrolled durable product history.
There is no commissioning seed, original lease owner or journal publication here.

## Failed initial profile and bounded correction

Initial source `29690051b7ca6534cf9d419ae51c96f67e28d831` passed portable
review but failed FreeBSD 15.1 parent setup at 2.04 seconds: the existing fixed
property collector rejects a single-leaf pool. Inspection confirms
`VerifyPoolTopology` accepts homogeneous mirrors or RAIDZ2 groups only. No child
completed; the run remains unqualified. Host/device/base/runtime and supervisor
teardown passed; incomplete in-guest parent cleanup is not promoted to acceptance.
The 14.5 attempt was not run for this failed source. Preserve that receipt.

Keep the production restriction and correct only the fixture topology. The scoped
seven-disk profile and complete independent mirror controls are declared before
corrected native execution. Existing five-disk fixtures retain their defaults;
no deadline, runtime requirement or hardware support claim changes. Failing-first
profile regressions require separation of manifests, default-profile preservation,
unknown-profile refusal and removal of all seven owned disks on preservation
failure. The actual failed native candidate is the single-leaf regression evidence.

## Inventory and bounds declared before native execution

The exact selector is `^TestNativeMultiplePolicyFacts$`. One parent and its one
normally waited `^TestMultiplePolicyFactsChild$` child are required per release.
The child runs with `GOMAXPROCS=1`, a 150-second test timeout and one shared
90-second caller context across both collections and denial probes. Contexts are
not renewed per policy. The parent keeps eight minutes; the test process has
18 minutes. These are this new prerequisite's bounds, not enlarged deadlines for
the existing root gate or product. Child output is bounded to 16 KiB.

| Marker | Required controls |
|---|---|
| `POLICY_FACTS_CHILD_RESULT` (12) | native_baselines, distinct_pools, distinct_members, first_pass, second_pass, last_member_denied, last_scope_denied, zero_eligibility, no_policy_mutation, records_preserved, bounded_context, cancelled_zero |
| `POLICY_FACTS_PARENT_RESULT` (9) | child_waited, child_inventory, policy_bytes, sentinel_first, sentinel_second, pools_removed, mounts_restored, swap_restored, inputs_removed |

All 21 controls must pass in the completed parent. Missing, duplicate, skipped,
failed, extra or unfinished parents/markers cannot qualify. Host qualification
also requires the checksum-bound binary and exactly one successful device-cleanup
marker. Cleanup verifies each pool's original GUID before destruction, absence
of both pools and private inputs, unchanged mounts/swap, and owned guest/device/
key/artifact/supervisor removal. Failed runs remain unqualified even if teardown
succeeds. Synthetic failing-first parser tests establish rejection behavior only.

## Review and remaining scope

Run required `make check lint test secrets` through signed clean-source
`make review`; shared qualification tooling changes require its full portable
fallback. Native runs are serial, never concurrent with race/fuzz qualification.
Record exact source/base/title, compiler/artifact and original log/record hashes.
Preserve all six earlier retained-policy manifests unchanged.

No production behavior, public API, persistent format, helper registration,
configuration or service changes. Normal binaries exclude these tagged fixtures.
This prerequisite does not qualify multiple-policy original review, activation,
observation or retirement, lease/census/concurrency/publication faults, workers,
launch permission, browser/client/hardware/performance or release acceptance.
The existing single-policy root gate and its historical limitations remain
unchanged. Issue 305 and its parent remain open; runtime writes remain disabled.
