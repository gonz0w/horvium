# Dashboard roadmap: uPlot and the storage data story

Status: Planned direction; implementation deferred. Owner: Cam (`@gonz0w`).
Date: 2026-10-10. Authority: maintainer selected uPlot and requested roadmap
planning for the build chat to pick up later.

This records the library choice and desired experience, not an approved
implementation PRD/EDD/Test Design packet, telemetry contract or qualification
evidence. Preserve current M1/M2 work and acceptance gates. Follow the
[milestones](milestones.md), [UI baseline](../ui/baseline.md),
[dependency policy](../engineering/dependency-policy.md) and
[document lifecycle](../process/document-lifecycle.md).

Live dashboard design pickup, priority and dependencies are in [issue #166](https://github.com/gonz0w/horvium/issues/166)
and the [roadmap index](github-roadmap-index.md). The design obligations below
remain authoritative planning material, not a second progress checklist.

The first [Overview/Capacity adopted packet](../design/dashboards/PRD.md) inventories
[current public facts](../design/dashboards/inventory.md), proposes the
[presentation contract](../design/dashboards/EDD.md), [Test Design](../design/dashboards/TDD.md)
and [uPlot pin/integration ADR](../adr/0010-progressive-dashboard-charts.md), adopted
by maintainer merge of PR 349. Adoption authorizes its bounded design/slices,
not dashboard qualification. Current comparisons
precede a labelled synthetic chart fixture; real history needs separate review.

## Operator story

Each view answers one question and leads to supporting evidence:

1. **Orient:** what needs attention, and how fresh is the evidence?
2. **Compare:** what changed within the selected time and resource scope?
3. **Locate:** which pool, dataset, device, share or job accounts for the change?
4. **Explain:** what exact observations and related events can be inspected?
5. **Act deliberately:** follow an existing authorized workflow when available.
   A chart grants no mutation, recovery or privileged authority.

Place a short factual summary above the main chart, naming resource, interval
and evidence. State uncertainty; coincident spikes do not establish causation.
Collection success is not host health. Use the existing navigation sections.

## Dashboard set and prerequisites

Views depend on reviewed public contracts and actual qualification, not just a
milestone number. Show current facts first; add trends when sources are available.

| View / operator question | Visual sequence | Drill into | Prerequisite |
|---|---|---|---|
| Overview: what needs attention? | Freshness/concerns, capacity comparisons, available trends, related jobs/events | Resource or issue with evidence time/scope | M1 current inventory; accepted history/domain contracts for trends |
| Storage capacity: where is space going? | Pool allocated/free comparison, capacity history, ranked dataset usage with exact values | Pool then dataset, properties and own history | M1 capacity; bounded history; M2 quota/property integration |
| Storage performance: when and where did pressure rise? | Aligned throughput, operations and latency; read/write comparison | Measured pool/device/dataset attribution, interval and related events | New qualified FreeBSD metrics/history contracts |
| Storage health: what is degraded or uncertain? | Reported state, scrub/resilver progress/findings, meaningful error history | Original pool/device identity, scan/finding evidence | M2 public health/scan integration; separate history/device-health qualification |
| Shares: which service or share is affected? | Availability, measured traffic/errors, attributable ranked activity | Protocol/share and authorized interval detail | M3 service/client qualification and share attribution |
| Protection: can I demonstrate recoverability? | Snapshot/replication freshness, receiver/backlog status, independently verified restore evidence | Dataset, backup peer, job or restore receipt | M4 contracts and restore qualification; replication success is not restore proof |
| Jobs: what ran, stalled or needs review? | Labelled lifecycle timeline, measured elapsed/progress, related observations | Retained history and separately labelled recovery assessment | M2 public job/recovery contracts; private prerequisites are not runtime availability |
| System: is the host or observer the constraint? | Separate aligned CPU, memory, network and collection-freshness plots | Interface/resource, collection errors/events | Qualified metrics/history; physical memory inventory is not utilization |

Current inventory supports capacity comparisons, not durable history, IOPS,
latency, traffic, ARC efficiency or backup success. M1's two-generation cache is
not a time-series store. Omit nonfunctional charts and explain unavailable data.
Label synthetic previews and isolate them from operator views. Client identities
are not exposed by default in Shares comparisons.

ZFS dataset usage/availability overlaps through hierarchy, snapshots and shared
accounting. Do not sum those values into claimed pool allocation/free capacity.
Explain accounting basis. Forecasts need adequate history, method and uncertainty.

## Visual and color direction

Continue BSD Classic: system sans-serif text, monospace technical values, neutral
surfaces, fine rules, restrained warm red branding and coherent light/dark views.
Make supporting tables as legible and polished as the plots.

- Use uPlot for time/numeric series, synchronized cursors and interval selection.
  Align small multiples for different units; avoid misleading dual-axis overlays,
  decorative gauges, 3D, gradients and smoothed counter/rate curves.
- Use project-owned HTML/CSS or bounded Go-generated SVG for capacity bars,
  rankings, statuses and job timelines. No second chart engine is implied.
- Use semantic color roles: cool blue for primary/read/receive, teal for
  secondary/write/send, amber for attention, red for verified error and gray
  for unavailable context. Give resources a separate categorical palette so
  ordinary comparison series are not mistaken for alarms.
- Derive colors from theme tokens with deliberate light/dark variants. Validate
  text, controls and essential chart marks against applicable WCAG 2.2 AA criteria.
  Pair names with solid/dashed strokes, symbols or explicit text; color alone
  never communicates meaning. Keep series identity stable across related views.
- Limit initially visible series; selection/pagination handles large inventories.
  Sequential lightness encodes magnitude; diverging scales encode only meaningful
  signed change. Reserve status colors for status.
- Label units, scales, intervals, aggregation and coverage. Magnitude bars start
  at zero; narrowed line scales are explicit. Missing/reset/invalid intervals are
  breaks, never zero. Do not conceal gaps or other series with area fill.

## Progressive drill-down and data integrity

Range/resource selection leads to an exact-value table and resource details.
Provide equivalent keyboard-operable forms/links, reset, breadcrumbs and full-page
navigation. Preserve scope/time/generation throughout; explain expiry instead of
silently substituting newer observations. Pause automatic movement during history
inspection and offer explicit return to current. No-JavaScript views retain useful
summaries, tables and navigation; Canvas is an enhancement.

Go owns validation, authorization, aggregation and bounded materialization.
Browser selection is input, never authority. HTTP reads approved cached management
contracts; handlers/templates never start native commands or privileged collection.
Bind resources to qualified identity, not names; handle rename/reuse/restart.

Keep canonical large integers and identifiers exact. Plot coordinates may be
approximate; table/export values remain exact. Derive rates from actual elapsed
intervals, distinguish reset/wrap/gaps/clock changes and never connect across them.
Resolution changes preserve gaps and relevant peaks; zoom obtains bounded source
detail rather than invented interpolation. Percentiles need a qualified distribution
source, not arithmetic on precomputed percentiles. Event overlays show correlation
without alleging cause.

## Dependency and integration direction

Selected upstream: [uPlot](https://github.com/leeoniya/uPlot), under the
[MIT license](https://github.com/leeoniya/uPlot/blob/master/LICENSE). Retain its
copyright/license; Horvium's code remains BSD-3-Clause. Upstream documents zoom,
cursor synchronization and hooks; panning is an additional integration choice.
Reverify features against the selected release before implementation.

Pin a reviewed release/source and JS/CSS hashes, retain notices and record actual
compressed bytes. Serve assets locally and load only where needed. A small
project-owned adapter translates theme tokens, initializes page/fragment charts
and disposes charts/listeners/observers before removal. Use
[HTMX lifecycle events](https://htmx.org/events/) while preserving existing CSP,
script/evaluation restrictions and authentication. No runtime Node, CDN, SPA,
frontend framework or animation engine is required.

Reuse conventions across NAS, DNS, firewall and BSD cloud when separately adopted
there. This roadmap changes Horvium only; it specifies no shared package, new
repository, API routes, config schema or CLI flags.

## Build chat pickup sequence

1. Inventory public data/evidence. Start with Overview and Storage capacity,
   preserving current inventory semantics.
2. Prepare linked PRD, EDD, Test Design and dependency/integration ADR for review.
   The uPlot preference is settled; select concrete release, lifecycle and first
   history contract. Set payload, points/series, retention, cadence, memory and
   interaction budgets before implementation. Upstream benchmarks are not Horvium
   evidence; avoid unbounded telemetry infrastructure.
3. Build a labelled synthetic fixture for theme/interaction/lifecycle qualification.
   Introduce pinned assets/notices with the reviewed build slice.
4. Deliver current-data comparisons, then qualified history. Performance/System
   follow metrics; Health/Jobs, Shares and Protection follow domain integration.
   Do not delay current M2 safety work for the full dashboard set.
5. Carry planned IDs below into the active packet, record results/limitations,
   update compatibility evidence and leave PRs for review.

## Acceptance targets

These are planned obligations, not passing results or an approved build packet.

| Requirement | Planned test ID | Required evidence |
|---|---|---|
| DASH-REQ-001 Honest story | DASH-STORY-001 | Question answered, freshness/scope visible, unavailable explicit; no unsupported causal/health/restore claim |
| DASH-REQ-002 Exact data | DASH-DATA-001 | Large integers, ZFS accounting, zero/missing, identity reuse, resets/clocks/unequal intervals and aggregation |
| DASH-REQ-003 Drill-down | DASH-DRILL-001 | Scope/range/generation parity, authorization denials, JS-disabled/keyboard equivalents, expiry/reset/back |
| DASH-REQ-004 Accessible theme | DASH-UI-001 | Real Safari/Firefox, light/dark, color-vision simulation, keyboard/focus, screen reader, reduced motion, narrow/native zoom |
| DASH-REQ-005 Lifecycle | DASH-LIFECYCLE-001 | Repeated HTMX swaps/navigation release charts/listeners/observers; refresh preserves selection; hidden/removed views stop owned work |
| DASH-REQ-006 Performance | DASH-PERF-001 | Actual assets, many-resource fixtures, point/series bounds, resize/zoom cost, browser memory and existing service budgets |
| DASH-REQ-007 Safe delivery | DASH-SAFETY-001 | Asset/hash/notices, hostile data, existing CSP/auth/CSRF and no privileged HTTP path; disposable native qualification for new collectors |

Use failing behavioral/regression tests where practical. Run `make check lint test
secrets` and appropriate `make review`; `make review-full` at milestone/releases.
Apply disposable FreeBSD native gates when collection changes. Presentation fixtures
cannot establish native telemetry, protocol, hardware or whole-NAS performance.
