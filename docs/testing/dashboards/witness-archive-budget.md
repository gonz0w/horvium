# Aggregate witness archive exhaustion through the broker

Status: inactive test-only control; candidate results pending.
Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[accepted EDD](../../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-007 / DASH-WIT-NATIVE-005.

The original archive allowance is 256 MiB per guest candidate, shared across
native/projection modes. The existing direct-store controls exercise that bound;
this control sends a real manager collection through the actual broker after
counted, physically captured originals consume all but 80 bytes of that allowance.
No arbitrary counter, reduced cap, smaller mount or replayed workload supplies
pressure. Synthetic pressure is distinct from real producer bytes.

## Original bytes and ordering

First run a normal native-small product helper/manager through the observer.
Confirm its actual publication and export every original/index by object/hash.
Count scheduled originals too. Preserve those actual exported bytes as the prior
archive amount; a manual refresh count is never a collection count.

Before starting the next management process, capture fixed zero streams in a
fresh root-only store on a separate owned **320 MiB** tmpfs. Each real capture is
bounded by the unchanged raw cap (67,508,864 bytes) and attempt limit; exact EOF
makes these malformed synthetic originals incomplete without a storage failure.
Recheck each original object/content and sum its actual recorded byte count with
the preceding normal originals. The total must be exactly 268,435,376 bytes,
leaving 80 of the unchanged 268,435,456-byte archive allowance. Portable boundary
checks reject negative/out-of-range amounts, overrun, missing or repeated slots.
Arithmetic itself grants no custody or qualification authority.

Pass that measured aggregate through the existing private `PriorArchiveBytes`
bootstrap, using a fresh empty observer archive. The actual pressure originals
stay sealed on their original tmpfs until the real producer collection finishes;
they are not exported to guest ZFS during that collection. All original deadline,
schedule, frame, raw, archive and tmpfs bounds remain unchanged. The pressure
setup precedes management launch and consumes none of its collection window.

Require actual original management/helper credentials and a reaped connector,
then an 80-byte incomplete producer prefix with `archive-limit`. This is observer
capacity failure, not a source model-limit success or physical ENOSPC. There is
no forward, publication or original index for that failed capture. The original
copy boundary has no syscall error. A single authenticated refresh (or its genuine
busy join) must stay denied without another connector or changed original/boundary.
Retain the actual manager failure status and unavailable response separately.

Count the 80-byte prefix with the prior actual originals: exactly 256 MiB total,
never more. Reopen/export all normal, pressure and failed-prefix bytes before
identity-checked unmount. Actual pressure files are synthetic zero streams, not
whole product replies, native accounting witnesses or publications. Their source
class, root/object/mode/hash, original captured sizes and aggregate receipt remain
explicit. No claim that a normal product producer emits 256 MiB is made.

## Reproduction and teardown

After exact signed-source review, original image/QEMU/firmware/compiler/artifact
and guest ABI preflight, use the fixed short selection on each pinned release:

```sh
cd /root/work/cmd/horvium
env HORVIUM_NATIVE_VM=1 HORVIUM_PERF_VM=1 /root/work/.tools/dashboard-native.test -test.v -test.count=1 -test.timeout=15m -test.failfast -test.run='^TestNativeDashboardWitnessArchiveBudget$'
```

The fifteen-minute allowance is an outer diagnostic limit for normal capture,
physical pressure and export, not a changed collection deadline. The host archive
transfer has its own 180-second allowance; the original 45-second collection
window is unchanged. The original archive still fits the 272 MiB export ceiling.
Run both pinned disposable releases sequentially, keeping failures/skips without
same-source retries. Reopen original archives, strict receipt frames, source/build/
probe/actor bindings and actual producer versus synthetic bytes independently.
Verify original sockets/processes/configuration/token, both tmpfs mounts, owned
ZFS/md and host image/key/listener/firmware teardown. Export/cleanup observations
must survive assertion failure; uncertain cleanup remains unqualified.

## Boundaries

This verifies an actual broker's aggregate remaining-byte path using a measured
mixture of normal producer and synthetic pressure originals. It does not establish
full-scale ordinary producer traffic, projection-specific byte exhaustion, the
single-original raw limit through a normal caller, all admission orderings or a
resource workload. Direct-store capacity and physical broker ENOSPC retain their
separate evidence. Remaining cancellation/recovery, cadence/overhead, full resource/
idle/latency/activation, browser/manual accessibility, hardware and release gates
remain open. Canonical resource instrumentation stays inactive; runtime storage
writes remain denied. No product interface, dependency, service or Action changes.
