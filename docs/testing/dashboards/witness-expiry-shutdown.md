# Original collection expiry and orderly manager shutdown

Status: inactive test-only controls under [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md) and
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-004 / DASH-WIT-NATIVE-005.

Run `TestNativeDashboardWitnessExpiryAndShutdown` on both pinned disposable releases
with the existing signed clean review and source/artifact/ABI preflight. Six cases
keep the original product artifacts, 30-second interval, 45-second collection
window and all byte/record/attempt/archive/tmpfs limits. An 18-minute outer diagnostic
allowance covers six separate owned fixtures; it does not extend a collection or
change a resource budget. Normal controls confirm exact-ID native-small and
projection-10,000 publication without any fault channel.

For each negative case, hold the initial real manager call at the existing sealed
original barrier, after complete capture and connector reaping but before delivery.
The sealed event now records that connection's actual start and absolute deadline.
The normal/timed path gains no fault channel or metadata request. Other fault
stages retain their existing behavior and bounds.

Orderly-shutdown cases send SIGTERM to the original manager, observe its actual
successful exit/wait and ended lifetime, then release while the original observer
window is still open. Require actual broken-pipe errno 32 with zero accepted bytes,
a complete undelivered original, and no observer timeout/cancellation flags.
This observes signal-driven process shutdown, not cancellation inside a continuing
manager lifetime; it does not inspect the manager's internal context error.

Expiry cases wait until the **original absolute connection deadline**, then release
the sealed source without resetting any socket or context deadline. Require an
actual deadline-class forwarding error with zero accepted bytes, a crossed original
window, unchanged complete source and no delivery/publication. Preserve actual
observer context flags rather than reconstructing them from elapsed time. The real
manager must remain alive, report its collection timeout, and return unavailable
for the initial generation. Retain the original timeout status bytes/hash and unavailable response bytes/
HTTP status. Status requests are named negative-control observations, not resource
workload samples or request-free idle evidence.

Both negative families deny original lookup, retain the failed candidate and exact
forwarding facts, then stop/reap/export and independently verify original bytes,
missing versus present sidecars, actual actors/helper/socket, both configurations/
token, and guest/host teardown. An original sealed reply is not automatically an
eligible or published witness. Failed timing, barrier, status or cleanup observations
remain failed; never retry the workload or infer missing flags.

These are source-bound short controls for expiry and orderly shutdown at the sealed
boundary. They do not qualify continuing-manager cancellation, other interruption
stages, raw/archive broker byte limits, later storage-failure stages, whole admission
ordering, cadence/calibration/overhead, full resource/idle/latency, browser/manual,
hardware or release acceptance. Failed historical candidates remain unchanged.
Canonical resource runner stays inactive; runtime writes stay denied.

The first candidate passed all six controls on 14.5, but 15.1 failed the combined
manager/status assertion in native-small expiry after four passing cases. Projection
expiry was skipped by fail-fast. The retained source proves original deadline
forwarding failure; the actual manager/status predicate values were lost before
the fatal assertion, so their cause remains unknown. Guest/host cleanup passed.
A separate signed diagnostic candidate must retain observations on failure before
any predicate is asserted; an isolated diagnostic is not the six-case matrix.
