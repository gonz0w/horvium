# Original observer interruption controls

Status: inactive observer-interruption implementation submitted for review; full native
005 and resource acceptance remain open.
Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
[Accepted ADR](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../../design/dashboards/native-witness-binding-TDD.md):
DASH-WIT-003 / DASH-WIT-004 / DASH-WIT-NATIVE-005.

PR 390 supplied independently owned sampling and original coverage. It did not
accept the complete interruption matrix, cadence or resource procedure. This
batch exercises **the actual observer** at partial capture, sealed-before-forward,
partial forward and original-metadata readback, each with SIGTERM, SIGKILL and loss
of the private fault-control channel. Controller/connector death and the complete
ancillary/cap matrix remain separate; this is not full native 005 acceptance.

## Boundaries and observable assertions

Fault barriers exist only when an explicitly configured private fixture case
provides an inherited root-owned channel. Normal and timed runs have no channel,
barrier or delay. The actual protected observer artifact and kernel lifetime,
parent credentials, source request ID and case stage identify every boundary.
The capture reader and delivery writer retain actual first bytes rather than
synthesizing a source. Portable failing behavioral tests cover interrupted and
released byte paths before implementation.

| Boundary | Actual observation |
| --- | --- |
| Capture | Original producer prefix written; remaining capture blocked before the next read. |
| Sealed | Complete original captured/sealed; delivery has not begun. |
| Forward | Actual management endpoint has received a prefix; full bytes/EOF remain unconfirmed. |
| Metadata | Actual management publication exists; original observer readback is interrupted before confirmation. |

Each native case must refuse original lookup after uncertainty, retain all
original files and hashes including incomplete/unindexed prefixes, wait the
actual observer with positive original RSS/CPU usage, and prove its original
lifetime ended. A killed observer's surviving front socket is removed only after
checking its original inode. The actual helper socket/lifetime, both configuration
files and token digest stay unchanged. SIGTERM/channel-loss retain permanent
failure; SIGKILL cannot supply a graceful receipt and cannot acquire one later.

The matrix uses 20 disposable owned ZFS dataset identities, separate namespaces
and tmpfs per case. Parent ownership retains original namespace/mount identity;
all processes, descriptors, sockets, owned ZFS/md storage and host artifacts need
verified teardown. Never test on operator data. Private originals and actor/host
inventory are not committed; public receipts contain fixed case names and hashes.

An ordinary uninjected broker control also confirms an actual complete/delivered
original and all-started connector accounting without an inherited fault channel.
It verifies the normal path alongside the twelve adverse cases.

## Remaining acceptance

This fixture adds no product API/configuration, daemon, runtime dependency or
service. Canonical resource qualification remains inactive, runtime writes denied.
The earlier 14.5 sampling gaps, observer/connector overhead, 300-second idle,
full resource/latency, controller/connector interruption, browser, hardware and
release gates remain open. Run both pinned releases at one signed reviewed source;
keep failed attempts and source/base/title boundaries. Follow fresh check/lint/
test/secrets, affected-package review, and lab-stop/status; no paid Actions.

## Source-bound native observations

Behavioral source `6cc6e29c240202b56c69df0a1df7949df27a7d5d`, base
`420deb3cbc992704d0f0eed11e46b7b363d29b8d`, actual title
`test(dashboards): batch actual observer interruption controls`, passed signed
clean review, fresh check/lint/test/secrets, executed affected cmd race/original-wire
fuzz and a FreeBSD test cross-build before native execution. The
[immutable manifest](witness-observer-interruption-manifest.json) records the
ordinary broker and twelve adverse cases on both pinned disposable releases.

- FreeBSD 14.5: 378.12s, 12 cases and 15 independently reopened adverse originals; ordinary delivery/confirmation passed.

- FreeBSD 15.1: 506.28s, 12 cases and 15 independently reopened adverse originals; ordinary delivery/confirmation passed.

Independent safe archive/frame/object/hash verification binds observer/controller
artifact roles, actual-header probe, source/base/title and original custody/usage.
Capture SIGKILL preserves its actual 80-byte unsealed prefix; all signal cases
bind a reaped original observer and denied lookup. Signal cases do not retain a
graceful final observer receipt; only fault-channel EOF cases carry explicit
permanently failed receipts. The initial private validator's field-case/receipt
assumption was rejected before these distinctions were checked; no absent signal
receipt is relabeled a graceful acknowledgement.

Both original preflight observations match, pinned runtime/base bytes are
unchanged, and original descendants/descriptors/sockets/tmpfs/ZFS/md/host runtime
artifacts were cleaned. All 161 historical manifests and the seven primary
operator files remain untouched. Public evidence contains no PID/UID inventories,
private original payloads, credentials or operator data. Final documentation
review and lab-stop/status results accompany the PR; this is not review-full,
complete native 005 or release acceptance.
