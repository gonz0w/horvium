# Per-mode current-dashboard resource measurement records

Date: 2026-10-12. Status: test-only component awaiting maintainer review.
Owner: [issue 368](https://github.com/gonz0w/horvium/issues/368), under the open
[Capacity acceptance outcome 290](https://github.com/gonz0w/horvium/issues/290),
coordinated by [M2 gate 131](https://github.com/gonz0w/horvium/issues/131) and
[Overview outcome 289](https://github.com/gonz0w/horvium/issues/289).
GitHub rejected another direct child of #131 at its sub-issue capacity; the existing
acceptance hierarchy preserves historical links without a duplicate phase gate.
Authority: DASH-REQ-006 / DASH-PERF-001 and the approved
[resource Test Design](../../design/dashboards/resource-qualification.md).
Prerequisite: [native fixture bindings](native-fixture-bindings.md); format:
[strict offline receipt contract](resource-receipt-validator.md).

## Actual per-mode observations

The [existing synthetic fixture](../../../cmd/horvium/performance_freebsd_test.go)
and [built-reader fixture](../../../cmd/horvium/reader_resources_freebsd_test.go)
retain the four declared modes and prepared original-inventory workload. They now
supply complete per-mode `DASHBOARD_RESOURCE` observations after normal measured
pair shutdown and applicable native crash/recovery/boot-state/socket checks. This
component does not execute those fixtures or publish native results.

Warm-up retains its full 60-second timer and records the actual monotonic elapsed
interval. Small idle retains the complete 300-second request-free pause between
two independently read paired CPU projections. Its observed wall envelope includes
both bounded process probes. Large modes explicitly leave idle unmeasured at zero;
no shortened observation, phase pacing or performance retry is introduced.

The controller invokes fixed `/bin/ps -p <two-owned-PIDs> -o pid= -o usertime=
-o systime=` under `LC_ALL=C`, with the existing two-second probe deadline. Both
owned positive, distinct PIDs must appear exactly once. Separate cumulative user
and system counters are parsed with strict bounded numeric/clock syntax. Counter
regression, incomplete rows, missing/duplicate/unexpected processes and arithmetic
overflow fail. Deltas divided by the actual monotonic wall envelope produce the
percentage of one core; no averaged `%cpu` field or fabricated zero component is
used. PID values and command output are withheld from published receipts.

FreeBSD's [14.5 keyword source](https://github.com/freebsd/freebsd-src/blob/releng/14.5/bin/ps/keyword.c)
and [15.1 keyword source](https://github.com/freebsd/freebsd-src/blob/releng/15.1/bin/ps/keyword.c)
expose both projections. Their [14.5 print implementation](https://github.com/freebsd/freebsd-src/blob/releng/14.5/bin/ps/print.c)
and [15.1 print implementation](https://github.com/freebsd/freebsd-src/blob/releng/15.1/bin/ps/print.c)
read separate rusage counters and round displayed times to hundredths of a second.
The nanosecond fields preserve conversion of these displayed rounded observations;
they do not imply nanosecond kernel precision. Each command reads its process rows
non-atomically, and the wall envelope bounds probe time rather than identifying an
exact simultaneous sample instant. No new uncertainty tolerance or CPU budget is
added. Native execution must still establish that the fields behave as expected
on the checksum-pinned images. Without `-S`, child CPU is excluded from these
paired-parent measurements, as before; controller, OS/ARC and file daemons are
also outside their scope.

RSS retains the existing 100 ms sampler, peak/count and actual maximum gap.
Waited getrusage high-water values now remain individually identified for management
and helper. Native crash/recovery preserves the conservative maximum of the original
and restarted helper before summing with management. Budget RSS is the maximum
of sampled combined RSS and this high-water sum. The source-model/byte-exhaustion
boundary pair stays separately reported outside normal budgets. Existing probe
capture, subtree exit, stale retention, recovery, boot-state and socket assertions
remain; recording per-mode peaks does not attest to their execution.

## Full mode format and failure retention

The [test-only builder](../../../cmd/horvium/dashboard_resource_helpers_test.go)
requires original mode/count/origin/digest, observed warm-up and CPU, positive
individual/sample RSS, sample count/gap and exactly three completed replacements.
It reconstructs both phase receipts from raw outcomes and compares them with the
supplied phase summaries, rejecting modified or stale serialization. Returned
phases own their arrays/maps. Every large initial case and aggregate must meet
the unchanged <250 ms p95 budget; small RSS <=128 MiB / CPU <1% of one core and
large RSS <=256 MiB remain unchanged. Missing measurements, arithmetic overflow,
non-finite or mismatched CPU, invented large idle and either sampled/high-water
budget excess prevent a passing record. No deficit can hide behind an aggregate.

The JSON uses the existing strict `validate_mode` format: mode and original fixture
metadata, complete procedure and resource fields, initial/replacement samples and
separate success/expiry summaries. Procedure constants describe the reviewed driver,
one authenticated session, fixed fixture config and sample interval. Phase counts
and replacement completion come from observed controller results. These assertions
are source-bound fixture inputs, not independently inferred from final stdout.
Future candidate/control/host binding must independently verify the actual source,
original fixture context, complete native suite and teardown before qualification.

Legacy `PERFORMANCE_RESULT` / `NATIVE_READER_RESULT` fields are derived from these
same resource observations and initial aggregate quantiles, preserving microsecond
truncation. Historical records and their original combined CPU projection remain
unchanged; they are not reinterpreted with this new source's separate-counter basis.
Passing modes no longer emit the partial `DASHBOARD_WORKLOAD` marker. Failed workloads
retain that sanitized original marker and raw outcomes. Failed per-mode resources
retain `DASHBOARD_RESOURCE_FAILURE` with observed measurement units and phases,
followed by failed test status; missing earlier setup observations are not invented.
Both failure markers deliberately fail the strict complete transcript contract.
No inventory names/GUID lists, response bodies, tokens, cookies or credentials are
emitted. No whole-candidate or cleanup acceptance is manufactured by a mode builder.

## Portable validation, execution gate and rollback

Behavioral tests cover CPU rounding/units and paired identities, bounded malformed
inputs and overflow, actual wall/delta/regression, detached four-mode serialization,
revalidated raw phases and replacement counts, small/large budgets at and beyond
boundaries, conservative sampled/high-water accounting, separate expiry and legacy
rounding. Go-generated records for all four modes are passed to the independent
Python `validate_mode`; those inputs are explicitly synthetic, not native metrics.

Required signed clean `make review` runs `make check lint test secrets` and affected
command-package race checks. The package has no discovered portable fuzz targets.
FreeBSD/amd64 cross-compilation verifies the changed fixtures compile; no VM,
operator storage or service is used. Owned teardown is verified after completion.
Historical manifests, validators, runner/guest execution guards, runtime, modules
and workflows remain unchanged.

The existing performance execution and result entry points remain gated. Next is
full independently bound candidate/compiler/artifact/release provenance, actual
native predicate/control completion and fixture/guest/host supervisor teardown,
then fresh sequential both-release execution on the reviewed assembled candidate.
Broader #131/#289/#290, lifecycle #305/#139 and historical startup uncertainty,
browser/manual, continuity, hardware/commercial and release gates remain open.
Runtime writes stay disabled. Rollback restores test helpers and the two native
fixture files without changing product behavior or historical receipt interpretation.

## Recorded portable source

[Receipt](resource-measurement-records-manifest.json): signed behavioral source
`31a9d55177b38b6116c3f1f6d77a2eb328326760`, base
`c895d987d1cc140fe52418b62b1bfed2bf113489`, actual title
`test(dashboards): publish per-mode resource observations`. Required checks passed
with 271 tooling tests, ordinary Go tests and fresh affected command-package race
checks; no portable fuzz targets were discovered for that package. Nine new
behavioral parents and independent four-mode Python contract checks passed.

The source-bound FreeBSD/amd64 cross-build compiled but was not executed and was
removed. Owned teardown is verified. All 142 historical manifests and execution
guards/validators/runtime/modules/workflows remain unchanged. A private source
preservation audit with a mismatched old/new code marker was corrected; it was
not native execution or acceptance. A signed documentation-only followup requires
a clean current-head review and may reuse only this completed exact-input deep
evidence. Candidate/control/teardown binding and both-release evidence remain.
