# Witness rejection custody and recovery diagnostics

Date: 2026-10-14. Status: inactive test fixture; native candidate evidence pending.
Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[accepted EDD](../../design/dashboards/native-witness-binding-EDD.md) and
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-NATIVE-003 / DASH-WIT-NATIVE-004 / DASH-WIT-NATIVE-005.
The preceding [fixture record](native-witness-fixture.md) and its manifest retain
all original candidate boundaries, including the unexplained recovery rejection.
PR 388 was merged; this follow-up stays in its owning issue.

## Corrections and behavioral evidence

Failing portable regressions reproduced three problems: interrupted capture and
archive exhaustion discarded the already-written prefix from the original index;
publication confirmation discarded its rejection stage; and a successful metadata
return could confirm after its caller cancelled. The corrections preserve those
bytes as a sealed, hashed rejected original, carry only allowlisted failure stages
through readback/publication, and recheck cancellation and permanent registry
failure before confirmation. Underlying errors, paths, IDs, inventory values and
response bodies never appear in classification. The first rejection remains visible.

A capture failure has an explicit `capture_failure` in private fixture receipts.
It is never complete, delivered or publication authority. Archive-prefix writes
stay within the original budget; the overflow byte used to detect exhaustion is
not archived beyond that cap. Failure permanently denies the candidate. A producer
EOF with malformed/incomplete bytes remains its original rejected reply; it is
separate from observer I/O uncertainty. Successful byte copying with failed write-half
EOF does not establish delivery. Raw files remain exclusive, sealed mode 0400 and
rechecked by original descriptor/object/hash. Physical write/seal failure can still
prevent a trustworthy retained receipt and remains unqualified.

The observer now identifies control framing, original lookup/custody/decoding/hash,
metadata failure/mismatch and actor registration separately. Receipt custody failures
latch observer failure even if bytes are subsequently restored. These diagnostics
are test-only; product helper frames, API, configuration and runtime stay unchanged.
They do not retrospectively explain the earlier `6525631` recovery failure.

## Native scope and remaining gates

The reduced native short suite additionally runs four injected custody cases on
its original disposable root-only tmpfs: cancelled read, failed read, seeded archive
budget exhaustion, and failed write-half EOF. Each retains one private original
and object receipt, denies eligibility/resumption and removes its owned directory.
These fixed injected wire fixtures are separate from actual native producer originals.
Seeded byte accounting is not a physical 256 MiB or tmpfs exhaustion measurement.

Both pinned FreeBSD releases require a new clean signed reviewed source, actual
candidate builds, probe/artifact/preflight verification, exported original readback
and independent guest/host teardown before any native result is reported. Actual
process cancellation at each transport stage, ancillary replay/cap faults, complete
adverse controls, direct/interposed calibration, instrumentation-inclusive RSS/CPU
and independently reopened sample coverage remain required. The performance runner
remains inactive on its existing oracle; no resource acceptance, causal recovery fix,
hardware claim, browser waiver or runtime write authority follows from this slice.
