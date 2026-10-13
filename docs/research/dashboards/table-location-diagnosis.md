# Original table-check and maximum-page diagnosis

Status: Bounded research; no changed resource acceptance contract.
Date: 2026-10-13. Owner: Cam (`@gonz0w`).
Refs [issue 382](https://github.com/gonz0w/horvium/issues/382),
[Overview 289](https://github.com/gonz0w/horvium/issues/289) and
[Capacity 290](https://github.com/gonz0w/horvium/issues/290).
Authority: adopted [PRD](../../design/dashboards/PRD.md),
[EDD](../../design/dashboards/EDD.md), [Test Design](../../design/dashboards/TDD.md)
and [resource supplement](../../design/dashboards/resource-qualification.md),
DASH-REQ-006 / DASH-PERF-001. Preserve the
[earlier focused diagnosis](native-focused-diagnostics.md) and original failures.

## Question and procedure

Which exact table check fails after the original native-small warm-up/idle, and
where does maximum-page rendering spend CPU in the portable authenticated path?
Fixed controller-only diagnostics distinguish structure, headers, empty forms,
row count/shape/name/link/selection, cell style and individual fixed columns.
They retain no observed value, name, GUID, response body or session material.
Unknown diagnostic keys fall back to the generic table failure. The independent
original expectation and every rejection criterion remain unchanged. Failed
observations still cannot construct a successful resource receipt.

Run one separately scoped native-small diagnostic per signed clean reviewed
candidate, on disposable FreeBSD 15.1 with the existing QEMU pins and owned
supervision. Preserve 100 independently owned datasets, 60-second warm-up,
300-second idle, ten readers and 1,000 initial requests. Use an eight-minute Go
test and ten-minute owned command deadline. Keep the private scoped controller,
20-second research transfer connection window and provenance from the earlier
spike; canonical qualification orchestration remains unchanged. A failed run
stays failed, with missing complete-parent evidence explicit. No same-candidate
retry to obtain a pass. FreeBSD 14.5 and complete both-release resource execution
remain separate acceptance work after a justified correction.

Extend the existing portable authenticated-page benchmark to default 100 and
maximum 500 limits for both 100 and 10,000 generated datasets. Each fixture uses
ten concurrent readers and 1,000 complete handler responses per iteration.
Profile CPU/allocation separately from QEMU measurements, with exact source,
compiler, commands and original output hashes. Direct portable handler execution
excludes native helper, transport and TCG; it can identify local costs but cannot
explain native latency or establish the resource budget. No runtime profiling
route, dependency, configuration or service is added.

## Preservation and remaining gates

Independently reopen source/build/preflight/transcript/supervisor records and
verify owned host process/listener/key/image/variables disposal and original
base/QEMU/firmware preservation. Preserve all earlier manifests and primary
operator edits. Run required checks, signed clean local review and lab-stop/status.
Retain actual results and limitations in the adjacent source-bound manifest.
Do not refresh expectations from responses, relax predicates or increase budgets.
Broader dashboard, browser, native recovery, hardware and release gates remain
open; runtime writes stay disabled. Research becomes architecture only through
reviewed adoption, and a merged diagnostic is not resource acceptance.

## Native observations and correction boundary

Signed diagnostic source `d13e41ea2583677048118f4a151bc1f3dd9bc4b8`, base
`ea4efd6031e06cee7e6cfeda61ac055c342db29d`, passed clean local review before
execution. Actual FreeBSD kernel/userland were 15.1-RELEASE-p3, with Go 1.27.1;
QEMU remained qemu64/two CPUs/6144 MiB/TCG. The native-small fixture retained
115 total datasets, 100 independently owned, original 60-second warm-up and
300-second idle. All 1,000 initial HTTP-200 predicates passed (200 per case).
The replacement workload failed: 104 successes and 896 HTTP-200 failures.

| Replacement case | Success | First failed exact check |
| --- | ---: | --- |
| Overview | 22 | 178 Allocated cells |
| Pools/default | 21 | 179 Allocated cells |
| Pools/maximum | 19 | 181 Allocated cells |
| Datasets/default | 26 | 158 Used and 16 Available cells |
| Datasets/maximum | 16 | 171 Used and 13 Available cells |

The whole selected parent failed in 432.60 seconds; native-small failed in
427.12 seconds. Replacements reached the workload; native crash/recovery and
resource-budget publication did not. Projection was omitted, so the complete
parent was never eligible. No complete parent cleanup receipt or after-preflight
was reached; independently verified host discard does not supply either.
Original command/outer worker groups completed cleanly with exit 1, no escalation
or cleanup failure. Host QEMU/listener/key/image/variables removal and original
base/QEMU/firmware preservation were independently reopened and verified.

This identifies exact value mismatches, rather than a shape/header/link failure,
but only the first failed check in each response is known. It does not establish
which pool/dataset changed, whether expected native facts were invariant, or a
product accounting defect. Prior failed initial-phase observations stay unchanged;
this candidate's successful initial phase does not promote that history. Changing
the adopted constant-original oracle needs an evidenced, reviewed fixture contract
correction. It is an unresolved acceptance boundary, not permission to refresh
expectations from management responses or relax checks. The old native failure
and the new replacement failure remain open under #382.

## Portable page allocation change

The existing benchmark now covers both default and maximum limits. Its original
profile contains four 1,000-page fixtures, ten readers and one iteration each;
maximum pages contain 500 rows only for the 10,000-dataset fixture. The 100-row
fixture remains 100 rendered rows even with requested limit 500. Combined CPU
samples and allocation totals cover all four fixtures, not a maximum-only profile
or retained memory measurement. Runtime waiting/allocator activity dominates flat
CPU samples; template reflection/escaping, buffer growth and row-link construction
also appear. This does not identify the native/TCG latency cause.

Inspection found pool/dataset detail facts were constructed for every row despite
being rendered only for the selected row. The allocation-only correction builds
those facts only for a matching selection. Exact tables, comparisons, escaping,
links, generation/query validation and selected details remain unchanged. It adds
no cache, runtime interface, native collection change or higher limit. Existing
visible behavior already passed, so a failing behavioral assertion would invent
a behavior change: the meaningful evidence is the original allocation baseline,
plus exact full/HTMX selected and unselected 500-row regressions before and after.
A fresh source-bound benchmark records the effect; native execution above belongs
to the earlier diagnostic source and does not qualify this later optimization.


## Measured allocation result and review

The [source-bound manifest](table-location-diagnosis-manifest.json) retains actual
source/base/title, original native/build/preflight/transcript hashes, failure
counts, completed replacements, clean supervisors, profile/comparison hashes and
unexecuted FreeBSD cross-build. The optimization source is
`301db4d4abe1869ca12b4554eed12884b2408838`. Portable benchmark binaries used
Go 1.27.1, Darwin/arm64, Apple M5 Max, GOMAXPROCS 18 and CGO enabled; the native
candidate is cgo-free. Both comparison commands ran after QEMU and other review
work stopped, with the same fixture workload and no CPU profiling enabled.
Each B/op and allocs/op below covers **1,000 pages**, not one page or retained memory.

| Datasets / requested limit | Before bytes / 1,000 pages | After bytes / 1,000 pages | Before allocations | After allocations |
| --- | ---: | ---: | ---: | ---: |
| 100 / 100 | 705,610,928 | 689,010,960 | 10,452,686 | 9,953,198 |
| 100 / 500 | 705,554,344 | 688,938,992 | 10,453,512 | 9,953,852 |
| 10,000 / 100 | 802,970,616 | 783,991,968 | 11,047,616 | 10,349,759 |
| 10,000 / 500 | 3,028,209,736 | 2,933,517,512 | 48,054,298 | 44,566,090 |

For 10,000/500 this is 3.127% fewer allocated bytes and 7.259% fewer allocations.
Observed p95 was 8.275 ms before and 9.174 ms after; the single-cohort comparison
shows no latency improvement. The earlier CPU-profiled cohort's maximum-page p95
was 18.72 ms, with profiling overhead; it is a separate observation. These are
portable diagnostic costs, not native budget or statistical performance acceptance.

Both signed behavioral candidates passed required checks, 306 tooling tests and
fresh affected-package race review for cmd/horvium, internal/console and
internal/service; no eligible fuzz targets. Exact selected/unselected maximum-page
and full/HTMX regressions passed before and after the optimization. Final prose
followups rerun the required baseline and may reuse only those exact behavioral
inputs. Owned portable/native groups are stopped and verified; benchmark and
cross-built test binaries are removed. All 151 historical manifests and primary
operator edits retain their original bytes. Runtime writes remain disabled.

This PR combines the diagnostic, concrete allocation correction, regressions and
evidence. It ends before resource acceptance because the remaining native value
invariance/cause needs an evidenced fixture-contract decision and the large native
latency failure is still unresolved. #382, #289/#290/#131, manual accessibility
and startup/recovery gates stay open. Next unblocked work is to isolate original
native value changes by controlled resource ownership/generation, without using
management responses as new expectations, then review a justified correction and
execute both complete unchanged resource procedures. No historical result is upgraded.
