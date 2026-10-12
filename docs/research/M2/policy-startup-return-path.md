# One fresh startup return-path diagnostic

Owner: [issue 347](https://github.com/gonz0w/horvium/issues/347), under
[acceptance owner 305](https://github.com/gonz0w/horvium/issues/305) and
[parent 139](https://github.com/gonz0w/horvium/issues/139). Date: 2026-10-12.
Time box: one fresh disposable FreeBSD 15.1 candidate; stop after that attempt.
Authority: the [adopted method/Test Design](policy-startup-timeout-provenance.md),
approved [M2 PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[TDD](../../milestones/M2/TDD.md), retained-policy contract and ADR 0009 linked
there. The [fixture/runner prerequisite](../../testing/M2/deadline-provenance-fixture.md)
was reviewed and merged in PR 346, merge
`256b6bc5ab0520a182cace0566b98d18e990e1cd`.
Requirements: M2-POLICY-DEADLINE-PROVENANCE-004–006 executed-diagnostic subsets,
with adopted 001–003 prerequisites. This is evidence, not runtime authority.

## Question and fixed candidate

Which original context is expired at the failing startup return, and how does
that immutable error observation propagate through the original return path?
Record the operation outcome and independent final journal separately from
diagnostic completeness; native lifecycle acceptance always remains false.

Use the merged `policy-startup-provenance` suite with one prebuilt tagged
FreeBSD/amd64/v1 binary. Bind actual clean signed/DCO source, pinned compiler,
ordinary/tagged file selections and cross-build checksums. The profile remains
qemu64/TCG, one CPU, 6 GiB, seven owned devices, caller 180s, child 210s,
parent 480s and process phase 18min. Preserve original product/collector/command
allowances, fixed 30/75/110s samples, 64 KiB raw cap, 5s join, 30s cleanup,
16 KiB child output, remote 7200s and supervisor 9000s/grace15s.

Exactly one original factory call, the original facts child, one parent and two
normally waited children must supply the exact 50-control diagnostic inventory.
Retain immutable pre-cancel frames, honest sample skips/overlaps, final original
journal hash/state/generation, ledger/nonjournal preservation, released locks,
raw snapshot removal and original metadata/fact teardown. No missing observation
is inferred from a successful host cleanup. Unknown, failed, partial or incomplete
records remain unqualified and are retained in private evidence.

Execute with the existing owned runner:

```sh
.tools/venv/bin/python3 tools/local-qualification.py policy-startup-provenance 15.1 --run
```

No retry, second release, full lifecycle, CPU comparison, altered budget,
producer wrapper, context renewal or failed-seed reuse is authorized by this
candidate. Preserve all 27 preceding receipts and existing Go/runner sources.
Base/runtime/device backing preservation and guest/host/image/key/binary/process
teardown are independent gates. Finish with `make lab-stop` / `make lab-status`.

## Recording and interpretation boundary

The candidate has not executed in this signed preparation. Add the actual
source-bound receipt and observations only after the single attempt finishes.
If it fails, retain that attempt and identify unverified controls explicitly.
Map sanitized point frames to actual executed project/compiler source before
interpreting them. Return-point context constraints do not establish earliest
cancellation delivery, precise command duration, performance or emulation cause.
Preserve active journal history; releasing an owner is not durable idle retirement.

Both full lifecycle release gates, worker/backend/global launch, manual browser/
client, hardware/performance and release acceptance remain open. Runtime writes
remain disabled. Earlier failed/skipped/unverified receipts retain their original
candidate boundaries. GitHub owns live task status; this document owns the
candidate contract and its eventual evidence.
