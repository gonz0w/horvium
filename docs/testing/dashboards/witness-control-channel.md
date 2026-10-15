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
