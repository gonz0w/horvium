# Offline dashboard resource-receipt validator

Date: 2026-10-12. Status: portable implementation awaiting maintainer review.
Owner: [issue 362](https://github.com/gonz0w/horvium/issues/362), under
[M2 gate 131](https://github.com/gonz0w/horvium/issues/131).
Authority: DASH-REQ-006 / DASH-PERF-001 and the adopted
[resource Test Design](../../design/dashboards/resource-qualification.md).
This is a development evidence contract, not a runtime schema or a native result.

## Scope and trust boundary

[The module](../../../tools/dashboard-resource-evidence.py) exports
`validate(candidate, runs)` for future reviewed runner wiring. It has no CLI,
service, subprocess or network activity. The current runner, native fixtures and
[historical validator](../../../tools/check-performance-evidence.py) are unchanged.
Historical manifests remain readable and bound to their original source. Passing
them does not supply current-dashboard coverage.

The caller must first verify the actual clean signed/DCO candidate and review,
resolved base and actual proposed PR title. It must independently capture the
source archive, pinned compiler, built management/helper/test artifacts, original
fixture expectations, base/runtime hashes and owned supervisor records. These
inputs must not be obtained by copying assertions from the transcript under test.
The module checks their structure and agreement; it does not perform Git signature
verification, rehash files it cannot access, authenticate flags, or attest to the
truth of developer-controlled evidence. `review_verified: true` records a caller
prerequisite; setting it cannot substitute for the existing signed review gate.

The caller retains failed/interrupted records and actual transcripts privately.
The validator raises `ValueError` on rejection and returns only release/mode/request/
expiry counts on success. It never returns native or dashboard acceptance. Offline
synthetic test transcripts are parser fixtures, not measurements or native execution.

## Candidate and independent run inputs

All new contract objects have exact keys; unknown fields require a reviewed change.
Hashes are lowercase SHA-256; source/base are full 40-character Git IDs. Counts and
nanosecond/byte quantities are unsigned integers, with booleans excluded. Numeric
CPU/legacy milliseconds must be finite and nonnegative. Duplicate JSON keys,
non-finite constants, malformed UTF-8 and transcripts over 32 MiB fail closed.

`candidate` contains `source_commit`, `base_commit`, `pr_title`, `go_version`
(`go1.x.y`), `compiler_sha256`, `archive_sha256`, `review_verified`, and `artifacts`.
The artifacts map has exactly `15.1` and `14.5`, each with SHA-256 values for
`management`, `helper`, and `fixture`. The caller verifies those artifacts and
compiler against the actual reviewed source and pinned tools before calling.

`runs` contains exactly one run per release, each with these inputs:

| Key | Meaning |
|---|---|
| `release` | Exactly `15.1` or `14.5` |
| `pins` | Independently checked hashes for `base`, `qemu`, `firmware_code`, `firmware_vars` |
| `fixtures` | Independently prepared original `datasets`, `owned_datasets`, `inventory_sha256`, `origin` for each of the four modes |
| `record` | Captured candidate, transcript SHA-256, provenance, before/after preservation observations and guest/host cleanup |
| `transcript` | Actual complete stdout bytes, including measurements, assertions and suite completion |
| `supervisor` | Independently read owned process result normalized to `exit_code: 0`, `group_cleaned: true`, `stopped_verified: true`, bound to `source_commit`, `release`, `transcript_sha256` |

The host `record` has `release`, `candidate`, `transcript_sha256`, `provenance`,
`preservation` and eight required true flags: `guest_assertions`, `host_teardown`,
`fixture_key_removed`, `image_removed`, `variables_removed`, `listener_removed`,
`qemu_exited`, `qualified`. Each preservation entry has `before` and `after`, both
matching the independent pin. These normalized fields do not replace or rewrite
existing runner/supervisor records; future wiring must retain originals and derive
the projections only after their actual verification and complete teardown. The
supervisor projection must bind the executed command/owned fixture to the source,
release and final transcript; copying success flags from an unrelated group fails.

## Transcript records

Exactly one `DASHBOARD_PROVENANCE` JSON record contains the candidate, release,
release artifact hashes, actual `kernel`, `userland`, `zfs`, `guest_go`, `host_os`,
`host_arch`, `qemu_version`, `arch`, `cpu_model`, `cpu_count`, `memory_mib`,
`accelerator`, `physical_bytes`. Guest kernel/userland must match the named release
including optional patch suffix. The adopted VM procedure remains amd64/qemu64,
two CPUs, 6144 MiB and TCG. Actual versions and physical bytes are recorded;
image labels alone are insufficient. Host provenance must agree with stdout.

Exactly four `DASHBOARD_RESOURCE` JSON records describe `synthetic-100`,
`synthetic-10000`, `native-small`, `projection-10000`. Each has `mode`, `datasets`,
`owned_datasets`, `inventory_sha256`, `origin`, `procedure`, `resources`, `phases`.
Native small has at least 100 datasets, 100 owned, and `native-original` origin;
other modes have the exact declared 100/10,000 counts, zero owned, and `generated`
origin. Every count/digest/origin must agree with the independent original fixture.
No names, GUID inventory dumps, bodies, cookies or credentials are recorded here.

The procedure requires ten clients, one session, 1,000 requests per phase, exactly
three completed replacements, zero retries/hidden API reads, 60-second warm-up
with observed `warmup_wall_ns` at least that long, small-mode 300-second idle,
30-second scheduled collection, default 100/maximum 500 table limits, 20 comparison
rows, the unchanged 32 MiB response ceiling, and predicate controller outside the
measured pair. Large idle is explicitly unmeasured (zero), not a shortened idle.

Resources retain sampled combined RSS, each process's `management_high_water_bytes`
and `helper_high_water_bytes`, their conservative sum, the maximum used for budget,
sample count, actual `max_sample_gap_ns`, and nominal 100,000,000 ns sampling.
No new maximum-gap budget is invented. Small idle preserves accumulated combined
`idle_user_ns`, `idle_system_ns`, actual `idle_wall_ns` at least 300 seconds, and
recomputed `idle_cpu_percent_one_core`. Small RSS <=128 MiB / CPU <1%; large RSS
<=256 MiB. The controller, browser, OS/ARC and file daemons remain excluded.

Each `initial`/`replacement` phase contains `samples` and `summary`. All 1,000 raw
samples have `index`, `case`, `outcome`, `status`, positive integer `duration_ns`,
complete `body_bytes` within the ceiling, and `predicate`. Indices are unique
0..999; case is index modulo five in the adopted driver order. HTTP 200 success
requires `original-inventory`, meaning the reviewed HTML predicate actually passed.
Only replacement Overview HTTP 409 permits `expiry` / `safe-expiry`. No failed,
skipped, unclassified or all-expiry case qualifies. The validator checks predicate
receipts; it does not revalidate response bodies, which are deliberately absent.
Future wiring must call the reviewed predicates before emitting these outcomes.

The summary contains `all` and each of the five cases, with `scheduled`, `success`,
`expiry`, `success_latency`, `expiry_latency`. Quantiles have integer `p50_ns`,
`p95_ns`, `p99_ns`; an empty expiry set has null latency. Counts and nearest-rank
quantiles are recomputed from every raw observation. Large initial p95 <250 ms is
required for every case and aggregate. Replacement expiry is reported separately,
never used to lower successful latency. Absence of a natural expiry timing race is
allowed; deterministic eviction coverage remains separate.

Exactly one `DASHBOARD_CONTROLS` JSON record retains 8,000 completed predicate
assertions per release, 100 independently observed owned native datasets and GUID
assertions, unchanged native boot state, separately reported boundary RSS/high-water/
sample count/gap, boundary outside normal budgets, guest cleanup, pair descendant
removal and socket removal. Future native assertions must emit actual observed
counts after successful completion, not these literal expectations at boot.

The unchanged legacy validator must also pass: both synthetic and both built-helper
records, model cap/exhaustion/recovery and native crash/stale/recovery/boot-state
markers. Their counts, resources and initial aggregate percentiles must agree with
new reports, retaining legacy microsecond truncation. The original boundary marker
must include matching raw RSS/high-water/count/gap observations. Both native parent
RUN/PASS markers must occur exactly once in order, followed by exactly one resource
suite PASS. Earlier discovery-suite output remains intact and cannot complete
the resource suite; any FAIL/SKIP in the complete transcript or
unknown/malformed dashboard record rejects coverage. A green VM exit alone fails.

## Behavioral validation and remaining gates

[Offline tests](../../../tools/tests/test_dashboard_resource_evidence.py) use two
explicitly synthetic transcripts and exercise missing/duplicate/unknown releases,
modes, cases/phases, legacy-only/boot-only/skipped runs, procedure/predicate/body
bounds, numeric/JSON types, raw identities, safe expiry, incorrect ranks, hidden
per-case budget failures, provenance mismatches, preserved controls and uncertain
cleanup. Test fixtures use a boundary pair above the normal budget to preserve its
separate scope; they are not observed FreeBSD performance values.

Required checks and conservative shared-tool portable review follow the existing
[local review gate](../../engineering/local-review.md). Native wiring must be a
subsequent reviewed slice. Only then run a clean assembled candidate sequentially
on both pinned disposable releases, retaining failures and verifying teardown.
Broader #131/#289/#290, native lifecycle #305/#139, browser/manual, continuity,
hardware/commercial and release gates remain open; runtime writes stay disabled.
Rollback removes the inactive module/tests/docs without changing runtime or old
receipts. No runtime dependency, Actions workflow, host service or operator storage
is added or used.
