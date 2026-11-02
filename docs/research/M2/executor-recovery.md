# Spike: original executor identity and recovery

Status: Executed; observations accepted in PR 106; runtime recovery unqualified.
Date: 2026-10-08.
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

## Executed observations

Signed source `de3f028a8e579ece4843bcca59a8cbeb06d4588f` passed both phase
parents and all seventeen cases on disposable FreeBSD 15.1-RELEASE-p3/OpenZFS
2.4.2 and 14.5-RELEASE/OpenZFS 2.2.9. Both used the same prebuilt Go test executable,
SHA-256 `bbff7f1540fa6ae382bc0dd192a2201b9cbb4c777558e96b906655f7e2a2acf7`.
The [source-bound manifest](../../testing/M2/executor-recovery-manifest.json)
records images, actual compiler/header/utility hashes, private log hashes,
independently observed reboot markers and complete teardown. Total native runs
took 331.411 and 234.567 seconds respectively on the recorded Mac/QEMU profile;
these are lab timings, not product performance evidence.

The guest-compiled observer returned a 1088-byte `kinfo_proc` on both images.
It checked its own native structure size, selected PID/start/parent/group/session,
root UID and stopped/zombie fields, and queried native reaper status. Observed
header hashes match the pinned FreeBSD primary sources:
[14.5 user.h](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/sys/user.h),
[14.5 procctl.h](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/sys/procctl.h),
[15.1 user.h](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/sys/user.h),
[15.1 procctl.h](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/sys/procctl.h).
These interface observations establish no other architecture or runtime ABI
support and do not enumerate every descendant.

Actual containment observations were identical on both releases. The owner held
the original metadata lock and launched the unchanged native guard with an owned
blocking probe and its escaped child. Original identities were observed before
signals and checked again when signaling or attributing survivors.

| Owner/guard cut | Same singleton lock reacquired | Original probe live | Original escaped child live |
|---|---|---|---|
| Owner death | Yes | No | No |
| Guard stopped before owner death | Yes | No | No |
| Owner stopped, guard killed, then owner killed | Yes | Yes | Yes |

The final row is an actual counterexample to treating lock reacquisition as
executor quiescence. Both surviving fixture processes were subsequently killed
and verified stopped. The stopped-guard row records the observed result without
claiming to explain why those descendants exited.

The property cases used the actual private coordinator, backend and guarded
fixed native control against explicitly owned nonboot data and metadata pools.
Four separate fixture ledgers allowed independent cuts on one data pool; they
do not implement the shared runtime launch owner. Clearance was a test-only
successful callback, and ownership metadata was fixture-seeded.

| Actual witnessed boundary | Retained job event | Independent quota observation |
|---|---|---|
| Queued acknowledgement, before handoff | Queued | Original value/source; no effect |
| Applying acknowledgement, before native launch | Applying | Original value/source; no effect |
| Normal native completion and durable outcome | Succeeded | Literal `2097152`, local source |
| Native control returned successfully, before outcome publication | Applying | Literal `2097152`, local source |

The normal-completion case restored the fixture quota before the final effect
case. Killing the actual owner at each interrupted boundary did not restore its
volatile claim on reopening. A real guest reboot changed independently collected
boot observations. Exact retained job/policy bytes and original lock inodes
survived; filesystem GUID and final quota/value source survived independently.
Interrupted records required reconciliation, every retained record kept its pool
hold, including success, and a fresh state-source collection denied eligibility.
Only declared fixture GUIDs/devices were imported for readback and teardown.

## Failed attempts and portable review

The initial 15.1 attempt failed an overly strong fixture assertion that both
stopped-guard descendants must survive; nine other before cases passed and no
reboot phase ran. The corrected fixture records each actual descendant outcome
independently while still requiring a live original descendant for the guard-first
counterexample. The first corrected 14.5 attempt failed SSH banner exchange during
source transfer before any native test executed. A fresh retry passed with the
same source and deadlines. Both failed records remain unqualified in the manifest,
with verified teardown.

Working-tree runner rejection tests preceded implementation. The signed tested
source passed `make check lint test secrets` through `make review`, including all
144 tooling tests and a fresh full portable review of eleven race packages and
49 bounded fuzz targets. Shared runner/checker changes required that conservative
plan. Existing non-test product files and all eighty historical M2 manifests
retain their bytes. All 695 owned sessions were stopped with verified teardown
at evidence capture; Actions remained disabled.

## Mechanism followup and remaining runtime gates

An active property-command/kernel interruption was **not exercised**. The final
effect cut witnessed completed control before durable outcome publication; the
blocking escaped descendant tested containment separately. Graceful guest reboot
did not test power loss or a reboot during an active storage command. The owned
sentinel is fixture preservation evidence, not independent file-client continuity.

The observed tuple is useful attribution evidence for these owned processes,
without becoming runtime clearance. The accepted PR 107 EDD/ADR selects concrete
helper/supervisor identity, containment and the independent restored-state boundary.
Implementation must qualify those gates before startup can grant authority, account
for the surviving descendant counterexample, bind clearance to the original state lease and keep
retained holds separate from global launch capacity. No new runtime schema or
adapter is adopted by this experiment. M2-RUNTIME-001 retains its implementation/native delivery gates.

PR 106 accepted these observations at merged source
`318ef8ac155f4004051ad6f4287fa8de2109a5a5`. The accepted
[mechanism amendment](../../milestones/M2/property-recovery-mechanism.md) and
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md) were accepted in PR 107 at merged source
`02a995e5681f38b4a40141402dd2d0a9f0244933`; the original native manifest is unchanged.
The [private supervisor qualification](../../testing/M2/property-supervisor.md)
now supplies source-bound boot/clock/reaper/consumer evidence separately.

Boot time/start timestamps are observed identifiers, not cryptographic provenance
or proof of kernel quiescence. The later supervisor evidence tests clock-independent
raw boot identity, without qualifying general clock/restore recovery. PID reuse,
restored-state provenance, full attribution, power loss, root adversaries and
production recovery remain separate gates. The accepted clearance design
still needs its independent journal/startup/global implementation. M2 runtime writes remain disabled.

The [October 19 source correction](../../testing/M2/property-supervisor.md#source-correction-and-blocked-fixture-cuts-2026-10-19)
removes the historical metadata-to-PID signal helper. Original receipts above
retain their exact source and scope. The successor deliberately fails the cuts
whose original guard capability is unavailable, before launching those subtrees
or proceeding to later effects/reboot publication. Stable-capability fixture
redesign and both-release native qualification remain open; new source/checker
coverage does not relabel historical execution or create runtime authority.

The [stable original-guard successor](../../testing/M2/property-supervisor.md#stable-original-guard-fixture-successor-2026-10-19-not_run)
now implements those four cuts through retained original birth/ancestry and
pre-Wait gates in isolated self-reaper controllers. It preserves actual kernel
STOP/KILL/CONT, survivor and drain predicates, with separate fixture disposal.
This is source-only and NOT_RUN on both releases; historical observations above
are unchanged and cannot qualify the successor.
