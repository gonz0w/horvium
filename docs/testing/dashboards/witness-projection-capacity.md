# Projection attempt-capacity qualification

Status: inactive test-only qualification under [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md) and
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-007 / DASH-WIT-009 / DASH-WIT-NATIVE-005.

The [original full-capacity attempt](witness-broker-admission.md) failed on 14.5
after five projection captures. Its underlying cause remains unknown; the passing
[focused diagnostics](witness-forward-diagnostics.md) do not qualify that gate.
Use a fresh signed candidate with those diagnostics for one complete existing
projection attempt-cap procedure on each pinned release, starting with 14.5.
Never rerun an old failed candidate until it passes or reconstruct its missing facts.

## Bounded procedure

Select the existing projection subtest without reducing its workload:

```sh
cd /root/work/cmd/horvium
env HORVIUM_NATIVE_VM=1 HORVIUM_PERF_VM=1 /root/work/.tools/dashboard-native.test -test.v -test.count=1 -test.timeout=30m -test.failfast -test.run='^TestNativeDashboardWitnessAdmission$/projection-10000$'
```

Require the existing native source/artifact/ABI preflight, exact source/base/title
and candidate context binding. Keep all 32 actual collections, including scheduled
ones, observed publication IDs, attempt 33 failure and permanent followup denial.
The manager's last complete publication must remain; cached witness eligibility
must be denied after failure. Each control receipt stays bounded by 64 KiB.
Preserve all originals, exact forwarded-prefix observations, original indexes,
five separate control snapshots, actual helper/management incarnations, both
configurations and token digest, then reap/stop and verify guest/host teardown.

The portable attempt-cap regression checks all 32 actual store captures/forwards
and preservation of their diagnostic census after attempt 33 and repeated denial.
It measures the reference census only; complete native metadata-envelope bounds,
real broker/caller admission and publication require this separate native gate.
This adds coverage for existing behavior, with no product or broker logic change.

Use the new original forwarding receipt only for its matching request/incarnations.
A last successful forward's receipt cannot explain a later pre-forward failure.
If a native gate fails, retain its actual error class/errno, prefix bytes/hash,
original deadline/elapsed/context observations, diagnostics and fallback originals.
Complete capture and accepted socket bytes remain separate from delivery/EOF and
observed publication. Missing evidence stays unknown. A concrete correction needs
regression coverage, a new signed review and relevant fresh execution.

Keep the 30-second interval, 45-second collection deadline, 32-attempt limit,
64 KiB relay/control, 64 MiB/100,000 normal wire, 67,508,864 raw-byte ceiling,
256 MiB archive and 320 MiB tmpfs unchanged. No artificial pacing, queued retry,
cap increase, fixture-count reduction or budget change is permitted. The outer
30-minute diagnostic allowance is the existing full-cap test allowance.

This is a projection-emitter capacity control on the small owned native topology;
it is not 10,000 actual ZFS datasets, the 100-owned-dataset resource workload or a
hardware support claim. Native-small/control-failure and transport observations
retain their previous source boundaries. Raw/archive/physical/overlap controls,
remaining recovery/cancellation, cadence/calibration/overhead, full resources,
browser/manual accessibility, hardware and release acceptance remain separate.
Canonical resource runner stays inactive and runtime writes remain denied.

## Candidate evidence

Record independently reopened candidate outcomes after execution, preserving
failed and incomplete observations without upgrading older evidence.
