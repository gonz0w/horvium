# Research: original startup deadline and proposed stack diagnostic

Status: offline analysis completed; diagnostic method Proposed, not adopted.
Date: 2026-10-11. Owner: Cam (`@gonz0w`). Time box: one development session.
Decision [issue 321](https://github.com/gonz0w/horvium/issues/321);
implementation/qualification [issue 305](https://github.com/gonz0w/horvium/issues/305);
parent [139](https://github.com/gonz0w/horvium/issues/139).
Requirements: M2-RUNTIME-002, M2-STATE-001, M2-RECOVERY-001, M2-OWN-001,
M2-DRIFT-001, M2-PRIV-001 and M2-NATIVE-001.
Authority: [M2 PRD](../../milestones/M2/PRD.md),
[EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[adopted policy-only design](../../milestones/M2/retained-policy-startup.md),
[adopted fixture budgets](../../milestones/M2/retained-policy-fixture-budgets.md)
and [ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).

## Question and declared offline method

What does the retained 120.01-second startup failure establish about the actual
original factory, and what observation is missing before considering a remedy?
Analyze the private original log/result for executed source
`6c64143132bae6fdbde33c71c646ababb8987633`, bound by the unchanged
[PR-320 candidate receipt](../../testing/M2/retained-policy-lifecycle-budget-manifest.json).
Require exact source/artifact/log/result hashes, one-CPU amd64/TCG profile,
330/1080/40 fixture limits, unqualified status, three completed prerequisite
parents, failed UFS lifecycle parent, one failed startup stage and skipped 14.5.
Verify independent cleanup, separately from missing in-guest vectors.

Inspect that executed Git source for the fixture caller, original root composition,
single-use registry, storage review/activation and error/cleanup propagation.
Record file hashes and exact source-relative line assertions. Check for any
internal stage timing or captured stack in this log; do not invent either.
Preserve all twelve preceding policy receipts byte-for-byte. Record a sanitized
research witness without raw arguments, stack addresses, device/pool names,
inventory or private paths. No new VM, instrumentation or product/fixture change
is included in this research PR. Mandatory signed local review and owned-session
teardown remain required.

## Verified observation and source boundaries

The [sanitized witness](policy-startup-deadline-manifest.json) verifies the original
failed result/log bytes, artifact/profile, 26 source assertions across eight
historical files and all twelve unchanged preceding receipts. No new native run
was performed. The original 15.1 prerequisites passed at 26.81s (facts), 291.17s
(UFS review) and 393.35s (ZFS review). The first UFS lifecycle parent failed at
200.22s after its child reported `startup` at 120.01s with
`context deadline exceeded` and failed the `registered` assertion. There was no
whole-child timeout, captured goroutine stack or internal review/activation
timing. Neither complete lifecycle control vector appeared. 14.5 was skipped.
Independent host/preservation/device/supervisor cleanup passed; incomplete
in-guest parent teardown remains unverified.

The following locations are from **executed source**
`6c64143132bae6fdbde33c71c646ababb8987633`; they describe code, not an observed
trace of this failing call:

| Historical source | Verified behavior |
| --- | --- |
| `internal/host/freebsd/execution_policy_lifecycle_fixture_test.go:104` | One 120-second caller covers the actual original factory, followed by the combined error/nonnil-owner assertion at line 110. |
| `internal/host/freebsd/execution_restart_freebsd.go:26` | Original policy review and activation (line 30) receive the same context `c`; activation failure joins the original review's `Close` at line 32. |
| `internal/host/freebsd/execution_owner.go:79` | The process registry is reserved before `factory(ctx)` at line 82; the result joins caller context error at line 87. Failed construction never resets registration. |
| `internal/storage/execution_startup_freebsd.go:59` | Review performs original inspection; its inspection context is clamped to 90 seconds at line 87. Failed review joins pair release at line 61. |
| `internal/storage/execution_restart_freebsd.go:23` | Activation has one 90-second child of its received caller; complete inspections occur at lines 50, 75 and 100, around original publication at line 84. |
| `internal/storage/execution_restart.go:63` | Activation joins its error with the received context error; the error does not retain internal timing or location. |
| `internal/host/freebsd/execute.go:78` | A failed native command propagates context error after its original cleanup, without returning captured output on error. |

The compiler-matched pinned Go context source confirms that a child timeout cannot
extend an earlier parent deadline. If activation starts at time `A` relative to
original caller creation, its effective deadline is at most
`min(120 seconds, A + 90 seconds)`. The caller permits 120 seconds for the combined
review/activation work, not 90 + 90 seconds independently. For example, activation
starting at 40 seconds can have at most 80 seconds remaining; this is hypothetical
budget accounting, **not an observed review duration**.

The 120.01-second stage report is consistent with the original caller ceiling,
but it does not select review, activation, a particular native probe, publication
revalidation or error cleanup as the cause. Nested operations propagate the same
error; synchronous cleanup can consume time after another context has expired.
No caller-context observation or stack was separately captured. Source permits
activation failure after publication, so a failed registration/nil returned owner
does not establish that the journal stayed closed or that leases released. The
failed parent supplied no independent final-byte/hold vector. Do not transfer the
different historical finish stack or primitive collection timings onto this call.

The proposed snapshots below supply the missing point observations; they do not
optimize startup or adopt a runtime change. Unchanged full-suite retries would
only reproduce the already bound error without resolving its location.

## Proposed next diagnostic, pending explicit adoption

Run one separate diagnostic-only original-factory attempt on disposable
FreeBSD/amd64 15.1 with the existing one-CPU qemu64/TCG, 6 GiB, seven-device profile.
Do not rerun all three acceptance phases merely to reproduce the known failure.
Reuse the original two-policy fact/data and UFS metadata fixture, original closed
seed and original native topology restrictions. A new tagged test-only parent
starts one checksum-bound fresh child that invokes
`activateNativeClosedExecutionOwner` exactly once under the original registry,
leases, policy producers, backing/system/boot collectors and publication pipeline.
No substitute/wrapped producers, reset registry, caller subset or cached facts.

The diagnostic child captures bounded Go goroutine snapshots around this original
startup call. Use Go's standard-library `runtime.Stack` into one reusable 64 KiB
buffer, with at most three samples scheduled at **30, 75 and 110 seconds** from
the original caller's creation. Record actual monotonic capture start/end times,
call-return time and caller deadline/error as scalar observations. Timers are
observers, never operation contexts; delays do not renew or extend any deadline.
Snapshots are point observations, not a timing profile or collector counter.

Keep raw snapshots only in a child-owned 0700 guest directory with predeclared
0600 files; at most three 64 KiB files (192 KiB). Reject full-buffer/truncated,
missing or duplicate files and unsafe paths/permissions. Log only fixed sample
IDs, bounds, scalar timing/status and SHA-256 hashes inside the existing 16 KiB
child-output cap. Never emit raw stacks or arguments to stdout or commit them.
Use fixed test selection rather than an arbitrary operator flag or output path.
No new runtime API, dependency or tracking automation.

If startup returns before a scheduled sample, label that sample skipped with the
actual return time; never wait out the schedule or synthesize a stack. On return,
stop the sampler and join it before child exit; a five-second join ceiling is
included in the original child allowance, not a new product grace. An incomplete
join invalidates the diagnostic and the whole-child/outer ownership limits still
terminate it. `runtime.Stack` and local syscalls are not guaranteed to return by
a timer; record overruns instead of claiming a hard syscall wall-time ceiling.

Retain a **120-second startup caller**, all original **90-second product** and
native collector/command limits, a **150-second diagnostic child**, default
**480-second parent work context**, **18-minute diagnostic phase**, independent
**30-second original cleanup invocations**, **7200-second remote command** and
**9000-second supervisor / 15-second termination grace**. This new diagnostic
does not borrow the lifecycle-only 330/1080/40 amendment. Check actual selected
budgets and original absolute deadlines. No timeout renewal, retry, CPU tuning
or further budget increase. Sampling overhead can alter scheduling and the
outcome; record that limitation even if the original failure does not recur.

After startup, use only the actual returned original owner for release, if one
exists. Release through its original `Close`; capture any error, without calling
an alternate retirement or assuming quiescence. The parent waits for the child
and independently inspects the final validated journal against its original
closed seed and original ledger bytes. Record exact unchanged/changed metadata
and decode uncertainty; a nil owner/error does not prove no publication. Preserve
original metadata and fact/data cleanup, mount/sentinel checks, then independent
host/image/device/key/binary/supervisor teardown and base/runtime preservation.
Cleanup failure invalidates completeness; do not retry or erase failure evidence.

Sanitize only function symbols and source-relative line numbers from any unique
goroutine containing the actual root startup factory. Bind frames to the executed
Git source or recorded Go compiler. If no unique matching block exists, capture
overlaps return, a sample overruns, or a line cannot be verified, label the sample
unlocated/ambiguous and preserve that limitation. Do not infer a deadlock, command
duration, emulation causality, successful active publication or an absent owner
from a snapshot. A frame can distinguish an observed review/activation/cleanup
path at that instant; it cannot apportion the preceding 120 seconds.

A complete diagnostic may include the original startup failure as its observed
outcome only if the stack/provenance/parent/cleanup inventories themselves pass.
Keep diagnostic completeness separate from operation success and native
acceptance. Missing snapshots due to an early return are explicit limitations,
not successful localization. A separately scoped 14.5 comparison can follow if
the observation justifies it; this one-target research cannot qualify either
release's original five-parent, 289-control, thirteen-child acceptance gate.

## Proposed failing-first Test Design

| Test ID | Observable criterion before implementation/execution |
| --- | --- |
| M2-POLICY-STARTUP-TRACE-001 | Replay exact failed log/result and executed source hashes; reject changed source/artifact/profile, qualified relabeling or missing failed/skipped observations. |
| M2-POLICY-STARTUP-TRACE-002 | Portable sampler/parser tests reject oversized/full-buffer, missing/duplicate/unsafe snapshot files, raw argument/address leakage, unsupported source locations and ambiguous factory blocks. |
| M2-POLICY-STARTUP-TRACE-003 | Fixed schedule stops on early return; cancellation does not renew caller/product deadlines. Actual timestamps distinguish late/overlapping samples; incomplete stop/join fails. No future timer keeps the child alive. |
| M2-POLICY-STARTUP-TRACE-004 | Tagged native diagnostic invokes the actual factory once with the original producers/registry/leases/publication; verifies actual child/parent/caller bounds and complete existing 21 fact/data controls, original seed/final history and mount/metadata cleanup. Normal builds exclude it. |
| M2-POLICY-STARTUP-TRACE-005 | Strict evidence consumer independently tracks diagnostic completeness, startup success/failure, skipped/unlocated samples and native acceptance false. Missing final parent/cleanup vectors cannot pass on partial snapshots or host cleanup alone. |
| M2-POLICY-STARTUP-TRACE-006 | Signed source/artifact/compiler/log/result/snapshot/source hashes and independent owned teardown/preservation are bound in a sanitized manifest. All earlier receipts retain their bytes; no raw stacks, keys or inventory are published. |

Before implementing, the maintainer must explicitly adopt this method through
review and merge of its design PR, recording source/merge identities in issue 321.
Its merge accepts a research method, never native behavior or architecture.
Concrete diagnostic-only control names and strict parser inventory must be
declared in the implementation contract before its failing-first code and run.
Any remedy to product/caller budgets, collectors, guard, caching or runtime
authority needs a distinct adopted design and relevant evidence.

## Replay

On the evidence-owning checkout, locate the ignored original
`policy-startup-lifecycle-15.1-2qvd59c1` session. Keep its raw files private.
SHA-256 `result.json` and `qualification.log` and require equality with the witness
and the unchanged PR-320 receipt. Compare source, prebuilt artifact/compiler,
one-CPU profile, fixture bounds, failure and all seven independent cleanup flags;
verify removal of owned image/key/devices/binary. Require the exact three passed
parents, one failed UFS parent and single startup/error row recorded above, absent
whole-test panic, goroutine stack and completed lifecycle vectors.

For every `historical_sources` file, read `git show <analyzed-source>:<file>` and
verify its Git blob and SHA-256. Match all 26 exact line/count assertions; verify
review precedes activation, registry reservation precedes the factory and the
three activation inspections surround original publication. Match the recorded
Go command checksum to the native compiler identity before checking its pinned
local `src/context/context.go` hash and parent-deadline rule. That rule describes
context semantics, not the failed call's stage timing. Verify all twelve previous
receipt bytes/hashes against analysis base
`139f193fe18aa62785a925b518e88aca309cd143` and the current checkout.

These standard Python/Git provenance checks need no new VM. Without the original
private log/result a reviewer can inspect source assertions and Git receipts,
but cannot independently re-extract the raw observation. No current status,
proposal approval or broader acceptance can be inferred from the sanitized file.
