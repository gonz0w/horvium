# Retained-policy child operation budgets

Status: fixture correction candidate; native qualification pending.
Date: 2026-10-10. Owner: Cam (`@gonz0w`).
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305) and
[M2-RUNTIME-002](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted design](../../milestones/M2/retained-policy-startup.md),
[PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).

## Observed failure and correction

Merged PR 308 source `ad38166a66c233c078635c2e88dfcb32829af621`
completed the FreeBSD/amd64 15.1 UFS lifecycle parent in 242.33 seconds.
The ZFS parent then failed at its clean child's `closed` assertion after
120.02 child seconds (154.71 parent seconds). The underlying close error was
not included in that assertion's output. The timing coincides with the shared
two-minute caller context; it does not independently prove the product's close
error or qualify any part of this incomplete run. Guest/device/host cleanup,
base/runtime preservation and supervisor teardown passed.

The child previously charged construction, observation and retirement to the
same two-minute context. These are separate owner calls with their own product
operation bounds. Give each independent fixture call its own caller context:
construction retains two minutes, while observation/finish and denial probes
have 90 seconds. Every caller context is shared across the entire invoked
operation, including all policies, collectors and publication checks. There is
no per-policy renewal or retry. The original 150-second Go child test timeout
continues to bound the complete lifecycle in one normally waited process.

This changes the fixture's effective cumulative context allowance from 120
seconds to the existing 150-second whole-child limit. It does not claim that the
old shared budget remains unchanged. Parent (480 seconds), child test (150
seconds), each product operation (90 seconds), policy/retained process (34 minutes
each) and host controller (7200 seconds) limits remain unchanged. Production
contexts, fixed native producers, strict seed publishers, process registration,
publication semantics and persisted formats are unchanged.

Log each independent call's stage, elapsed time and returned error. Parent
failures retain that bounded child output, making operation failures distinct
from fixture context exhaustion. All original mode assertions and the 16 KiB
child-output limit remain required. Copy, decoded, retired and UID denial probes
receive fresh unexpired contexts; they cannot pass merely because an earlier
operation exhausted the caller context.

## Regression qualification and limits

The first independent-context candidate
`fe23046685a0c54f2e1ae536ee220a6af484939a` remained unqualified on
FreeBSD 15.1: UFS construction succeeded in 99.62 seconds, observation succeeded
in 24.45 seconds, copied/decoded probes denied immediately, then the child hit
its unchanged 150-second test timeout during finish. Fresh contexts alone did
not resolve the aggregate execution time. Owned guest/device/artifact removal,
base/runtime preservation and supervisor teardown were verified; no incomplete
in-guest parent result is promoted to acceptance.

The next fixture candidate pins `GOMAXPROCS=1` only for the normally waited
policy child and checks that setting inside that child. This bounds Go execution
to one thread to test whether emulation scheduling overhead can be reduced.
It does not change the two-vCPU QEMU profile, parent/retained test processes,
product configuration or any deadline. OS processes, actual root credentials,
advisory locks and fixed producers remain real. This candidate must demonstrate
complete native passes before the lab setting is treated as effective; no
throughput, parallel execution or commercial performance claim follows.

The actual failing merged-source run is the regression observation. Qualify the
corrected artifact with the complete `policy-startup-owner` suite, serially on
both pinned disposable FreeBSD/amd64 15.1 and 14.5 releases. Require the same
artifact hash, both normally completed bounded phases, ten parents and 580
controls per release, exact histories, original locks and independent complete
teardown. Partial or historical passes do not accept this change. Portable
review and cross-compilation alone cannot establish kernel or filesystem behavior.

This fixture correction adds no product feature. Expanded multi-policy,
mixed/same/other-pool history, concurrency, policy-aware census/object and bounds
acceptance remains pending in issue 305. Controlled closed seeds do not qualify
shutdown after admitted work. Configured runtime, worker/backend assembly,
native launch, client/browser/hardware/performance and release gates remain open;
runtime writes stay disabled. Earlier failed receipts remain authoritative in the
unchanged [correction manifest](retained-policy-root-manifest.json).
