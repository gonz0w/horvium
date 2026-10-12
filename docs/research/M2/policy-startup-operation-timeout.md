# Original startup operation timeout investigation

Status: completed diagnostic; startup failed after publication.
Native lifecycle acceptance remains false; maintainer review pending.
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

## Executed candidate and observations

The [source-bound receipt](policy-startup-operation-timeout-manifest.json) binds
signed protocol/native source `c63ad0614c4922a5edcf67d8a3dad23bba6e944a`, base
`647912303c660529292bac9be44a81e383b07ebc`, pinned Go 1.27.1, QEMU/firmware/base
hashes and original profiles. The tagged binary SHA256
`e816430ce0bbfd40e03838e345e97f656d3c4ea29a5db233c587409de7f302d4` is identical
to PR 333's failed lifecycle binary. No product, fixture or tooling source changed.
This fresh standalone candidate does not reproduce the full suite's preceding
facts/review workload or reconstruct its failed seed.

The single UFS parent completed at **200.29 seconds**, with two normally waited
children and all **42 diagnostic controls**. Startup itself failed at
**121.635881999 seconds**: deadline operation error, caller error `none`, unchanged
absolute 180-second caller and no returned owner. Diagnostic completeness is
true; operation success and native lifecycle acceptance are false.

| Sample | Capture start/end after caller creation | Bytes | Source-located work |
| --- | --- | --- | --- |
| 1 | 30.008719–30.017926s | 8,789 | Original review inspection, domain system version reader (`execution_startup_freebsd.go:207`, `system_use_freebsd.go:9`). |
| 2 | 75.004420999–75.006222999s | 9,545 | Activation second pre-publication inspection, domain backing version reader (`execution_restart_freebsd.go:75`, `persistent_backing_freebsd.go:74`). |
| 3 | 110.002551999–110.003725999s | 10,153 | Activation post-publication inspection, per-policy journal backing version reader (`execution_restart_freebsd.go:100`, `execution_policy_freebsd.go:70`, `persistent_backing_freebsd.go:74`). |

All three samples were captured and uniquely located: 103 frames across 26 files,
70 project frames and 33 compiler-matched Go frames. At each point the original
factory goroutine was waiting on the fixed `/bin/freebsd-version -kru` command
path. Exact branches select that version probe at these points, but do not
identify the final failing command, same invocation or duration. Raw guest
snapshots were independently verified, sanitized and removed.

Source order plus the first two samples place activation context creation after
30.008719s and no later than 75.006222999s. Its nominal 90-second deadline is
therefore after 120.008719s and no later than 165.006222999s, before the original
180-second caller. **This caller did not clip activation's own context in this
observed call.** This is a conservative point/source deduction, not a measured
phase boundary or identification of the earliest expired activation/collector/
command timer. Publication occurs between the second pre-publication and third
post-publication observations, not at an independently measured timestamp.

Independent final journal readback found **active generation 7**, changed from
closed generation 6, with exact hashes in the receipt. Original ledger/nonjournal
bytes were preserved and child locks released. The unsuccessful attempt reached
activation's post-publication inspection. There is no returned owner, durable
idle retirement or retry authority; fail-closed registration/store handling and
lease release remain distinct from clean shutdown. These observations establish
this candidate's final state, not the earlier full-lifecycle failure's state.

The offline replay verifies 43 source assertions across 15 files, the prior
actual failed scalar/log/artifact and two earlier strict diagnostic receipts. All
twenty historical policy manifests remain byte-identical. Their failed, skipped
and unverified limits are preserved. Tagged native compilation and normal runtime
fixture exclusion were verified. Initial actual-title documentation `make review`
passed mandatory check/lint/test/secrets, all 228 tooling tests, ordinary Go tests,
signatures and DCO with clean supervised teardown. No unchanged race/fuzz suite
was rerun. The final evidence head requires fresh current-title baseline review.

Original metadata/fact teardown and independent host/device/image/key/executable/
supervisor cleanup, base/runtime/device-backing preservation all passed. Before
final evidence review, `make lab-stop` and `make lab-status` verified all 108 owned
records stopped. Historical incomplete lifecycle guest vectors remain unverified.
No automatic repeat, second release or full-lifecycle attempt followed.

Next bounded work is a reviewed diagnostic design for timeout provenance within
activation's post-publication checks. Distinguish the activation allowance from
nested collector/command expiry before changing product limits or optimizing
repeated safety checks. Preserve original producers, fail-closed publication,
registration and teardown. Both full native lifecycle gates and broader 305/139
acceptance remain pending; this evidence uses Refs only.
