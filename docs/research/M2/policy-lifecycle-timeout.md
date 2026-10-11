# Research: preserved original lifecycle timeout stack

Status: bounded evidence analysis completed; original lifecycle remains unqualified.
Date: 2026-10-11. Owner: Cam (`@gonz0w`). Time box: one development session.
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305) and
[parent 139](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted policy-only design](../../milestones/M2/retained-policy-startup.md),
[original lifecycle contract](../../testing/M2/retained-policy-native-lifecycle.md)
and the maintainer's request to continue the investigation.

## Question and declared method

Where was the actual original child executing when its existing 150-second
whole-process limit expired? Analyze the already retained Go timeout stack from
the failed FreeBSD 15.1 candidate
`5336a7be00c59e55cd35faeb373889ab64b9ebea`. Do not rerun QEMU, wrap producers,
add instrumentation, change deadlines or modify any product or fixture code.
This is a single historical snapshot, not a duration profile or successful gate.

Before interpreting the stack, require the private original `result.json` and
`qualification.log` bytes to match the record/log hashes in the unchanged
[failed lifecycle receipt](../../testing/M2/retained-policy-native-lifecycle-manifest.json).
Require that candidate's source, artifact checksum, two-CPU amd64/TCG profile,
unqualified result and independent host teardown/preservation facts to match.
Require exactly one 150-second panic and exactly one stack containing the
original `finishIdle` call. Match startup/observation times to that receipt.

Extract only source-relative file names, line numbers and Go function symbols
from that stack. Never publish addresses, argument values, raw trace text,
device/pool names, inventories or private paths. Resolve each Horvium frame
against its **historical executed Git source**, not today's line numbers, and
record original file blob hashes. Read the surrounding control flow to distinguish
pre-publication inspection from post-publication revalidation. Fail the analysis
if any provenance, frame, line or expected control-flow assertion is missing.

Preserve all nine existing retained-policy receipts and the independent primitive
diagnostic receipt byte-for-byte. Record a sanitized analysis manifest and link
the findings from the documentation index and compatibility matrix. Run mandatory
signed clean-source local review; eligible prose/evidence changes require no new
race/fuzz or native execution. Verify supervised teardown and no active owned
sessions before PR review. GitHub owns live status; this document owns the method
and evidence limits.

## Interpretation boundaries

A blocked `Cmd.Wait` snapshot cannot establish how long that command was running,
which kernel/child activity consumed time or that a lock was deadlocked. Reaching
a post-publication source line does not independently verify final journal bytes,
durability, exact closed successor, released leases or normally completed child
controls. No partial stage or historical stack qualifies unused retirement,
second-process restart, either complete release gate or issue 305/139 acceptance.

Do not project this snapshot onto the other two startup-deadline failures: they
returned through a different path and have no corresponding captured stack.
Do not add primitive timings and label their sum actual factory runtime. The
snapshot does not establish emulation causality, performance, supported capacity,
SMP behavior or another architecture. Runtime writes remain disabled.

Any next native run must retain the adopted safety checks and disposable-fixture
ownership/teardown. A hardware-accelerated amd64 lab needs a declared, verified
profile first; the current runner is TCG-only. Any proposed fixture-budget change
needs a separate reviewed contract amendment with cumulative operation/child/
parent/process/controller bounds and failure/cleanup inventories, before code
or execution. This analysis adopts neither option and changes no current bound.

## Verified historical observation

The [sanitized analysis receipt](policy-lifecycle-timeout-manifest.json) binds the
original failed candidate, executable, record/log bytes, 29 extracted frames
(19 Horvium frames), historical source-file hashes and ten source/control-flow
assertions. The original log and record exactly match the failed lifecycle
receipt. All ten preceding policy receipts remain byte-identical. No new native
execution, producer wrapper or stack sampling was necessary.

The one matching original lifecycle stack follows this path. Line numbers refer
to executed source `5336a7be00c59e55cd35faeb373889ab64b9ebea`:

