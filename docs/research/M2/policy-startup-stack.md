# Diagnostic: bounded original startup stacks

Status: implementation candidate declared; no execution or completeness yet.
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
