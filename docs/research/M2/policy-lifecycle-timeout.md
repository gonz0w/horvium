# Research: preserved original lifecycle timeout stack

Status: bounded evidence analysis declared; no new native experiment.
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
