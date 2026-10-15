# Broker attempt capacity and permanent admission denial

Status: inactive test-only fixture; corrected native observations pending.
Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-007 / DASH-WIT-NATIVE-005.

The custody component already exercises its full byte/attempt/physical limits.
This batch adds actual broker/caller attempt controls. It also tests admission
after a permanent observer failure: a failed candidate must reject subsequent
connections before creating a connector or another original. The admission latch is checked under the original observer mutex; evidence
readback remains available after failure. Same-UID unrelated
peer rejection remains a separate nonfatal denial; it cannot clear a failure.

## Observable controls

- Control failure: first confirm one actual native publication and original,
  then send the typed unsupported control operation through the real channel.
  A subsequent single authenticated manager refresh must be rejected without
  another connector, original or publication, even though the store itself has
  not failed. Retain the exact injected control bytes.
- Native small and projection 10,000: capture 32 actual collections through the
  real manager, unchanged helper and observer. Confirm observed publication IDs,
  including first and boundary IDs, and every original in the final census.
  Scheduled collections count too; intermediate publication IDs not observed by
  the controller are not retrospectively confirmed. Attempt 33 must fail the unchanged store
  limit without creating original 33. A subsequent manager refresh must be denied
  at admission without connector 34. Cached original eligibility stays denied.

These explicit diagnostic refreshes are single requests. A refresh that finds a
scheduled collection already busy joins that original operation after its existing
409 response; it never retries the POST or queues extra work. No refresh retry,
background collection pacing, seeded counter/byte state, cap increase or reduced
mount is allowed. The existing 30-second interval and 45-second trusted deadline
stay unchanged. An incidental extra collection fails the expected census. These
controls do not measure workload pacing or resource performance.

Keep healthy pre-fault, failed boundary, failed followup and terminal receipts
separate. Observe the real manager's failed collection status and retained prior
complete generation. Preserve original actors/artifacts, both configurations and
token digest. Reopen all raw originals and index sidecars by exact object/hash;
export before unmount. Reap the original observer and every started connector,
verify front removal, owned tmpfs/ZFS/md and host runtime teardown.

## Evidence and remaining gates

Run the new regression against the old admission behavior before correcting it.
Preserve any failed source, native exit, actual original bytes and cleanup outcome.
A corrected source needs signed clean review and both pinned FreeBSD releases;
passing later runs never upgrade failed predecessors.

Full broker raw/archive/physical capacity and overlap, remaining cancellation/
recovery, cadence/overhead/augmented budgets, historical 14.5 gaps, 300-second idle,
original large latency failure, full resource/browser/manual accessibility/hardware/
release acceptance remain open. Actual controller interruption is covered by its
[separate surviving-owner evidence](witness-controller-interruption.md).
Canonical resource runner stays inactive; runtime writes remain denied. No
runtime API/configuration/dependency, service installation or Action is added.

## Incomplete full-cap predecessor

Source `08fd369f893cd24d4ecf95d0bc639bcc0def2705` passed the actual
control-failure regression on 15.1, then failed readiness in both capacity cases
(759.70 seconds overall). Fallback cleanup retained 31 complete/delivered native
originals and one complete/delivered projection original, with original indexes.
Those cases have no complete case/publication receipts or exported terminal failure
reason, so the exact earlier failure boundary remains unproven. Guest and host
teardown passed. 14.5 was not run; the full-cap predecessor remains failed.

Review also found two fixture weaknesses: counting manual refresh calls does not
count actual scheduled collections, and combining four full control receipts can
exceed the unchanged 64 KiB metadata ceiling. Count actual captured originals
and retain every captured attempt, allow genuine request-ID/publication gaps, and
reference separate bounded immutable receipt frames. Add failure-only diagnostic
readback before fallback cleanup so future early failures retain their actual
reason. These corrections do not establish the unknown predecessor cause. The
outer diagnostic allowance becomes 30 minutes for full-scale I/O; trusted product/
collection/connector deadlines, scheduling and all byte/attempt bounds are unchanged.

Source `eff95c79f0430dff7a048a06a3775b49cec444fb` then failed on 15.1
startup (83.59 seconds): the revised loop called the ready-only overview helper
before the first inventory publication. The early diagnostic records a healthy
but active observer with connector 1; fallback preserved one complete, undelivered
original. Guest/host teardown passed. No admission outcome, capacity cases or 14.5
execution is established. Restore the existing initial publication wait before
using the overview helper; no product readiness or deadline changes are needed.
