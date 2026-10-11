# Native multiple-policy storage startup

Date: 2026-10-11. Owner: Cam (`@gonz0w`).
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305) and
[M2-RUNTIME-002](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted retained-policy Test Design](../../milestones/M2/retained-policy-startup.md),
[M2 PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md) and [ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
Requirement IDs: M2-RUNTIME-002, M2-STATE-001, M2-OWN-001, M2-DRIFT-001,
M2-RECOVERY-001.

## Bounded outcome and original seed

Extend the accepted policy-only storage seed from one policy to two, using the
original fresh commissioned journal/ledger leases. Publish each ownership through
the accepted original journaled publisher (pending, ledger successor, confirmed
complete census), then publish the strict controlled closed successor. Never
write encoded records directly or reconstruct a publication acknowledgement.
There is no admitted worker, executor or qualified shutdown after real work.

The synthetic policy fixtures describe distinct pool GUIDs 100 and 200, scopes,
members and providers, with a complete supplied system-use view of both pools.
Both policies must independently pass existing fact validators. Actual UFS
without softupdates and ZFS files, leases, census, journal sync/acknowledgements
and namespace snapshots are native. Physical-use/policy/boot transfers remain
synthetic; fixed-root producers and independently waited root-factory inspection
remain separate acceptance.

## Inventory declared before execution

Keep the existing selector
`^TestNative(ClosedStartup(Publication|Retirement)(UFS|ZFS|Cuts)|PolicyStartup(UFS|ZFS))$`,
eight parents and eighteen-minute process bound. Preserve all 870 controls from
the complete-history gate and add 60 per policy parent: 223 policy controls per
filesystem, 446 policy plus 544 retained controls, 990 total.

The twelve `multi_` controls are `catalog`, `reviewed`, `activated`, `observed`,
`closed`, `lineage`, `ledger`, `all-policies`, `leases`, `shared-deadline`,
`retired`, `reopened`. They require two exact canonical policies and zero jobs,
one-time activation and unused close with exactly two added generations, changed
owner identity and unchanged bindings/census/ledger bytes. Every operation must
visit both policies under both actual leases and one shared operation deadline;
finish retires authority and independent diagnostic reopen retains the catalog.

The six `last-{review,activate,observe,finish,post-activate,post-finish}_` groups
each require `denied`, `last-policy`, `leases`, `deadline`, `preserved`, `sticky`,
`released`, `diagnostic`. The first policy must pass before the last producer
fails. Pre-publication failures retain exact journal and ledger snapshots.
Post-publication failures retain the visible expected generation+1 active/closed
successor, but grant no successful activation/close result; the exact ledger and
conservative visible journal remain unchanged through repaired-source retries.
Entered lifetimes remain retired and both leases release. Failed initial review
returns no handle; repeating its same failed producer remains denied with exact
metadata. Fresh diagnostics must read both original policies and no jobs.

Pin the additional inventory in a failing-first evidence-consumer test. Missing,
failed, duplicated or old-only controls cannot qualify. That parser red is not a
reproduced native/product defect. Product behavior is unchanged.
Qualification is pending until the complete checksum-bound gate passes on both
pinned FreeBSD/amd64 15.1 and 14.5 targets with independent owned teardown.

The initial signed native candidate
`3ca51368b613d791d159e58888c40601d53971e2` failed the UFS valid-review setup:
the fixture compared a canonical decoded policy against construction-order
slices using structural equality. It returned a fixture denial before the new
lifecycle assertions. Native cleanup and base/runtime preservation passed;
the complete run remains unqualified. A signed failing-first portable regression
reproduces that fixture mismatch. The correction compares the complete strict
canonical encodings, preserving revision/member/scope validation and rejecting
drift; it changes no production policy validator or startup consumer.

## Acceptance limits

Two supplied policy facts do not qualify actual multiple data-pool composition,
all sparse histories, every drift variant or commercial capacity. Fixed-root
multiple-policy/mixed-history composition, remaining original lineage/concurrency,
census/object and context/capacity families remain open under issue 305. No worker
shutdown, configured startup, native launch, client/browser, performance, hardware
or M2/release acceptance is added. Historical receipts retain their bytes and
original candidate-specific limits.
