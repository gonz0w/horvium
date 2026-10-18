# Authenticated refresh overlap through the native witness broker

Status: inactive test-only control under [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md) and
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-004 / DASH-WIT-007 / DASH-WIT-NATIVE-005.

Run `TestNativeDashboardWitnessRefreshOverlap` on both pinned disposable releases
from clean signed review, with original source/artifact/ABI preflight. Run native-small
and projection-10,000 separately. Hold the real manager's initial collection at the
existing sealed-original barrier, after its one connector has been reaped and
before delivery/publication. Dispatch exactly three concurrent authenticated
`POST /api/refresh` requests using the original session and CSRF protection.
Retain bounded raw response bodies, status and start/finish times; require actual
409 busy replies within the collection's original 45-second deadline.

Before and after these requests, require the identical one complete, undelivered
original, one started/reaped connector, no active connector, unchanged denial count
and no broker/store failure. Release the barrier once. Observe the exact initial
request ID through the real manager's overview, then independently decode its
immutable original. Send one later refresh request: require 202 refresh_requested
or a genuine 409 busy join, with no retry. Observe a distinct complete publication
and resolve both its original and the retained initial original.

The unchanged 30-second schedule can collect additional originals after release.
Export every actual original/index and report any additional count separately;
do not infer its trigger from delivery alone. Only the two overview observations
confirm publication. Require every captured original complete and delivered, exact
forward hash/EOF, original deadline and no permanent failure. Stop/reap the observer,
preserve the helper/socket, configurations and token digest, and independently
verify guest and host teardown. Do not change caps, deadlines, scheduling or retry
failed candidates. Failure cleanup retains partial HTTP/control observations.

Portable behavioral tests first require three simultaneous HTTP requests, exact
busy/accepted response contracts, bounded reads, transport/partial-read failure,
no retry and draining all terminal replies when one fails. These are fixture
controls, not a new runtime API or product logic change. The existing manager
single-flight test separately covers its no-queue contract.

This adds authenticated original-manager overlap coverage beyond the
[unrelated-peer rejection and graceful drain](witness-overlap-drain.md) controls.
It does not qualify metadata coalescing, all admission ordering, arbitrary
cancellation, numerical invariance, resource/idle/latency/overhead, browser/manual,
hardware or release acceptance. Canonical resource instrumentation remains inactive;
runtime writes stay denied. Historical failed/skipped/unknown limits remain intact.

## First native candidate

Source `ce334499a103286b33f139ffc6e74a7180b86159` failed FreeBSD 15.1 observer
readiness (41.66s overall / 16.74s native-small). The exported child log explicitly
reports `observer configuration`; the new refresh labels were absent from the
existing fault-label allowlist. No collection or refresh request was exercised,
projection did not start, and 14.5 was not started. Guest and host teardown passed.
The correction registers only the two exact fixture labels and checks rejection
of near-miss/path-like labels; no product behavior, caps or deadline changed.
A corrected source requires fresh signed review and both native releases.

## Corrected candidate evidence

Source `e5c62c393c578fb571ad57b334d54f4da713cd44` passed both cases on each pinned
FreeBSD/amd64 release after fresh signed review. The [immutable manifest](witness-refresh-overlap-manifest.json)
binds source/base/title, compiler/artifacts/ABI, actual commands/controllers,
preflight, original bytes/indexes/actors, bounded HTTP observations and teardown.

| FreeBSD | Native suite | Busy replies | Confirmed publications / originals / indexes | Later refresh statuses | Additional originals |
| --- | --- | --- | --- | --- | --- |
| 15.1 | 146.32s | 6 | 4 / 4 / 4 | 202 / 202 | 0 |
| 14.5 | 122.71s | 6 | 4 / 4 / 4 | 202 / 202 | 0 |

Independent readback reopened every original's framing, echoed request ID, end
count and native/projection dataset counts, exact hashes/forward EOF, sealed
object/index and actor/artifact lineage. The native fixture independently decoded
both published originals and rechecked the retained initial one. Before/Held
receipts retained exactly one undelivered original and one reaped connector, with
unchanged denial counts and all three busy replies inside the original deadline.
Both later requests per release returned 202; no request retry occurred.

Guest storage/process/socket/tmpfs and host process/listener/key/image/variables
teardown passed. Fresh signed corrected-source review executed baseline checks
and affected-package race/fuzz; final evidence documentation repeats the baseline
and may reuse only that exact-input deep evidence. All 178 historical manifests
and seven operator files remain unchanged. The first failed candidate stays failed
with projection and 14.5 unstarted, rather than acquiring the corrected source's
coverage. Broader gates and inactive/denied runtime boundaries above remain.
