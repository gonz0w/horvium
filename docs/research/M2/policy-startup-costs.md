# Diagnostic: retained-policy native collection costs

Status: experiment declared before execution; no lifecycle qualification.
Date: 2026-10-11. Owner: Cam (`@gonz0w`). Time box: one development session.
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305) and
[parent 139](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted retained-policy design](../../milestones/M2/retained-policy-startup.md)
and the maintainer's request to continue investigating the merged
[failed lifecycle gate](../../testing/M2/retained-policy-native-lifecycle.md).

## Question and method

Which read-only native collectors and guarded command launches consume time in
this disposable amd64/TCG lab? Measure independent primitives, not an alternate
root factory or a relaxed lifecycle. All product checks, command containment and
deadlines stay unchanged. Do not cache facts, share observations between policies,
remove repeated reads, change the CPU profile again or infer production latency.

Use one new tagged checksum-bound executable on both pinned FreeBSD/amd64 releases,
serially, in the existing one-CPU, 6 GiB, seven-device disposable lab. One parent
retains the original eight-minute fact/data fixture and all its 21 controls. Its
callback measures nine independent stages in a fixed order:

1. Eight direct `/usr/bin/true` commands, used only as a launch comparison.
2. Eight originally guarded `/usr/bin/true` commands, with unchanged base timeout
   reaper and descendant containment. Direct commands cannot replace product reads.
3. Complete system collection with both original data pools present.
4. First policy pool properties, then its actual provider collection.
5. Second policy pool properties, then its actual provider collection.
6. Actual fixed persistent-backing collection from owned UFS and ZFS directories.

The five system/property/provider stages instrument only their existing private
reader seams, forwarding every call and result to the unchanged original native
reader. Count calls and elapsed nanoseconds; log no names, arguments, output or
inventories. Such wrappers are diagnostic, never fixed-root acceptance evidence.
The two actual fixed backing collectors are opaque: record stage wall time only,
with zero reader count/time denoting **unmeasured**, not zero native commands.
Each no-op sample has a five-second context; collectors keep their existing bounds.
Every stage must return successfully. No timing threshold can establish support.

Require all nine distinct ordered timing rows and the two metadata fixture domain/
mount cleanup controls, all 21 original fact/data controls, one normally completed
parent/process, exact kernel one-CPU profile and executable checksum, independent
host/device/guest/artifact/key/supervisor cleanup and base/runtime preservation.
Missing, extra, duplicated, failed or skipped rows/parents/controls or incomplete
cleanup invalidate the diagnostic. Preserve all nine prior retained-policy receipts.
The normal build excludes this fixture. Add failing-first consumer tests before
implementation and run signed clean-source mandatory portable review.

## Interpretation and next work

Code inspection shows each policy inspection performs complete domain/system
observations before and after per-policy collection; activation independently
inspects before entropy, before publication and after acknowledgement. These
repetitions protect distinct drift/publication boundaries. Primitive timings do
not measure the actual factory, show where a failing call expired, isolate TCG
cost from filesystem/kernel cost, or justify removing a check. No public operation,
product scheduling, persistent format, dependency, service or runtime authority.

Actual factory activation/observation/retirement and the second clean process
cycle remain unqualified on both releases. If primitives indicate launch overhead,
repeat the unchanged acceptance gate on a suitable disposable amd64 lab before
changing product design. An ARM accelerated guest would be a different declared
architecture; it cannot substitute for the existing amd64 acceptance receipts.

## Rejected initial diagnostic and corrected inventory

Source `d36e309c2c101135a9aebf19c3ea43e15200b4fb` completed all nine stages,
metadata cleanup and the fact/data parent on 15.1, but the consumer rejected its
system read count. It expected 22 reads (two pools); the declared cached guest has
an imported root ZFS pool in addition to both data pools. The original system
collector correctly read all three: two passes of five global probes plus three
probes per imported pool, **28 reads**. Root exclusion does not omit root-pool
observation. Preserve the rejected attempt as unqualified with complete host
teardown; 14.5 and portable review were not run for that source.

Before rerunning, correct only the expected diagnostic inventory to 28 system
reads, retaining eight direct/eight guarded launch calls, sixteen property and
sixteen provider reads for each policy, and opaque backing counts of zero
(unmeasured). All existing cases, deadlines, root exclusion, collector behavior
and independent cleanup stay required. A new failing-first consumer test must
accept the actual 28-read fixture and refuse a missing or incorrect count.
