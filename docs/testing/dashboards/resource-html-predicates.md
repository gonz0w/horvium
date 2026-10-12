# Original-inventory dashboard resource HTML predicates

Date: 2026-10-12. Owner: [issue 360](https://github.com/gonz0w/horvium/issues/360),
under [M2 gate 131](https://github.com/gonz0w/horvium/issues/131).
Authority: [adopted resource Test Design](../../design/dashboards/resource-qualification.md),
[PRD](../../design/dashboards/PRD.md), [EDD](../../design/dashboards/EDD.md) and
[Test Design](../../design/dashboards/TDD.md).
Requirements: DASH-REQ-006 / DASH-PERF-001 with DASH-STORY/DATA/DRILL/SAFETY-001.
Foundation: portable inactive [request driver](resource-request-driver.md),
adopted by maintainer merge of PR 359, source `0efe0aa1cedec644bca1fb2716f5f9498c924387`.

## Prepared expectations and validation

The [test-only predicates](../../../cmd/horvium/dashboard_html_helpers_test.go)
prepare a validated detached copy of declared typed inventory and the exact
1,000-request plan before requests start. Expectations are cached by request path:
original name/GUID order, full page rows and exact byte cells, first 20 comparisons,
independent pool ratios and dataset maximum across every row of that page.
Arbitrary-precision integer arithmetic supplies the independent ratio reference;
no browser floats or product comparison helper generates expected values.
Nothing is inferred from a name prefix or copied out of the response to become
expected inventory. Changed requests or malformed inputs fail before validation.

The returned callback fits the reviewed driver. It makes no API or native request,
has no mutable cross-request state, and runs after request/read/close timing stops.
The controller's allocations/time remain outside the paired measured PIDs, while
validation affects workload pacing. Inventory values must remain constant during
the declared fixture replacements, as the Test Design requires. Future native
wiring must supply original observations, not assert that generated rows represent
native sources or silently rebaseline changing values from each response.

Every successful response requires a complete full document and the correct view,
unique workspace, generation, exact table and bounded comparisons. Table counts,
column roles, every original name/GUID/uint64 cell and original-GUID links must
match; a correct first row cannot hide a truncated 100/500-row page. Links retain
the generation and original default/maximum query, including HTMX targets.
Filter/sort/order/page-size controls and pagination agree with that same scope.
Unexpected tables, selected-row/detail state, scripts or extra comparisons fail.

Comparison validation checks coverage, row count, original identity, approximate
percentages, independent zero-origin rectangles, fixed tracks/viewports and exact
local scale including table rows beyond the first twenty. Zero pool size or all-zero
Used uses the explicit unavailable-ratio/scale form, without fake SVG bars. An
observed empty section uses its empty table/comparison form. An unavailable section
suppresses comparisons and retains unknown counts; it cannot claim observed zero.
Overview additionally checks original host facts and storage-section counts/status/
links plus its first-by-name exact pool table and pool-specific comparison freshness.

Generation and timestamps are response-specific. Generation syntax and agreement
across the header, coverage, forms, resource/section/pagination links are checked.
Read times must be valid and agree with the section header or Overview pool-row
last-success time; successful section read and last-success times agree. Both
Observed and Stale labels are accepted for `ok`, with exact non-OK status labels.
This proves consistent presentation, not independent native clock accuracy,
collector freshness or availability. Existing collector/native readiness gates
still need execution; a constant original snapshot is not future measurement evidence.

Only an Overview 409 may satisfy the safe expiry predicate: exact generation-expired
code/explanation and current-Overview reload link, with no successful generation,
host facts, table, list or SVG projection. The driver separately prohibits expiry
in its initial phase and requires successful completions in every replacement case.
Login/error/wrong-status pages never become successful workload samples.

## Bounded parser and actual regression fixtures

The controller uses Go's standard-library XML raw-token decoder to inspect the
current rendered HTML grammar, including boolean/void HTML elements and escaped
text. It explicitly checks the original closing-tag stack instead of accepting
automatic markup repair. It is not a browser DOM, accessibility evaluator or new
runtime HTML parser. Bounds are 32 MiB complete bodies, 20,000 elements, depth 64
and 8,192 decoded bytes per attribute. Duplicate attributes/IDs, prefixed elements,
processing instructions, extra roots, mismatched closes and truncation fail.
Exact parser limits have accepted-boundary and rejected-limit+1 regressions.
These controller bounds do not change any product or inventory budget.

[Behavioral tests](../../../cmd/horvium/dashboard_html_test.go) use the actual
console, authentication and observation manager with generated disposable input.
They cover 610 pools and 610 datasets, complete default/maximum pages, first twenty
comparisons, page rotation, hostile escaped names, values above 2^53 and uint64
maximum, non-additive pool accounting and a dataset maximum outside comparisons.
They also exercise empty/zero/unavailable host/storage/stale observations and a
real expired-generation response after two manager replacements.

Semantic mutations change counts, names, GUIDs, integer cells, generations,
links/HTMX links, query controls, pagination, host/section facts, freshness times,
scale, coordinates, tracks/viewports, selection, table/comparison bounds and status.
Missing/duplicate/truncated tables or comparison rows and unsafe expiry projections
are rejected. Mutation preconditions fail if a change has no effect.

A 1,000-request, ten-reader in-memory HTTP transport runs the reviewed driver
against the real authenticated handler and predicates: exactly 200 successes per
case and no hidden API calls. It supplies the same incoming-request form as the
server; it does not bypass authentication/origin policy. These are portable
integration assertions, not network, process-pair, service-latency or FreeBSD
measurements. Temporary manager/auth sessions are cancelled/closed by test cleanup.
No helper, VM, browser, host listener/service or operator storage is used.

Use `make test-focus PKG=./cmd/horvium TEST='^TestDashboardHTML'` during development.
Signed clean review runs required check/lint/test/secrets and affected-package
race/fuzz checks. FreeBSD test cross-compilation is build evidence only. The
source-bound [receipt](resource-html-predicates-manifest.json) records actual review execution.

## Remaining gates and rollback

The native performance loop and historical validator are unchanged and inactive
with respect to this new workload. Next review strict candidate/transcript/release
coverage receipts and wire the complete controller into all four native modes,
preserving original crash/cap recovery, warm-up/idle, replacements, RSS/CPU and
teardown requirements. Only then execute a clean signed assembled candidate on
both pinned releases. Historical receipts retain their original failed/skipped
limits and dataset-only workload; none gains dashboard coverage from this slice.

Overview/capacity #289/#290/#131, native lifecycle #305/#139, browser/manual
accessibility, independent file-daemon continuity, hardware/commercial and release
acceptance remain open. Runtime writes remain disabled. Rollback removes the two
test files and their documentation; runtime, native fixtures and old validators
are unchanged. No dependency, public route/flag/config/schema, Action or service
is introduced.

## Recorded portable candidate

[Receipt](resource-html-predicates-manifest.json): signed implementation `dcd6df529a10d4e3c3fa4c271947a170ccd082de`,
merged base `0efe0aa1cedec644bca1fb2716f5f9498c924387`, actual title
`test(dashboards): validate resource workload HTML`. Required check/lint/test/secrets
passed, including 258 tooling tests and ordinary Go tests. Seven HTML behavioral parents
passed, followed by fresh affected command-package race checking. No portable
fuzz targets exist in that package. The conservative plan marks native review
required; this inactive portable slice does not satisfy native resource acceptance.

FreeBSD/amd64/v1 test compilation passed with pinned Go 1.27.1 and cgo disabled;
the checksummed artifact was removed without execution. All 138 historical
manifests and runtime/native fixture/driver/legacy validator/tooling/module/workflow
inputs preserve their original bytes. 194 owned records were stopped with verified
teardown. The new HTML fixtures use in-memory HTTP; no VM/browser/helper/management
pair or host service was started. Final receipt-only review reruns the baseline;
deep reuse must match these original implementation inputs and supervisor evidence.
