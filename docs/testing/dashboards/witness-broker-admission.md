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
- Native small and projection 10,000: publish 32 actual collections through the
  real manager, unchanged helper and observer. Confirm every explicit published
  generation and final original census. Attempt 33 must fail the unchanged store
  limit without creating original 33. A subsequent manager refresh must be denied
  at admission without connector 34. Cached original eligibility stays denied.

These explicit diagnostic refreshes are single requests. No refresh retry,
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
