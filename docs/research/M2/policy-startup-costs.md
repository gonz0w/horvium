# Diagnostic: retained-policy native collection costs

Status: diagnostic completed on both pinned releases; actual lifecycle remains unqualified.
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
6. Actual fixed persistent-backing collection from owned ZFS then UFS directories.

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

## Release-specific root and metadata setup correction

The corrected 15.1 diagnostic at source `b119f961a52efe630f0dad75162c28f1e740f51f`
completed with 28 system reads. The same source's 14.5 attempt observed 22 reads
and failed during ZFS metadata setup after the UFS stage (parent 27.60 seconds).
Preserve the failed run and verified independent host teardown. Do not infer its
ZFS backing or accept an incomplete in-guest parent/control vector. The command
failure alone does not establish its underlying cause.

Declare the correction before another run: require the original system collector's
actual root mount type and complete imported-pool count, recording only those two
scalars. The pinned profiles permit UFS root/two pools or ZFS root/three pools;
require system reads to equal `2 * (5 + 3 * pool_count)` (22 or 28). Reject unknown
or contradictory roots/counts; no inventory names are logged or pools omitted.
All non-system stage counts remain exact. Run the owned ZFS backing stage before
UFS on the single declared spare, avoiding the UFS-to-ZFS setup sequence without
forcing creation, clearing devices or changing the shared metadata harness. The
ordered stage/cleanup inventory becomes ZFS then UFS. No time limit or product
check changes; both corrected release runs still must complete independently.

## Completed diagnostic and evidence limits

Signed source `e2b0360d9422ef864a4d94ab0a50eabfd3f9c968` completed all nine ordered stages,
the original 21 fact/data controls and six independent metadata cleanup checks on
both pinned FreeBSD/amd64 releases, using the same checksum-bound executable.
The original system view reported ZFS root/three imported pools/28 system reads
on 15.1 and UFS root/two imported pools/22 reads on 14.5. Both ZFS-first and UFS
backing collection completed. Complete host/device/guest/key/artifact/supervisor
cleanup and base/runtime preservation passed. The [receipt](policy-startup-costs-manifest.json)
records original source/base/compiler/artifact/log/record hashes, all stage values,
the earlier successful single-release diagnostic and both rejected candidates.
All nine earlier retained-policy receipts remain byte-identical.

| Independent stage | 15.1 seconds | 14.5 seconds |
|---|---:|---:|
| direct-launch | 0.167 | 0.127 |
| guarded-launch | 0.293 | 0.208 |
| system | 3.736 | 2.855 |
| first-properties | 2.497 | 2.331 |
| first-providers | 2.329 | 2.168 |
| second-properties | 2.566 | 2.304 |
| second-providers | 2.372 | 2.338 |
| zfs-backing | 1.870 | 1.747 |
| ufs-backing | 1.501 | 1.374 |


Launch values are totals for eight fixed no-op samples. Backing stages are opaque;
zero reader metrics mean unmeasured. The two roots and versions differ, and fixed-
order single samples are not a controlled benchmark or production performance
comparison. Reader wrappers forward actual native results but do not measure the
actual fixed factory, its lock/object/publication work or the failing call's
location. These times cannot be added and presented as measured factory runtime.

### Source analysis, not execution tracing

Inspection of the unchanged successful two-policy code path finds six complete
system collections, twelve fixed backing collections, two pool-property and two
provider collections in each policy inspection. Initial review inspects once;
activation independently inspects three times, observation once and unused finish
twice. For UFS metadata and the observed two/three imported pools, this implies
244/280 guarded reader calls per inspection, or **976/1,120 for startup**. The
estimate excludes file/ACL/boot syscalls, publication work and command-internal
processes; it is not a measured call trace or supported-capacity claim.

The primitive samples and source repetition support investigation of cumulative
collector work. They do not prove that emulation causes the failures or that the
base reaper alone dominates; guarded no-op samples are much shorter than complete
collector stages. Every repeated read protects an adopted drift/publication
boundary. Do not remove it, reuse facts, renew contexts or enlarge deadlines based
on this diagnostic.

The useful next comparison is the unchanged actual acceptance gate on a dedicated
disposable amd64 lab. The existing runner is TCG-only; a hardware-accelerated lab
profile requires its own declared configuration, preservation/containment checks
and source-bound receipts before acceptance. No host or persistent runner was
installed. ARM acceleration would be a different architecture; the existing boot
identity implementation is explicitly amd64-only and requires separate design/
qualification before another architecture can provide that authority.

Signed clean-source local review passed 207 tooling
tests, ordinary Go tests, all eleven race packages and 54 fuzz targets. The final
evidence-only head requires a fresh baseline with exact-input deep reuse. This
qualifies diagnostic completeness and the dry-run CPU-description correction only.
Actual lifecycle, broader issue 305/139 acceptance, configured startup/workers/
backend/global launch and browser/client/hardware/performance/release gates remain
pending. No production behavior, time bound, guard, schema, dependency or runtime
permission changes; runtime writes remain disabled.
