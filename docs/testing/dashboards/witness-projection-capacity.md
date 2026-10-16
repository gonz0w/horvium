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

Signed reviewed source `e5949969c894455b26bc40bb04e1469963521fc1` passed the full selected
projection attempt-cap gate on both pinned FreeBSD/amd64 releases. The
[immutable manifest](witness-projection-capacity-manifest.json) binds both runs to
that source, base, exact PR title, original review, compiler/artifacts, native probe,
command/controller and before/after preflight. Baseline checks and affected cmd
race/original-wire fuzz actually executed at this source before native execution.

| FreeBSD | Native test duration | Complete/delivered originals | Observed publication IDs | Largest complete snapshot, bytes including header |
| --- | --- | --- | --- | --- |
| 15.1 | 1191.60s | 32 | 32 | 29392 |
| 14.5 | 1139.10s | 32 | 32 | 29392 |

Each run preserved all 32 originals and sealed indexes, denied original 33, and
rejected a subsequent actual manager refresh before starting connector 34. Cached
witness eligibility stayed denied and the manager retained its last complete
publication. Independent readback reopened all producer frames, exact 10,000 dataset
counts, full delivered hashes/EOF observations, original object/actor/incarnation
bindings, publication metadata, five bounded snapshot shards, raw/index census,
helper/configuration/token preservation and guest/host teardown. Observed publication
IDs include the first and last originals; intervening IDs not observed by the
controller are not retrospectively confirmed.

15.1's healthy Before snapshot records `active=true`, one complete/delivered
original and one already reaped connector (`connector_active=0`). The existing
native gate does not require that pre-fault snapshot to be quiescent. Preserve this
flag explicitly; both releases' capacity, boundary, followup and final snapshots
are inactive, with no active connector. Independent readback distinguishes these
states rather than treating Before as terminal cleanup evidence. Both untouched
original receipt sets pass that predicate; fourteen mutations introducing terminal
activity, a live connector or unreaped-connector counts are rejected.

The intended terminal failure is the attempt-33 capture limit, after 32 successful
forwards. The retained terminal forward receipt belongs to successful original 32;
it is not a forwarding failure for attempt 33. All source facts and failure states
remain separate. Both candidates remain globally native/resource unqualified.

This is new source-bound component evidence. It does not establish the cause of
the historical 14.5 fifth-original failure or change that failed observation.
The complete three-case admission sequence and its cumulative preconditioning/order
were not executed at this source. Full broker/resource activation and broader
acceptance remain open.

The host dispatcher stopped before launching 15.1 because exclusive log creation
protected an older existing output file. 14.5 was already complete and independently
verified. The first 15.1 execution resumed at the same signed source/controller
with a fresh output path; no native test was replayed and the old log was preserved.