| Historical frame | Location | What it establishes at the snapshot |
|---|---|---|
| `executionStartupNative.finishIdle` | `execution_restart_retirement_freebsd.go:105` | Post-publication inspection, after the closed-successor publisher, acknowledged-object checks and `n.finished = true` |
| `executionStartupNative.inspectLocked` | `execution_startup_freebsd.go:189` | Its initial complete domain observation, before that inspection's per-policy validation |
| `ExecutionDomainLeases.observeLocked` | `execution_domains_freebsd.go:202` | Its final system collection, following repeated journal/ledger backing checks |
| `SystemUseSource.Collect` | `system_use.go:96` | The second native system pass |
| `readSystemNative` | `system_use_freebsd.go:9` | The fixed `/bin/freebsd-version -kru` probe, through the unchanged native guard |
| `executePrepared` / `os/exec.Cmd.Wait` | `execute.go:70` / Go `os/exec/exec.go:944` | Waiting for the launched guarded command; the stack continues into FreeBSD `wait6` |

This locates the **whole-child** timeout during revalidation after closed
publication. It is not the product's 90-second finish deadline or proof of a
successful close. The child never returned its complete lifecycle control row;
the parent did not independently validate the closed record or complete its
cleanup inventory. Independent host/device/guest/supervisor cleanup did pass,
as the historical receipt preserves. Publication uncertainty and original
retirement/release rules remain authoritative.

That child had already reported successful startup at 91.04 seconds and
observation at 28.02 seconds. Subtracting these two rounded stage observations
from the 150-second whole-child budget leaves **at most approximately 30.94
seconds** for all other child work, including finish. This is budget accounting,
not a measured finish duration: setup, denial checks and logging also consume
time. No completed finish timing was recorded. The snapshot alone cannot say
whether the current version command was unusually slow or ordinarily in flight
when the aggregate child limit arrived.

The two other 15.1 attempts expired during startup and have no equivalent stack.
This finding cannot locate their failure or clear either complete release gate.
The two-CPU historical profile differs from today's one-CPU lifecycle profile;
do not silently relabel the evidence or infer a scheduling improvement.

## Replay and next review boundary

On the evidence-owning checkout, locate the historical
`policy-startup-lifecycle-15.1-iyjnozpf` session under ignored `.tools/lab-runs/`.
Keep its raw log and record private. To reproduce the analysis with standard
Python/Git facilities:

1. SHA-256 the original log and record and require equality with both this
   manifest and the matching source entry in the unchanged failed receipt.
   Compare source, artifact checksum, profile, failure and independent cleanup
   fields. Require unqualified results in both records.
2. Require one `panic: test timed out after 2m30s`; compare the startup/observe
   stage rows with the original receipt. Split its goroutine blocks and require
   exactly one containing `executionStartupNative).finishIdle(`.
3. Parse that block's Go source-location lines and preceding function symbols,
   removing argument lists, addresses and creation notices. Require the exact
   ordered 29-frame vector in the sanitized manifest. Do not infer elapsed time
   from goroutine IDs or wait-state annotations.
4. Read each Horvium file with `git show <executed-source>:<relative-file>`.
   Require its SHA-256 and each of the ten exact line/control-flow assertions
   in the manifest. The Go standard-library locations remain tied to the
   compiler identity recorded in the original receipt; they are not Horvium
   source blobs. Verify all ten preceding receipt hashes against the analysis
   base and current Git contents.

These are developer-owned, independently replayable provenance checks. The
committed sanitized witness contains no raw inventories; a reader without the
private original log can review its source mapping and historical receipt hashes
but cannot independently re-extract the stack.

The useful next implementation pickup is a declared accelerated amd64 lab profile
with unchanged lifecycle criteria, if suitable disposable hardware is available.
An ARM Mac cannot accelerate this existing amd64 guest using the same architecture
profile, and the current boot-identity implementation is amd64-only. No home-lab
host has been selected or installed. If work stays on this Mac, a proposed
fixture-budget amendment must first explain the aggregate deadline conflict,
inventory every retained control and preserve caller/product bounds and cleanup;
its review cannot be inferred from this research PR's merge. Do not rerun the
unchanged failing full suite solely to reproduce this already-preserved timeout.
Remaining native fault/mixed-history families, configured startup, workers,
backend/global launch and broader client/browser/hardware/performance/release
gates stay open.
