# Original startup operation timeout investigation

Status: declared focused diagnostic; execution evidence pending.
Date: 2026-10-11. Owner: Cam (`@gonz0w`). One development session.
Owning [issue 305](https://github.com/gonz0w/horvium/issues/305),
Refs [parent 139](https://github.com/gonz0w/horvium/issues/139).
Authority: approved [PRD](../../milestones/M2/PRD.md),
[EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[retained-policy contract](../../milestones/M2/retained-policy-startup.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md), and
the [adopted inspection diagnostic](policy-startup-inspection.md),
M2-POLICY-STARTUP-ENVELOPE-001–006 (PR 327).
M2-RUNTIME-002, M2-STATE-001, M2-RECOVERY-001, M2-OWN-001,
M2-DRIFT-001, M2-PRIV-001 and M2-NATIVE-001 remain authoritative.
The [PR 332 caller amendment](policy-lifecycle-caller.md) applies to the full
lifecycle fixture; this investigation reuses the already adopted separate
180/210 diagnostic without changing either profile.

## Question and source review

The [merged PR 333 candidate](../../testing/M2/retained-policy-lifecycle-caller.md)
returned an operation deadline error at 126.708043619 seconds, with its original
180-second caller still unexpired and no returned owner. The scalar was captured
before cancellation. It rules out expiry of that caller at capture, but does not
identify the nested timeout or establish final publication state. The failed
candidate and its incomplete guest vectors remain unqualified.

The original factory opens/reviews the protected domains and activates through
the same caller. Review inspection has one 90-second context; activation creates
another 90-second context shared by three complete inspections. Its first two
inspections precede publication; its third follows publication before returning
the owner. These checks protect different intervals. They cannot be removed,
cached or treated as redundant solely because they repeat observations.

Native backing, pool, provider and system collectors have their own 45-second
collection and 10-second command limits. Context deadlines inherit an earlier
parent deadline. The command executor waits for the process and its cleanup,
then checks its supplied context error. Factory error joining and scalar
classification do not carry a unique originating timer. Mutex/file operations
also mean a configured context allowance is not a promise of exact wall time.
An error after active publication can return no owner while active history
remains visible; lease release is not durable closed publication.

The earlier 120/150 trace established caller clamping only in its own sampled
call. The earlier separate 180/210 success at 84.44 seconds preceded the old
120-second limit. Neither establishes the cause of the latest failure or stable
performance. Preserve their candidate boundaries and all twenty preceding
policy receipts byte-identical.

## One unchanged diagnostic

After signed clean-source actual-title local review, run the existing suite once
on a fresh disposable FreeBSD/amd64 15.1 fixture:

```sh
.tools/venv/bin/python3 tools/local-qualification.py policy-startup-envelope 15.1 --run
```

Keep one original factory call, absolute 180-second caller, 210-second child,
480-second parent, 18-minute process phase, one qemu64/TCG CPU, 6 GiB and seven
owned devices. Keep product 90-second operations, nested native limits, sample
offsets 30/75/110 seconds, 64 KiB snapshot cap, 5-second sampler join, 16 KiB child
output, cleanup 30 seconds, remote 7200 seconds and supervisor 9000/grace15 seconds.
No new fixture, timer, instrumentation, producer wrapper, reset, budget or CPU
profile is introduced. Do not retry the failed seed or reinterpret this fresh
standalone startup as the failed full-lifecycle candidate.

Require the existing strict consumer: one completed parent, two normally waited
children, all 42 controls, three honest captured/overlap/skipped sample records,
bounded sanitized source-only frames, independent final journal hash/decoded
state, ledger/nonjournal preservation, released locks, raw guest snapshot removal
and original metadata/fact teardown. Rebind frames to executed Git/compiler
sources and record exact binary/source/log/result hashes. No raw stacks, command
arguments, keys or host inventories enter Git.

Stop after this one candidate. No automatic second release, lifecycle rerun,
alternate CPU or changed-profile attempt. Verify independent host/device/image/
key/executable/supervisor teardown, base/runtime preservation, `make lab-stop`
and `make lab-status`, including any failed/incomplete candidate.

## Interpretation and remaining gates

Keep operation outcome, diagnostic completeness and native lifecycle acceptance
separate. The existing consumer always records native acceptance false. Point
samples locate work at capture; they do not measure stage/collector/command
duration, identify the earliest expired timer or prove deadlock, emulation,
performance or capacity causality. Final journal readback belongs only to this
new candidate and cannot reconstruct the earlier failed startup's journal.

This is evidence-only work using an unchanged adopted fixture. No fabricated
failing product test is needed; the original fixture's red/green receipts remain
authoritative. Mandatory documentation `make review` still runs check/lint/test/
secrets, signatures, DCO and current title/source verification. No unchanged
race/fuzz suite needs repeating for this documentation change.

Both complete five-parent/293-control/thirteen-child/forty-scalar lifecycle gates,
second clean cycle, retained fault families, worker provenance, backend/global
launch, manual browser/client, hardware/performance and release acceptance remain
open under 305/139. Runtime writes remain disabled. No service, deployment,
dependency, infrastructure or hosted automation is added.
