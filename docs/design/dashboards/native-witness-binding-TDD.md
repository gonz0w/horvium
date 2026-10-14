# Test Design supplement: original native witness binding

Status: Review; proposed test-only qualification mechanism. Date: 2026-10-13.
Owner: Cam (`@gonz0w`), [issue 382](https://github.com/gonz0w/horvium/issues/382).
[PRD](PRD.md), [EDD](native-witness-binding-EDD.md),
[ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[requirement correction](native-invariance-correction.md),
[original resource procedure](resource-qualification.md).
DASH-REQ-002 / DASH-REQ-003 / DASH-REQ-006 / DASH-REQ-007 and their existing
DASH-DATA-001 / DASH-DRILL-001 / DASH-PERF-001 / DASH-SAFETY-001 gates remain.
Maintainer adoption of the complete packet authorizes implementation, not native
execution/qualification without the separately reviewed integrated fixture.

## Portable behavioral tests before implementation

| Test ID | Observable acceptance |
| --- | --- |
| DASH-WIT-001 / original ID | Capture whole explicit request ID from original actor; every response record matches. Wrong/duplicate/out-of-order ID, cross-management/helper lifetime and same-values/wrong-ID replay fail. Counter gaps do not invent missing witnesses. |
| DASH-WIT-002 / independent decode | Parse original framed bytes without product helper/management decoder or renderer math. Duplicate/unknown fields, zero/oversized/truncated header/body, noncanonical integers, invalid GUIDs/statuses/counts, wrong begin/end/EOF and depth/field/byte/record limit+1 fail before oversized allocation. uint64 maximum remains exact. |
| DASH-WIT-003 / custody and forward | Exclusive original creation, sealed descriptor/object/hash/size/index and exact forwarded bytes/EOF. Changed inode/content, symlink/adopted file, mutable earlier witness, partial delivery and ancillary extra/truncated descriptors fail. Rejected producer prefix is forwarded unchanged; observer failure never becomes a source-limit success. |
| DASH-WIT-004 / publication | Complete/delivered source is not published automatically. Real manager cancellation before publication, clone/model error, overwritten pending state and expired/unconfirmed ID fail lookup even with matching values. Confirmed current/retained IDs resolve only their immutable original. |
| DASH-WIT-005 / dynamic values | Actual native accounting changes select independently captured original values for that generation. Deliberately corrupt a numeric cell/bar/scale/link/generation: it still fails. No management numeric field or later collection enters the expected values. Synthetic values and projected datasets stay frozen. |
| DASH-WIT-006 / races and controls | Three overlapping replacements, retained latest/last-complete, partial publication, eviction between Overview reads and background generations. One coalesced metadata operation per new ID, no control during request-free idle, no per-request API, refresh, artificial pacing, retry or extra allowed expiry. Failed confirmation permanently denies qualification. |
| DASH-WIT-007 / limits | One active connector; immediate capacity rejection, 32 attempt cap, 256 MiB archive, 64 KiB relay/control bounds and trusted normal wire limits. No silent eviction/truncation or cap increase. Keep separate raised-cap synthetic model tests on their original direct path. |
| DASH-WIT-008 / accounting | Exact paired and conservative augmented RSS/CPU calculations; transient connector high-water maximum and cumulative CPU, not omitted or counted as simultaneous. Missing samples, unknown descendants, NaN/negative measurements, altered budgets, incomplete cadence/calibration or observer failure cannot qualify. |
| DASH-WIT-009 / evidence | Source/base/title/compiler/product/fixture/native-probe/original/forwarded/publication/sample references match independently reopened originals. Swapped release/mode/candidate, altered originals, missing/wrong witness and old coverage masquerading as native-witness evidence fail. |

Use the existing request plan, response ceiling, parser bounds, integer comparison
oracles, real authenticated console/manager and test-only generated source fixtures.
Test both successful changed generations and forged equal values with wrong lineage.
Do not write a test that merely regenerates the proposed record from the same
implementation or accepts management agreement as numerical authority.

## Native short gates before activation

Run both pinned disposable FreeBSD releases sequentially, with signed clean review
and original image/QEMU/firmware/compiler/artifact preflight. Use only the prepared
owned md/ZFS fixture and temporary guest storage. No product host installation.

| Test ID | Required actual native observations |
| --- | --- |
| DASH-WIT-NATIVE-001 / credentials | Actual base-header version/size/offset/UID/PID expectations match fixture observation. Original management/helper identities and process lifetimes match, with forged UID/PID payload and same-UID unrelated process rejected. Unknown ABI or descriptor origin is failure. |
| DASH-WIT-NATIVE-002 / descriptor handoff | Connector drops credentials before connect. After SCM_RIGHTS transfer, original helper still sees expected management UID; observer independently sees actual helper UID/PID. Descriptor count/CLOEXEC, reaped connector, EOF/half-close and deadlines are observed, including sender exit and forged/replaced endpoint. No root-connect fallback. |
| DASH-WIT-NATIVE-003 / originals | Actual source frames captured before successful delivery; product bytes unchanged. Root-owned tmpfs/file/index/object/hash custody, positive accounting changes, correct per-generation HTML plus deliberate table mutations. 100 actual owned dataset identities remain independently witnessed. |
| DASH-WIT-NATIVE-004 / native death | Capture original helper command/guard; kill actual helper, confirm subtree exit and stale old generation/witness. Only actual stale socket is removed. Register new helper incarnation, observe recovery with new request/witness, preserve both configuration bytes and token digest. Front proxy socket is not reported as helper death. |
| DASH-WIT-NATIVE-005 / cancellation and cleanup | Stop controller/observer/connector during capture, seal, forward and metadata confirmation; fill cap or tmpfs; close/replay ancillary channel. No accepted witness on uncertainty. Independent original descendants/descriptors/front+actual sockets/tmpfs/ZFS/md and host artifact cleanup pass or remain unqualified. |
| DASH-WIT-NATIVE-006 / overhead | Direct/interposed diagnostic timing and scheduler cadence, observer/connector RSS/high-water/CPU and tmpfs/controller occupancy reported separately, with unchanged product artifacts. No proof that added work necessarily preserves cadence or yields conservative latency. Resolve unexplained differences before activation. |

Development ABI probes use FreeBSD base headers/compiler, not a new runtime C
component. Archive actual source/compiler/header/binary hashes. Unsupported lab
ABI fails this qualification without narrowing the product hardware charter.
Each candidate is source-bound; failed/refused/incomplete attempts remain retained.
A corrected candidate needs new signed review and relevant new execution.

## Complete resource gate after integration review

All four original modes on 15.1 and 14.5: original 60-second warm-up, full 300-second
small-mode request-free idle, ten readers, 1,000 initial observations, 1,000 during
exactly three completed replacements, 200 per case/phase. Keep first-20 comparisons,
complete 100/500-row pages, independent exact ratios/scale/links, valid expiry and
all original native crash/stale/recovery and model/byte-boundary controls.

Require original product-pair <=128 MiB small / <=256 MiB large and idle CPU <1%
plus the EDD's same-budget augmented instrumentation checks. Large initial per-case
and aggregate p95 remain <250 ms; failed/expiry timing never improves a percentile.
Keep nominal 100 ms RSS sampling and original sample/gap/high-water evidence.
Record named supplemental metadata operations and every observed generation's
original witness reference. No metadata requests during idle or API-derived numeric
expectations. Missing lineage fails an observation; no workload retry or reduced
count to compensate. Parser work stays outside timing but its pacing costs remain
explicit. Calibration/short gates are not the complete measurement.

Strict new witness coverage and unchanged legacy validators independently reopen
actual original evidence for the same candidate on both releases. Require all
original source/controls and verified guest/host teardown. Existing schemas and
failed historical candidates retain their original boundaries; they cannot acquire
witness provenance retroactively. The earlier large latency failure remains open.

## Delivery and approval

After this packet is adopted, keep broker, decoder/registry, metadata binding,
short native controls, supplemental receipts/validator, runbook and compatibility
in one coherent implementation PR. Internal helpers use signed incremental commits,
not default additional PRs. Short native gates may validate the still-inactive
fixture, but the performance runner does not activate it before integrated review.
Then execute one reviewed candidate's complete original procedures on both releases.
Follow required check/lint/test/secrets and focused review; full review at milestone/
release. Stop owned fixtures and verify lab-stop/status. No paid Actions/automation.

Adoption/implementation merge does not close #382, wider dashboards, native safety,
manual accessibility, hardware or release acceptance. Runtime writes stay disabled.
