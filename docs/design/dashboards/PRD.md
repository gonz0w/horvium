# PRD: Overview and Storage capacity dashboards

Status: Approved design; stage-A implementation beginning. Owner: Cam (`@gonz0w`).
Date: 2026-10-12. Phase: M2 presentation research; no new release phase.
Owner: [issue 166](https://github.com/gonz0w/horvium/issues/166), under
[gate 131](https://github.com/gonz0w/horvium/issues/131).
[EDD](EDD.md), [Test Design](TDD.md), [dependency ADR](../../adr/0010-progressive-dashboard-charts.md),
[source inventory](inventory.md), [maintainer direction](../../roadmap/dashboards.md).
Maintainer merge of PR 349 adopted this packet for its bounded implementation slices.
Approval does not accept either dashboard, M2 or the NAS release.

Approval: maintainer merge of [PR 349](https://github.com/gonz0w/horvium/pull/349),
source `4713aef87cca23183b949ba2ed5a56130b3b3186`, on 2026-10-12.
Adoption authorizes the scoped stage-A design and separately bounded stage-B
research, not dashboard/native/browser/resource or release acceptance.

## Problem and outcomes

An operator needs to compare observed storage capacity, find the resources behind
it and inspect exact values without mistaking an attractive chart for health,
backup proof or historical telemetry. Existing current-generation tables remain
the authoritative fallback. FreeBSD is the runtime; four disks and 16 GB are a
reference lab, not cardinality or hardware assumptions.

Overview answers “what is observed, how fresh is it, and where should I look?”
Storage capacity answers “what do these pool and dataset accounting values say?”
A current observation cannot answer “what changed last week?” or explain IOPS,
latency, utilization, quota headroom or impending failure. Show these gaps plainly.

## Delivery boundaries

| Stage | Proposed outcome | Prerequisite / exclusions |
| --- | --- | --- |
| A: current observations | Overview links/freshness plus bounded pool comparisons; pool and dataset capacity comparisons alongside exact tables | This packet adopted; existing public inventory only; no new assets, collectors or history |
| B: chart integration research | Labelled synthetic numeric-series fixture with pinned uPlot, progressive controls, themes, lifecycle and CSP evidence | Separately bounded reviewed fixture slice; no operator trends or synthetic product mode |
| C: qualified history | Operator trend plots tied to a separately approved identity/time/retention contract and actual sources | Fresh PRD/EDD/Test Design for history and collection, then native/browser/resource qualification; not authorized by this packet |

A and B are independent of runtime write enablement. Keep native safety work
moving independently. Existing [Overview 289](https://github.com/gonz0w/horvium/issues/289)
and [Capacity 290](https://github.com/gonz0w/horvium/issues/290) own broad delivery;
decompose into reviewed bounded issues on pickup, preserving their dependencies.
Do not create eight implementations or a history service from roadmap direction.

## Requirements and acceptance

These retain the roadmap IDs and narrow their applicability by stage.

| Requirement | Observable outcome | Test IDs / scope |
| --- | --- | --- |
| DASH-REQ-001 Honest story | Each view names its question, generation, section observation time/status and stale/partial/unavailable limits; no unsupported health/cause/restore/trend claim | DASH-STORY-001; A and B |
| DASH-REQ-002 Exact data | Bytes and GUIDs remain exact; pool and dataset accounting never masquerade as an additive total; chart coordinates are labelled approximate | DASH-DATA-001; A and B; real historical rates remain C |
| DASH-REQ-003 Drill-down | Links/forms preserve resource identity, generation and available query scope; explicit expiry/reset/back; keyboard and JavaScript-disabled equivalents | DASH-DRILL-001; A and B; historical range requires C |
| DASH-REQ-004 Accessible theme | BSD Classic light/dark, textual meaning beyond color, usable tables/focus and real Safari/Firefox/assistive-technology observations | DASH-UI-001; all delivered stages |
| DASH-REQ-005 Lifecycle | Fragment removal/navigation releases owned work; enhancements are idempotent; resource inspection never silently changes generation | DASH-LIFECYCLE-001; A navigation, B charts |
| DASH-REQ-006 Performance | Materialization, rendered rows, charts, series, points, assets and owned browser work obey explicit bounds; existing service budgets remain | DASH-PERF-001; A and B |
| DASH-REQ-007 Safe delivery | Cached unprivileged reads only; escaping, authentication, CSP, provenance/notices and denial paths preserved | DASH-SAFETY-001; A and B; C requires native source qualification |

Each delivered stage must pass all applicable rows. A scoped current-data receipt
must mark history/uPlot cases outside that candidate rather than complete C or
either broad delivery gate. Stage B is research evidence until separately adopted.

## Compatibility and proposed budgets

Preserve M1's default 30-second collection, 90-second stale threshold, two volatile
generations, 100-row default/500-row maximum pages, 32 MiB response ceiling,
configured reader/connection limits and whole-application resource gates. See
[EDD](../../milestones/M1/EDD.md), [performance evidence](../../testing/M1/performance.md)
and [qualification limits](../../testing/M1/qualification-status.md).
No persistent history, additional collector cadence or background browser timer
is introduced in A. The retention contract is explicitly zero durable samples;
two replaceable generations are inspection cursors, never a trend buffer.

| Adopted additional bound | A | B synthetic fixture |
| --- | --- | --- |
| Visible comparisons | At most 20 rows per comparison; exact table pagination remains 100/500 | Same fallback |
| Materialized comparison inputs | At most one bounded page per source per view, never drain every page | Fixed generated fixture only |
| Plot instances / numeric series | Zero | At most two simultaneous plots / four Y series total |
| Points | Zero historical points | At most 2,048 per Y series; 8,192 total; no unbounded append |
| Encoded chart data | None | At most 512 KiB per page, reject before plot allocation |
| Added compressed asset budget | None | At most 32 KiB gzip for uPlot JS, CSS and owned adapter combined |
| Browser responsiveness | Existing page/focus behavior retained | After warm-up, p95 initialization/resize/reset under 100 ms; 30 measured iterations each |
| Owned browser memory | No charts | Retained heap increase <=64 MiB from warmed fallback; after 100 swaps/removal <=5 MiB above that baseline where measurable |

These are review targets, not measured results or whole-browser/whole-NAS limits.
The Test Design specifies browsers, workload and missing-memory instrumentation
handling. Existing small <=128 MiB / large <=256 MiB paired Go-process budgets,
small idle <1% of one core and large HTTP p95 <250 ms are retained; new views need
new candidate-specific measurements before inheriting those claims. Ten thousand
synthetic rows do not qualify ten thousand real datasets or commercial hardware.

## Non-goals and unresolved delivery prerequisites

No privileged action, quota/property editor, disk health inference, backup proof,
share traffic, public job/recovery projection, new API route/config/CLI schema,
time-series persistence or collector is added. Existing JSON numbers are not
changed into strings by this design; an enhancement must not parse uint64 JSON
numbers through JavaScript and call them exact. The EDD defines a separate display
projection without changing existing internal JSON consumers.

The first history producer, persistent resource identity binding, clocks/gaps,
retention/storage format and native sampling cost are unresolved C prerequisites.
They require a separately reviewed packet, not an arbitrary retention choice here.
uPlot's strict-CSP and actual browser behavior remain B qualification questions;
no security policy relaxation is implicitly approved. Manual accessibility stays
pending by the maintainer's existing choice; approval of this packet does not
waive that gate or invent completed observations.
