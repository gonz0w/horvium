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
