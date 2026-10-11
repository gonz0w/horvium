# Fixed-root retained-policy startup qualification

Status: fixture correction merged in [PR 308](https://github.com/gonz0w/horvium/pull/308);
the executed layout remains unqualified; child-context correction is pending.
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

The first split candidate `46d3a26b6526f7b4d7c59b9c9a7aee1d05237e56`
completed all four policy parents on FreeBSD 15.1: lifecycle UFS 437.67 seconds,
lifecycle ZFS 474.53 seconds, denial UFS 140.32 seconds and denial ZFS 161.35
seconds. These consumed over twenty minutes before retained acceptance began.
Its retained ZFS parent also passed at 409.58 seconds. These individual
observations do not qualify an incomplete whole run or the final revised layout.
The combined run then exhausted its unchanged 34-minute process limit during
retained UFS. The remaining parents did not complete. Host/device cleanup and
base/runtime preservation passed; this whole run is unqualified.
The initial portable count assertion incorrectly expected 54/36 rather than
51/39 controls; its failed baseline remains attached to that source. Corrected
source `5d7301a04a58ce9de96198fbf254cc368b40c249` passed the full portable review.

Split the final fixtures into clean lifecycle (`clean`, `clean-again`), revocation
(`late`, `uid`) and invalid ownership (`legacy`, `member`, `scope`, `missing-pool`)
on each filesystem. Every original mode assertion stays required. Independent
data sentinel, domain teardown and mount restoration assertions run for **each**
group. Fixture construction still uses original fresh commissioning, journaled
ownership publication and strict acknowledged closed successor publication. No
publisher, seed bridge, native producer, production behavior or format changes.

## Inventory and bounds declared before final execution

| Actual parents, one each on UFS and ZFS | Required controls per parent |
|---|---|
| `TestNativePolicyStartupLifecycleOwnerUFS` / `ZFS` | 31 |
| `TestNativePolicyStartupRevocationOwnerUFS` / `ZFS` | 23 |
| `TestNativePolicyStartupDenialOwnerUFS` / `ZFS` | 39 |
| Four unchanged original review/closed-startup UFS/ZFS parents | 394 total |

All ten parents and 580 controls are required per release. Each parent keeps its
eight-minute deadline; child context remains two minutes, child test timeout 150
seconds and runtime operation bound 90 seconds. Policy and retained cohorts run
serially in the same owned guest, using the same checksum-bound binary, in two
normally waited test processes. Each process keeps the 34-minute limit. Total
test time can therefore reach 68 minutes; this deliberately replaces the prior
single-process whole-suite limit. The existing host SSH/controller bound stays
two hours. No runtime, byte, record, parent or child bound is enlarged.

The exact selectors are recorded in
[the implementation qualification](retained-policy-startup.md).
`tools/check-policy-startup-evidence.py` requires each group's distinct controls
and independent teardown, exact parent placement within both completed bounded
phases, then delegates retained evidence to its unchanged historical checker.
Missing groups/phases, old two/four-parent output, unknown/duplicate parents,
skipped/failed controls and incomplete host-device cleanup cannot qualify.

Synthetic parser regressions first failed against the previous checker, then
passed with the complete group/phase requirements. A separate failing regression
proved that a policy parent could previously be moved into the retained phase;
the revised checker rejects that placement. Selector coverage independently
requires six policy and four retained parents, disjoint and collectively complete.
Parser observations and cross-compilation are not native qualification proof.

## Required candidate review and limits

Run the final checksum-bound `policy-startup-owner` gate serially on disposable
pinned FreeBSD/amd64 15.1 and 14.5. Record exact executed source, binary/compiler
hashes, original record/log hashes, failed candidates and independent cleanup.
Run `make check lint test secrets` through signed clean-candidate `make review`;
shared checker/selector changes require the conservative full portable fallback.
Earlier or partially completed native results cannot accept the final layout.

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

## Recorded final correction review

The [sanitized correction manifest](retained-policy-root-manifest.json) preserves
both unqualified native runs, their completed parents and exact source/compiler/
artifact/log/record hashes, the failed count baseline and parser regressions.
Source `d6555e3bba46c5d6617b412d99f5ec065b91399a`, base
`3de0cd25e2689abcc62f2b895e9870d35664b5dc`, title
`test(host): bound retained-policy startup qualification` passed `make review`:
`make check lint test secrets`, eleven race packages and all 54 portable fuzz
targets, signed/DCO/title verification and supervised teardown. The fixture
cross-build passed using pinned Go 1.27.1 for FreeBSD/amd64, ordinary build
selection excluded the bridge, and the temporary compiled artifact was removed.

The six-parent policy/two-phase layout ran on merged source
`ad38166a66c233c078635c2e88dfcb32829af621` on FreeBSD 15.1. Its UFS
lifecycle parent passed at 242.33 seconds; the ZFS clean child failed its close
assertion after 120.02 seconds. That whole run remains unqualified and its other
parents did not complete. Complete owned teardown passed. See the
[child operation budget correction](retained-policy-child-budget.md) for the
observed timing, diagnostic limitation and explicitly changed fixture context
scope. The corrected artifact still needs complete native qualification on both
releases; cross-compilation and partial passes do not qualify it. Issue 305 and
its parent remain open, including the separately outstanding expanded families.
