# Bounded pool-page capacity comparisons

Date: 2026-10-12. Owner: [issue 350](https://github.com/gonz0w/horvium/issues/350),
under [Storage capacity delivery 290](https://github.com/gonz0w/horvium/issues/290)
and [M2 gate 131](https://github.com/gonz0w/horvium/issues/131).
Authority: [adopted PRD](../../design/dashboards/PRD.md), [EDD](../../design/dashboards/EDD.md),
[Test Design](../../design/dashboards/TDD.md), [ADR 0010](../../adr/0010-progressive-dashboard-charts.md),
maintainer merge of PR 349, source `4713aef87cca23183b949ba2ed5a56130b3b3186`.
This is a stage-A pool-page implementation subset of DASH-REQ-001–007,
not qualified dashboard or native/release delivery.

## Contract and safety boundary

The existing authenticated Storage → Pools full page and HTMX fragment show up
to 20 comparisons from their original filtered/sorted/paginated detached page.
No extra manager query, collector or native command is introduced. Coverage names
the comparison row interval, matching count, filter/order, generation and section
read/last-success time. Each link uses the original page query/GUID, preserving
the exact table, selection and generation expiry behavior. Internal JSON/CLI
uint64 fields, collector, privilege and file-service boundaries remain unchanged.

Allocated and free are separate zero-origin measurements against reported size,
never forced into additive segments. Coordinates/percentages use overflow-safe
128-bit intermediate integer arithmetic, rounding to the nearest 0.1% with ties
up; exact uint64 byte values and GUIDs remain in the table. Large values can share
a rounded coordinate without being equal. Zero reported size supplies an explicit
unavailable ratio without an SVG. Empty successful pages show no invented data;
unavailable sections suppress comparisons. The display guard also rejects partial
status; the current inventory validator already rejects partial pool observations.
Stale successful pages retain their original labelled observation/time.

SVG is decorative to assistive technology with no focusable controls; named HTML
links and exact tables remain usable without JavaScript. Project-owned CSS supplies
text labels plus blue/teal light/dark bars. Inputs are escaped, numeric coordinates
are typed, and no inline style, new script, uPlot asset, native source, route/schema,
mutation capability or background timer is introduced. Existing authentication,
CSRF, self-only CSP and HTMX restrictions remain intact.

## Behavioral tests and evidence boundaries

Failing-first focused tests originally detected absent comparisons; a first draft
fixture incorrectly supplied unsupported partial pool metadata and a section time
outside its snapshot. Those fixture errors were corrected before implementation;
the corrected red run failed specifically on missing comparisons. The initial
sandbox run could not bind the owned supervisor's local control socket and is not
behavior evidence. Failed records remain private and never qualify this slice.

The pool comparison tests cover full/fragment agreement, exact 2^53 and uint64
values, non-additive accounting, nearest/tie rounding, zero-size and stale/empty/
unavailable cases, defensive non-success rejection, 20-row bounds at ordinary/max
page size, preserved exact tables/query/GUID links, JSON parity, escaping, expired
generations, anonymous denial and invalid selection. Existing ordinary console,
authentication/cache/response/CLI tests remain required. Test outcomes, exact source
and fresh review/cleanup records belong in the candidate receipt accompanying this
contract; proposed browser/native/performance gates are not inferred from Go tests.

## Remaining acceptance and rollback

Overview and dataset comparisons are separate implementation slices. This does
not qualify uPlot/CSP integration, operator history, Safari/Firefox/manual assistive
technology, zoom/contrast assessment, fresh native service/resource/performance,
physical hardware/commercial scale, runtime writes or release acceptance. Required
DASH and M1/M2 gates stay open under their delivery/acceptance issues. Historical
native/browser/resource receipts retain their original sources and limitations.
No skipped or unexecuted test becomes passed. The earlier native startup deadline
remains unexplained and full lifecycle gates remain open under 305/139.

Rollback restores the original pool page/embedded CSS and removes the private
presentation helper; no storage state, journal, schema or configuration migration.
Development previews use labelled generated inventory with owned process teardown.
Do not enable host services or use operator disks/datasets/backups for validation.
