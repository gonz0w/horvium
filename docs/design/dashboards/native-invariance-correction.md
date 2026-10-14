# Proposed Test Design correction: native values during replacement

Status: Draft requirement refinement; neither adopted nor an executable fixture
mechanism. Date: 2026-10-13. Owner: Cam (`@gonz0w`).
Refs [issue 382](https://github.com/gonz0w/horvium/issues/382),
[Overview 289](https://github.com/gonz0w/horvium/issues/289),
[Capacity 290](https://github.com/gonz0w/horvium/issues/290).
Authority: [PRD](PRD.md), [EDD](EDD.md), [Test Design](TDD.md),
[resource supplement](resource-qualification.md), DASH-REQ-002 / DASH-REQ-006 /
DASH-DATA-001 / DASH-PERF-001. The approved resource supplement remains in force.
Actual findings belong to the [native invariance research](../../research/dashboards/native-value-invariance.md), not to this proposed future procedure.

## Problem and preserved requirement

The adopted procedure expects original native numbers to stay constant during
all collection replacements. Previous source-bound candidates failed exact
Allocated/Used/Available cells. A real current native collection can legitimately
contain different accounting from an earlier collection: a presentation oracle
must distinguish original producer values for the displayed generation from a
fixed fixture assumption. The proposed correction preserves independent exact
values; comparing HTML to another management response would not establish that.
It does not retroactively accept a failed candidate or prove a product bug.

## Proposed oracle rule

| Mode | Original invariants | Proposed exact numerical expectation |
| --- | --- | --- |
| Synthetic 100 / 10,000 | Entire independently declared inventory remains constant | Original deterministic values for initial and replacement phases, unchanged |
| Native small | Original 100 owned dataset membership plus complete guest census; GUID/name/record identity and declared unchanged fields | Independent original native producer witness for the exact displayed generation |
| Projection 10,000 | Original native host/pool census and independently declared generated datasets | Native pool accounting from an independent producer witness for the displayed generation; deterministic projected dataset values stay constant |

A retained generation retains its own original witness. A replacement cannot
mutate earlier expectations. Exact table values, ratios, dataset-local maximum,
links, scope, section metadata and same-generation checks all remain mandatory.
Missing, ambiguous, changed or evicted witnesses cause explicit denial of
qualification; no management/API-derived fallback, prefix membership, tolerance,
retry, skipped mismatch or silent current-generation refresh is allowed.
Original changes to supposedly invariant facts still fail. Explicit generation
expiry retains only the existing allowed replacement-Overview exception.

## Mechanism review required before implementation or execution

This document proposes the requirement, not a producer instrumentation interface.
A separately reviewable EDD/Test Design mechanism must resolve all of the following:

- Obtain original native command/source evidence before cached management
  consumption, with independent bounded parsing for expected accounting.
  An independent later collection is insufficient for a generation's exact values.
- Unambiguously bind complete original inputs and successful collector lineage to
  the actual published generation, including multiple outstanding replacements,
  retained old generations, cancellation/partial collection and failed publication.
  Clock coincidence, a matching value digest or management agreement alone is
  insufficient provenance.
- Preserve the actual native product helper/artifact role or explicitly design and
  disclose any test-only instrumented artifact. A fixture substitution cannot
  silently inherit real-helper crash, protocol, latency or memory evidence.
- Bound witness bytes, records, retention, outstanding generations and controller/
  helper memory. Measure and report instrumentation CPU/I/O/RSS effects inside or
  outside paired-process budgets; writing witnesses must not masquerade as source
  invariance. Over-cap collection keeps the existing separate boundary scope.
- Protect private originals from writable management inputs, preserve original
  file/object/content receipts and root/session separation, and remove owned
  witnesses/descendants/sockets/storage with independently bound cleanup.
  Publish only sanitized hashes, counts, measurements and failed/skipped limits.

No new runtime route, protocol, schema, configuration, CLI option, source mode,
package, collector cadence, persistence or dependency is proposed. Any mechanism
that requires changing runtime behavior needs its own explicitly adopted design
and applicable native safety gates. Do not instrument production/operator hosts.

## Required tests and unchanged acceptance

Before using the corrected oracle, write failing behavioral tests for wrong/stale/
duplicate/altered/missing witnesses, generation races, retention/expiry, resource
identity changes, producer/management disagreement, unknown lineage, incomplete
native commands, failed publication, count/byte limits and interrupted cleanup.
Demonstrate that deliberate numeric table/chart mutations still fail and that
legitimate native source changes do not corrupt original older expectations.
Synthetic expectations must remain frozen. Independent parsing cannot reuse
management values or the product renderer's ratio helpers as its oracle.

Both pinned disposable FreeBSD releases remain required, using four modes, original
60-second warm-up, full 300-second small-mode idle, ten readers, 1,000 initial and
1,000 overlapping replacement observations, and three completed replacements.
Retain all native crash/stale/recovery and 100,000-record/byte-exhaustion controls.
Keep each-case and aggregate large p95 below 250 ms, small combined RSS <=128 MiB,
large <=256 MiB and small idle CPU <1% of a core. Strict and legacy validators must
consume original candidate-specific evidence with complete guest/host cleanup.
No result here satisfies those gates; the earlier large-mode latency failure still
needs a measured correction. Browser/manual accessibility, startup/recovery,
hardware and release claims remain separate, and runtime writes remain disabled.

## Review boundary

Maintainer review may accept or revise this requirement direction. It does not
approve an unspecified witness mechanism or authorize immediate qualification.
Keep #382 open. The next bounded design outcome is the concrete original-producer/
generation binding with artifact/resource/cleanup evidence above, followed by its
reviewed implementation and fresh both-release execution. GitHub owns live status;
this packet preserves requirement and evidence boundaries without a progress board.
