# Generation-bound Overview pool comparisons

Date: 2026-10-12. Owner: [issue 354](https://github.com/gonz0w/horvium/issues/354),
under [Overview delivery 289](https://github.com/gonz0w/horvium/issues/289)
and [M2 gate 131](https://github.com/gonz0w/horvium/issues/131).
Authority: adopted [PRD](../../design/dashboards/PRD.md), [EDD](../../design/dashboards/EDD.md),
[Test Design](../../design/dashboards/TDD.md) and [ADR 0010](../../adr/0010-progressive-dashboard-charts.md),
maintainer merge of PR 349. Reuses the pool comparison foundation merged in PR 351;
the independent dataset slice merged in PR 353. This is the portable stage-A
Overview subset of DASH-REQ-001–007, not broad dashboard or release qualification.

## Composition and identity contract

Authenticated Overview resolves its detached metadata once, then requests exactly
one bounded pool page using that returned generation, name order, no filter,
offset zero and the existing default limit of 100. It compares the first at most
20 rows alongside an exact table for those same rows. It does not drain pages,
rank all resources or find every degraded pool. Coverage names the generation,
number compared, observed pool count and whether more pools are available beyond
the comparisons. All-resource list links remain in the storage-section table.

Comparison and exact-table links lead to the existing Pools page with the original
GUID, generation and default query, on a page containing that row. No pool-to-dataset
membership is inferred. Current-generation reset remains explicit. Full pages and
HTMX fragments use the same presentation; existing status polling only updates
collector status and cannot silently replace the displayed generation.

Eviction between the metadata and pool reads returns the original generation-expired
error, discards both projections and uses the existing reload explanation. No retry
with a newer generation, stale combination or partial successful Overview. Failed
initial resolution does not query pools. Cancellation and cached reader admission
retain their original safe errors. The private composition helper accepts the two
existing typed read contracts; no public API/CLI/config/schema/interface is added.

## Accounting, freshness and fallback

Allocated/free retain separate zero-origin ratios against each pool's reported
size, never additive segments. The existing overflow-safe integer rounding helper
produces approximate 0.1% labels; exact uint64 bytes and GUIDs remain in the table.
Zero size retains an explicit unavailable ratio. Empty successful pool observations
show zero observed pools and no invented comparisons. Unavailable/partial status
suppresses the comparison section; the existing observation table retains unknown
counts and section freshness. The inventory validator rejects partial pool sections.
Stale successful pools retain original labelled read/last-success times even when
host metadata is fresh. Reported pool state implies neither disk health nor backup
proof. No history, telemetry, utilization guarantee or commercial scale claim.

SVG stays decorative and non-focusable. Semantic HTML links and the exact table
remain primary without JavaScript, using existing project-owned light/dark CSS.
Names are escaped; no observed trusted HTML, new script, inline style, dependency,
timer, native collector or privileged execution is introduced. Internal JSON/CLI,
authentication, authorization, CSRF, strict CSP and HTMX policy remain unchanged.
Overview now performs the one additional bounded cached query authorized by the
EDD; old resource measurements cannot qualify this candidate's extra work.

## Behavioral verification and remaining gates

Failing-first HTTP tests detected absent comparisons before implementation. Tests
exercise 125 generated pools with a 20-row comparison/exact-table cap, original
100-row query and same-generation GUID drill-down, hostile names, 2^53/uint64 exact
values, non-additive accounting, full/fragment parity, empty/unavailable/stale
sections and anonymous/malformed/expired query denial. A private test reader delegates
to the real observation manager and deterministically replaces its cache between
reads: zero/one replacement retains the original generation; two replacements
require expiry with both projections discarded. Initial expiry prevents the second
read, and cancellation between reads also discards the projections. Tests assert
exactly one Overview and one fixed same-generation pool query, with no retry.
Shared pool/dataset regressions preserve accounting and query behavior.

Source-bound results belong in the candidate receipt; synthetic fixture tests are
not FreeBSD execution. Real Safari/Firefox/manual assistive technology, contrast/
zoom/narrow, browser lifecycle/resource and fresh assembled native service/resource
measurements remain open under #289/#131. M1 manual accessibility remains pending
by maintainer choice. uPlot research/history/native sources are separate scopes.
Historical failed/skipped evidence and the unexplained startup deadline remain;
full native lifecycle #305/#139 is open and runtime writes stay disabled.

Rollback restores the prior Overview composition and embedded presentation;
no storage, configuration, journal or schema migration. Development previews use
labelled synthetic inventory with supervised process/tab teardown, never operator
pools, datasets, backup history or host service activation.

## Recorded portable candidate

[Receipt](overview-capacity-manifest.json): signed implementation `bcb50419eb31838e5e36a01cc3c8dc34369d99fc`,
merged base `84913b3ea10ed0b4a9834868f017d73b1deeea2a`, actual title
`feat(console): compare generation-bound Overview pools`. Required check/lint/test/secrets passed,
including 258 tooling tests and ordinary Go tests. Fresh focused race checks passed
for console, service and command consumers; these packages have no portable fuzz
targets. Four Overview behavioral parents passed, including five intervening-read
subcases. Pool/dataset regressions also passed. No full milestone suite selected.
All 136 historical manifests and collector/cache/native/privilege/storage/tooling/
module/workflow inputs preserve their original bytes.

Console/command tests cross-compiled FreeBSD/amd64/v1 with pinned Go 1.27.1 and
cgo disabled; checksummed artifacts were removed without execution. Limited
labelled in-app synthetic light inspection at 1280 pixels showed exact pool values,
33.3%/66.7% ratios and 333/667 coordinates; client/scroll widths were both 1280.
Keyboard Enter reached the original GUID/generation on Pools and retained comparison
focus, with no warning/error console entries observed. Browser engine/version was
not captured; this is not Safari/Firefox/manual accessibility or resource acceptance.
Screenshot is private/checksummed. Owned tab/preview closed/stopped, build artifacts
removed and 172 owned records stopped with verified teardown. Final receipt-only
review reruns required baseline checks; deep reuse requires unchanged behavioral
inputs from the original implementation source.
