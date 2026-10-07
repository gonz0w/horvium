# M2 private fixed native property control

Date: 2026-10-07. Status: native prerequisite qualified; accepted in [PR 81](https://github.com/gonz0w/horvium/pull/81).
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0006](../../adr/0006-storage-ownership-and-plans.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-PROPERTY-001, M2-PRIV-001, M2-DRIFT-001 and M2-NATIVE-001 evidence.

## Private control boundary

`propertyControl` has no public constructor or runtime consumer. A future root
backend must bind its authorization callback to the original live plan/session,
root lease, complete native eligibility, exact current durable applying record,
recovered-state safety and global/pool reservations. A captured intent, root UID,
stored record, matching observation or reconstructed control supplies none of
that authority. Only explicitly owned disposable fixtures bind it in this slice.
The existing native sources keep false eligibility; no flag is promoted.

After adapter, context and root checks, a control consumes one attempt, including
authorization denial. Validate the
immutable intent, independently obtain the exact two-event applying job, compare
the entire intent binding and fresh baseline/policy, then recheck cancellation,
clock rollback, recorded time and original review expiry. Read-only observations
can establish a baseline comparison without becoming write authority. Inputs and
outputs remain owned; unknown/malformed/incomplete/drifting facts deny launch.
The callback has a 145-second outer bound, and every native collection keeps its
existing limits. Concurrent calls on this object cannot repeat the control;
global launch/pool scheduling and cross-process authorization remain separate.

The private native runner checks actual kernel EUID zero and accepts only a
validated typed immutable intent. Build exactly `set`, one allowlisted
quota/refquota/compression `property=value` argument, and the captured filesystem
address selected by GUID. Full fresh baseline matching includes that address.
No executable, environment, arbitrary property/map, flag, recursion, shell,
inheritance/reset, rollback, destroy or force command is accepted. The fixed
executable is `/sbin/zfs`; no-limit remains the explicit accepted `none` value.

## Guard acknowledgement and uncertainty

Use the accepted fixed FreeBSD base timeout/reaper with a ten-second command
deadline and 4 KiB combined stdout/stderr budget. Its ordinary successful return
waits for the child and reaper descendants, rather than inferring quiescence from
the ZFS child's exit alone. The new native barrier tests leave a descendant alive
after its parent exits, both inside and outside that process group; success must
remain withheld until the descendant is released and stopped. The guard's
existing helper-death/cancellation behavior remains unchanged.

Require a successful guarded invocation with no stdout for this narrow positive
quiescence acknowledgement. Every timeout, cancellation, nonzero exit, output
overflow, missing executable, permission or other error returns false/unverified.
Do not infer no effect or a safe retry. This primitive publishes no outcome,
releases no hold and grants no generic process or kernel-operation cancellation.
The [execution sequence](property-execution.md) still needs a qualified original
live root backend before it can invoke this control in an operator workflow.

Pinned primary sources describe the base guard and ZFS argument handling:

- [15.1 timeout](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/bin/timeout/timeout.c)
  and [14.5 timeout](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/bin/timeout/timeout.c).
- [15.1 ZFS command](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/cmd/zfs/zfs_main.c)
  and [14.5 ZFS command](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/cmd/zfs/zfs_main.c).

## Executed qualification

Signed source `6ef01cb5fbd3f3015a6fc8c9a5a8ccc4302031aa`, based on merged PR 80
`252e07e3cd2412a698b3b0c6c4651ca47db1331f`, passed `make review` baseline checks,
122 tooling tests, eight affected race packages and all nine selected fuzz targets,
including `FuzzPropertyControlSelection`. The fixture correction changes only
disposable setup/cleanup; product control and guard behavior are unchanged.

Both FreeBSD 15.1 and 14.5 passed all 88 original pool assertions, all earlier
postcondition/retained-assessment/native-source subgroups, all eleven new control
children/group and both guard children, without FAIL/SKIP. Parent timings were
201.42/193.37 seconds; new control groups took 37.14/45.63 seconds.
Both used the same independently rebuilt binary:
`6d8c7e8735f18b547bc19d5ff5d85d597efb6af8fea16a88cea92444c323b02e`.
These fixture timings are not NAS throughput or scale evidence.

Actual native effects, fresh postconditions, retained applying records, repeat
refusal, ledger/native/expiry denial, kernel EUID denial and normal guard descendant
waiting all passed. Both guests used an explicitly mounted ZFS metadata child,
with actual backing/mount checks. Complete storage/source/GUID/metadata/anchor/
pool/provider/sentinel/base/runtime/process preservation and teardown passed;
`make lab-status` verified no active or unverified owned sessions. Historical
56 M2 manifests and 334 retained paths were byte-identical.
See the [sanitized execution manifest](property-control-manifest.json) for exact
source, primary-source, input, runtime, review/log and teardown hashes.

The initial 15.1 attempt at `924ce12dc77c883d36e89679d8aafb2c8bc5593c`
was operator-stopped before control qualification when source review identified
that a private directory on 14.5's UFS softupdates root could not support actual
publication. It remains unqualified with successful owned teardown recorded
separately. The corrected fixture declares a GUID-bound mounted ZFS metadata
child on both guests; the accepted product UFS softupdates refusal is unchanged.
No earlier evidence is relabeled or gate relaxed.

## Failing-first and native fixture gates

Portable tests first require the absent control contract, then require fixed
argument selection, whole intent/applying/fresh baseline binding, root/expiry/
clock/context/missing-adapter denials, unverified errors, owned copies and 32
concurrent single-attempt calls. Bounded selection fuzzing cannot widen arguments.

The existing opted-in five-minute `pool-properties` parent retains all 88 original
assertions, all old postcondition/assessment/source subgroups and GUID/backing/
sentinel/host teardown. Add `native-control` after those checks; require its eleven
children and group once, without FAIL/SKIP: `preflight`, `quota`, `refquota`,
`no-limit`, `compression`, `expired`, `ledger-drift`, `native-drift`, `wrong-uid`,
`guard-quiescence`, `preserved`. Require both guard children `same-group` and
`escaped` and the actual dropped-UID child PASS inside the parent's assertion.

Declare one exclusive unmounted child filesystem with its recorded GUID and a
second GUID-bound ZFS child mounted at an exclusive private `/root` anchor for
the metadata directory/lease. Verify the actual ZFS backing and mount; neither
guest root filesystem nor UFS softupdates publication is assumed supported.
Only test capture uses synthetic eligibility
to create intents. Actually publish/read queued/applying records and ownership
through the root store; the fixture claim intentionally does not establish full
runtime eligibility/admission. Execute the private fixed control on that child,
independently compare fresh postconditions, prove repeat denial and unchanged
applying metadata, then restore exact values/sources. Expired plans, modified
ledger and native context drift must leave storage unchanged. Actual EUID 65534
must be denied before native invocation. No durable success is inferred here.

Cleanup independently closes/removes only declared metadata and verifies absence;
verify both child GUIDs before destroy, verify absence, then remove only the empty
unmounted anchor and verify the original complete raw pool
projection. Parent pool/provider/backing, base/runtime/guest/host/process cleanup
remains mandatory. No live operator fixtures, daemon or persistent service runs.

```sh
make test-focus PKG=./internal/host/freebsd TEST='^TestPropertyControl'
PR_TITLE='feat(freebsd): bound native property controls' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

Original live root authorization/backend, complete launch eligibility and recovery,
native applying/effect/outcome crash cuts, global scheduling, reservation release,
transport/UI/CLI and complete M2 acceptance remain open. No runtime writer is added.
