# Test Design: progressive capacity dashboards

Status: Approved Test Design; execution gates remain candidate-specific. Owner: Cam (`@gonz0w`). Date: 2026-10-12.
[PRD](PRD.md), [EDD](EDD.md), [inventory](inventory.md),
[ADR](../../adr/0010-progressive-dashboard-charts.md).
Owner: [issue 166](https://github.com/gonz0w/horvium/issues/166).
TDD means Test Design Document; implementation uses failing behavioral tests first.

Approval: maintainer merge of [PR 349](https://github.com/gonz0w/horvium/pull/349),
source `4713aef87cca23183b949ba2ed5a56130b3b3186`, on 2026-10-12.
Adoption authorizes the scoped stage-A design and separately bounded stage-B
research, not dashboard/native/browser/resource or release acceptance.

## Traceability and observable gates

| Requirement / test ID | Required scenarios | Stage and evidence |
| --- | --- | --- |
| DASH-REQ-001 / DASH-STORY-001 | Question, coverage, generation and per-section freshness visible; empty vs partial/unavailable/stale; no health/backup/history claim | A HTML/full/fragment integration; B labelled synthetic entry; C separately designed |
| DASH-REQ-002 / DASH-DATA-001 | Zero, 2^53-1/2^53/2^53+1 and uint64 maximum; exact GUID/text values; allocated/free not additive; zero denominator; overlapping datasets; numeric sorting/ties | A Go behavior and real-browser tables; B string/coordinate validation; no new current JSON schema |
| DASH-REQ-003 / DASH-DRILL-001 | Same-generation row identity/filter/order/page/selection; eviction between Overview and pool read; unknown selection/malformed query; reset/back/expiry; no invented pool membership | A full/HTMX/no-JS parity; B keyboard/range controls with synthetic scope only |
| DASH-REQ-004 / DASH-UI-001 | Light/dark, keyboard/visible focus, names beyond color, screen reader, reduced motion, narrow/zoom; fallback with JS/Canvas/assets unavailable | Actual Safari/macOS and Firefox/Linux; manual VoiceOver and Linux screen-reader findings, not automated-only acceptance |
| DASH-REQ-005 / DASH-LIFECYCLE-001 | 100 repeated swaps/selection/reset/navigation cycles; exactly-once init/dispose, hidden/removed resize silence, pagehide/pageshow, no stale registry/data/listeners | A unchanged polling/selection; B instrumented owned adapter and actual browsers |
| DASH-REQ-006 / DASH-PERF-001 | 100/10,000 generated rows; default/max page limits; at most 20 bars; oversized response/series/points/payload denial; initialization/resize/reset and retained memory | A fresh service workload for new presentation; B actual asset/browser measurements at PRD bounds |
| DASH-REQ-007 / DASH-SAFETY-001 | Anonymous/session-expired, foreign/null origin and CSRF denial; hostile names/labels; self-only assets/CSP, eval/insertion/history disabled; hashes/notices; no native calls in HTTP | A regression/integration; B exact pinned source + strict CSP real browsers; C new native gates |

DASH IDs are preserved from roadmap direction. A receipts must mark B/C cases
not in that candidate; a fixture result cannot close qualified operator-history
or full dashboard gates. New collectors/public property/job integration have
separate accepted designs and native requirements. M1 manual acceptance stays open.

## Fixtures and test-first sequence

Before A implementation, add failing behavioral assertions against actual console
and cached-query boundaries: bounded rows, exact values, empty/failure states,
concurrent generation eviction and links to real original GUID rows. Reuse
synthetic helpers only through labelled development fixtures, never a product
fake-source mode. Use deterministic host/pool/dataset names and bytes, not exported
operator inventory. Include repeated names with distinct GUIDs across generations
and identity/name reuse; selection must stay scoped or explain expiry.

Expected bar count/scope and exact table text must be independent assertions,
not implementation-generated expectations. Test sums deliberately differing from
size, uint64 arithmetic overflow edges, partially available sections, cancellation,
reader admission and hostile Unicode/markup. Native collectors and internal JSON
responses retain their existing contracts. The original review packet contained no behavior implementation. Each adopted
implementation slice adds meaningful regression tests and its own source-bound receipt.

Before B implementation, define one bounded fixture owning at most two plots,
four Y series and 2,048 points per series with exact string source values. Include
deterministic null gaps, reset/clock-discontinuity segments, duplicate/malformed
and oversized data, theme changes and finite-coordinate denial. Synthetic gaps
exercise presentation only, not a real history producer. Reject a limit+1 before
allocating plots; do not silently truncate series/values or smooth gaps.

## Browser and lifecycle procedure

Record actual Safari/macOS, Firefox/Linux, drivers/OS, viewport, device pixel ratio,
CPU/RAM and exact fixture/asset source. Use installed authorized Safari settings;
unavailable automation is an explicit blocker, never a passed case. Inspect JS
on/off, light/dark, keyboard selection/reset/back, 390-pixel narrow viewport and
200%/400% zoom. Provide exact-value table captions/headers and usable controls when
Canvas fails. Record real VoiceOver and a supported Linux screen-reader experience,
including focus, textual status/labels and table traversal. Color-vision simulation
and measured WCAG 2.2 AA contrast supplement human inspection; they do not replace it.

B runs under actual unchanged response CSP and HTMX configuration. Capture console
violations and behavior for initialization, legends, cursor/selection/resize/theme,
fragment removal and full/bfcache navigation. Inspect same-origin network traffic;
no CDN, telemetry or background trend requests are expected. Test denied login,
expired generation and asset failure with useful fallback, not hidden exceptions.

Instrument owned registry/listeners/observers/queued frames/synchronization membership
in the fixture, not product routes. After 100 alternating subtree swaps and final
removal, require zero owned instances/observers/queued frames/data references and
no resize/render callbacks against removed/hidden elements. Show adapter counts
return to baseline; uPlot destroy alone does not prove all adapter resources freed.
Repeat with delayed resize, duplicate load and pagehide/pageshow. No leak claim
based solely on one successful image or console silence.

## Resource procedure and missing instrumentation

The [current-dashboard resource supplement](resource-qualification.md) is a
refinement of stage-A workload coverage and strict receipts adopted in PR 357.
The [portable request driver](../../testing/dashboards/resource-request-driver.md) is
the first fixture slice, followed by [original-inventory HTML predicates](../../testing/dashboards/resource-html-predicates.md).
Receipt validation, native wiring and fresh both-release execution remain required.
Adoption and portable regressions are not qualification.

A reruns affected console/cache checks plus fresh assembled service measurements
for its candidate using existing small/large native resource procedure on both
pinned disposable FreeBSD/amd64 releases before dashboard qualification. Preserve
original limits/exclusions and independent file-daemon continuity gates. Pure
presentation work adds no collector and cannot inherit a larger hardware claim.

B records raw/gzip asset bytes (gzip level 9, mtime 0) and complete loaded page
bytes including its adapter. After 10 warm-up iterations, measure 30 each of init,
resize and reset at the maximum points/series, report all samples and nearest-rank
p50/p95/p99 against the PRD p95 target. Event completion includes the next finished
render, not just a function call returning. Fixture source and workload remain fixed.

Measure retained JS heap relative to warmed table-only baseline, during maximum
series, after 100 cycles and after final disposal/collection. Do not use Chrome-only
memory APIs as Safari/Firefox evidence. If a browser's profiler cannot measure
retained heap consistently, record unavailable with versions/procedure, keep the
memory gate open, and rely on ownership counters only for their bounded structural
claim. Browser process RSS and OS pressure are reported separately when available;
heap delta is not whole-browser RSS. Synthetic browser measurements do not prove
FreeBSD native collection cost, disk/ARC behavior or commercial hardware capacity.

## Recording, teardown and acceptance

Every future receipt names exact signed source/base, actual PR title, selected
suite/build/assets, release/browser/fixture inputs, requirement/test IDs, complete
counts, failures/skips and independent teardown. Existing historical failures and
M1 accessibility limitations retain their original boundaries. Status lives in
GitHub; receipts and designs live in Git. Both dashboard delivery issues stay open
until applicable real candidate gates pass and maintainer accepts their scope.

Run `make check lint test secrets` and clean signed `make review`; use focused
race/fuzz selection for affected packages and `make review-full` at milestones/
releases. New sources need separately approved native disposable tests; no operator
storage/credentials or host service activation. Supervise preview/measurement
processes, close owned browser fixtures and verify `make lab-stop`/`make lab-status`.
Design adoption alone supplies no execution evidence. Candidate receipts record
actual scoped tests; full dashboard/browser/resource/native acceptance remains separate.
