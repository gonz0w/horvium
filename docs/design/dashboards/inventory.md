# Dashboard public facts and evidence inventory

Date: 2026-10-12. Status: source review, not new native/browser qualification.
Review base: `38d9366c510eed999c1e422467b29dd596a7d90f` (merged PR 348).
Owner: [issue 166](https://github.com/gonz0w/horvium/issues/166).
[PRD](PRD.md), [EDD](EDD.md), [Test Design](TDD.md).
All source links below bind this inspected base; future behavior needs its own receipt.

## Existing runtime-facing facts

“Public” here means the authenticated management read model available to existing
HTML/CLI adapters, not a stable external release API or unauthenticated endpoint.

| Source / contract | Actual available facts | Limits for dashboard use |
| --- | --- | --- |
| [Inventory model](https://github.com/gonz0w/horvium/blob/38d9366c510eed999c1e422467b29dd596a7d90f/internal/inventory/model.go) | Pool name/GUID, size/allocated/free uint64 bytes, reported pool health | No vdev/device health, scan progress or physical failure prediction |
| Same model | Filesystem name/GUID, used/available/referenced uint64 bytes | No structured pool/parent membership, quota/reference quota, ownership, snapshot breakdown or public property source |
| Same model | Host kernel/userland/ZFS versions, architecture, optional physical memory; GEOM graph/size/identity confidence | Memory capacity is not utilization; provider IDs are collection-local |
| [Observation manager](https://github.com/gonz0w/horvium/blob/38d9366c510eed999c1e422467b29dd596a7d90f/internal/observation/manager.go) | Detached Overview counts/host facts, section status/start/end/last-success/staleness; current/previous generation | Counts need successful section status; per-section success is not host health; no history persistence |
| [Queries](https://github.com/gonz0w/horvium/blob/38d9366c510eed999c1e422467b29dd596a7d90f/internal/observation/query.go) | Exact numeric sorts, literal name filters, bounded pages and current/previous resolution | Default 100/max 500 rows, generation expiry; no pool-GUID dataset selection or historical range contract |
| [Console adapter](https://github.com/gonz0w/horvium/blob/38d9366c510eed999c1e422467b29dd596a7d90f/internal/console/console.go) | Full/fragment/JSON reads, escaped exact decimal byte tables, selected GUID in displayed page | Overview has counts, not pool rows; pool comparisons require a separate bounded cached query bound to the same generation |
| [Console template](https://github.com/gonz0w/horvium/blob/38d9366c510eed999c1e422467b29dd596a7d90f/internal/console/assets/console.html) | Semantic forms/links, generation-bearing storage navigation, exact table/detail facts and explicit not-collected sections | No current charts, public job timeline or trend control |
| [Owned enhancement](https://github.com/gonz0w/horvium/blob/38d9366c510eed999c1e422467b29dd596a7d90f/internal/console/assets/console.js) / [HTTP policy](https://github.com/gonz0w/horvium/blob/38d9366c510eed999c1e422467b29dd596a7d90f/internal/admin/http.go) | Local HTMX 2.0.11; eval/script insertion/history snapshots disabled; self-only CSP and gated reads | An adapter must preserve these restrictions and authentication; native commands never belong in HTTP |

The existing inventory validator checks allocated/free each against size; it does
not assert their sum equals size. Draw separate zero-origin measurements, not a
stack that forces additivity. Dataset values overlap through hierarchy/snapshots
and inherited availability. Name matching cannot prove pool membership or resource
identity across observations. GUIDs plus the exact observation identify a row for
inspection; they confer no write eligibility, durable-history lineage or ownership.

## Available evidence and retained gaps

[M1 console evidence](../../testing/M1/console.md),
[qualification record](../../testing/M1/qualification-status.md) and
[performance evidence](../../testing/M1/performance.md) preserve merged source IDs,
actual two-release native observations, synthetic scale results, Firefox flows
and failed/skipped limitations. They establish the original inventory behavior,
not future dashboard browser/memory measurements. Safari/VoiceOver/manual usability
remains pending. Neither the MicroServer nor larger physical/commercial workloads
are qualified by QEMU projection fixtures.

The approved [M2 PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md) and [runtime contract](../../milestones/M2/property-runtime.md)
contain private property/planning/recovery prerequisites. They are not public
quota, launch, job or recovery APIs. [Owner 305](https://github.com/gonz0w/horvium/issues/305)
and [gate 139](https://github.com/gonz0w/horvium/issues/139) remain open. The
[one-candidate diagnostic](../../research/M2/policy-startup-return-path.md) passed
its bounded observations but did not reproduce/explain the earlier timeout or
qualify full native lifecycle; runtime writes remain disabled.

## Explicit gaps and proposal

| Desired story | Gap | Proposed treatment |
| --- | --- | --- |
| Current pool comparison | Overview query supplies no pool rows | One bounded pool page for its resolved generation; truthful coverage caption; fail clearly on intervening eviction |
| Ranked dataset usage | No verified dataset-to-pool relation in public model | Compare current filtered dataset page, labelled scope; do not fabricate pool drill-down membership |
| Quota headroom/properties | Private projections only | Keep absent until separately assembled/qualified public contract |
| Capacity trend/forecast | No durable samples, persistent lineage, time/gap/retention contract | Unavailable in A; labelled synthetic fixture only in B; separate C design |
| Performance, health, jobs, shares/protection | Metrics/public domain integration absent or unqualified | Follow existing domain issues; no placeholder data or inferred success |
| Chart accessibility/lifecycle/resources | No uPlot asset/adapter candidate or browser execution | Proposed ADR/Test Design; qualify before product adoption |

This inventory completes source inspection only. None of the planned DASH tests
has executed on a dashboard candidate. No historical receipt is rewritten.
