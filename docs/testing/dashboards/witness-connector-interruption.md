# Original connector interruption controls

Status: inactive test fixture; native 005 and resource acceptance remain open.
Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-008 / DASH-WIT-NATIVE-002 / DASH-WIT-NATIVE-005.

The actual credential-dropped connector exits before producer capture, seal,
forward and metadata confirmation. There is no live connector to stop at those
later observer boundaries. These controls cover its real lifetime: after original
kernel/artifact/credential confirmation but before helper connection, and after
SCM_RIGHTS handoff plus independent helper-peer validation but before acknowledgement.

Each boundary exercises actual SIGTERM, SIGKILL, original connector-channel EOF
and successful release. Private inherited fault control reports the original
source request ID and independently observed connector actor. Only that bound
lifetime is signalled. Signals, EOF's ordinary failure exit and successful release
must match distinct original wait statuses. EOF closes the actual connector IPC;
it does not cancel the process context before waiting. All started connectors
must be reaped and accounted with positive original RSS/CPU. Missing usage denies
qualification; no guard is relaxed to accommodate short process lifetimes.

Adverse cases must permanently refuse original lookup and create no producer
original because interruption precedes the first producer request. Successful
releases must confirm actual complete/delivered originals. An uninjected broker
control has no inherited fault channel or boundary record and confirms ordinary
publication. Reopen retained private original frames, probe/artifact/context,
wait receipts and archive census independently before publishing sanitized results.
Fault barriers exist only in explicitly configured owned disposable tests; normal
and timed runs have no barrier, delay or additional channel.

Every case preserves the actual helper process/socket, both configuration files
and token digest. Original observer/connector lifetimes end, front sockets are
removed, and the owner verifies all original descendants, sockets, tmpfs, owned
ZFS/md and host artifacts. Twenty disposable dataset identities are used; no
operator data. Private actor IDs, original payloads and inventory stay uncommitted.

Run both pinned FreeBSD releases at the same signed reviewed source; retain failed
attempts with their original boundaries. Run required check/lint/test/secrets,
affected-package review, FreeBSD test cross-build and lab-stop/status. This batch
adds no product API/config, dependency, service, paid automation or Actions.
Canonical resource activation and runtime writes remain disabled. Controller
interruption, remaining ancillary/cap controls, historical 14.5 cadence gaps,
observer/connector overhead, augmented budgets, 300-second idle, original large
latency failure, browser, hardware and release gates remain open.

## Retained failed predecessor and receipt correction

Source `1d920fdac20c75c8e24685abd01cec44ca00f426` passed signed portable review
before a failed 15.1 native attempt (592.23 seconds). The uninjected, two release
and two EOF controls passed; four signal cases timed out at control readback.
Original archive/transcript, failed exit and verified guest/host cleanup remain
retained. No 14.5 execution or signal wait acceptance follows from this candidate.

The original receipt put Go's non-exit `-1` sentinel into a control frame whose
unchanged grammar requires unsigned numbers. A failing portable codec regression
reproduces that rejection. The correction represents availability and actual exit
separately, preserves the raw kernel wait status/signal, and retains an exclusive
original wait receipt before control readback. It does not permit negative source
numbers or treat a missing wait as success. The failed predecessor did not retain
those signal wait receipts, so it cannot prove each original signal outcome;
a corrected candidate requires fresh review and both-release execution.

The next source `1443bae6a75c94de377e64396f3124fb350979e7` also remains failed
on 15.1 (483.46 seconds): six boundary controls and the uninjected control passed,
but both post-handoff signal cases returned ordinary failure exit 1 / raw status
256. Original wait receipts now prove those mismatches; they are not signal
passes. Accounting and context remained valid. The earlier negative-number
representation is corrected, but this run does not accept the full matrix.

The controller's boundary-time full executable rehash added work while the child
was already waiting under its original ten-second deadline. The correction checks
kernel PID/UID/start lifetime at the boundary, using the actual-header ABI. Full
artifact/path verification remains in the connector and observer before that
boundary, with protected candidate artifact preflight; no deadline or artifact
check there changes. Original final context is now exported before outcome
assertions as well. The failure receipts identify exit 1, not the child's exact
read error; they do not retrospectively prove a timeout cause. Fresh execution
must establish the corrected controls. No artificial delay or retry is added.
