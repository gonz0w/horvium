# Diagnostic: bounded original startup stacks

Status: corrected 15.1 diagnostic complete; original startup returned a deadline
error after active publication. Native lifecycle acceptance remains pending.
Date: 2026-10-11. Owner: Cam (`@gonz0w`). Time box: one development session.
Owning research [issue 323](https://github.com/gonz0w/horvium/issues/323);
Refs [305](https://github.com/gonz0w/horvium/issues/305) and
[parent 139](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted sampling contract](policy-startup-deadline.md),
[M2 Test Design](../../milestones/M2/TDD.md), approved PRD/EDD and ADR 0009 linked
there. PR 322 adopted the method at merge `c481cbbb7b17c592f61bb7cff506f475c071cbd4`,
signed proposal source `9cbd5b01df263cc823bd53a31f3917a50832ed6f`; issue 321 is done.

## Declared implementation and inventory

M2-POLICY-STARTUP-TRACE-001–006 require failing-first portable scheduling/file/
sanitization and strict evidence tests, followed by the original one-call native
diagnostic. Use one tagged UFS parent, one fresh original-factory child and the
original independently waited facts child: **one parent, two children, 42 controls**.
Keep all 21 existing fact/data controls. Add eight child controls:
`cpu_affinity child_budget caller_deadline original_call sampler_joined
samples_bounded owner_release deadline_unchanged`.
The parent retains each as `child_<name>` and adds thirteen actual controls:
`cpu_affinity parent_budget seed_closed child_waited child_inventory
snapshots_verified final_journal_observed ledger_preserved nonjournal_preserved
child_locks_released raw_removed domain_teardown mounts_restored`.

Use fixed samples 1/2/3 at 30/75/110 seconds, one reusable 64 KiB buffer, three
private 0600 files in a child-owned 0700 fixture directory and exclusive creation.
Reject empty/full/truncated buffers, duplicate/unexpected/unsafe files, wrong
permissions/owners/link counts or changed hashes. Raw stacks stay inside the
owned guest. Only sanitized source-relative frame vectors and scalar hashes/
timings/status enter the private qualification log and sanitized receipt.
Bind Horvium frames to the executed source and standard-library frames to the
compiler-matched Go source. Unsupported/ambiguous/return-overlapping observations
stay unlocated with explicit limitations. Do not manufacture locations or durations.

Actual startup return and caller deadline/error are separate scalar fields;
samples after early return are skipped. Stop/join within five seconds after
return; the existing 150-second whole-child ceiling still applies. A missing
scheduled snapshot, stop/join failure or incomplete parent invalidates completeness.
Record original final journal hash, unchanged/changed flag and decoded state (or
invalid/uncertain), original ledger/nonjournal equality and released child locks.
An operation error or nil owner does not prove absence of publication.

Profile marker: `cpu_count=1 child_seconds=150 parent_seconds=480 phase_minutes=18`.
Retain caller120/product90, all native bounds, original cleanup30, remote7200,
supervisor9000/grace15 and child-output16KiB. The fixed native suite is
`policy-startup-trace`, first and only declared target 15.1. Complete diagnostic
observations may include startup failure, but **native acceptance is always false**;
both complete lifecycle release gates remain required. No retry or full-suite rerun.

Require actual fixed factory/native producers/registry/leases/publication, normal
build exclusion, all thirteen prior receipts byte-identical, signed clean-source
mandatory review (shared-tooling full fallback), serial execution, original
metadata/fact cleanup and independent host/device/key/binary/supervisor teardown
and base/runtime preservation. No product source, caller/product limit, guard,
cache, CPU experiment, runtime authority, service or dependency changes.
Sampling overhead may change scheduling/outcome; snapshots cannot apportion the
preceding time or establish a deadlock, performance/capacity or emulation cause.


## Historical first candidate and preserved setup failure

The [sanitized receipt](policy-startup-stack-manifest.json) binds original native
source `ec92537b281c0d4465c3a3cfda2b56979e4f93b7` and base
`c481cbbb7b17c592f61bb7cff506f475c071cbd4`. The single declared 15.1 UFS parent
failed at **14.95 seconds**, in original seed setup with `storage scope denied`.
The startup child and original startup factory were never reached. There are no
captured samples, caller/return observations, final journal/ledger vector or
complete 42-control inventory. Diagnostic completeness and native acceptance are
both **false**. This is a setup failure, not a reproduced startup timeout, early
startup return or successful localization. 14.5 was not attempted.

Executed runner registrations omitted `HORVIUM_CLOSED_STARTUP_OWNER_VM=1`.
The original tagged seed explicitly rejects its absence before commissioning
(`internal/storage/execution_policy_fixture_freebsd.go:17`). The candidate now
supplies that existing opt-in in both registered commands and requires it in the
prebuilt shell preflight. A failing-first runner regression reproduces the
omission; all six focused consumer/profile tests pass after correction. No seed
or product guard was weakened. **The corrected native setup has not executed**;
review it before another separately recorded diagnostic attempt.

Independent host/image/device/key/binary/supervisor cleanup and base/runtime
preservation all passed. The incomplete guest parent supplied no complete
metadata/fact teardown vector; that remains unverified despite host cleanup.
Temporary native artifacts are removed. No raw stacks were captured or copied;
all thirteen preceding policy receipts remain byte-identical.

Original implementation source passed mandatory portable review: 215 tooling
tests, ordinary Go tests, eleven race packages and 54 fuzz targets, with verified
supervised teardown. Earlier failing-first sampler/file/sanitizer and consumer
observations, the portable temporary-directory setup failure and rejected review
whose CPU-selector expectation omitted the adopted one-CPU suite remain recorded.
These portable results do not qualify native scheduling, original factory
execution or the corrected opt-in path. The final signed head requires its own
actual-title review; later evidence retains the executed source separately.

This draft uses Refs for issue 323, owner 305 and parent 139. Neither incomplete
research nor a merged sampler satisfies their remaining native acceptance.
Runtime writes remain disabled. No caller/product/native budget, CPU profile,
producer, registry, cache, retry, service, dependency or runtime authority changed.


## Historical corrected runner portable review

Corrected source `840c662264d111754400d376cc7179ae482bb389` passed a fresh full
portable fallback: mandatory check/lint/test/secrets, ordinary Go and 215 tooling
tests, eleven race packages and 54 fuzz targets, actual-title/signature/DCO checks
and supervised teardown. Its [receipt](policy-startup-stack-manifest.json)
binds the source/tree/base and review/check/deep-log hashes separately from the
failed native source. Subsequent evidence-only head verification may reuse those
exact unchanged inputs; it does not create corrected native evidence.


## Corrected original-factory qualification

PR 324 merged the correction at `2d78311dbf7c71678b02f48f2a3488a4c8a737fa`,
whose tree exactly matches reviewed signed head
`026aa9cb682516cfa951af8831fa2744437d969a`. This follow-up executed one focused
15.1 UFS diagnostic from that clean merged source, with no fixture, runner,
product, deadline or CPU change. The
[new source-bound receipt](policy-startup-stack-qualification-manifest.json)
preserves the earlier failed setup receipt and all fourteen preceding policy
receipts byte-identical; historical pending/draft statements above describe only
those earlier candidates.

The parent passed at **216.09s** with both normally completed children and all
**42 controls**, original metadata/fact cleanup, released child leases, unchanged
ledger/nonjournal bytes, restored mounts and independent host/device/key/binary/
supervisor teardown and base/runtime preservation. All three snapshots were
captured, verified, source-sanitized and deleted in the owned guest. Their 97
frames (64 Horvium / 33 compiler-matched Go) are source-mapped; no raw stack or
argument was copied to the host or committed. Post-deletion independent raw
re-extraction is unavailable by design.

| Sample | Actual capture start/end after caller creation | Bytes | Located original path |
| --- | --- | --- | --- |
| 1 | 30.008717–30.012838s | 9,647 | Review inspection, guarded persistent-backing version reader (`persistent_backing_freebsd.go:74`; `execution_startup_freebsd.go:59`). |
| 2 | 75.008148–75.009936s | 8,693 | Activation first pre-publication inspection, guarded system-use version reader (`system_use_freebsd.go:9`; `execution_restart_freebsd.go:50`). |
| 3 | 110.000572–110.002112s | 8,713 | Activation post-publication inspection, guarded system-use pool path topology reader (`system_use_freebsd.go:23`; `execution_restart_freebsd.go:100`). |

Each unique root factory goroutine was in the original native command waiting
path. Source checks bind the observed reader branches and activation's original
publish at line 84 before the final inspection at line 100. These are point
observations: no internal stage boundary times, command duration, failing probe,
deadlock, production capacity or causal emulation conclusion is inferred.
Capture intervals exclude subsequent snapshot file writes; total observer cost
is unmeasured and sampling can change scheduling/outcome.

Original startup returned at **120.010624s**, with both caller and startup error
classified `deadline` and **no returned owner**. Independent final journal bytes
changed from closed generation **6** to active generation **7**; exact before/after
hashes are in the receipt. The 110-second snapshot is already in post-publication
inspection. This demonstrates publication before this observed startup failure;
a nil owner/error cannot be treated as no mutation or a safe reusable closed
seed. Original ledger/nonjournal bytes and released leases were independently
checked. Those facts do not grant launch authority, establish restart/retirement
acceptance or permit another startup attempt on this fixture history.

**Diagnostic completeness is true; original startup success and native lifecycle
acceptance are false.** This supports closing only bounded research issue 323
when its evidence PR is accepted. Owner 305 and parent 139 remain open; neither
release's full five-parent/289-control/thirteen-child lifecycle gate, second clean
cycle or broader recovery/runtime/client/hardware/performance/release gate is
qualified. 14.5 remains outside this adopted one-target diagnostic and was not
attempted. Runtime writes remain disabled.

The next bounded research/design work is to review the original repeated native
inspection path and post-publication deadline/error recovery against these exact
observations before any changed runtime or fixture contract. Preserve all drift,
lease, publication/uncertainty and fail-closed requirements. These snapshots
support investigation, not caching, removed checks, wider deadlines or retries.
