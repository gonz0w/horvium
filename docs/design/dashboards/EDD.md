# EDD: progressive capacity presentation

Status: Review; proposed, not approved implementation. Owner: Cam (`@gonz0w`).
Date: 2026-10-12. [PRD](PRD.md), [Test Design](TDD.md), [inventory](inventory.md),
[ADR 0010](../../adr/0010-progressive-dashboard-charts.md).
Refs [issue 166](https://github.com/gonz0w/horvium/issues/166).
This refines presentation only after adoption; existing M1/M2 contracts and
native recovery gates remain authoritative. No new runtime interface is named.

## Current-data contract (stage A)

HTTP continues to query the unprivileged immutable observation manager after
existing session/provenance and reader admission. No handler/template collects,
executes native commands, opens protected storage or admits a write. Preserve
existing full/fragment/CLI query semantics, strict validation and output limits.
There is no new wire version, route, flag or configuration key in this packet.

Resolve Overview once, then request at most one pool page with that exact
returned generation, name order, default 100-row limit and no filter. Display
comparisons for its first 20 rows, labelled “first N pools by name in observation
G; X observed pools; more available” when applicable. Count X is usable only
with successful pool section status. No comparison claims to find every degraded
pool or rank capacity across all resources. Links lead to the existing pool
list with the same generation; row selection uses the original GUID on a page
containing it. If another collection evicts G between calls, render explicit
expiry/reload, never combine a new pool page with the old Overview.

On Storage pool/dataset views, compare at most 20 rows of the existing filtered,
sorted, paginated result. Reuse its detached rows, do not scan every page to build
a chart. Caption names generation, filter/order, displayed interval of rows,
matching count only when known, and section freshness. Ranking means the chosen
page/order, not an unqualified top-N for a different scope. Existing default sort
remains name; an operator can explicitly choose Used descending for dataset ranking.

Dataset links inspect original GUID in the same displayed page/generation. Pool
links inspect a pool; do not turn a name prefix into verified dataset membership.
A later pool-to-dataset public relation needs qualified identity and a reviewed
contract. Provider local IDs never become durable dashboard resource identifiers.
Neither selection nor a GUID from a chart grants native authority.

## Exact values and visual accounting

Go retains uint64 bytes and decimal-string GUIDs. Server-rendered exact tables
and details remain canonical; no chart converts an identifier to a JavaScript
Number. Current internal JSON uint64 fields remain unchanged. Any B fixture/data
projection uses canonical decimal strings for byte values/IDs, with typed Go
validation and separately derived finite approximate plot coordinates; it does
not feed existing numeric JSON through browser parsing and call it lossless.
No new export API is promised; exact table text can be inspected/copied.

A uses bounded Go-rendered SVG with project CSS for capacity comparisons, avoiding
new JavaScript or inline CSS. Templates escape resource text; coordinates/colors
come from fixed validated presentation values, never arbitrary observed markup.
No foreignObject, scripts, hyperlinks derived from raw names or trusted-HTML cast.
A chart is decorative to assistive technology when the adjacent labelled exact
HTML table conveys the full data and links. Do not duplicate a misleading set of
interactive controls inside inaccessible SVG/Canvas.

Draw allocated and free as separate bars against the original pool size; each
starts at zero. Do not force allocated+free=size or manufacture “other” from an
underflowing subtraction. Zero size has exact zero values and an explicit
unavailable ratio, not division or a fake 100% fill. Ratios use overflow-safe
integer arithmetic before conversion to bounded presentation coordinates; label
rounded percentages as approximate. Values near uint64 maximum remain exact in
tables even when two bars cannot show their one-byte difference.

Dataset Used, Available and Referenced are independent accounting observations,
not disjoint slices. Rank one selected metric; do not sum datasets into pool
allocation or show Used/(Used+Available) as a pool utilization guarantee. State
that hierarchy, snapshots and shared space can overlap. A page-local normalized
ranking states its local maximum/scale; it never becomes a quota-headroom gauge.

## Freshness and missing data

Bind charts and tables to the same source section. `ok` with zero rows is an
observed empty section; `partial`/`unavailable` is unknown and must not claim zero
resources or render an apparently complete comparison. For A, suppress comparisons
when that section is not `ok`; retain existing honest table/error behavior.
A stale `ok` page may retain comparisons clearly marked stale with original time;
never relabel it current because another section succeeded. Section problem codes
are safe typed metadata, never raw stderr. Each view exposes observation generation
and relevant section start/end/last-success as existing contracts permit.

Refresh continues to request one deduplicated collector through the existing
CSRF-protected POST. Background status polling changes only its status element;
it does not replace charts/table selection or silently move resource inspection
to another generation. Current-generation/reset is an explicit full-page link.
Back/forward resolves the recorded query or explains expiry. No local/session
storage caches authenticated inventory; no history snapshots or query credentials.

## uPlot ownership (stage B only)

The [ADR](../../adr/0010-progressive-dashboard-charts.md) pins proposed upstream
inputs and identifies observed destroy/label/style behavior. A separate fixture
slice must verify those bytes before embedding; serve only local assets, no Node
runtime, CDN, SPA, plugin ecosystem or new Go module. Load chart assets only on
pages that use them; fallback remains independent of their availability.

One owned adapter manages an idempotent instance registry keyed by original DOM
container, not names or arbitrary observed IDs. Initialize on initial document
load and `htmx:load` for added subtree roots. Before `htmx:beforeCleanupElement`
removes/disables a subtree, dispose its instances exactly once. Call uPlot destroy,
unsubscribe owned synchronization membership, disconnect ResizeObserver and
remove owned listeners, queued animation frames and data references. Keep only
one bounded global lifecycle listener set. Full navigation/pagehide also disposes;
bfcache pageshow reinitializes only connected eligible containers. No refresh,
polling or remote fetch timer is added by the adapter.

Hidden containers stop resize/render work; return to visible revalidates layout
without duplicating ownership. Coalesce resize notifications into at most one
pending frame per instance, cancel on disposal and never allocate plots at zero
or unbounded dimensions. Theme change derives colors from project-owned tokens
and redraws existing instances without spawning duplicates. Synchronize cursors
only between the fixture's two plots with identical synthetic time scope; no
global registry bridges authenticated pages or different resources.

Canvas interactions are enhancements. Named keyboard-operable forms/selectors,
explicit reset and exact tables implement every supported selection without
JavaScript. Pointer zoom/cursor synchronization uses pinned upstream facilities;
panning is deferred. All B controls manipulate labelled synthetic data only.
They do not invent an operator historical range API. Exact values/units, interval,
coverage and aggregation accompany plots; missing points are gaps, never zero.

## Security, failure and operations

Keep existing self-only script/style/connect policy, authentication, CSRF,
allowEval=false, allowScriptTags=false, includeIndicatorStyles=false and disabled
HTMX history snapshots. Proposed upstream uses dynamic style-property assignment;
static source inspection cannot qualify it under our actual CSP. B must test
actual strict headers, theme/layout/cursor/resize paths and console violations
in both selected browsers. Failure blocks adoption; propose a separately reviewed
compatible adaptation if needed. Do not add unsafe-inline/eval or silently bypass
security checks to make a chart display.

Untrusted names/labels enter escaped HTML or textContent, never innerHTML. Adapter
inputs allow only approved finite coordinates, exact canonical strings and literal
series roles; reject unknown fields, oversized/malformed data, duplicate ownership
and over-budget series/points before allocation. An enhancement failure retains
the exact fallback plus a safe unavailable explanation; it must not retry or
replace the table. Authentication expiry follows existing sign-in/denial behavior;
discard owned chart state on workspace removal/logout navigation.

A adds no persistence/migration. B introduces only removable local fixture assets
and the owned adapter in its separately reviewed slice. Rollback restores tables
and removes assets/registration; no live data, native journal, daemon or storage
change is required. Development fixtures use supervised ownership and verify
lab-stop/status. Never use operator datasets/backups or enable host services.

## History boundary and future review (stage C)

Retention is zero persistent samples for A/B; B uses a fixed deterministic
synthetic series, labelled at every entry. Two volatile generations are not history.
Before any operator history, review the original source/units/measurement scope,
qualified resource lineage across rename/reuse/restart/import, collection interval,
monotonic elapsed/civil time handling, bounded persistence and retention/eviction,
point/series/payload limits and measured memory/disk/collection costs. Define gaps,
reset/wrap and clock-change segmentation, unequal intervals, resolution changes
and bounded drill-down without invented interpolation or percentile arithmetic.
Counter rates require actual elapsed intervals and distributions require qualified
sources. That future contract needs native disposable evidence on both declared
releases and fresh browser/resource gates; none is supplied by this packet.
