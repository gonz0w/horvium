# Original unused restarted-lifetime retirement

Status: failing-first private prerequisite; native qualification pending; runtime denied.
Date: 2026-10-10. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted recovery mechanism](../../milestones/M2/property-recovery-mechanism.md),
[accepted startup activation](closed-startup-activation.md).

## Contract and Test Design

M2-STATE-001/M2-RECOVERY-001 require an acknowledged closed lifecycle before
releasing a clean incarnation. This consumer closes only the original unused
restarted lifetime: it has never admitted or launched work, retains both original
leases and the exact new active acknowledgement, and has a confirmed empty
census. It does not drain workers or close a reconstructed/advanced incarnation.
The initial generation-one close and its historical evidence stay separate.

FinishIdle excludes observation and release. Original pointer lineage and a
non-nil live context are mandatory. Nil/cancelled/copy/decoded pre-entry calls
have no effect. Once entered, observation/finish authority is terminal, including
collector, publication, cancellation or lost acknowledgement failures. Both
leases stay held until explicit idempotent release; releasing alone still leaves
active unclean history. The process-wide native owner reservation never resets.

Under the original pair/journal/ledger locks, repeat the exact active receipt,
complete empty census, directory/lock/mount snapshots and native system/backing/
raw boot facts. Publish only generation+1 and active-to-closed through the
existing actual successor pipeline. Capture the acknowledged namespace inside
the original directory-sync operation before outer fault wrappers/native
producers; compare it again before retaining that publication baseline. Repeat
all original native facts against that exact closed successor. Every entered
outcome retires both stores; visible active/staged/closed history is preserved.
The reserved final generation may close once but can never restart again.

T1/T2 require four portable parents: exact framing/seven unsafe records/final
capacity, original lifecycle/copy/serialization/context, five terminal failure
variants and exclusion of observation/release during finish. The four evidence
consumer tests remain, extended with two regressions rejecting activation-only
proof and missing/duplicate/failed retirement controls or teardown.

T3 extends the existing focused publication gate to six exact parents: retain
all 268 activation/cut controls, add 126 retirement controls per actual UFS/ZFS
profile and 24 actual UFS cut controls. Twenty-five original precondition,
collector, file pipeline, acknowledgement and late object/context failures are
terminal. Pending/disarmed record mutations and the final-capacity historical
seed are explicitly controlled fixtures, not operator recovery or fabricated
publication acknowledgement. Four exact-binary children are killed and waited
after actual file sync, directory sync, complete acknowledgement or actual UID
loss with inherited root descriptors. Closed cuts may receive original
metadata-only review; no historical publication acknowledgement is reconstructed.
Storage physical-use/boot transfers stay synthetic; filesystem/durability is real.
Require six independent backing teardowns and every control within eighteen minutes.

The fixed-root gate retains both original startup parents/all 186 controls and
extends the activation owner parents to 104 controls each. Ten normally waited
children per profile exercise actual native boot/backing/system producers and
separate production registries. The clean child now proves exact original close,
context/copy denial, retirement and release. A final child consumes only that
successful closed successor, then injects a native system failure after close
publication; it must preserve uncertain closed bytes and release both locks.
The prior finish-denial/active-preservation expectations are deliberately replaced
by closed-receipt expectations; earlier activation-only proof remains bound to
its original source and is rejected by the current reader. All other original
registration/denial/UID/data/domain gates stay required. Keep every original
eight-minute parent deadline and the existing 34-minute four-parent process bound.

Both separate suites run serially on both pinned disposable FreeBSD/amd64
releases. Require exact source/executable hashes and independent history,
lease/backing/data/device/guest/host/supervisor teardown. Full unrelated storage
corpora retain their actual historical evidence; fresh affected portable review
is mandatory, with exact-input reuse only for final prose. All failed runs remain
unqualified. No nonempty recovery, worker shutdown, native launch, configured
startup, operator route/service, actual reboot, power loss, kernel interruption,
whole-host rollback, performance or commercial-scale qualification is implied.


## Bounded fixture correction

The implemented ZFS root parent exhausted its unchanged eight-minute deadline
when the final close-post child began; that observed failed run was stopped and
retains outer guest/host/device/supervisor cleanup, with native completion
unqualified. The fixture now consumes its already-established original closed
seed for the clean child, rather than commissioning a duplicate. After independent
parent verification, retain the legitimate generation-four closed successor for
boot/system denial and the final original close-post child; retain its exact
visible generation-six closed history afterward. The active sentinel stays exact.

Cancelled observation proves unchanged bytes/original unblocked lineage and is
followed by actual successful native FinishIdle, avoiding a duplicate full native
observation. Ordinary paired metadata observation gathers declared disjointness
inside its real system collector, avoiding one separate full collection. Every
actual callback still collects fresh native facts; no producer is cached, mocked
or relaxed. Keep all ten original-registry children, all 104 controls, both
independent history/lock/data/domain/device gates and every original deadline.
Product implementation and the storage executable remain unchanged.
