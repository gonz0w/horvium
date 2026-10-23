# Test Design: read-only package install and lifecycle

Status: Draft; scenarios unexecuted. Owner: Cam (`@gonz0w`). Date: 2026-10-23.
[PRD](PRD.md), [EDD](EDD.md), [#159](https://github.com/gonz0w/horvium/issues/159).
TDD means Test Design Document. No execution grant follows from this packet.

## Requirement traceability and proportional checks

| Requirement | Stable test | Scope and actual acceptance needed |
| --- | --- | --- |
| M5-PKG-001 | M5-BUILD-001 | #160: clean pinned poudriere builds on FreeBSD15.1/14.5 amd64; exact source/private distfile/module/Go/Ports-tree/jail/ABI hashes and license closure; missing/mismatched input fails before installation |
| M5-PKG-002 | M5-STAGE-001 | #160: expected/extra/missing payload, ownership/modes, sample validity after explicit identity substitution, no runtime toolchain/source/token, clean staging; no install-time service/account/credential or unrelated config changes |
| M5-PKG-003 | M5-SETUP-001 | #161: actual dedicated identity and protected ancestors; root config0640/service group; token setup persists digest only, refuses replacement, invalid UID/GID/symlink/hardlink/mode/unknown entry cases |
| M5-PKG-004 | M5-LIFE-001 | #161: original helper/root and management/non-root identities, startup order, safe socket, real login/CLI same-generation inventory, root management denied, missing digest/helper/listener failure, initial unavailable inventory distinct |
| M5-PKG-005 | M5-LIFE-002 | #161: normal and interrupted start/stop/restart, duplicate/concurrent start, changed/stale pidfile and replacement socket, trusted administrator control plus independent original fixture waits/descendant retirement, session invalidation, no unrelated signal/removal or file-service interruption |
| M5-PKG-006 | M5-LOG-001 | #161: local sink delivery/redaction, slow/blocked sink, finite shutdown, no raw data or setup token in exported records; no durable-audit claim |
| M5-PKG-007 | M5-UPGRADE-001 | #161: compatible package pair and explicit quiescence; config/digest/identity/dataset preservation; unknown format/unknown live owner refuses; deinstall removes only declared payload; rollback does not restore an old verifier |
| M5-PKG-008 | M5-ACCEPT-001 | #161: same reviewed candidate on both targets, complete original evidence and independent fixture retirement, failed/skipped limitations, required source/security review and explicit scoped acceptance; parentM5/M6 remain separate |

For #159 itself, run the existing static content/configuration/license/local-link
check and redacted secret scan, verify source-to-command/permission consistency,
and independently review this coherent design. No new test framework or product
test is added solely for prose. Full required repository/current-source review
gates remain applicable; held process-bearing checks are recorded as held, never
passed from this static selection. Do not rerun historical campaigns for a doc edit.

## Future fixtures and sequence

Use separately authorized disposable native targets with existing finite owned
qualification infrastructure, never operator pools/accounts or production data.
Bind actual OS/kernel/userland/compiler/package source and all guest/host artifacts
before execution. Run target releases sequentially; no new VM/resource/capacity
envelope is approved here. Process/guest/security/credential effects require their
exact future reviewed grant and a proven safe host preflight.

Each installed fixture owns a fresh private config anchor, selected non-login test
identity, runtime directory and token state on boot/system storage. Independently
retain before/after metadata and digests for operator-state sentinels, package
manifest and separately managed file-service sentinels. Generated plaintext tokens
stay in protected test input; never export tokens, cookies or raw inventory.

Before the first service-control fixture, review exact rc/daemon source and the
selected stock composition. Verify protected root-owned supervisor pidfiles,
expected command matching, no broad PID/name search, startup refusal for foreign
or unlocked stale entries, disabled crash restart and finite stop-wait failure.
Inject changed/missing/unlocked pidfiles, interrupted supervisor/child, unknown
socket/listener and unrelated sentinels. Record actual refusals and limitations;
the stock administrator control path does not promise an immutable capability or
eliminate every PID-reuse race. It must not claim such protection from these cases.

The independent qualification supervisor retains its own born-process/wait
ownership and original sentinels; rc status or a later PID/name snapshot cannot
substitute for that fixture custody/terminal evidence. Do not reproduce the known
#414 vulnerable cleanup path or claim it fixed by package metadata.

Normal sequence: inert install -> explicit protected setup/token -> helper start
-> non-root management -> HTML and CLI inventory -> management/helper stop ->
restart/re-authentication -> compatible quiesced upgrade/rollback -> explicit stop
-> deinstall. Verify state and unrelated file-client continuity at each boundary.
Old sessions must fail after restart; the operator's same retained token still
works unless explicit rotation occurred. An unknown/stale socket refusal is an
expected denial, not an instruction to unlink it automatically.

Fault cases remain in this same lifecycle slice: partial payload install, occupied
listener, unreadable/invalid config, missing/unsafe digest, helper exits during
management startup, interruption between component starts/stops, concurrent start,
full state storage and log sink blockage. Preserve the first failure and actual
retirement uncertainty. No success from a supplied boolean, new census of lost
historical IDs, automatic retry, package fallback or swallowed stop error.

## Upgrade, rollback and custody

Require two real signed packages with an explicitly supported config/ABI pair;
one newly invented predecessor cannot establish upgrade acceptance. Compare
active config/verifier bytes and ownership, independent dataset/file-service
sentinels and service identity before/after. Unsupported downgrade and unsafe
state fail without repair. Removal preserves active config even if it equals the
sample; modified and unmodified cases both matter. Retained accounts/verifiers
need a documented operator-owned later disposal policy, not a package purge hook.

## Evidence and current result

Future receipts identify exact source/base, tag or lab version, original archive
and package SHA256, module/SDK/Ports/jail/OS identity, installed plist/modes, ordered
phase exits, original owned process outcomes, independent terminal observations,
raw private evidence and sanitized acceptance summary. Native acceptance needs
all named cases on both targets and applicable browsers/clients; M1 manual/browser
and broader NAS gates are not closed by package testing. Keep the existing runtime
resource/security budgets unchanged; native envelope approval is separate.

Current result: design prepared only. No package build, source distfile, stage,
installed service, new credential, native run, VM or original fixture retirement
was produced. #160/#161 remain open and not pulled. Historical evidence, UNKNOWN
states, owner_clean:false and consumed attempts retain their original meanings.
