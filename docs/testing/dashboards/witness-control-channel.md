# Native witness control-channel containment

Status: inactive test-only fixture; broader native/resource acceptance remains open.
Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-006 / DASH-WIT-007 / DASH-WIT-NATIVE-005.

A malformed or closed controller channel is observation uncertainty. Stopping
admission and cleaning up must also permanently mark the candidate failed. An
already decoded/cached original cannot remain eligible after a failed control.
Preserve its actual producer bytes and complete/delivered flags as historical
facts; those facts do not grant candidate qualification.

The existing bounded private control codec is shared with portable tests without
changing its wire representation. Exercise exact 64 KiB payload acceptance and
successive frame boundaries, zero/oversized/truncated frames, duplicate/unknown
fields, trailing JSON and depth rejection. Oversized headers reject before reading
the body; oversized outbound controls emit no partial frame. No limit is raised.

The native batch starts the actual product helper/manager through the original
owned observer. Confirm a real published original, then inject each malformed
control, EOF or unknown operation into the original inherited channel. Require a
failed final receipt, removed front socket, reaped observer/connectors, refused
followup control and refusal of the cached original while its root is still open.
An exact-cap valid readback followed by normal stop remains a successful control.
Actual helper identity/socket and both configuration files/token digest must be
unchanged. Export every original and final receipt before exact tmpfs/ZFS/md and
host image/key/listener/firmware teardown.

Run one signed clean reviewed source on both pinned disposable FreeBSD releases,
sequentially. Independently reopen framed receipts and full producer bytes, object/
hash/source/actor/ABI references and original preflight/teardown. Preserve failed
regression attempts and missing observations without promotion to acceptance.
Portable checks alone establish no actual process/cancellation behavior.

This covers controller-channel loss, not killing the actual controller with a
separately surviving verified owner. Full broker/caller capacity, remaining
cancellation/recovery, cadence/overhead/augmented budgets, historical 14.5 gaps,
300-second idle, original latency/full resources, browser/manual accessibility,
hardware and release gates remain open. Canonical resource runner stays inactive;
runtime writes remain denied. No product API/config, dependency, service or Action.

## Retained failing regression

Signed reviewed source `6eca553dc70b1c57a2bc6e5fb77f82ff21f3abbf` failed the
focused 15.1 EOF regression after 68.35 seconds. Its actual terminal receipt was
unfailed with no failure reason, and the previously cached complete original
remained eligible after a refused followup control. Its original producer bytes,
receipt and failed exit remain retained. Guest and host teardown passed. This
source has no full-matrix or 14.5 execution. Later fixes cannot upgrade that result.

The observer now latches failed control reads/writes before final cleanup. The
fixture caller permanently latches control errors and rechecks that latch before
and after original decoding, including cache hits. Cleanup/control reads can still
collect failure evidence; they cannot restore original eligibility.

## Typed control receipt correction

The first corrected full matrices at `a4e4e78314abe10e305cb0e940ee97c4971c7aca`
exercise actual unsupported-operation refusal, but that case's injection hash/size
describe a compact request template. The ordinary typed caller also sends zero
actor fields. Those records retain this representation limitation; their original
producer files and observed containment results remain unchanged.

A followup records bytes only after successful writes through the original
control channel, using the same caller and failure latch. Export the recorded
typed frame in an exclusive root-only receipt and independently reopen its full
encoding, size and hash. Run the affected unsupported-operation case on both
releases at the new signed reviewed source. The earlier full matrices are not
relabeled as executions of the followup source.

## Source-bound native observations

The [immutable manifest](witness-control-channel-manifest.json) preserves five
attempts, including the failed EOF regression. Full matrices at reviewed source
`a4e4e78314abe10e305cb0e940ee97c4971c7aca` passed nine adverse controls and one valid exact-cap control
on each pinned disposable FreeBSD/amd64 target:

- FreeBSD 15.1: 555.09s.
- FreeBSD 14.5: 498.24s.

The actual typed-control receipt correction at reviewed source
`543a61752db6b8aa036d668ce55cf4c1a1130abb` passed focused unsupported-operation checks on both targets:

- FreeBSD 15.1: 69.70s.
- FreeBSD 14.5: 59.92s.

These focused runs are not full matrices of the followup source. The earlier
unsupported-operation template hash/size retain their original limitation; both
new receipts bind independently reopened bytes from the actual typed writes.

Every adverse case produced a failed final receipt, refused a followup control and
denied the already cached original before closing its root. The normal exact-cap
control retained eligibility and stopped successfully. Actual complete/delivered
producer originals remain preserved with original object/hash references.
Independent readback reopened every original byte/frame/index, terminal receipt,
source/actor/ABI context, original preflight/runtime pins and exact guest/host cleanup.
The actual helper/socket and configuration/token stayed unchanged. These short
controls establish no full native 005, resource, browser or release acceptance.
