# Witness rejection custody and recovery diagnostics

Date: 2026-10-14. Status: inactive corrections checked on both disposable targets; complete gates pending.
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

## Candidate receipts and independent readback

Signed candidate `e9c42a4caba4efe2ccc47cae104ed5259717b6c1`, base
`0993c77407432e313974e08d8ddae15f765e07b7`, passed required checks and focused
cmd/horvium race/original-wire fuzz review before sequential Mac QEMU runs:

| Pinned release | Reduced suite | Final producer originals | Injected rejected originals |
| --- | --- | --- | --- |
| FreeBSD 15.1 | Passed, 123.58 seconds | 7 | 4 |
| FreeBSD 14.5 | Passed, 99.33 seconds | 6 | 4 |

Both retained actual ABI probes, unchanged product-role artifacts, original raw
files/object receipts, matching before/after preflight observations and verified
bounded guest/host teardown/pinned preservation. Each checked ten actual HTML
responses per mode, exact lineage/mutations, owned identities/fixed projection,
same-UID denial, actual native command/guard death, stale retention and helper epoch
recovery. Four injected custody cases per guest retained bounded rejected originals;
these are not an entire process-cancellation or physical exhaustion matrix.

Independent readback reopened both private archives and decoded all **21** indexed
originals with the independent original-wire decoder: thirteen producer attempts
and eight injected faults. Fifteen-one's final archive contains one additional
complete delivered projection original beyond its pre-stop two-original summary.
It was retained during the interval between that summary and completed drain; it
supplies no extra publication or workload acceptance. The initial private readback
harness incorrectly assumed twenty files and rejected the archive count. Replacing
that assumption with the independently verified complete census allowed all
originals to be decoded; no raw record was removed, repaired or relabeled.

The [sanitized manifest](witness-rejection-custody-manifest.json) names exact source,
base/title, original hashes, probe receipts, counts and remaining limitations. Raw
inventories and actor identities remain private. All 155 earlier manifests retain
their exact bytes, including the three failed native candidates and unresolved
`6525631` recovery rejection. No new failure was observed in these reduced suites;
they establish no causal explanation of that prior failure, complete adverse gate,
calibration/augmented budget, full resource acceptance or writer authority.

This follow-up continues the partial fixture merged in PR 388. Keep subsequent
fault/calibration/accounting/coverage work in the continuation PR while it is open;
full integration/native short acceptance still precedes resource activation.
