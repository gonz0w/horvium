# Original two-policy lifecycle qualification follow-up

Status: first-target native qualification failed; both complete release gates remain pending.
Date: 2026-10-11. Owner: Cam (`@gonz0w`).
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305) and
[parent 139](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted retained-policy design](../../milestones/M2/retained-policy-startup.md),
[fixture budget amendment](../../milestones/M2/retained-policy-fixture-budgets.md),
[PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md) and
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
Requirements: M2-RUNTIME-002, M2-STATE-001, M2-OWN-001, M2-DRIFT-001,
M2-PRIV-001 and M2-NATIVE-001.

## Candidate and unchanged gate

After acceptance of the separate [caller-envelope diagnostic](../../research/M2/policy-startup-envelope.md)
in [PR 329](https://github.com/gonz0w/horvium/pull/329), run the original
`policy-startup-lifecycle` gate from merged source
`c9b01533a8c3e343c2cc9dd176ce35f12ad9aed4`. Its tree exactly matches signed,
DCO-reviewed head `f5c4b7944b2dd30e2429bda89c5be46e17e694d7`.
The [new candidate receipt](retained-policy-lifecycle-qualification-manifest.json)
binds executed source, tree, pinned compiler, binary, QEMU/firmware/base hashes,
original profile, private log/result digests and cleanup observations.

Keep facts/review/lifecycle phase selectors, five parents, **289 controls and
thirteen normally waited children per release**. This run uses one qemu64/TCG CPU,
6 GiB and seven owned disposable devices, with actual CPU 0 lifecycle affinity.
Child/parent-work/lifecycle-phase limits remain 330 seconds / 1,080 seconds /
40 minutes. Startup uses the original shared **120-second caller** across review
and activation. Each original product operation remains 90 seconds; observation
and finish retain independent 90-second callers. Native, output, cleanup, remote
and supervisor limits remain unchanged. The separate 180-second diagnostic caller
does not apply to this gate. No product, fixture, producer, guard, registry,
profile, service or dependency change accompanies this follow-up.

## Actual first-target result

| FreeBSD/amd64 15.1 parent | Result | Seconds |
| --- | --- | ---: |
| Standalone multiple-policy facts | Passed | 13.57 |
| Original UFS review | Passed | 157.82 |
| Original ZFS review | Passed | 180.70 |
| Original UFS lifecycle | Failed at `restart_child_waited` | 372.68 |
| Original ZFS lifecycle | Not attempted after failure | — |

The strict retained facts/review consumer accepts only the completed prefix:
three parents, 135 controls and seven normally waited children. The full consumer
rejects the run. No complete lifecycle vector, five-parent gate or 289-control
release acceptance is established.

The failed child's own log identifies mode `restart`, startup
`context deadline exceeded` after **120.02 seconds**, and child failure at
120.05 seconds. No 330-second whole-child timeout occurred. The parent failed
when the second child's successful-wait criterion was false.

**Source-order inference:** the original parent reaches `restart` only after
normally waiting the `first` child, checking its exact 21-control vector and
independently checking the closed generation+2 successor, new owner/current boot,
unchanged ledger and released original leases. Thus the first cycle reached those
checks before this failure. The receipt binds this interpretation to the executed
fixture hash and assertion lines. Successful child output is not independently
retained by this fixture, and its stage times are unavailable. This partial
observation is not added to the strict qualified inventory or accepted as a
complete lifecycle parent.

There is no independent final journal readback after the second startup failure.
Do not infer whether that attempt published an active successor, identify its
failing native probe, or treat the first closed successor as current retry
authority. This log establishes neither deadlock nor a causal scheduling,
emulation, performance or capacity explanation.

Following the adopted stop rule, **14.5 was not attempted**. No retry, reseed,
wider deadline, cached facts, removed guard or new CPU experiment was used.

## Preservation and teardown

All seventeen preceding policy receipts remain byte-identical, including failed,
skipped and historically unverified boundaries. Independent host, device,
image/key/executable and supervisor fixture/group teardown passed, as did
base/runtime/device-backing preservation. The owned temporary artifacts were
removed. `make lab-stop` and `make lab-status` verified all 100 existing owned
sessions stopped before documentation review; that review requires its own
supervised teardown. These are independent host checks. The incomplete UFS
lifecycle metadata/fact cleanup vectors remain **unverified**, even though the
owned guest and disposable backings were removed. Later clean runs cannot
retroactively accept any failed candidate.

This follow-up changes evidence and documentation only. Existing signed fixture
and evidence-parser red/green records remain authoritative; no fabricated red of
unchanged product behavior is claimed. The final signed evidence head requires
fresh actual-title `make review`, including `make check lint test secrets`.
The documentation plan does not repeat unchanged race/fuzz or native suites.

## Remaining work

Both original complete release gates and issue 305 remain open under parent 139.
Next bounded work is review of the second-cycle startup failure using this exact
source and the existing startup inspection/diagnostic records before any new
experiment or changed contract. Point observations and one prior successful
diagnostic do not demonstrate that another run will fit the original caller.
An accelerated amd64 lab remains an alternative requiring a separately declared
and reviewed preservation/containment profile and actual qualification evidence.
No further caller/product/fixture limit or hardware assumption is adopted here.

Controlled initial closed seeds remain fixture setup, not worker shutdown
provenance. Fault/mixed-history, credential/lineage/concurrency/census/publication/
context/capacity families, configured startup, workers/backend/global launch,
browser/client/hardware/performance and release acceptance remain separate.
Runtime writes remain disabled; this evidence PR closes neither 305 nor 139.
