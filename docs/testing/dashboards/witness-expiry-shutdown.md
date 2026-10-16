# Collection reply deadlines and native shutdown controls

The helper reply completion path enforces its existing absolute deadline even when
context cancellation is not yet visible. The native witness controls below remain
inactive test-only qualification under [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md) and
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-004 / DASH-WIT-NATIVE-005.

Run `TestNativeDashboardWitnessExpiryAndShutdown` on both pinned disposable releases
with the existing signed clean review and source/artifact/ABI preflight. Six cases
keep the candidate's real product artifacts, 30-second interval, 45-second collection
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
for the initial generation. Retain original timeout status bytes/hash and the unavailable response bytes
and HTTP status. Status requests are named negative-control observations, not resource
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

## Reply-deadline correction

The receipt-retention candidate's 15.1 projection expiry retained an alive manager,
HTTP 503 and original status `protocol`, with one connector and no followup denial.
The observer recorded the original deadline crossing and zero-byte forwarding
failure. Its original native-small expiry passed; 14.5 was not started after the
projection failure. Both failed candidates remain separately retained.

The client previously checked only `ctx.Err()` after reading a reply. A socket's
deadline can expire before context cancellation becomes visible; the decoder then
reports a protocol failure, or a fully buffered late reply can escape the deadline
check. This ordering is a source-supported explanation, not a measurement of the
old manager's internal callback. Deterministic regressions reproduce both outcomes
with an actually expired Unix socket and a complete buffered late reply while the
context error is not yet published.

The reply completion path now checks the existing absolute context deadline as
well as explicit cancellation, rejecting late output as `DeadlineExceeded` before
publication. No new timer, wait, timeout extension or retry is introduced. On-time
malformed replies remain protocol failures; on-time valid data and explicit
cancellation retain their behavior. Authority also includes the approved
[M1 EDD](../../milestones/M1/EDD.md) and [Test Design](../../milestones/M1/TDD.md),
M1-REQ-002 / M1-REQ-006, M1-FAIL-001 / M1-SCALE-001. The candidate evidence
below binds the correction to both pinned native releases.

## Candidate evidence

Reviewed source `59bdb60828ef83ba9f91cdbd16821472e5b0bc87` passed all six short controls on
each pinned FreeBSD/amd64 release. The [immutable manifest](witness-expiry-shutdown-manifest.json)
binds exact source/base/title/review, compiler/artifacts/native probe, command/
controller, original bytes/indexes/status receipts, preflight and teardown.

| FreeBSD | Six-case native duration | Complete producer originals | Confirmed normal publications | Shutdown / expiry negatives |
| --- | --- | --- | --- | --- |
| 15.1 | 352.17s | 6 | 2 | 2 / 2 |
| 14.5 | 335.30s | 6 | 2 | 2 / 2 |

All exported complete originals were independently reopened for original framing,
echoed ID, object/owner/mode/hash and actor binding. Every projection stream contains
10,000 dataset records. Only the normal controls have delivered originals, sealed
sidecars and observed publication IDs; failed forwards retain complete undelivered
sources with zero accepted bytes and no write-half EOF/index/publication authority.

All four orderly-shutdown controls observed the original manager's successful
SIGTERM exit and reaping before release, then actual EPIPE errno 32 while the
original observer window was still open, without observer cancellation/timeout
flags. All four expiry controls crossed the original unchanged absolute 45-second
window and recorded actual deadline-class socket failure. Original timeout status
bytes/hash and unavailable JSON/status were reopened; the manager remained alive.
Actual context flags and elapsed times are preserved individually in the manifest.
No continuing-manager cancellation or internal manager-context claim is inferred.

Guest/host teardown, original helper/socket/configuration/token and pinned runtime
preservation passed. All 172 historical manifests and seven operator files remain
unchanged. The reply completion correction enforces the existing absolute deadline
without extending it. Deterministic expired-socket and late-reply regressions failed before
the correction and passed after; on-time valid/malformed and explicit cancellation
controls retain their behavior. Native negatives use the unchanged window and
budgets. No runtime write authority or service installation is introduced.

The earlier source's 15.1 native-small expiry failed a combined manager/status
predicate after four passing cases; projection expiry was skipped. Its actual
manager/status values were lost before the fatal assertion. The retained original
proves deadline forwarding failure only; the underlying predicate cause remains
unknown. Original 14.5's six passing controls and 15.1's failed/skipped boundaries
remain separately preserved in the manifest, including the fallback byte stream
and its export-generated reference (not an original sealed index). The corrected
candidate persists observations in cleanup before any assertion can discard them.
A second receipt-retention source subsequently recorded 15.1 projection status
`protocol` with an alive manager, HTTP 503, one connector and no followup denial;
its original native-small expiry passed, and 14.5 was not started after failure.
Both predecessors' guest/host cleanup passed. The socket/context-callback ordering
model is reproduced deterministically; callback scheduling inside the old manager
was not observed. The correction enforces the absolute reply deadline regardless
of callback ordering. New passing evidence does not reconstruct the first source's
missing predicate values or retroactively qualify either failed 15.1 gate.
Broader qualification and activation gates listed above remain open.
