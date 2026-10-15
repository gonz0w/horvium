# Original ancillary-channel rejection

Status: inactive native witness fixture; full native 005 and resource acceptance open.
Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-007 / DASH-WIT-NATIVE-002 / DASH-WIT-NATIVE-005.

A valid first SCM_RIGHTS handoff alone cannot establish that the original private
channel contains exactly one descriptor message. After acknowledging and waiting
the original credential-dropped connector, the observer now requires actual channel
EOF before sending the producer request. Queued bytes, replayed/extra descriptors,
truncation, timeout or any read uncertainty deny qualification permanently. All
received extra descriptors are closed; the first independently validated endpoint
retains its original ownership. No producer response is accepted after ambiguity.

Portable failing behavioral regressions send actual queued rights/bytes and an
unclosed channel, then verify denial while the first endpoint still transfers its
original bytes. Existing sender-closure, write-half EOF, CLOEXEC, GC/finalizer,
extra/truncated/wrong-tag/non-socket controls remain part of native execution.

## Native matrix and ownership

The same actual connector artifact receives only a fixed private fault whitelist.
Normal bootstrap omits it. The fault is injected after original UID/PID/artifact
binding, under the configured management credentials; no substitute connector,
root-connect fallback, product API/config or shell interpolation is added.

| Case | Required original observation |
| --- | --- |
| EOF / missing rights | No descriptor can be accepted. |
| Wrong tag / extra rights / truncated rights | Initial handoff is rejected and all received descriptors close. |
| Non-socket / wrong Unix endpoint | Descriptor type or original helper peer identity is rejected. |
| Replay before ACK / replay after ACK / extra byte | First endpoint is valid; post-wait channel EOF check rejects the queued second message. |
| Uninjected control | All connector channels end exactly once; actual complete/delivered original confirms publication. |

Every rejected case requires an original failure stage, all-started connector
usage/reaping, empty producer custody, original observer exit and front removal.
The actual helper/socket, both configuration files and token digest remain unchanged.
Original facts are retained before semantic assertions so failed cases are not
silently dropped. Public evidence omits private actor IDs and original payloads.

A separate descriptor-only scope bounds its own soft file-descriptor limit to
128 and compares every descriptor/flag before and after each complete control group.
It restores the exact original limits before starting product processes. This is
an exhaustive census of newly receivable descriptors in that bounded scope, not a
new product capacity, full process inventory or a replacement for later descendant,
socket, tmpfs, ZFS/md and host cleanup. Owned disposable datasets only.

The new EOF read uses the existing overall 45-second collection deadline after
the original connector has been waited. Bootstrap/handoff deadlines are unchanged.
No caller retry, workload pacing, cap increase or product budget adjustment follows.
Both pinned FreeBSD releases need signed clean review, original artifact/preflight
binding, independent archive readback and verified guest/host teardown. The bounded
outer diagnostic allowance covers eleven broker cases; it changes no product deadline.

## Remaining acceptance

Canonical resource activation and runtime writes remain disabled. Actual controller
interruption, complete cap/physical-exhaustion controls, historical 14.5 sampling
gaps, cadence/overhead/augmented budgets, 300-second idle, full original resource/
latency, manual browser/accessibility, hardware and release gates remain open.
No new dependency, service, paid automation or Actions; use required check/lint/
test/secrets, affected-package review, FreeBSD test build and lab-stop/status.
