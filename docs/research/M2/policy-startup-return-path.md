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

The signed preparation preceded the one attempt recorded below. Its contract
requires an actual source-bound receipt, including failed attempts and explicit
unverified controls; recording a result never manufactures missing observations.
Map sanitized point frames to actual executed project/compiler source before
interpreting them. Return-point context constraints do not establish earliest
cancellation delivery, precise command duration, performance or emulation cause.
Preserve active journal history; releasing an owner is not durable idle retirement.

Both full lifecycle release gates, worker/backend/global launch, manual browser/
client, hardware/performance and release acceptance remain open. Runtime writes
remain disabled. Earlier failed/skipped/unverified receipts retain their original
candidate boundaries. GitHub owns live task status; this document owns the
candidate contract and its eventual evidence.

## Recorded native candidate

The [receipt](policy-startup-return-path-manifest.json) binds actual executed
signed source `e4e77ac5f30a131a02b6f96315c73de5c3fef6a7` to review base `256b6bc5ab0520a182cace0566b98d18e990e1cd`,
pinned Go 1.27.1 and the actual source-isolated FreeBSD/amd64/v1 tagged build.
One fresh 15.1-RELEASE-p3/OpenZFS 2.4.2 lab ran; no retry or second release
followed. Real ordinary/tagged compiler selections and both artifact checksums
match the reviewed fixture's offline build. Existing Go/runner sources and all
27 prior receipts remain unchanged. This establishes this instrumented candidate,
not production support or a controlled comparison.

| Observation | Actual result | Boundary |
| --- | --- | --- |
| Original startup | Success at 105.111880999s; owner returned | Earlier unsuccessful startup is not reproduced or explained |
| Original caller | 180s absolute deadline; error none at return | No renewal or larger native allowance |
| Immutable error trace | `success`, zero frames | No failing return context or non-nil native error delegation observed |
| 30s sample | Located; 35 sanitized frames | Activation first inspection, original backing command path |
| 75s sample | Located; 35 sanitized frames | Activation second inspection, original backing command path |
| 110s sample | Skipped; zero bytes/frames | Operation returned before that sample |
| Original final journal | Changed from closed generation 6 to active generation 7 | Actual independent readback; no idle retirement claim |
| Parent/children/controls | Parent 173.35s; two normally waited children; all 50 controls passed | Diagnostic completeness only |
| Independent preservation/teardown | Guest, host, base/runtime/device backing, raw snapshots, metadata, image/key/binaries and owned process groups passed | No historical missing vector is upgraded |

All 70 point frames map to 22 actual project/installed compiler source files.
At 30s the original activation call is at
[its first inspection](https://github.com/gonz0w/horvium/blob/e4e77ac5f30a131a02b6f96315c73de5c3fef6a7/internal/storage/execution_restart_freebsd.go#L51);
at 75s it is at
[its second inspection](https://github.com/gonz0w/horvium/blob/e4e77ac5f30a131a02b6f96315c73de5c3fef6a7/internal/storage/execution_restart_freebsd.go#L76).
Both locate the original persistent-backing command chain. These are point
locations, not continuous command durations, probe targets or expired timer
observations. The trace is empty because the actual operation succeeded; forcing
a failure or repeating it would violate the adopted single-candidate scope.

The original ledger/nonjournal content, released locks, raw snapshot removal and
metadata/fact teardown all passed. Base/runtime/device backing preservation and
host image/key/executable/supervisor cleanup independently passed. Raw snapshots
were guest-only, sanitized and deleted before metadata cleanup; later independent
re-extraction is unavailable. Private full/canonical/serial logs and result
checksums are retained without committing raw errors, GUIDs, credentials or
host inventories.

The signed pre-run actual-title `make review` passed 258 tooling tests and
mandatory check/lint/test/secrets with the documentation-only plan. Reviewed
implementation race/fuzz/native-build prerequisites retain their exact earlier
source; no new Go or runner input justifies repeating deep portable work here.
The independent post-run verifier revalidated actual candidate/file bindings,
full transcript/50-control completeness, source locations, immutable receipts
and teardown. Its owned process completed and cleaned its group. Final receipt
changes require fresh mandatory review and final lab-stop/status.

## Remaining interpretation and acceptance

The adopted attempt is exhausted. Its success completes the bounded observation
record, while the earlier failing deadline's return context remains unobserved.
Do not infer a fix, earlier cancellation order, command duration or native
non-nil error propagation from this empty trace. Any further native qualification
or method investigation is a separately scoped decision under 305 after review;
this PR adds no retry or changed profile.

Both complete 293-control/thirteen-child/forty-scalar lifecycle release gates,
second clean cycle and fault families remain pending. A released owner and
active generation 7 do not establish durable idle retirement, worker/backend/
global launch, recovery or release acceptance. Native acceptance stays false;
runtime writes stay disabled. Manual browser/client, hardware/performance and
all broader gates remain open, with previous failed/skipped evidence intact.
