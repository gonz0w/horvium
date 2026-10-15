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
