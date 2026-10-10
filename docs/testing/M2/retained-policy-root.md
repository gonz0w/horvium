# Fixed-root retained-policy startup qualification

Status: fixture correction candidate; native qualification pending.
Date: 2026-10-10. Owner: Cam (`@gonz0w`).
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305) and
[M2-RUNTIME-002](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted design](../../milestones/M2/retained-policy-startup.md),
[PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).

## Failure and bounded correction

Merged PR 307's source `3de0cd25e2689abcc62f2b895e9870d35664b5dc`
failed its FreeBSD/amd64 15.1 fixed-root UFS parent at 480.24 seconds while
commissioning the final `uid` fixture: `context deadline exceeded`. The eight
modes shared one eight-minute parent. The run remains unqualified: no ZFS or
retained-parent completion is inferred. Host, device, base and runtime teardown
passed. Earlier signed red and successful storage receipts remain unchanged in
the [implementation manifest](retained-policy-startup-manifest.json).

Split lifecycle (`clean`, `clean-again`, `late`, `uid`) and denial (`legacy`,
`member`, `scope`, `missing-pool`) into separately owned parents on each
filesystem. Each parent keeps the eight-minute deadline; child context remains
two minutes, child test timeout 150 seconds, operation bound 90 seconds and whole
suite bound 34 minutes. Every original mode assertion stays required. Independent
data sentinel, domain teardown and mount restoration assertions run for **each**
group. Fixture construction still uses original fresh commissioning, journaled
ownership publication and strict acknowledged closed successor publication. No
publisher, seed bridge, native producer, production behavior or format changes.

## Inventory declared before execution

| Actual parent | Required controls |
|---|---|
| `TestNativePolicyStartupLifecycleOwnerUFS` | 51 |
| `TestNativePolicyStartupLifecycleOwnerZFS` | 51 |
| `TestNativePolicyStartupDenialOwnerUFS` | 39 |
| `TestNativePolicyStartupDenialOwnerZFS` | 39 |
| Four unchanged original review/closed-startup UFS/ZFS parents | 394 total |

All eight parents and 574 controls are required per release. The exact selector
is recorded in [the implementation qualification](retained-policy-startup.md).
`tools/check-policy-startup-evidence.py` requires each group's distinct controls
and independent teardown, then delegates retained evidence to its unchanged
historical checker. Missing groups, old two-parent output, unknown/duplicate
parents, skipped/failed controls and incomplete host-device cleanup cannot qualify.
Synthetic parser tests first failed against the old checker, then passed with the
new complete group requirements; these parser observations are not native proof.

## Required candidate review and limits

Run the checksum-bound `policy-startup-owner` gate serially on disposable pinned
FreeBSD/amd64 15.1 and 14.5. Record exact executed source, binary/compiler hashes,
original record/log hashes, every failed candidate and independent owned cleanup.
Run `make check lint test secrets` through signed clean-candidate `make review`;
shared checker/selector changes require the conservative full portable fallback.
No enlarged timeout or reused old native receipt accepts this correction.

Actual fixed native producers supply the positive pool, filesystem, member,
provider and system-use facts. The late-drift denial modifies a fact only after
actual native collection. The tagged controlled closed seed remains fixture setup,
not shutdown after admitted work. Normal product binaries exclude the bridge.

This records only the existing single-policy root suite. Expanded multi-policy,
mixed/same/other-pool history, concurrency, policy-aware census/object and context
bounds remain outstanding in issue 305. Its parent runtime gate, configured
startup, retained-job recovery, workers, launch permission, public operations,
client/browser/hardware/performance and release acceptance remain open. Runtime
writes remain disabled. No service, deployment, hosted automation or dependency
is added.
