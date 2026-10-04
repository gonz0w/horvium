# M1 assembled HTTP abuse and recovery qualification

Status: native qualification passed; under maintainer review. Owner: Cam (`@gonz0w`).
Date: 2026-10-04. Requirements M1-REQ-003,004,006,007; tests M1-HTTP-001,
M1-AUTH-001, M1-SCALE-001, M1-NOCHANGE-001 and M1-RESTART-001.
Complements [administration](security.md), [helper denial](helper-events.md)
and [runtime](runtime.md) evidence; this is selected adversarial coverage,
not an independent penetration test or complete security certification.

## Bound and regression

Response admission already bounded queries/rendering, but accepted TCP clients
could accumulate before a handler. A failing loopback regression reproduced
excess accepted slow connections. The foreground runtime now wraps its listener
with `max_readers + 8` accepted connection slots, covering incomplete headers,
active responses and idle keep-alive connections. Default capacity is 24; trusted
configuration yields 9–72. Eight additional slots accompany the existing session
bound but are not reserved by identity. No HTTP client can change this bound.

The accept loop waits for a slot before allocating another net/http connection
handler. Excess connections stay in the kernel's finite backlog; this is not an
HTTP 503 promise. Existing five-second header, ten-second read/write and
30-second idle timeouts release capacity. A saturated loopback connection flood
can delay legitimate administration. The cap bounds application work and leaves
independent file daemons untouched; it does not guarantee availability under
arbitrary hostile kernel/network load. Listener close wakes a capacity waiter,
and idempotent connection close releases each slot once. Portable regression
and race tests cover overflow, release/recovery and saturated shutdown.

## Assembled native procedure

`TestNativeBuiltHTTPAbuseAndRecovery` provisions only a disposable md pool and
root-owned boot config/digest fixtures. It runs the actual root helper and built
management binary under the configured non-root UID. Requests exercise the real
TCP parser, authentication, query/HTML router, logger and native source.

Denial cases include missing/duplicate session cookies, fragment bypass, Host
rebinding, absolute request targets, Origin/Fetch Metadata conflicts, GET
mutation/body, unknown command paths, missing CSRF, wrong media type, conflicting
lengths, malformed chunking, oversized body/header and duplicate query fields.
Challenge floods must throttle without invalidating the admitted session; concurrent
refresh floods must return admitted/deduplicated outcomes and recover.

Thirty-two incomplete clients accompany an independent guest-only descriptor
count. It must remain within the configured connection bound plus two descriptors
for concurrent collector activity. Closing clients must restore authenticated
queries. Separate stalled-header and stalled-body requests must expire under the
actual transport deadlines. Fixture input, fstat output and inventory stay inside
the guest; only statuses and assertions leave it. Helper/management event records
must use the existing eight-field schema and disclose no fixture token, cookie,
CSRF, hostile marker or owned storage identifiers. Config/digest remain unchanged,
owned dataset properties/teardown pass, and both processes shut down cleanly.

## Execution record

The connection regression failed before implementation. Portable focused race
checks pass; FreeBSD cross-compilation is build verification only. Actual native
run IDs, exact source/versions and results will be recorded before this additional
gate is qualified. Required browsers and independent SMB/NFS continuity remain
separate gates. No operator host service or live data is used.

## Native execution

Functional source `20edb928cdb0eaef9474ded38103782c4d090441` passed all five
executed jobs in [run 37230688573](https://github.com/gonz0w/horvium/actions/runs/37230688573).
The optional performance job was intentionally skipped; this run is functional
security/recovery evidence, not resource qualification. Both actual built-runtime
suites passed all seventeen denial cases, challenge/refresh floods, descriptor
bound, independent slow-header/body expiry, recovery, eight-field redacted events
and unchanged boot config/digest. Duration: 16.32 seconds on 15.1 and 16.27 on 14.5.
Existing native reader/helper/cache/admin/CLI/runtime/SFTP suites also passed.

Observed guests: 15.1-RELEASE-p3 and 14.5-RELEASE, installed/running kernel and
userland amd64; bundled ZFS 2.4.2-1-FreeBSD_g6330a45b0 and
2.2.9-FreeBSD_g079ba86d7 respectively. Both used Go 1.27.1, x/sys v0.48.0 and
local HTMX 2.0.11. Ubuntu 24.04 hosted the pinned VM Action
`a2f9a41fa97f6848b8c3b791087dfcdaa5b473ff`, builder 2.2.8, 6144 MiB with default
CPU/image settings. No immutable performance or physical-hardware claim follows.

Two earlier native runs failed test assumptions: immediate overview before the
first collection, and expecting 405 instead of the existing router's 404 for GET
/api/refresh. The final suite checks that all denial cases preserve generation.
No allowance was added to authentication or resource budgets. Source file hashes
and traceability are in the [manifest](../../research/M1/source-manifest.json).
Local `make check lint test secrets build go-check` and subsequent focused
cross-compilation/checks passed, including race and five bounded fuzz targets.
Required browser/manual accessibility and SMB/NFS gates remain separate.
