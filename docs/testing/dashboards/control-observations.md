# Original dashboard fixture and control observations

Date: 2026-10-12. Status: test-only implementation awaiting maintainer review.
Owner: [issue 378](https://github.com/gonz0w/horvium/issues/378), under
[Capacity gate 290](https://github.com/gonz0w/horvium/issues/290).
Authority: DASH-REQ-006 / DASH-PERF-001, adopted dashboard PRD/EDD/TDD and
[resource Test Design](../../design/dashboards/resource-qualification.md).

## Original inputs and actual validated counts

The [portable controller ledger](../../../cmd/horvium/dashboard_controls_helpers_test.go)
is wired into the existing guarded synthetic and native-reader resource parents.
Each parent captures its two original prepared fixtures before timed requests:
mode, dataset/owned counts, origin, canonical original-inventory digest and the
number of successful owned GUID comparisons. These values are detached copies,
never obtained by parsing the resource transcript or rereading management HTML.
Native preparation still uses the independent original collector and owned ZFS
witness; generated modes still use their original declared generators.

The [prepared adapter](../../../cmd/horvium/dashboard_adapter_helpers_test.go)
increments the native GUID comparison counter after each distinct owned witness
matches its original dataset GUID. A completed native-small fixture has 100 such
comparisons; other modes have zero. This describes fixture-controller validation,
not 100 separate native command invocations or a new storage membership API.

After workload completion and process resource observation, the ledger rechecks
original fixture identity and recomputes both phase receipts from the actual raw
samples through the existing resource validator. It records their combined
predicate count and completed replacements only on success. Each completed mode
has 2,000 validated outcomes and three replacements. Safe Overview expiry is a
validated safe-expiry predicate, counted separately from success in the original
phase summaries; it is not retried or treated as successful HTML. Two modes yield
4,000 predicates per parent, hence 8,000 only when both original parents complete.
Budgets, warm-up, idle, case distribution, original values and raw phase validation
remain unchanged. This second validation happens after measured work, outside the
paired PIDs; it adds controller work, not hidden HTTP requests or workload pacing.

Missing declarations, wrong-parent modes, changed originals, duplicates, failed
phases/resources or invalid mutation permanently invalidate that ledger. A later
successful input cannot repair the same failed observation. Returned records and
stored boundary/native controls own their values independently of caller mutation.
The two parents run their existing sequential subtests; no concurrent ledger or
process-global cross-parent accumulator is introduced.

## Existing boundary and native recovery assertions

The synthetic boundary observes four ordered steps, only after the existing
assertions succeed: accept the declared 100,000-object model; reject 100,001
objects while retaining stale original data; reject the declared padded-byte
fixture while retaining stale original data; recover to the original small model.
The object count is computed from the existing generator's fixed sections and
controlled dataset count, rather than copying the legacy transcript's constant.
Padding changes encoded bytes, not object cardinality; the controller does not
allocate a second padded/oversized dataset model merely to count it. This is the
declared synthetic fixture count, not a new native inventory measurement.

After that boundary pair stops, the observation retains actual sampled RSS, sum
of management/helper getrusage high-water RSS, sample count and maximum gap. All
must be positive and the sum cannot overflow. The boundary remains explicitly
outside the normal-size RSS budget. Original record/byte rejection, stale retention,
recovery and `PERFORMANCE_LIMITS` assertions/output remain intact.

Native-small records the original verified helper/guard/command identities privately
after actual kernel-event probe capture. It counts subtree execution exit only
after the existing checks on the captured command and timeout guard return. Those
checks accept a gone process or zombie execution state; they do not prove every
pair descendant has been removed/reaped. It then records actual same-generation
stale retention and a distinct complete recovered generation. Finally it compares
the original configuration/token-digest bytes with their post-control bytes before
recording unchanged boot state. Identities, generations, boot bytes and credentials
never enter the published observation. The original native crash and stale-socket
identity/recovery checks remain intact; no stronger cleanup claim follows.

## Parent callback and receipt boundary

A parent cleanup callback emits one `NATIVE_DASHBOARD_CONTROLS` JSON observation
only after its subtests and their cleanup callbacks finish successfully. Any
failed/skipped/incomplete parent or missing control suppresses success or fails
the parent. Each observation contains its parent name, two original mode summaries,
validated predicate/replacement/GUID counts and its own completed boundary or
native recovery counters. The fixture's existing scoped storage cleanup callback
still runs separately. No raw inventory, names, GUID list, PID or response body is
published. These are development observations supplied by the trusted controller;
portable event inputs cannot attest that native commands actually ran.

This is deliberately distinct from `DASHBOARD_CONTROLS`. It does not supply full
source/release/provenance, independent host fixture bindings, complete-suite PASS,
whole guest cleanup, all pair descendants/socket removal or host QEMU/listener/key/
image/variables teardown. Parent callbacks alone do not establish those facts.
The strict offline validator and all performance execution guards remain unchanged;
partial observations cannot qualify resources. A future reviewed runner must bind
original controller context, both parent completions and all independent cleanup
observations before constructing the complete strict receipt.

## Portable verification and remaining work

Seven portable behavioral parents exercise both complete parent contracts;
missing/duplicate/wrong-parent controls; original mutation and failed raw phases;
ordered boundary/native controls; changed stale/recovery/boot values; safe expiry;
immutable stored/returned copies; and original-generator object counts. Inputs are
synthetic and no native command is executed. The initial missing-ledger red log is
retained privately. An early negative test expected to reuse a failed ledger; it
was corrected to require permanent failure and independent test fixtures instead.

Required signed clean review runs `make check lint test secrets` and proportional
command-package race/fuzz checks. FreeBSD/amd64/v1 cross-compilation checks native
wiring only; no binary is executed and outputs are removed. Historical receipts
remain unchanged. Actual both-release native observations, original host fixture
binding, process/socket cleanup, transfer/launch and host teardown assembly and
fresh resource execution remain pending. Overview/Capacity, M2 lifecycle #305/#139,
manual accessibility, hardware/commercial and release gates stay open. Runtime
writes remain disabled. Rollback restores the three fixture/adapter files and
removes the two ledger/test files without changing product behavior or history.
