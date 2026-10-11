# Portable retained-policy admission coverage

Date: 2026-10-11. Owner: Cam (`@gonz0w`).
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305) and
[M2-RUNTIME-002](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted Test Design](../../milestones/M2/retained-policy-startup.md),
[M2 PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md) and
[Test Design](../../milestones/M2/TDD.md).
Requirement IDs: M2-RUNTIME-002, M2-OWN-001, M2-DRIFT-001 and M2-STATE-001.

## Scope and observable checks

This test-only slice extends the portable checks of supplied policy/native facts
and complete decoded catalogs. Production files, native fixtures, selectors,
timeouts and historical receipts remain unchanged.

| Parent | Cases | Observable criteria |
|---|---|---|
| `TestExecutionStartupPolicyFactsDrift` | 22 drift cases | A valid positive control passes first; changed pool/scope name or parent, foreign/missing scope/member, serial provenance, geometry, duplicate/incomplete providers, missing/mismatched imported pool/topology, system providers, protected mount, swap and another pool's overlapping disks each fail. All supplied facts, policy and eligibility remain exact after refusal. |
| `TestExecutionStartupPolicyFactsEquivalentCaptures` | One positive case | Reordered members/providers and changed capture-local graph IDs still pass with an ordinary `/srv/data` mount; supplied facts remain exact and `Eligible` stays false. |
| `TestExecutionStartupPolicyCatalogEncodedHistory` | 42 format/state cases | Strict accepted ownership/job codecs and the actual catalog builder feed startup refusal. Each of seven states in both job formats denies on the owned pool and another pool. A progressively combined catalog retains two policies and fourteen jobs across two pools, including eight nonterminal and six terminal records. Exact job identity/submission/revision/state/byte summaries, encoded histories and all projected holders remain intact. |

The catalog fixtures contain synthetic metadata, not acquired enrollment or
admission authority. Property histories receive distinct synthetic review IDs;
their full baseline still passes the accepted binding codec. The mixed records
must also pass the reservation decoder, which refuses duplicate plans or
submissions. No actual native pool facts are inferred from these codec positives.

## Development validation

The unchanged implementation already satisfies these portable behaviors, so this
is additional coverage rather than a product fix with a reproduced behavioral
failure. The first development run failed because the combined property fixtures
reused a review identifier. That fixture-construction failure is not product red
evidence. Distinct fixture IDs corrected it; the focused development loop passed.

Three temporary Go overlays independently remove member validation, system
validation or job refusal. Every mutant reaches the relevant new behavioral
assertions and fails as expected. Overlays never modify the tracked production
source and are never qualification candidates. All subprocesses use the existing
owned-process supervisor. Required signed/title/DCO and affected-package review
must be recorded against the committed candidate before handoff.

## Remaining acceptance

These checks exercise pure validators, accepted codecs and metadata projections.
They do not qualify complete native namespace capture, original leases, filesystem
immutability, physical use, publication/retirement, borrowed-lock concurrency,
sticky cancellation, operation capacity or multi-policy lifecycle. In particular,
an in-memory refusal with preserved supplied records does not establish native
history/hold preservation under both original leases.

The existing [storage receipt](retained-policy-startup-manifest.json) and
[single-policy root receipt](retained-policy-root-qualification-manifest.json)
retain their original executed sources and limitations. Expanded native
same/other-pool/mixed-history, multiple-policy and remaining adopted families
still need their own candidate-specific evidence on both pinned FreeBSD releases.
Issue 305 and its parent remain open. No configured runtime, worker shutdown,
performance, client, hardware or release support claim follows from this slice.
