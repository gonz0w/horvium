# Test Design: native discovery and read-only console

Status: Approved. Milestone: Validating; manual acceptance pending.
Owner: Cam (`@gonz0w`). Date: 2026-10-03.
Approval: maintainer merge of [PR 3](https://github.com/gonz0w/horvium/pull/3),
source `527a330f4b1069fa0b7a1414f4d3c6dbab54f5a5`, on 2026-10-03.
[PRD](PRD.md), [EDD](EDD.md).
TDD here means Test Design Document; implementation uses test-first development.
The research spike does not satisfy these product gates. Completed portions and
remaining cases are tracked in [implementation status](implementation.md).

## Traceability and gates

| Requirement | Test ID / behavior | Level / environment |
|---|---|---|
| M1-REQ-001 | M1-NATIVE-001: compare real helper/CLI/UI discovery with independently captured fixed native projections on both releases | T3 / E2 |
| M1-REQ-001 | M1-PARSE-001: empty, valid, large, malformed, truncated, overflow, control-byte, unknown-enum and hostile XML fixtures | T1 / E1, T3 / E2 |
| M1-REQ-002 | M1-FAIL-001: missing executable/OID, permission denied, timeout, output cap, helper disconnect, partial section and stale previous generation | T2-T3 / E1-E2 |
| M1-REQ-002 | M1-IDENTITY-001: duplicate/missing serials; provider vs disk; GUID retained on fixture rename; graph IDs never persisted as identity | T1,T3 / E1-E2 |
| M1-REQ-003 | M1-PRIV-001: native Go peer wrapper allows configured UID, denies another UID/forged fields, checks helper UID/version, rejects unsafe socket permissions | T1-T3 / E1-E2 |
| M1-REQ-003,007 | M1-NOCHANGE-001: before/after fixture config/properties/services show no discovery side effects; mutation/general-command requests denied | T2-T3 / E1-E2 |
| M1-REQ-004 | M1-AUTH-001: token generation/rotation/digest/file modes, invalid/expired session, throttling, logout/restart, no credentials in logs/argv/URLs | T1-T3 / E1-E2 |
| M1-REQ-004 | M1-HTTP-001: cross-origin/missing CSRF, login CSRF, DNS rebinding Hosts, forwarded-header spoof, GET mutation, body/header limits, fragment bypass, CSP | T2 / E1; browser / E2 |
| M1-REQ-005 | M1-CONTRACT-001: CLI/full page/fragment reflect identical generation, pagination and honest errors; expired generation cannot mix rows | T2 / E1 |
| M1-REQ-005 | M1-UI-001: keyboard/focus/labels/status/reduced-motion/light-dark/responsive; JS disabled and HTMX enabled; malicious resource names escaped | T2 + manual browser / E1-E2 |
| M1-REQ-006 | M1-SCALE-001: one collector under concurrent refresh/read; cancellation/deadlines; resource caps before allocation; incomplete result explicit | T1-T2 / E1 |
| M1-REQ-006 | M1-PERF-001: measure RSS/CPU/latency on declared small and 10,000-dataset fixtures, plus 100,000-row cap/limit-exhaustion scenario | T8 / E1-E2 |
| M1-REQ-007 | M1-RESTART-001: kill helper/management mid-collection, recover fresh state, invalidate sessions, preserve config and independent file daemons | T3,T6 / E2 |
| M1-REQ-008 | M1-BUILD-001: pinned native toolchain/module/assets, license notices, clean builds with no runtime Python/Node/cgo, accurate version/unsupported-host result | T0,T3 / E0-E2 |

All listed product tests are M1 acceptance gates. Foundation checks also remain
required. Security-denial and native failures block acceptance; skipped capabilities
remain unqualified, never silently green. An approved justified scope change must
update PRD/EDD/Test Design together.

## Fixtures and procedure

E1 parsers use small redacted fixtures derived from native format observations,
not committed customer inventory. Generated fixtures cover larger counts and
hostile strings. Assertions focus on typed values, provenance, bounds and visible
failure rather than reflecting implementation internals. Fuzz parsers/framing
with corpus/time/memory limits; record seed/reproducer for any failure.

E2 uses disposable QEMU FreeBSD 15.1 and 14.5 amd64 guests. The current research
[fixture/procedure](../../research/M1/native-spike.md) is a starting point; product
tests must call the actual Go adapter/helper. Use only fresh file-backed md devices
and test-owned pools; keep mountpoints disabled and resource names generated.
No force pool creation, live disk discovery-to-mutation chain or external host
device input. Denial tests use a separate non-login UID, actually dropping identity
before connect. Teardown verifies pool disappearance and md detachment and reports
any leak as failure; always discard the guest.

For independent-service testing, explicitly provision disposable file daemons and
a local fixture file, then confirm they remain running/reachable through management
restart. This verifies separation, not SMB/NFS/SFTP client compatibility. Complete
client/ACL/locking/restore gates are M3/M4/M6. No user data is transmitted.

Browsers: Safari on macOS and Firefox on Linux first; record actual OS/browser
versions, viewport, JS mode and accessibility procedure. BSD CLI tests run on the
native guests. Manual screen-reader/keyboard findings are recorded with defects;
an automated accessibility scan alone cannot claim WCAG conformance.

## Failure and restart

Inject command stalls, oversized native output, unknown protocol version, broken
stream, concurrent manual refresh, external dataset rename, disappearing provider,
invalid config and permission changes. Verify bounded work, deterministic error
classes, no zero-as-success, no publication of truncated complete state, no stale
section accidentally marked fresh and no arbitrary native command execution.

Kill each process during collection. Verify service configuration/token digest
survive and sessions/cache do not. Test a valid boot/system state directory with
the data pool unavailable. M1 has no on-disk database migration or storage restore;
assert absence of writes rather than inventing upgrade/rollback coverage. Package
upgrade/removal and actual file restoration remain later release gates.

## Performance and result record

Measure after 60-second warm-up; idle CPU over five minutes; collect process RSS
including transient peak while replacing a generation. Pin QEMU CPU/memory/image,
host runner, source/suite revision and synthetic seed. Small workload: four
providers, one pool, 100 datasets; large workload: 10,000 datasets and ten clients,
1,000 cached page requests. Report p50/p95/p99 and maximum RSS, not averages alone.
Separately test 100,000-row and byte-cap exhaustion. Compare the [PRD budgets](PRD.md)
with actual values and explain VM limitations. Synthetic rows do not prove native
disk throughput or commercial scale.

Each record includes exact OS kernel/userland, OpenZFS, Go/modules/assets,
browser/client versions, command/procedure, status (pass/fail/skip/unavailable),
sanitized logs, deviations and linked requirement/test IDs. Acceptance requires an approved revision and actual
evidence, followed by updating the compatibility matrix.
