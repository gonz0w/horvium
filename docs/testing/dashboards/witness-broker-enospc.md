# Broker failure on an exhausted witness mount

Status: inactive test-only qualification under [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md) and
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-007 / DASH-WIT-NATIVE-005.

Run `TestNativeDashboardWitnessBrokerENOSPC` on both pinned disposable releases
with signed clean review and the existing source/artifact/ABI preflight. Keep all
original limits and product artifacts. Two normal controls confirm native-small
and projection-10,000 original capture, delivery and exact-ID publication.

Two separate cases fill the actual owned 320 MiB tmpfs before launching management.
Use real writes at decreasing granularity through one byte; observe actual ENOSPC
and refusal of a new one-byte allocation, then reopen/hash the actual filler and
verify its original object. Filler bytes are outside the original archive and do
not seed its counters. All filling/readback occurs before a collection starts;
the trusted 45-second collection deadline is unchanged. Do not shrink the mount,
change the 256 MiB archive cap or use operator storage.

Launch the unchanged manager/helper through the real observer/connector. Require
actual capture ENOSPC, failed observer/store, one reaped connector and no complete,
delivered or published original. The bounded capture diagnostic records original
request/actor incarnations, failing operation, acknowledged copy bytes and error
class/errno without paths or raw errors. A failed open has no captured original;
copy acknowledgements do not substitute for sealed original file size/hash. No
capture diagnostic grants witness eligibility. Later failed admission must retain
that original diagnostic unchanged. Portable tests cover real exclusive-open
failure, interrupted-source cancellation, successful capture and repeated denial.

Remove the owned filler and prove a fresh small-file allocation succeeds on the
same mount. Join the original failed collection, make one refresh request (or join
an already busy scheduled attempt), and observe permanent denial with no second
connector/capture or publication. No workload retry, counter seeding or eviction.
Readback remains available for failure evidence; original lookup remains denied.
Stop/reap, export every actual original prefix/index and independently verify
original helper/socket, both configurations/token, guest and host teardown.

The filler itself is reopened and hashed in the guest before removal; its hundreds
of MiB are not exported as a producer original. Receipts preserve that distinction.
A zero-byte/incomplete original is failure evidence, not a valid source or a
source-model boundary success. Normal controls and ENOSPC cases have separate
actors, publications and originals; do not inherit a control's qualification.

This covers physical exhaustion before capture on the actual full-size broker
mount. Raw/archive byte limits, exhaustion during later seal/index/forward stages,
whole admission ordering, actual deadline expiry and remaining cancellation/
recovery, cadence/calibration/overhead, full resource/idle/latency, browser/manual,
hardware and release acceptance remain separate. Historical failed/skipped evidence
is unchanged. Canonical resource runner stays inactive; runtime writes stay denied.
