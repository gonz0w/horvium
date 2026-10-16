# Broker overlap rejection and graceful drain

Status: inactive test-only controls under [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md) and
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-007 / DASH-WIT-NATIVE-005.

Run `TestNativeDashboardWitnessOverlapAndDrain` on both pinned disposable releases
with the original source/artifact/ABI preflight and clean signed local review.
Each real native-small and projection-10,000 manager's initial collection is held
at the existing sealed-original fault barrier. The connector has already exited;
the complete original has not yet been forwarded or published.

Two actual unrelated processes run the existing impostor control under the
management UID/GID. Both must reach the front listener and receive rejection
within the existing five-second socket bound, while the original call remains
held. The denied count increases by two, with no new connector, queued capture,
original mutation or latched candidate failure. Release the barrier; the accepted
original must forward unchanged, publish its exact request ID and resolve through
original metadata lookup. Stop and reap the owned observer normally.

Separate drain cases repeat that overlap control, then send exactly one normal
stop request while the original is held. Bounded socket-connect probes observe
listener closure before release; they carry no inventory request and are not
workload retries. Require ECONNREFUSED/absence and the observer's original lifetime
still alive, then release. Stop must drain the existing call, retaining one
complete/delivered original and one reaped connector, unchanged byte hash and
write-half EOF, the original 45-second deadline and no deadline/cancellation flag.
Drain delivery does not confirm publication: there is no post-stop metadata lookup
or publication claim for these two cases. Record any closure-probe admissions
separately from the two fixed impostor denials.

Export original bytes and sealed indexes, Before/Rejected/Final control facts,
original actors, deadline and forwarding observations, helper/socket preservation,
both configurations and token digest preservation, then independent guest/host
teardown. Preserve failed candidates without retry or changed caps/budgets.

These controls qualify unrelated same-UID rejection during an active legitimate
call and graceful owner completion at the sealed boundary. They do not establish
an overlapping authenticated original-manager request, raw/archive/physical broker
exhaustion, actual collection expiry, ordinary cancellation, signal/controller
loss at other stages, whole admission ordering or full resource acceptance.
Existing source-specific evidence remains separate. Canonical resource runner is
inactive; runtime writes stay denied. Hardware/browser/manual/release gates remain.

## Candidate evidence

Reviewed source `abacef732f249ab1edc3b64fd10ffe0186a32af1` passed all four controls on each pinned
FreeBSD/amd64 release. The [immutable manifest](witness-overlap-drain-manifest.json)
binds source/base/title, original review, compiler/artifacts/ABI probe, exact
command/controller, original byte/index readback and before/after preflight.

| FreeBSD | Four-case native duration | Complete delivered originals/indexes | Overlap peers rejected | Publication-confirmed originals |
| --- | --- | --- | --- | --- |
| 15.1 | 193.48s | 4 / 4 | 8 | 2 |
| 14.5 | 209.98s | 4 / 4 | 8 | 2 |

All eight case originals and their sealed indexes were independently reopened.
Their real framing, echoed IDs, exact full forward hashes/EOF, object/mode/owner,
actor incarnations and unchanged 45-second deadline were checked. Projection
sources each contain exactly 10,000 dataset records. In every case, Before and
Rejected retain the same one complete undelivered original and one reaped connector;
two actual unrelated management-UID peers are denied without queued work or
permanent failure. The overlap cases then confirm actual publication; the drain
cases first observe listener refusal while the original observer is still alive,
then release and retain delivery without claiming publication confirmation.

Guest cleanup and host process/listener/key/image/variables teardown passed, with
original helper/configuration/token and pinned runtime preservation. All 170 prior
manifests and seven existing operator files remain unchanged. This adds coverage
for existing test-only broker behavior, with no product or broker logic correction;
there is no fabricated failing regression. Broader gates listed above remain open.
