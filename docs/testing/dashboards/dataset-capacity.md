# Bounded dataset-page Used comparisons

Date: 2026-10-12. Owner: [issue 352](https://github.com/gonz0w/horvium/issues/352),
under [Storage capacity delivery 290](https://github.com/gonz0w/horvium/issues/290)
and [M2 gate 131](https://github.com/gonz0w/horvium/issues/131).
Authority: adopted [PRD](../../design/dashboards/PRD.md), [EDD](../../design/dashboards/EDD.md),
[Test Design](../../design/dashboards/TDD.md) and [ADR 0010](../../adr/0010-progressive-dashboard-charts.md),
maintainer merge of PR 349. Reuses the merged pool comparison foundation in PR 351.
This is the portable stage-A dataset subset of DASH-REQ-001–007; broad dashboard,
browser/native/resource and release acceptance remain separate.

## Display contract

The authenticated Storage → Datasets full page and HTMX fragment compare Used for
at most the first 20 rows of the existing detached filtered/sorted/paginated page.
The exact table remains complete at its existing default 100/maximum 500 rows.
There is no extra manager query, native collection or inventory materialization.
Coverage names the matching count, row interval, query, generation, section status,
read time and last successful read. Each link selects its original GUID using the
original query and generation; selection/expiry, JSON and CLI contracts are preserved.

The local scale is the largest individual Used value on that same table page,
including rows beyond the comparison limit. Its exact byte value is visible; it
may change with filtering, ordering or pagination. It is not a global top-N list.
Ordering follows the existing operator choice, including exact numeric Used sorting.
No new sort or query schema is added. Used, Available and Referenced can overlap
across parents, children, snapshots or shared availability; these values are never
summed into pool usage, utilization or quota headroom. Available/Referenced stay
independent exact table values. No name-based pool membership is inferred.

Zero-origin decorative SVG bars use the existing overflow-safe integer ratio
helper, rounded to 0.1% with ties up. Percentages refer explicitly to the page
scale, not total capacity. Exact uint64 bytes and GUIDs never pass through browser
floating-point numbers. A zero Used with a nonzero scale has a zero-width bar;
all-zero pages have an explicit unavailable scale and no SVG or invented ratio.
Empty successful pages show no data; unavailable sections suppress comparisons.
The display guard rejects partial/non-success metadata; the current inventory
validator already rejects partial dataset sections. Stale successful observations
retain their original labelled times. Invalid selection and expired generations
retain their existing denial/reset behavior.

Named semantic links and the exact table remain usable without JavaScript. SVG
is aria-hidden and non-focusable. Existing project-owned light/dark CSS styles the
single Used bar; labels explain the metric beyond color. No inline style, trusted
observed HTML, new script, timer, chart dependency, telemetry or history is added.
Authentication, authorization, CSRF, strict CSP and HTMX restrictions remain intact.
No shell, privilege, native command, disk/dataset mutation or service deployment.

## Behavioral verification and limits

Failing-first tests detect the missing presentation before implementation. Tests
cover exact 2^53/uint64 values and GUIDs, overlapping independent accounting,
page-local maximum outside the first 20 bars and a larger value outside the page,
20-row limits at ordinary/maximum page size, exact table and JSON preservation,
full/fragment agreement, original query/selection links, markup escaping,
zero/empty/stale/unavailable metadata, defensive partial/non-success rejection,
anonymous denial, invalid selection and actual cached generation eviction.
Pool regressions continue to exercise shared ratio rounding and overflow boundaries.
Candidate-specific results belong in the accompanying receipt, not live task lists.

No Safari/Firefox/manual screen-reader, contrast/zoom/narrow or browser lifecycle/
resource matrix is qualified by portable tests. No fresh native service/resource
measurement, real FreeBSD execution, commercial scale or hardware support follows.
Overview, uPlot synthetic research, identity-bound history and native sources are
separate slices. M1 manual accessibility remains pending by maintainer choice;
#290/#131 and native #305/#139 remain open. Historical failures/skips and the earlier
unexplained startup deadline retain their original boundaries; writes stay disabled.

Rollback removes the private dataset projection and embedded presentation changes;
there is no storage, configuration, schema or journal migration. Any development
preview uses labelled synthetic inventory and owned process/tab teardown.
