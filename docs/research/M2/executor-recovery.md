# Spike: original executor identity and recovery

Status: Authorized experiment; execution pending. Date: 2026-10-08.
Owner: Cam (`@gonz0w`). Time box: two development days.
Authority: maintainer merge of [PR 105](https://github.com/gonz0w/horvium/pull/105),
source `6854b5554f1104c4fc2f2ee0b4e36dd80b962a9f`.
[Runtime contract](../../milestones/M2/property-runtime.md),
[ADR 0008](../../adr/0008-property-runtime-recovery-gates.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Question and experiment

Which actual FreeBSD facts identify an original executor and show why singleton
metadata lock acquisition alone cannot clear interrupted work? Use serial disposable
15.1/14.5 guests, the existing five snapshot-backed disks and checksum-bound Go
test executable. No runtime product code or storage schema changes.

Require actual root-owned ZFS data/metadata fixtures, root exclusion, explicit
enrolled member geometry, actual coordinator/backend/control and durable records.
Cut the owning process after queued admission, acknowledged applying before launch,
and a returned real guarded property effect before its outcome is published. Run
normal durable completion independently. If the actual property command cannot be
held at a reliably observed active boundary, report that gap rather than claiming
a kernel interruption.

Observe selected owned processes through a research-only C executable compiled by
FreeBSD base `cc` against that guest's own headers. Query `kern.boottime`, exact
`KERN_PROC_PID`/`kinfo_proc` start timestamps and `PROC_REAP_STATUS`. Reject a size
mismatch; record process/reaper facts without command arguments or whole inventories.
This selects a native research mechanism, not a runtime dependency or portable ABI.

Separately exercise the unchanged guard with an owned blocking escaped descendant.
Require natural owner death, a stopped guard during owner death and guard death
before owner death. Independently attempt the same metadata lock while witnessed
descendants remain alive. A containment fixture is not a property mutation.

Retain real queued/applying/succeeded records, policy, native quota values and exact
lock objects across an actual guest reboot. Import only the explicitly owned fixture
GUIDs from their declared devices; this is test cleanup/readback, not production
recovery import. Verify new boot, exact unchanged bytes, conservative holds and
non-restoration of volatile claims before GUID-checked fixture teardown. Keep
failed experiments unqualified and verify all host/supervisor cleanup.

## Acceptance boundaries

The strict research checker requires both phase parents and all seventeen cases,
an independently observed changed boot and once-only phase results. The runner
uses the existing owned prebuilt artifact and cleanup path; no package install or
persistent service is needed. Test checker/runner rejection before implementation.
Run `make review`; shared tooling changes select the conservative portable suite.

Boot time/start timestamps are observed identifiers, not cryptographic provenance
or proof of kernel quiescence. PID-reuse, clock changes, restored-state provenance,
full process enumeration/containment attribution, power loss, root adversaries and
production recovery remain separate gates. Research findings need a followup
review before becoming clearance policy. M2 runtime writes remain disabled.
