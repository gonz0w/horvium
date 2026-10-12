# Startup deadline return-path diagnostic

Status: method and Test Design adopted by merged PR 336
(`37402d6a017dc28fb7fb6528fc9b92516fda1cfb`). The proposal and offline replay below
retain their historical review boundaries. The separate
[portable adapter slice](../../testing/M2/deadline-provenance-core.md), owned by
[issue 337](https://github.com/gonz0w/horvium/issues/337), implements the
prerequisite only; original-return instrumentation and native execution remain
under 305.
Date: 2026-10-11. Owner: Cam (`@gonz0w`). Time box: one offline development session.
Owning [research issue 335](https://github.com/gonz0w/horvium/issues/335), under
[parent 139](https://github.com/gonz0w/horvium/issues/139), coordinating
[acceptance owner 305](https://github.com/gonz0w/horvium/issues/305).
Authority: approved [PRD](../../milestones/M2/PRD.md),
[EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[retained-policy contract](../../milestones/M2/retained-policy-startup.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md), and the
[adopted separate envelope](policy-startup-inspection.md).
Requirements: M2-RUNTIME-002, M2-STATE-001, M2-RECOVERY-001, M2-OWN-001,
M2-DRIFT-001, M2-PRIV-001 and M2-NATIVE-001. The broader contracts alone did not approve the annotation method; PR 336
adopted the scoped method and Test Design below.

## Question and existing evidence

Which original context is expired at a startup deadline-error return, and how
does that error propagate through native collection and activation? The
[merged PR 334 investigation](policy-startup-operation-timeout.md) excludes
caller clipping in its observed call, reaches the post-publication inspection,
and independently reads active generation 7 without a returned owner. It does
not identify the earliest expired nested timer or final failed command.

Its actual signed native source is `c63ad0614c4922a5edcf67d8a3dad23bba6e944a`;
review base is PR 334 merge `df47fe999210e6b2f602778bdbbc8b052321c5c1`.
Startup failed at 121.635881999 seconds while its original 180-second caller
remained live. All three point samples were located; the 200.29-second UFS
parent/two children/all 42 controls established diagnostic completeness only.
Final ledger/nonjournal preservation, released leases, guest/host cleanup and
base/runtime preservation passed. Earlier failed lifecycle journal/guest vectors
remain unverified. Preserve all twenty-one preceding policy receipts, including
PR 334's receipt, byte-identical.

The [offline replay](policy-startup-timeout-provenance-manifest.json) rebinds
that exact log/result/artifact, all 103 frames/26 source files and original
compiler, 46 new source assertions across 16 files, all 43 prior source assertions
and original point deductions. This is source/evidence
verification, not a newly executed behavioral or native test.

## Actual timer and error path

| Actual startup boundary | Configured allowance | Return-point observation needed |
| --- | ---: | --- |
| Original caller across opening/review/activation | 180s in separate fixture | Existing pre-cancel operation/caller summary and unchanged deadline. |
| Original review inspection | 90s | Actual child/parent deadlines and errors before review cancellation. |
| Activation, shared by all three inspections | 90s | Same original context at each deadline-error exit, with the actual inspection/publication phase. |
| Persistent backing, pool, provider and system collectors | 45s each | Actual collector/parent context before collector cancellation. |
| Persistent backing, pool and system command captures | 10s each | Actual command/collector context before command cancellation; provider collection uses the existing system command capture. |

There are nine relevant timeout-constructor source sites, not nine timers per
startup: repeated calls create distinct contexts and inherit earlier deadlines.
The public joint-domain observer's 90s and single-domain observer's 45s timers
are outside this factory path. Startup uses their already-locked observers;
the native backing collector supplies its own 45s limit. Do not attribute those
unused public timers to these samples or introduce them through a wrapper.

Review opens original domains before its inspection context; activation enters
after review. Its first two inspections precede publication and its third
follows publication. Fixed producers, descriptor borrowing, repeated drift checks,
mutex order, original registry and sticky fail-closed publication/release stay
unchanged. Command execution still waits for its original process group and I/O,
then checks its supplied context. Classified errors lose this return-path context
today, and outer cancellation can destroy an inner context's useful observation.

## Adopted minimal instrumentation

Add a private annotation adapter in the existing storage/host layers, shared
through the existing internal storage dependency; introduce no new package or
external/runtime API. The ordinary-build adapter returns the exact original
error object and does not inspect contexts, collect clocks or allocate records.
The tagged build does the same unless a dedicated disposable fixture explicitly
enables the diagnostic on its original caller. Compile the annotation type,
enablement/extraction and scalar serialization only with the native fixture tag.
Keep the existing trace, envelope and lifecycle fixtures/consumers unchanged.

At an actual deadline-error return, the enabled adapter creates an immutable
bounded annotation around that same error. Its Error string delegates to the
original, and Unwrap retains original errors.Is/errors.As behavior. Do not
replace, classify afresh or suppress the underlying error. Annotate only known
deadline returns; non-deadline errors retain identity. Context creation remains
the original context.WithTimeout with the original parent, allowance and cancel.
Do not use WithTimeoutCause, timers, AfterFunc callbacks, goroutines or a global
mutable collector. The method observes error propagation rather than introducing
another cancellation mechanism.

Retain the actual original parent context locally before each existing timeout
constructor. Capture the derived and parent Deadline/Err at error return, before
that scope's deferred cancel. Preserve the original caller start solely for
relative monotonic observations. Do not construct a replacement context, renew
a deadline or read an inner context after its cancellation to fill a missing row.
The immutable inner snapshot travels with the error; outer scopes add their own
pre-cancel snapshot as it returns. Guard returns before a child exists must be
explicitly recorded as parent-only, never assigned synthetic child observations.

Each annotation carries only fixed source-site/phase identifiers, nominal
allowance where a child exists, actual effective child/parent deadline presence
and relative values, operation/context/parent error classes, and a capture
begin/end interval. Record error observations around that interval so cancellation
during capture is explicitly ambiguous; do not claim an atomic snapshot. No raw
error string, pointer, command arguments, probe target, policy/GUID, path,
credential, context value or host inventory enters the scalar record.

Bound annotation construction and extraction to twelve frames and 8 KiB encoded
evidence within the unchanged
16 KiB child transcript. Traverse only diagnostic annotations and original error
wrapping, with bounded traversal of joins. Preserve distinct annotated branches;
multiple competing deadline paths are ambiguous rather than flattened into an
invented cause. Stop constructing annotations at the cap: retain a bounded tagged
overflow indication around the original error, and propagate it unchanged through
later adapters rather than allocating an ever-growing chain. Unknown joins, cycles, excessive depth, encoding or frame overflow
invalidate diagnostic completeness without changing the operation's original
failure, resetting registration or granting a retry. An operation success yields
an explicitly empty error trace. An unannotated deadline is honestly unattributed,
not silently accepted as complete required provenance.

This records the observed failing return path and effective deadline constraints.
A child deadline earlier than its parent's with child DeadlineExceeded and parent
still live identifies a local expiry observation. Equal effective deadlines mean
inherited constraint; do not label a second independent timer expiry. Both expired,
cancellation overlap or ambiguous branches do not prove cancellation delivery
order. No return-point observation can promise the earliest wall-clock timer
event, precise command duration or performance/emulation/deadlock causality.

## Adopted separate fixture and inventory

Use a separately identified tagged parent/child, prebuilt marker and strict
consumer; choose development-only names during implementation after adoption.
Do not retrofit the earlier receipts or change their fixtures/consumers. Reuse
the existing fixed sampler, sanitizer, absolute caller helper and original
native factory/facts child without wrapping any native producer. The new method
changes observation overhead, so its timings are instrumented-candidate evidence,
never byte-identical production timing or a controlled comparison with PR 334.

Keep caller 180s, child 210s, parent 480s, process phase 18m; one qemu64/TCG CPU,
6 GiB and seven owned devices. Keep product/native allowances, samples 30/75/110s,
64 KiB raw cap, 5s sampler join, cleanup 30s, output 16 KiB, remote 7200s,
supervisor 9000s/grace15s unchanged. No command/probe selection or extra collection
is added. Perform one original factory call, release its actual owner if returned,
and independently read the final original journal. Release cannot stand in for
durable idle retirement or erase failed active history.

Retain all 42 original diagnostic/fact controls. Add five child controls for
explicit tag/fixture enablement, unchanged underlying error semantics, pre-cancel
trace snapshots, bounded exact trace inventory and original context/deadline
retention. Parent validates and carries those five controls, then adds three
controls for exact trace extraction, sanitized evidence and disabled-build/source
isolation. This makes thirteen child controls, twenty-nine diagnostic parent
controls plus twenty-one original fact/data controls: **50 total**, with one
parent and two normally waited children. Final control identifiers and exact
vectors must be declared and tested before native execution; fewer controls or
different accounting requires another design review.

Require the unchanged three honest sample records, independent final journal
hash/state or invalid uncertainty, ledger/nonjournal preservation, released locks,
verified raw snapshot removal and metadata/fact teardown. The new consumer always
records native lifecycle acceptance false, even if startup succeeds or the failed
return trace is complete. Consumer/controller must reject cross-profile/source/
artifact mismatches, unknown/duplicate/missing/partial fields, false attribution,
raw output and incomplete normally waited controls/cleanup. A deadline is an
acceptable operation outcome only when required diagnostic evidence is complete.

After adoption, run one fresh disposable 15.1 diagnostic under the signed reviewed
source and pinned compiler. Stop after that one candidate; no automatic retry,
second release, full lifecycle, CPU comparison, wider budget or failed-seed reuse.
Retain failures/unattributed/ambiguous/overflow states honestly. Independently
verify guest and host/device/image/key/executable/supervisor cleanup and base/
runtime/backing preservation, then lab-stop/status. Neither interpretation nor
host cleanup retroactively completes a historical incomplete guest vector.

## Adopted failing-first Test Design

These IDs refine the cited M2 requirements; they are plans, not passing tests.

| ID | Required observable behavior after adoption |
| --- | --- |
| M2-POLICY-DEADLINE-PROVENANCE-001 | Failing-first portable tests prove ordinary-build and tagged-disabled exact error identity; enabled annotations preserve original Error/Is/As and cancellation propagation. Normal build excludes annotation types, enablement, extraction and serialization. No diagnostic is enabled by a runtime config/route/CLI. |
| M2-POLICY-DEADLINE-PROVENANCE-002 | Deterministic original-context tests distinguish local/inherited deadline, parent-only guard, parent cancellation, both expired and capture overlap. Inner observations remain unchanged after deferred cancellation and outer error wrapping; no new/renewed context or post-cancel reconstruction. No slow sleeps or fabricated product red. |
| M2-POLICY-DEADLINE-PROVENANCE-003 | Failing-first bounded extraction/consumer cases cover twelve frames/8 KiB, original inner-to-outer path, joins, ambiguous branches, unknown/cyclic/overflow/duplicate/raw evidence and cross-profile/artifact/source mismatches. Explicit empty success and unattributed deadline are distinct; incomplete required provenance never qualifies completeness or native acceptance. |
| M2-POLICY-DEADLINE-PROVENANCE-004 | Exact new 50-control/one-parent/two-child inventory preserves every original factory, sampler, deadline, registry, native producer, lease, publication and original final-readback/teardown requirement. Original 42-control diagnostic and 293-control lifecycle sources/profiles/consumers remain intact and cannot satisfy the new trace gate. |
| M2-POLICY-DEADLINE-PROVENANCE-005 | One fresh signed-source checksum-bound 15.1 candidate separates operation outcome, trace attribution, diagnostic completeness and native acceptance. Actual source/compiler frames and original-context return records are sanitized and bounded. All guest/host cleanup passes or stays honestly unverified; no repeat, second release, alternate profile or broader acceptance claim. |
| M2-POLICY-DEADLINE-PROVENANCE-006 | Preserve all twenty-one preceding receipts, source/base/title/signatures/DCO and red/green provenance. Run current-head mandatory review with affected Go race/fuzz checks and conservative full fallback when shared tooling changes. Cross-compile and verify normal/tagged file selection; final evidence-only followup uses fresh baseline with only verified exact-input reuse. Verify owned-session teardown. |

## Decision and remaining work

PR 336 adopted the bounded fixture-only annotation method and Test
Design above. Alternatives are continued point-only diagnostics (which cannot
recover missing return context), or a separately reviewed accelerated lab profile
(which would change timing conditions and still not establish this candidate's
timer provenance). Changing product limits or caching/dropping safety observations
does not answer the current question and is outside this proposal.

Issue 335's decision criterion is satisfied by merged PR 336. Adoption covers
only the method, inventory and Test Design; implementation and one native
diagnostic remain separate work under 305. The proposal review ran no new VM,
instrumented product, changed deadline or service. The portable adapter slice
adds no product call sites or native execution.
Both complete 293-control/thirteen-child/forty-scalar lifecycle release gates,
second clean cycle, fault families, worker/backend/global launch, manual browser/
client, hardware/performance and release acceptance remain pending. Runtime
writes remain disabled. All PR references are Refs; neither broader gate closes.
