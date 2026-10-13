# Test Design supplement: current-dashboard resource qualification

Status: Approved Test Design; execution remains candidate-specific. Date: 2026-10-12.
Owner: Cam (`@gonz0w`), [issue 356](https://github.com/gonz0w/horvium/issues/356),
under [M2 gate 131](https://github.com/gonz0w/horvium/issues/131).
Delivery gates: [Overview 289](https://github.com/gonz0w/horvium/issues/289) and
[Capacity 290](https://github.com/gonz0w/horvium/issues/290).
Authority: adopted [PRD](PRD.md), [EDD](EDD.md), [Test Design](TDD.md),
[ADR 0010](../../adr/0010-progressive-dashboard-charts.md),
[M1 performance procedure](../../testing/M1/performance.md) and
[native reader procedure](../../testing/M1/native-resources.md).
This supplement refines DASH-REQ-006 / DASH-PERF-001 for stage A only. It changes no
product contract, budget or approval of stage B/C. Maintainer merge of [PR 357](https://github.com/gonz0w/horvium/pull/357), source
`82cce76f85fed1640388ea1c77c954171ba5f1e8`, adopts the future fixture/receipt
procedure; it does not produce measurement or acceptance.

## Observed gap and source boundaries

Merged presentation components are pool comparisons (PR 351), dataset Used
comparisons (PR 353) and same-generation Overview composition (PR 355). Their
[pool](../../testing/dashboards/pool-capacity-manifest.json),
[dataset](../../testing/dashboards/dataset-capacity-manifest.json) and
[Overview](../../testing/dashboards/overview-capacity-manifest.json) receipts qualify
bounded portable implementation only. Assembled review base is maintainer merge
`9a191915fe3c9b84fc5f03bf55591ca54c76cf86`, not a measured native candidate.
The workload inspection below uses that source.

`performanceSession.loadExpected` in
[the existing fixture](../../../cmd/horvium/performance_freebsd_test.go) performs
1,000 full HTML dataset requests at explicit limit 100, with ten concurrent readers.
[The native-reader fixture](../../../cmd/horvium/reader_resources_freebsd_test.go)
reuses that method. Their existing reports contain aggregate latency/RSS but do
not identify Overview, pool pages, maximum-limit pages or comparison predicates.
The [existing validator](../../../tools/check-performance-evidence.py) correctly
requires both source procedures, budgets, native crash and model-cap recovery.
It cannot establish current-dashboard coverage from historical report fields.

Overview now performs one additional bounded cached pool read. Dataset SVG and
page-local normalization run alongside the exact table. Existing numeric budgets
remain authoritative, but previous measurements cannot qualify this extra work.
No merged component or successful cross-build is fresh resource evidence.

## Candidate and fixture procedure

Implement the fixture/validator in bounded reviewed slices with failing behavioral
tests for workload and receipt contracts. The first [portable request-driver slice](../../testing/dashboards/resource-request-driver.md),
[issue 358](https://github.com/gonz0w/horvium/issues/358), remains inactive in the
native suite until original-inventory HTML predicates, strict candidate/release
receipt validation and native wiring have been reviewed. The
[HTML-predicate slice](../../testing/dashboards/resource-html-predicates.md),
[issue 360](https://github.com/gonz0w/horvium/issues/360), prepares detached
expectations and validates real portable console responses; it does not activate
the native suite or produce measurements. The
[offline receipt validator](../../testing/dashboards/resource-receipt-validator.md)
checks a distinct development evidence contract using synthetic transcript tests;
its runner wiring and actual both-release execution remain separate reviews. The
[prepared controller/phase adapter](../../testing/dashboards/workload-receipt-adapter.md)
binds original inputs and raw observations. The [native fixture bindings](../../testing/dashboards/native-fixture-bindings.md)
connect the four test-only modes while gating performance execution/qualification
until full independent host receipts are reviewed. The
[per-mode resource record slice](../../testing/dashboards/resource-measurement-records.md)
retains actual resource/procedure observations without opening that execution gate.
The [host candidate builder](../../testing/dashboards/host-candidate-build.md)
binds source/review/compiler and archive-isolated artifacts while leaving
transferred/executed artifact, control and guest/host teardown binding gated. The
[test-only guest artifact observer](../../testing/dashboards/guest-artifact-observations.md)
adds independently read artifact/compiler and running-kernel/userland environment facts
in a separate inactive preflight; trusted host comparison and actual launch,
control and teardown binding remain required. The
[inactive host preflight binding](../../testing/dashboards/host-preflight-binding.md)
rechecks the candidate's signed tree/archive/files and separately opened original
guest observation/supervisor; trusted runner transfer/launch, native fixture/control
and complete teardown assembly remain subsequent reviewed work. Leave
runtime, source model limits, collectors and file-service separation unchanged.
Then select one clean signed assembled candidate, actual PR title and pinned Go
compiler/artifacts for native execution. A failed candidate is retained with its
reason; fixes require a new reviewed source and new relevant execution, rather than
relabelling or silently rerunning to obtain a pass.

Use the existing local `performance` suite only after its workload/validator has
been extended and reviewed. Run both pinned FreeBSD/amd64 releases sequentially
on the Mac's disposable QEMU lab, using existing image/runtime/tool/firmware pins,
qemu64/two CPUs/6144 MiB/TCG and verified snapshot teardown. This supplement names no
new runner flag, suite, config or package and installs no host service or package.
Capture actual guest kernel/userland/ZFS/compiler and host/QEMU versions instead
of assuming the image label proves their values. Preserve base/runtime hashes.

Retain all four original separate paired-process measurement modes on each release:

| Mode | Input and claim boundary |
| --- | --- |
| Synthetic 100 | Existing deterministic typed helper model: one pool, four providers, 100 datasets; actual management executable and authenticated helper framing, not native command allocations |
| Synthetic 10,000 | Same generator/one pool/four providers with 10,000 dataset rows; not 10,000 real datasets or pools |
| Native small | Actual built helper and independently counted 100 test-owned ZFS datasets on disposable storage; preserve native command/crash and original boot-state observations |
| Projection 10,000 | Existing generated command-boundary projection through the built helper; zero owned datasets for this mode, preserving that explicit limitation |

Keep original generation replacement, native crash/stale-retention/recovery and
100,000 model-record/byte-exhaustion recovery gates. The raised transport-cap
boundary pair remains separately reported outside normal-size RSS budgets. Do
not delete legacy controls or replace native-reader execution with a typed helper.
Use existing fixture identities and declared datasets only; record no operator
inventory, credentials or private host configuration. A one-pool measurement
cannot qualify large pool cardinality. Existing portable bound tests retain their
own synthetic coverage; future larger hardware/source claims need separate evidence.

## Bounded workload and request predicates

For each measured mode, preserve 60-second warm-up, one authenticated session,
ten concurrent readers and 1,000 measured requests. Small modes retain the complete
300-second request-free idle observation with scheduled 30-second collection.
A second 1,000-request workload overlaps exactly three explicit completed
replacements. No shortened idle/warm-up or extra performance retry is authorized.

Distribute each 1,000-request phase evenly across these five cases: 200 per case.
Worker w owns global indices 100w through 100w+99; choose case i modulo 5. Dataset
page sequence uses j=floor(i/5), offset=(j*limit) modulo dataset count. Default
means the existing 100-row default; maximum means explicit 500. Pool pages use
offset zero. Requests use existing routes and exact typed query encoding only.

| Case | Request scope | Observable completion predicate |
| --- | --- | --- |
| Overview | Full `/`, current generation, no storage-page query options | Host/section metadata plus first at most 20 pools by name, exact comparison table, pool-specific freshness and same-generation original-GUID links |
| Pools/default | Full `/storage/pools`, name order, default limit | At most 20 comparisons, complete original pool table page, independent allocated/free accounting and correct coverage/links |
| Pools/maximum | Same pool route/order, limit 500 | Same comparison cap; table up to 500 rows, never comparison-driven truncation |
| Datasets/default | Full `/storage/datasets`, name order, default limit and calculated offset | At most 20 Used comparisons, complete page, exact page-local maximum including rows beyond comparisons and correct original query/GUID links |
| Datasets/maximum | Same dataset route/order, limit 500 and calculated offset | Same comparison cap and exact local scale; complete table up to 500 rows |

Derive expected names, GUIDs, counts and integer values from the declared fixture
and its original native observations, never browser Number conversions or string
prefix membership. Keep expectations outside the timed request loop where possible;
do not add a hidden API request per HTML request. Expected inventory values stay
constant during these replacement fixtures; generation/time are response-specific.
Assert generation agreement across each successful response's section, comparison,
table and resource links. HTTP 200 with an absent comparison, login/error body,
wrong row count/identity or truncated table is a failed request, not a timing sample.
For zero/no-data fixture cases use the documented explicit empty/unavailable forms;
never require 20 bars when fewer resources exist. Count visible measurement rows,
not SVG rectangles: pool rows have two independent bars and zero-size rows no SVG.

Timing includes request, complete bounded response read and close, using monotonic
elapsed duration. Verify a complete response within the unchanged product response
ceiling; reject ceiling+1 or read/transport failures instead of accepting a prefix.
Keep latencies for every scheduled request with case, phase and outcome. No response
bodies, session tokens/cookies or inventory dumps belong in the public receipt.
HTML predicate validation is fixture controller work outside the two measured PIDs;
record that limit, and include its effect on observed workload pacing.

The initial measurement requires 1,000 successful HTML responses and exactly 200
per case; no failed/skipped case is averaged into a pass. During the replacement
phase, Overview may legitimately return HTTP 409 with `generation_expired` if its
first generation is evicted between cached reads. Record that outcome separately,
require the safe expiry/reload form and absence of both successful projections,
and do not retry against a newer generation or count the denial as successful work.
Any other denial/transport/body failure fails the workload. Each replacement case
must also have successful completions; an all-denial case cannot qualify the view.
Report all 1,000 scheduled outcomes, successful and expiry counts per case and
successful-response latency separately. This contention observation is not a new
permitted expiry rate or availability guarantee. Portable deterministic eviction
regressions remain required; absence of a real timing race is not their replacement.

## Resource measurements and acceptance

Retain sampled combined RSS, sum of each owned process's getrusage high-water RSS,
conservative maximum of those two observations, sample count and maximum sample
gap. Preserve original 100 ms nominal sampling and observation limits. Idle CPU
uses combined accumulated user/system time divided by actual elapsed wall time,
as a percentage of one core. The controller, browser, OS/ARC, file daemons and
physical throughput are outside this paired-process measurement.

Small paired processes must remain <=128 MiB and idle CPU <1% of one core. Large
paired processes remain <=256 MiB and complete-response HTTP p95 <250 ms. Report
nearest-rank p50/p95/p99 for the initial 1,000 successes and each 200-success case;
require the large p95 budget for each case as well as the aggregate so fast pool
pages cannot hide expensive dataset/Overview work. For 200 samples the one-based
ranks are 100/190/198; for 1,000 they are 500/950/990. Retain raw sanitized durations
for verifier recomputation. Do not average percentiles across cases or releases.
Replacement successful-response timings and denial timings are separate observations;
no low denial latency may improve the initial qualified percentile. Keep failures
and missing measurements explicit, with no larger budget or shorter workload.

A source-bound receipt identifies every release/mode/case/phase, scheduled/success/
expiry totals, full procedure and observed counts, measured values/units and budget
results. Passing one release or only synthetic modes leaves resource acceptance
open. Complete source-model coverage and teardown are required before the bounded
resource gate can close; neither this review nor a future fixture-only PR closes it.

## Strict receipt validation and test-first pickup

The future validator must preserve the existing complete native/synthetic/crash/
limit-recovery requirements and add a distinct current-dashboard coverage contract.
Historical records remain unchanged, readable and source-bound. They cannot pass
the new coverage gate merely because they pass the legacy validator. Do not change
historical manifests, reinterpret their measurements or fabricate missing fields.

Before implementation, write failing offline tests for complete transcripts and:

- Missing/duplicate release, mode, case or phase; boot-only/skipped runs and legacy
  dataset-only logs presented as dashboard coverage.
- Wrong request distribution, warm-up/idle/clients/session/replacement counts,
  missing successful cases, unclassified denials and invented retries.
- Cross-generation/table/identity predicate failure, absent or oversized responses,
  changed page/comparison bounds and an expiry body that contains successful data.
- Non-finite/negative/string/bool measurements, wrong units, missing/duplicate raw
  durations, incorrect nearest-rank values or a single failed budget hidden by an
  aggregate. Successful/expiry counts must sum exactly to scheduled counts.
- Unmatched signed source/base/title/compiler/artifact/transcript hashes or a
  transcript from a different candidate; reused old measurements after new behavior.
- Incomplete sample/high-water evidence, missing preserved crash/cap controls,
  guest/host cleanup uncertainty and altered base/runtime artifacts.

Bind native stdout, actual assertion counts, complete-suite pass markers and
teardown independently. A green VM exit or parser-only fixture is not execution.
Fail closed on malformed/unknown coverage records; retain failure provenance.
Manifest/report formatting is a development evidence contract to be reviewed with
its implementation, not a new runtime schema. Shared verification/tool changes
require the existing conservative portable review plan; run no invented test that
merely restates an unexecuted proposal.

## Teardown, review and remaining acceptance

Run `make check lint test secrets` and signed clean `make review` for each bounded
change. Use affected-package checks for eligible implementation work and the existing
full fallback for shared tooling, with `make review-full` at milestones/releases.
Native execution belongs to the reviewed committed candidate and the existing owned
local lab. Run releases sequentially; wait for completion before `make lab-stop`,
then require `make lab-status` to report stopped/verified groups. Verify helper/
management/descendant/socket cleanup, disposable guest fixture cleanup, host listener/
QEMU exit, fixture key/image/variables removal and unchanged base/runtime artifacts.
Interruption stays unqualified with independently verified cleanup; do not enable
operator host services or use live storage/history as fixtures.

Approval of this supplement is followed by a bounded harness/validator PR, then
candidate-specific execution/receipt review. GitHub owns those live statuses and
native dependencies; this document owns the adopted procedure. Do not add duplicate
progress checklists, Actions, paid automation, runner services or synchronization.

Browser/Safari/Firefox/manual assistive-technology, zoom/contrast, browser lifecycle/
retained memory and file-daemon continuity remain separate gates. Manual M1
accessibility remains pending by maintainer choice. Stage B synthetic uPlot/CSP
research and stage C operator history/native-source contracts are outside this
supplement. Historical failed/skipped observations and the unexplained startup
deadline remain; full native lifecycle #305/#139 stays open, and writes stay disabled.
No hardware/commercial scale, whole-NAS performance or production support follows.
