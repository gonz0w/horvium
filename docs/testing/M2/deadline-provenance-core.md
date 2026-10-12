# Portable deadline provenance adapter

Status: portable prerequisite; native instrumentation and qualification pending.
Date: 2026-10-11. Owner: Cam (`@gonz0w`).
Refs [issue 337](https://github.com/gonz0w/horvium/issues/337),
[parent 305](https://github.com/gonz0w/horvium/issues/305) and
[broader gate 139](https://github.com/gonz0w/horvium/issues/139).
Authority: PR 336 adopted the [method and Test Design](../../research/M2/policy-startup-timeout-provenance.md),
refining the approved [M2 Test Design](../../milestones/M2/TDD.md),
[PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[retained-policy contract](../../milestones/M2/retained-policy-startup.md) and
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
Requirements: portable parts of M2-POLICY-DEADLINE-PROVENANCE-001–003 and 006;
M2-RUNTIME-002, M2-STATE-001, M2-RECOVERY-001, M2-OWN-001, M2-DRIFT-001,
M2-PRIV-001. These passes do not satisfy the complete six adopted test IDs.

## Behavior and boundaries

The existing internal storage package supplies an identity adapter in ordinary
builds. It returns the exact original error without context inspection, clocks
or allocations. The `horvium_native_fixture` build contains the diagnostic types,
enablement, extraction and serialization. Without explicit fixture enablement,
it only checks for that private context value and retains exact error identity;
no context errors/deadlines, clocks or record allocation. No runtime interface
can enable the diagnostic. No product return site calls the adapter in this slice.

A fixture carries its original monotonic `time.Now` origin using a private context
value; a zero or wall-clock-only origin cannot enable observations. This adds no timer, cancellation, goroutine, callback or shared collector.
Future instrumentation must pass each actual original child/parent before its
scope cancels them; this adapter cannot enforce correct placement by itself.
Guard annotations take no child and zero allowance. Child annotations accept
only the nine reviewed sites, fixed phases and original 90/45/10-second allowance.
The adapter observes actual effective deadlines, never constructs/renews them.

Enabled deadline errors receive immutable frames around the same original error.
Error/Unwrap preserve its original Error/Is/As behavior. Nondeadline errors retain
identity. Fixed scalars include source/phase, operation error class, effective
relative child/parent deadlines, error classes before/after and capture interval.
No raw error, arguments, context values, credentials or absolute clocks enter JSON.
Inner frames remain unchanged after cancellation or outer wrapping. Equal effective
deadlines mean an inherited constraint; both expired, canceled, unresolved or
capture overlap do not identify earliest timer delivery or command duration.

Construction and extraction cap twelve frames, 48 visited error nodes and 8192
encoded bytes. Terminal overflow/invalid annotations retain the original error
and propagate identically through subsequent adapters. Extraction retains distinct
join branches in inner-to-outer order, rejects duplicate annotations, cyclic/deep
wrappers, malformed records and unknown custom classification, and explicitly
separates success, not-deadline, unattributed, complete, ambiguous, invalid and
overflow. A complete trace is a diagnostic observation, not native acceptance.
The strict native consumer and its exact 50-control inventory remain separate.

## Failing-first and verification

Signed red source `2d8e5b0108d04fdb5406705cff8563fe86f0eee1` compiled the behavioral
specification against an identity-only tagged stub. The earlier red commit
`03c29c520ed435dd5931a7beef107cd9dda67c90` exposed the absent behavior and a typed
error test helper missing Unwrap; the second signed red corrects that helper.
The [receipt](deadline-provenance-core-manifest.json) preserves both actual failures.
Later regressions add disabled allocation, malformed-frame and inconsistent
expired-deadline checks, plus actual
immediately expired child/live parent cancellation retention. No slow sleep or
fabricated failure of an unchanged native product path is used.

Verification runs the ordinary identity case, all tagged storage tests, focused
tagged race checks, normal/tagged FreeBSD storage cross-compilation and normal/tagged
host file-selection inspection. The signed clean actual-title `make review` runs
mandatory check/lint/test/secrets and affected-package race/fuzz checks. A final
receipt-only followup uses a fresh baseline and only verified exact-input reuse.
All 22 preceding policy receipts and existing fixture/consumer sources remain
byte-identical. Owned processes are stopped and status verified.

No new VM, native run, service, daemon, runtime writer or deployment is introduced.
Original-return integration, strict fixture/consumer, one fresh 15.1 diagnostic,
both complete lifecycle release gates and all wider native/client/browser/recovery/
hardware/performance/release gates remain open. Historical limitations remain
bound to their actual candidates; this adapter adds no support claim.
