# M2 review and implementation sequence

Date: 2026-10-08. Status: implementing; design approved; implementation accepted through PR 101; configured property client in review; Actions disabled.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
[PRD](PRD.md), [EDD](EDD.md), [Test Design](TDD.md).

M1 automated work and the qualification record are merged. M1 is Validating;
Safari/manual accessibility and explicit milestone acceptance remain pending.
Cam (`@gonz0w`) approved M2 PRD/EDD/Test Design and ADRs 0006/0007 through
maintainer merge of [PR 20](https://github.com/gonz0w/horvium/pull/20) on 2026-10-04,
source `753f601960529cd17596d0ffc628afe38d041c82`. This authorizes the test-first
slices under [the repository instructions](../../../AGENTS.md), without accepting
M1, M2 or a deployable release.

## Approved slices and acceptance boundaries

| Order | Reviewable outcome | Meaningful proof before acceptance |
|---|---|---|
| 1 | Ownership model, scope policy, immutable plans and fresh identity/drift checks; mutations still disabled | Denial/model/property-source tests; real FreeBSD provider/topology/status observations; GUID reuse/rename and use-check cases |
| 2 | Root-owned bounded job records, duplicate admission, singleton writer and recovery projections; no storage writes | Native file ownership/sync/rename/lock/failure tests; acknowledgement cuts, metadata restore/version/full-store and capacity tests |
| 3 | Allowlisted creation/property operations through the root boundary | Actual pool/filesystem GUID/topology/properties; policy/UID denials; feature profile/cross-release and namespace gates; no force/rollback fallback |
| 4 | Health/scan adapters and attributed scrub lifecycle | Both bundled output formats, real progress/completion/errors, stop/conflict/ambiguity and bounded polling |
| 5 | Redundant replacement and created-pool reimport/reconciliation | Actual leaf/new-device binding, adequate redundancy/size, fault/reboot/helper death, independent content and no replay |
| 6 | UI/CLI review, job/recovery and cancellation workflows | Shared contracts, auth/CSRF/Host/Origin denials, real browsers/manual accessibility, JS-disabled parity and focus preservation |
| 7 | Assembled M2 qualification and operational recovery runbook | Every Test Design ID, independent protocol continuity, measured resources, teardown and compatibility; explicit maintainer acceptance |

Start with failing behavioral tests and the narrowest working boundary. Expose
an operation only after its native/privilege/recovery gates pass; a later slice
must not quietly assume an earlier unqualified capability. Keep service installation,
share configuration, snapshots/replication, destructive deletion and packaging out
of these slices. Reconcile actual findings through reviewed design amendments.

Use one open implementation PR at a time where practical. Branch the next from
merged `main` so squash merges do not recreate the previous stacked-PR conflicts.
Research/acceptance artifacts retain their actual tested revisions and file hashes
if commit history is rebased. Agents leave PRs open for the maintainer.

## Current evidence and slice progress

- Primary source inspection and SHA-256 capture: complete for the documented M2 questions.
- M2 assembled product mutation/recovery/browser/resource qualification: not started.
- Physical MicroServer/commercial/controller/client qualification: not supplied.
- M2 PRD/EDD/Test Design and ADRs 0006/0007: approved in PR 20 as recorded above.
- Slice 1 first subset accepted in [PR 21](https://github.com/gonz0w/horvium/pull/21),
  source `f334094eae3d3b6d8ca740465ac09cb5be08ba20`: `internal/storage`
  scope/property review model with
  failing-first denial/impact/immutability/expiry/drift/bound tests. See
  [contracts and partial evidence](../../testing/M2/planning.md). Runtime remains
  read-only; no policy schema, native adapter, transport or submitted job exists.
- Slice 1 property subset accepted in [PR 22](https://github.com/gonz0w/horvium/pull/22),
  merged source `3e40d7d278bc95809907d4813a42b142a3cf8c53`: bounded native property parsing and retained received
  values; [native projection evidence](../../testing/M2/properties.md) passed on
  both pinned FreeBSD guests (run 37242881791, source
  `2c7724d503584aa053acaf409ef28e50f3180c40`). No write eligibility follows.
- Slice 1 identity subset accepted in [PR 23](https://github.com/gonz0w/horvium/pull/23),
  merged source `4011e3d9848b2432c7ab30d7d70981a3e0fdd88c`: binds full dataset types/GUIDs to independent property
  rows and review version membership; see [identity contracts](../../testing/M2/identities.md).
  Both pinned guests passed all 20 property/identity assertions and teardown
  (run 37244653409, source `2d4478c8a8e34b0ac40657d7330b4d73f0df6b21`).
  No runtime collector or eligibility proof follows.
- Slice 1 filesystem state subset accepted in [PR 24](https://github.com/gonz0w/horvium/pull/24),
  merged source `be69a4ee846749e2d845dda5ce0d1633c14eb794`: adds complete mount/readonly/crypto constraints
  and review drift; [state contracts](../../testing/M2/dataset-state.md).
  Both pinned guests passed all 28 property/identity/state assertions and teardown
  (run 37246040343, source `6dd07d2b5a52569c97f9b7524476348ecd7faeaf`).
  No complete eligibility or runtime adapter follows.
- Slice 1 pool observation subset accepted in [PR 25](https://github.com/gonz0w/horvium/pull/25),
  merged source `9f6cc07d3c19f84b656e9130283b21eaa7c7ce13`: binds permanent/import identities and policy
  metadata to review; [pool contracts](../../testing/M2/pool-state.md).
  Both pinned guests passed all 36 property/identity/state/pool assertions and teardown
  (run 37247751495, source `552f203501c87822c96e9b712beea962b0f43d5a`).
  No complete eligibility or runtime adapter follows.
- Slice 1 feature subset accepted in [PR 26](https://github.com/gonz0w/horvium/pull/26),
  merged source `cb39ddfe9dbcd6505f8f9d3f0b257bbb511ee162`: requires a complete GUID-bound native feature table and
  immutable drift review; [feature contracts](../../testing/M2/features.md).
  Exact common profile bytes are checked without a product file reader or creation
  authority. Both pinned guests passed all 42 property/identity/state/pool/feature
  assertions and teardown (run 37249224834, source
  `f3d7c63918fc5a6ef2d5c3ec1078f413f3179aae`). Cross-release transfer/import remains open.
- Slice 1 topology subset requires complete paired native GUID/path status,
  homogeneous mirror/RAIDZ2 refusal checks and immutable membership/health drift;
  [topology contracts](../../testing/M2/topology.md). Both pinned guests passed all
  51 assertions and teardown (run 37252047398, source
  `5cfcfbbf357e93af3155f1e0d411a3f6df2b00c6`);
  [PR 27](https://github.com/gonz0w/horvium/pull/27) accepted this subset, merged source
  `7032ffba296ab6c7c2b807d52601866051ffebbb`. Initial native
  scan-omission failure and failing-first repair are retained in the evidence record.
  Full scan/JSON status and provider/system/device-use proofs remain open.
- Slice 1 device fixture prerequisite: [virtual disk identities/geometry](../../testing/M2/devices.md)
  under the approved Test Design. Source `069d38f98bcc39e94861b22bf7db40a4fac70969` passed all
  eight repaired native assertions and complete backing/base/runtime/group
  preservation/teardown on both targets through the Mac supervisor. A further
  actual controller-SIGKILL fixture stopped its VM/listener and removed all owned
  disks/image/key/variables while remaining unqualified. PR 28 accepted this fixture
  at merged source `27dafaba7a5d04a4896f27776ef79f4b1f4c2bbc`. Original hosted failures/unstarted jobs, the first Mac deadline
  failure and the first 14.5 root-control failure are retained. No product selector,
  complete device-use proof or creation authority follows from fixture qualification.
- Slice 1 subset accepted in [PR 31](https://github.com/gonz0w/horvium/pull/31),
  merged source `ef60c33489a2ddd30c99e2a8e865a76272473f2b`: [native disk correlation](../../testing/M2/disk-correlation.md)
  binds bounded diskinfo serial/geometry to a complete GEOM capture, refusing
  ambiguous identity and non-direct providers. Source `031dbe715c0c0d6d94b92c19689f4fd78123c23b`
  passed all twelve assertions and complete backing/base/runtime/group teardown
  on both targets through the Mac runner. See the [native manifest](../../testing/M2/disk-correlation-manifest.json).
  Runtime/authority wiring stays absent.
- Slice 1 subset accepted in [PR 32](https://github.com/gonz0w/horvium/pull/32),
  merged source `f53688f29d6ba85e47b7062bbada73a857f02378`: [disk attachments](../../testing/M2/disk-attachments.md)
  recognizes exact terminal DEV and whole-disk disk-ident LABEL correspondence,
  refuses PART/unknown/contradictory attachments and adds disposable GPT denial
  controls. Source `0c9882f45c3abc1a81c693af01720ece9e50a9e6` passed all seventeen
  assertions, both executed native tests and complete preservation/teardown on
  both targets. See the [native manifest](../../testing/M2/disk-attachments-manifest.json).
  No complete use proof, runtime collector or product partitioning follows.
- Slice 1 subset accepted in [PR 33](https://github.com/gonz0w/horvium/pull/33),
  merged source `bbf7fe22ea0d81cbd459171f633e99c3096f7ee0`: [system device use](../../testing/M2/system-device-use.md)
  correlates full GEOM, mount/swap and mounted-pool GUID/status captures to
  positive root/mount/swap backing facts. Disposable UFS and active swap tests
  extend the suite to twenty-three assertions and three executed native tests;
  source `3f97271f0a785d45baeb62311e8681e56d55b6a9` passed all twenty-three
  assertions, three executed native tests and complete preservation/teardown on
  both targets. See the [native manifest](../../testing/M2/system-device-use-manifest.json).
  No unused-device or write authority follows.
- Slice 1 subset accepted in [PR 34](https://github.com/gonz0w/horvium/pull/34),
  merged source `50d6399b58f201ea26ec007879fe56792993c312`: [imported-pool use](../../testing/M2/imported-pool-use.md)
  correlates the complete all-imported-pool catalogue and independent GUID/status
  captures to positive disk membership, including pools without mounts. Six new
  assertions extend the devices suite to twenty-nine assertions and four executed
  native tests. Signed source `5b0a94128aa081fbb3f5d8a79ed8ab560d1f4a09`
  passed all twenty-nine assertions, four executed tests and full preservation/
  teardown on both targets; see the [native manifest](../../testing/M2/imported-pool-use-manifest.json).
  Exported pools, complete signature absence and complete use clearance remain open.
- Slice 1 subset accepted in [PR 35](https://github.com/gonz0w/horvium/pull/35),
  merged source `78a93f2c82620d3185311725a8afd6fe956d5cf7`: [filesystem signatures](../../testing/M2/filesystem-signatures.md)
  binds successful base `fstyp -u` UFS/ZFS scalar captures to independently
  correlated direct disks. Seven new assertions extend the existing four native
  tests to thirty-six assertions. Signed source
  `9a363b053d9ba12d1a23da3465c66f438f72b8d9` passed all thirty-six assertions,
  four executed tests and complete preservation/teardown on both targets; see the
  [native manifest](../../testing/M2/filesystem-signatures-manifest.json).
  No blank-disk clearance or pool identity/label integrity follows from this
  positive probe.
- Slice 1 subset accepted in [PR 36](https://github.com/gonz0w/horvium/pull/36): [combined device-use review](../../testing/M2/device-use-review.md)
  joins selected independent disk correspondence, full mount/swap/imported-pool
  backing and optional positive signature facts into a private immutable baseline.
  Fifteen added assertions extend the existing four tests to fifty-one cases. Signed
  source `9dd27d946cb726da1b0a5215b6392b1524aaff76` passed all fifty-one
  assertions, four executed tests and complete preservation/teardown on both
  targets; see the [native manifest](../../testing/M2/device-use-review-manifest.json).
  PR 36 accepted the combined device-use subset. Missing signature evidence remains unavailable and
  unchanged positive facts never grant launch or unused-device authority.
- Slice 1 subset accepted in [PR 37](https://github.com/gonz0w/horvium/pull/37): [positive pool-label identity](../../testing/M2/pool-labels.md)
  binds bounded default zdb -l configurations and exported/native-state drift to
  independently observed disks. Signed source `be438496ab6316433af60d57d13aefc2cbfb6a86` passed all
  fifty-nine assertions, four native tests and complete preservation/teardown on
  both targets; see the [native manifest](../../testing/M2/pool-labels-manifest.json).
  The initial utility-path and native-format failures retain verified teardown.
  PR 37 accepted this positive subset at merged source `d52fa620ab76d5787d782c1ad9de6b4b415bee20`; full device eligibility remains pending.
- Slice 1 subset accepted in [PR 38](https://github.com/gonz0w/horvium/pull/38): [disk access](../../testing/M2/disk-access.md) requires
  identity-bound provider/consumer aggregate counts, access drift and disposable
  direct/alias-held-descriptor tests. Signed source `53b74a8844f5f0bce42b08ae044929aacdfaa92f`
  passed all sixty-eight assertions, four tests and complete preservation/teardown
  on both targets; see the [native manifest](../../testing/M2/disk-access-manifest.json).
  PR 38 accepted this access subset at merged source `dea2efefc3c9b9a50053bc4794a5a21685ad5132`; full device eligibility remains pending.
- Slice 1 remains incomplete: assembled native eligibility/provider proofs, full
  status/scan interpretation, complete-impact collection and device-use drift gates
  have not run. No later slice may treat
  model input booleans or old M1 inventory as qualified native write authority.

The approval adopts scope and gates, not passing native behavior or a NAS release.
Record actual slice results here and in the compatibility matrix; skipped fixtures
are never acceptance evidence. Full M1 acceptance remains a separate pending decision.

## Original worker uncertainty propagation

PR 86 accepted original-claim applying metadata acknowledgement at merged source
`fd817121c375f02d9c02700a7cd66a001c99787c`. Its native evidence remains bound to
the executed source in its [manifest](../../testing/M2/property-claim-applying-manifest.json).
The next focused repair connects execution publication uncertainty back to the
original coordinator. A worker previously stopped itself after a lost/foreign
acknowledgement while leaving new coordinator submissions possible. The
[handoff contract](../../testing/M2/property-handoff.md#worker-publication-uncertainty)
now denies subsequent admissions and issued-worker launches after such uncertainty.
This portable repair supplies no complete root backend, native serialization or
recovery clearance; runtime writes stay disabled. See the
[qualification record](../../testing/M2/property-handoff-uncertainty-manifest.json).
Signed repair source `3bb8e871fb3324d894d04c56e0ea95600f325703` passed the new
20-case regression and retained handoff tests, all mandatory baseline checks,
9 affected race packages and 38 bounded fuzz targets. FreeBSD compilation and
complete teardown passed; native fixtures were unchanged and not rerun. A repeated
final-baseline Git fixture cleanup race was then traced to detached automatic
maintenance. The accompanying fixture-only fix and failing-first child-process
regression passed fresh full portable qualification at signed source
`cdc6fd34bbf41a3a4885abd44e04047a14b01d9e`: 125 tooling tests, all 11 race
packages and 44 bounded fuzz targets, with verified clean teardown. The combined
change remains pending maintainer review and grants no native write authority.

## Independent slice 2 prerequisite

The approved durable-state boundary can be implemented and qualified while
slice 1's device/impact/status gates remain open; this work consumes none of
those unqualified gates and cannot admit or launch storage jobs. The
[state-directory lease](../../testing/M2/state-directory.md) opens only an
existing private root-provisioned local UFS/ZFS directory, establishes a stable
single-writer lock and synchronizes its empty lock file and parent directory.
Thirty-five native assertions, separate process death/reacquisition and full
preservation/teardown passed on both releases at signed functional/suite source
`123044324e75f6df27d43eee085c23435ddbcce1`; see the
[native manifest](../../testing/M2/state-directory-manifest.json). Maintainer
review accepted in PR 39 at merged source `b84e0572d389078ec0b8514ff4d7919b56b96b7f`; default-ACL disposable fixtures are the qualified scope.
Strict records, sync/rename/failure/power-cut tests, durable admission, reservation,
recovery, restored metadata, helper negotiation and UI/CLI work remain ahead.

## Strict ownership record prerequisite

The [ownership record codec](../../testing/M2/ownership-records.md) adds bounded
versioned JSON and damage checksums to the existing policy model, without file
operations, implicit enrollment or native write authority. Failing-first tests
cover exact/duplicate/missing/null fields, versions/types, scope/parent identity,
canonical bytes, uint64 precision, capacity refusal and immutable round trips.
It adds the twenty-first bounded portable fuzz target. Signed source
`e9f685c2f4460b7b4545bfa761c9089a2743439a` passed five behavioral tests, fifty-seven
subtests, complete local review and all twenty-one fuzz targets; see the
[manifest](../../testing/M2/ownership-records-manifest.json). PR 40 accepted the codec
at merged source `52a583b4a83ab44fc51fab85f36e48efc18601d4`; prior native manifests retain their original sources. Atomic
publication, job schemas, durable admission and all recovery/native gates remain
ahead. This independent prerequisite consumes no unqualified slice 1 gate.

## Native ownership read prerequisite

The [ownership reader](../../testing/M2/ownership-read.md) connects the accepted
codec to bounded private file reads through the held directory. Its twenty-eight
assertions extend the state suite to sixty-three lease/read cases, two native
tests, six unit tests and fifty-seven codec subtests. Both native targets passed
at signed source `94cf66e9c3a34184c76b84858cb7ef998e4bd2dc`, including complete
preservation and teardown; see the [manifest](../../testing/M2/ownership-read-manifest.json).
PR 41 accepted the reader at merged source `a6406670cb7804a1d15582a615114fefdbeac336`. It performs no file publication or runtime enrollment and
consumes no unqualified slice 1 observation. ACL provisioning, full store budgets,
atomic publication, durable jobs and recovery remain ahead.

## State ACL validation prerequisite

The [ACL boundary](../../testing/M2/state-acl.md) strengthens actual opened-object
validation before lock creation or record reads. It adds forty-five assertions
using owned memory-backed UFS fixtures with POSIX and NFSv4 ACLs, plus portable
policy tests. The state suite requires 108 assertions, ten parent tests and 57
codec subtests. Both targets passed at signed source
`33f4d91bc57adb71f27ce80bbe0552fb1e7685d2`, with complete teardown; see the
[manifest](../../testing/M2/state-acl-manifest.json). Initial native baseline
refusal and a deliberately stopped diagnostic source retain exact evidence.
PR 42 accepted this subset at merged source `73a47a04578cd14f8f8d1ef2a978b17ff21fdd8e`. No ACL modification
exists in product code; publication, runtime provisioning, full budgets and
recovery remain ahead.

## Ownership publication prerequisite

The [private publisher](../../testing/M2/ownership-publication.md) extends the
held lease and strict codec with same-directory exclusive temporary creation,
file sync/close, descriptor/name revalidation, rename and directory sync. Exact
previous revisions serialize cooperating updates. Failed final sync or cleanup
blocks further publication on that lease and requires reconciliation; reads stay
available. No runtime enrollment, storage jobs, total-store budgets or recovery
controller consumes this prerequisite yet. Both targets passed all 161 assertions,
fourteen parent tests and 57 codec subtests at signed functional/suite source
`eabf4691fc55b825db74ba73501d758126fda4e7`, with complete teardown; see the
[manifest](../../testing/M2/ownership-publication-manifest.json). The initial
large-allocation ENOSPC fixture failure retains its exact unqualified source and
teardown; tiny allocations plus zero-free-space verification repaired that fixture.
PR 43 accepted publication at merged source `d5c3c987f6363923a839904e87f080026d964c60`;
complete M2 state, recovery and mutation gates remain open.


## Bounded ownership inspection prerequisite

The [catalog contract](../../testing/M2/ownership-catalog.md) adds a bounded complete
ownership-only namespace observation through the held lease, with sorted summaries,
strict record validation, orphan/unknown refusal, byte/count limits and final drift
checks. It consumes no unqualified slice 1 authority and performs no publication,
repair, orphan deletion, durable admission or storage work. Its expanded state
suite requires 202 assertions, seventeen parent tests and 57 codec subtests on
both releases, with complete teardown. Both targets passed all 202 at signed source `5ea125fdc815fd7c6c54aba36824e50ea876b9f6`
with full teardown; see the [manifest](../../testing/M2/ownership-catalog-manifest.json).
PR 46 accepted the catalog at merged source `e4ae4b3cdfe43c88bdb233afe992d82a2d3d83ca`;
its original inspection evidence does not implement full-store admission/recovery.


## Ownership publication capacity prerequisite

The [capacity contract](../../testing/M2/ownership-capacity.md) connects complete
catalog validation to ownership publication, reserving all old published bytes
plus the full new temporary before creation and rechecking before rename. Unknown,
unsafe, damaged/newer records and orphan temporaries deny writes; only the actual
owned temporary can be excluded/cleaned. Cooperating publications serialize at the
last slot; replacement keeps its slot but reserves both old and new bytes. Fixed
64 MiB logical-byte/100,000-record ceilings are independent of hardware.
Require all 234 state assertions, nineteen parents and 57 codec subtests on both
releases, with complete preservation/teardown. Both passed at signed source
`bf80cb9f2e1c196eae330ae0d681d86e0c466033`; see the [manifest](../../testing/M2/ownership-capacity-manifest.json).
PR 47 accepted capacity at merged source `807420918f532ecbd31fa24f1d6b64184d61c88d`.
Runtime enrollment, mixed job-store allocation, durable admission/recovery and
populated scale/resource qualification remain ahead.


## Job lifecycle metadata prerequisite

The [job lifecycle contract](../../testing/M2/job-records.md) adds the existing
property intent subset, strict bounded versioned/checksummed metadata, complete
transition history and restart projections that never replay queued/applying/
observing work. Success/failure require an observation step; only queued work can
cancel; reconciliation stays nonterminal and cannot resolve without later gates.
This is in-memory metadata only, not complete durable native intent, admission,
job-file storage or recovery. The ownership catalog continues to refuse job files.
Signed source `a3791513f6423fe5860064e93f89399c21dc1445` passed seven tests/119 schema
subtests, baseline and focused storage race/all 15 fuzz checks; see the
[manifest](../../testing/M2/job-records-manifest.json). PR 48 accepted this at merged
source `09b00a646d67702c5bd9b5ef1636c7f74e9e50ba`; no changed filesystem/native fixture or support claim follows.


## Native job reader and successor prerequisite

The [reader/successor contract](../../testing/M2/job-read.md) adds root-only canonical
job-ID-selected native reads using shared ownership descriptor validation and a
pure append-only successor guard. Identity, intent and historical events cannot
change. The expanded state gate requires 275 assertions, twenty-eight parents and
176 codec/schema subtests. Both releases passed with full teardown at signed source `5241ae7f0bccde0423613af1dfcde9aed951ac34`; see the
[manifest](../../testing/M2/job-read-manifest.json). Accepted in PR 49. Job-file
publication, complete durable
intent, mixed publication budgets, admission/deduplication and actual recovery remain
open. Ownership publication still refuses all job files; no runtime caller exists.


## Mixed state census prerequisite

The [mixed catalog](../../testing/M2/state-catalog.md) validates the complete
ownership/job namespace through the held root lease, counts both kinds against
64 MiB, and refuses excess 64 nonterminal/1,000 terminal job records and conflicting
submission IDs. Require 336 state assertions, thirty-two parents and 176 codec/schema
subtests on both releases with complete teardown. Both passed at signed source
`b79ff0f03a85c719882b93e8080f562704cc6bac`; see the [manifest](../../testing/M2/state-catalog-manifest.json).
Accepted in PR 50. The next publication extension addresses mixed reservation;
complete intent, durable admission and recovery remain later gates.


## Durable job metadata publication prerequisite

The [job publisher](../../testing/M2/job-publication.md) reads exact durable
predecessors, appends immutable lifecycle history and uses the shared native
sync/rename pipeline. Both publishers require a complete mixed census and reserve
old plus new bytes and job class slots before temp creation and before rename.
Require 432 state assertions, thirty-five parents and 176 schema subtests on both
releases with full teardown. Both passed at signed source `dc789c4e3672f61e670e6c4352e1eae89b3fb13a`;
see the [manifest](../../testing/M2/job-publication-manifest.json). Accepted in PR 51
at merged source `bcc5086648b74eacefad350db8c3bebd56d2db23`. This metadata-only
boundary admits/executes no job; complete intent, admission/reservations, retention
and actual recovery/acknowledgement cuts remain later gates.


## Reviewed property intent capture prerequisite

The [capture/codec](../../testing/M2/property-intent.md) retains the complete current
property review and full policy binding after fresh plan/session/expiry validation.
It captures original values/sources/received values, affected and protected context,
version/state identities, pool/import policy, features and topology. Nanosecond UTC
metadata and a plan-scoped session digest retain binding without raw credentials.
Strict decoding checks internal consistency and canonical damage checksums; it
cannot restore a plan or provide native/root authority. Existing job v1/native
publication behavior remains unchanged. Seven parents/769 subtests, storage race
and all 17 storage fuzz targets passed at signed source `e66d59eec2e05137a11d8533b327206c3682e423`;
see the [manifest](../../testing/M2/property-intent-manifest.json). Accepted in PR 52 at merged source `77d48453cfedd4ec4dfaa1da62dbcfa75e0674e9`.
Job embedding/complete intent, admission/reservations, attribution/recovery and
native eligibility assembly remain later gates; runtime stays read-only.


## Property job embedding prerequisite

The [property job contract](../../testing/M2/property-jobs.md) binds the accepted
review baseline to matching job selection/history in one immutable bounded record.
Initial lifecycle time must follow capture and precede expiry; successor validation
preserves all review rows, policy/session bindings and prior events. A distinct
record kind leaves accepted ownership/job readers and publishers unchanged and
fail-closed. Portable qualification passed at signed source `ddec32450bfc533eca98c2e39d6a68ce09951404`:
six parents/859 subtests, required baseline, storage race/all 18 fuzz targets and
FreeBSD cross-compilation; see the [manifest](../../testing/M2/property-jobs-manifest.json).
Accepted in PR 53 at merged source `417afe1c965cd13ccde53934babaf67875f07635`. Native persistence, admission,
reservations, eligibility assembly, executor/recovery and M2 acceptance remain open.


## Native property-job store prerequisite

The [store contract](../../testing/M2/property-job-storage.md) retains complete
baseline/lifecycle bytes through typed root-only reads and publication using the
existing descriptor-relative sync/rename transaction. Mixed census and all three
publishers count both job formats against shared bytes/classes/submission IDs.
Typed predecessor checks refuse conversion, rewrites and stale history. The state
gate requires 555 assertions, fifty parents and 1,804 codec subtests on both
releases with full owned teardown. Both targets passed at signed source
 `5a0e67cd7f469cf398b1cd831284d8a44a3b67ce`
with full teardown; see the [manifest](../../testing/M2/property-job-storage-manifest.json).
Accepted in PR 54 at merged source `5d0fb0314de157f0ace3755e862480dea02c1c4f`. Runtime admission,
eligibility/executable attribution, reservations/recovery/cuts and M2 acceptance
remain open.


## Job reservation metadata prerequisite

The [reservation contract](../../testing/M2/job-reservations.md) strictly validates
the supplied complete job-record set, retains full property baselines, indexes
unique submission/plan identities and projects all whole-pool holders. Matching
property retries return the existing current record; conflicting meaning, reused
plans, pool holds and exhaustion deny new metadata. Terminal labels cannot release
holds. Concurrent checks do not atomically claim/admit work. Portable qualification passed. Full native namespace/provenance, authorization/eligibility assembly,
durable admission, executor/recovery/release and M2 acceptance remain open.

Reservation qualification: signed source `eec1ad7c34485bc2b548f8a1c88f68cdaaaae647`, six parents/thirty
subtests, required baseline, storage race/all nineteen fuzz targets and FreeBSD
cross-compilation; see the [manifest](../../testing/M2/job-reservations-manifest.json). Accepted in PR 55; no native admission/release claim follows.


## Native job reservation inspection prerequisite

The [native inspector](../../testing/M2/native-job-reservations.md) binds the
complete immutable job view to the held root lease's independently enumerated,
strictly validated ownership/job namespace. The sealed streaming builder retains
complete property baselines without another whole encoded-record set. Candidate
capacity includes ownership bytes; final ordinary object/namespace drift or any
unsafe record discards the entire view. Root/ACL guards, orphan/uncertainty
preservation and conservative terminal holds remain. Require 622 state assertions,
58 parents and 1,837 codec/model children on both native targets with full
teardown. Both targets passed at signed source
`cca3a07eed591ed3bab757fc530eada0a40686be` with full teardown. No runtime
admission/recovery/release or native effect is authorized by the result;
caller/policy/eligibility/session/clock and executor gates remain open.

Native reservation qualification: `cca3a07eed591ed3bab757fc530eada0a40686be`; see
the [manifest](../../testing/M2/native-job-reservations-manifest.json). Accepted
in PR 56; no runtime admission/release follows.


## Reserved property-submission publication prerequisite

The [contract/procedure](../../testing/M2/property-submission.md) combines strict complete retained
plan/submission/pool checks and initial queued property publication under one
root lease/mutex. Repeat full baseline/capacity checks before staging and rename;
new success follows final directory sync, while retries return existing current
metadata without rewriting or a fresh durability acknowledgement. Uncertain
publication denies even duplicate returns. Terminal holds remain; runtime native
admission, caller/policy/eligibility/session/clock, executor/recovery/release and
M2 acceptance stay open. Require 710 state assertions, 60 parents and 1,837
codec/model children on both releases with full owned teardown. Both native
targets passed at signed source
`1dc80019f5d68430ad49845ecdb666192922806f` with full teardown; accepted in PR 57 at merged source
`d79a55779f713e1209a930343fdb482d403c85ef`. Shared pipeline/evidence/supervisor changes required one
complete portable review; startup failure before worker creation must remain a
failure and clean prepared owned fixtures without inventing/signalling a PID.

Publication validates its exact renamed inode/bytes before final directory
sync. Native concurrency uses 32 callers per scenario. UFS softupdates publication
and retry calls are refused; supported metadata backing is ZFS or UFS without
soft updates. Twelve exact native backing/teardown controls and ten actual
softupdates refusal/read/preservation controls are required. Root filesystem and
metadata fixture backing are recorded separately. Strict census timestamps remain;
no product mount/tunefs change or root-fencing claim is added.

See the [execution manifest](../../testing/M2/property-submission-manifest.json); no runtime admission/release follows.


## Bounded job restart projection prerequisite

The [restart projection contract](../../testing/M2/job-restart.md) provides ID-sorted pages of 1..64
summaries from the complete immutable mixed-job view. Queued work projects
review-expired; applying/observing work projects executor-unverified, separately
from the unchanged last durable event. Both record formats, all terminal/uncertain
holds and full retry history remain retained. Unavailable/malformed cursors and
invalid page bounds expose no partial page; cursors require the same retained
view. No native observation, publication, startup consumer, replay or release is
added. Portable behavioral/race/fuzz and FreeBSD cross-compilation passed at signed
source `60db363d5b0f615dbdb7fd2f247b01e8ab12b788`; accepted in PR 58 at merged source
`caf21958b3d89dc0cde3f2ca0ad05787d63d5247`.
Native executor attribution/recovery and complete M2 gates remain open.

See the [execution manifest](../../testing/M2/job-restart-manifest.json); no native recovery/admission follows.


## Property metadata process-death qualification

The [process-cut contract](../../testing/M2/property-publication-cuts.md) kills/reaps the actual built metadata writer
at eight synchronized transaction boundaries, then reacquires the same lock and
independently verifies unchanged objects. Orphan temporaries remain preserved and
block publication; visible queued jobs project reconciliation and retain pool
holds. Matching retries are metadata readback without fresh sync/durability claims.
No native storage command, startup consumer or product behavior/schema changes.
Require all 777 state/backing assertions, 66 parents and 1,876 codec/model children
on both targets, including native restart-page execution and complete backing/
child/host teardown. Both targets passed at signed source
`a3a33d5929f7fba32fd9ab6364320984b49d9162` with full teardown; accepted in
PR 59 at merged source `62f4e7b79233fb945d1fe44010e27a8f2f7fde8b`. Shared evidence/selector changes required one complete portable
review. Reboot/power-cut and actual native
executor/reconciliation/restoration/release gates remain open.

See the [execution manifest](../../testing/M2/property-publication-cuts-manifest.json); no native executor/recovery authority follows.

## Property review and submission integration

The [coordinator contract](../../testing/M2/property-coordinator.md) connects the
accepted plan, capture, binding and publication interfaces. It owns 64 bounded
volatile plans, refreshes trusted policy/observations before submission, claims
each plan once and retains immutable admission acknowledgements for concurrent
confirms. Clock rollback and uncertain publication refuse further submissions;
restart cannot restore plans. Signed source
`a8868fa61bb60b80d3b83378a4f0e964e13e8028` passed the baseline, nine behavioral
parents, storage race/all 21 fuzz targets and an unexecuted FreeBSD cross-build;
see the [execution manifest](../../testing/M2/property-coordinator-manifest.json).
Accepted in PR 60 at merged source `1ae83a781ebaad932d4d7126cbebaf01e34b3cdf`. No native
policy/eligibility backend, kernel-authorized mutation transport, persistent retry
lookup, executor, native recovery or runtime writes are enabled. Those are the
remaining integration gates before an actual operator storage workflow.

## Native property observation integration

The [collector contract](../../testing/M2/native-property-collection.md) joins the
accepted pool/type/property/state/feature/topology parsers to fixed actual FreeBSD
reads. Two complete passes share bounded capture/record budgets and reject
ordinary native drift. Complete results keep eligibility false; root/system/device
exclusions, live policy/store binding, the full coordinator backend and mutation
transport remain open. Signed source
`57ac9dde8c87db90d6cda26c28e0f3ee55ea408e` passed all 16 assertions on both native
targets, the baseline/109 tooling tests, all 11 portable race packages and 30 fuzz
targets with complete teardown; see the
[execution manifest](../../testing/M2/native-property-collection-manifest.json).
Accepted in PR 61 at merged source `851928471a2330689ebd970ed832e9b8861f0fe3`. A focused native fixture and
checksum-bound prebuilt suite qualify only collection/preservation/teardown; shared
runner and cleanup changes require complete portable review.

## Root policy and native property binding

The [binding contract](../../testing/M2/property-policy-binding.md) connects the
held root ownership reader to the native collector. Requests select a GUID only;
private policy supplies the name and scopes. Two canonical policy reads bracket
collection, and complete graph validation checks every selected filesystem
against that policy. Failure returns no partial result; success retains false
eligibility. The focused fixture adds
six policy/lease denials and positive binding controls to the original 16 checks.
Native state reader/publisher behavior and existing evidence remain unchanged.
System/provider/device exclusions, atomic admission, full backend/transport and
executor/recovery still precede an operator storage workflow.

Signed source `cd1991c6a1cb36375f0fca7e7dd90228338390cc` passed all 22 assertions
on both native targets, the required baseline/109 tooling tests, all 11 race
packages and 31 fuzz targets with complete owned teardown; see the
[execution manifest](../../testing/M2/property-policy-binding-manifest.json).
Accepted in PR 62 at merged source `aebae03ab68dfe49490a1fc68537bb319e5b70ef`. The first writable-ancestor fixture failure and
superseded interrupted portable review retain unqualified original evidence;
the correction changes test setup only.

## Native property system-use exclusions

The [system-use contract](../../testing/M2/property-system-exclusions.md) joins
the accepted private policy/property observation to an actual bounded dual-pass
all-imported-pool/mount/swap/GEOM collection. Complete normalized views retain
native topology and positive physical backing; selection must agree with that
topology and remain disjoint from system/other mount/swap/pool backing. Own data
mounts are allowed. Every failure returns zero, and success stays ineligible.
The focused native fixture adds ten assertions using the accepted five snapshot
disks, including owned mount relocation, active disjoint swap, data preservation
and md refusal. Full provider/enrollment,
store/recovery/admission, transport and execution remain open. Runtime is read-only.

Signed source `c0f26eeef6e09e955f1edeb8d6fe15c59481b3cb` passed all 32 assertions
on both targets, the baseline/111 tooling tests, all 11 race packages and 33 fuzz
targets with complete teardown and unchanged snapshot disks; see the
[execution manifest](../../testing/M2/property-system-exclusions-manifest.json).
The initial UFS-only fixture assumption and stopped superseded review retain
unqualified evidence. The correction resolves actual mounted root pools without
changing product gates; a separate failing-first fix rechecks root policy after
the entire system collection. Accepted in PR 63 at merged source
`12ecb173e48099040f4f52662d3e1fba04de5dc6`.


## Native pool-provider integration

The [provider contract](../../testing/M2/property-provider-binding.md) connects
complete native topology to independent direct-provider serial/geometry and
attachment evidence. The composed safety source requires this gate before its
system collection and final root policy re-read. Two fresh passes reject ordinary
drift; every failure returns zero, and success remains ineligible. Signed source `1b6fef0cc54f304a88badffabeb5929ce7edd6d6` passed all 37 assertions
on both native targets, the baseline/112 tooling tests, all 11 race packages and
35 fuzz targets with complete owned teardown. See the
[execution manifest](../../testing/M2/property-provider-binding-manifest.json).
The exact imported ZFS consumer classifier preserves unused-device refusal;
provider identity/geometry must agree with the later fresh system graph. Failed
fixture/evidence attempts remain unqualified at their original sources. Accepted
in PR 64. Runtime enrollment, store/recovery/admission, transport and
execution still precede an operator storage workflow.

## Explicit property member baseline

The [version 2 contract](../../testing/M2/property-member-enrollment.md) adds exact
root-recorded member GUID, serial/provenance and observed geometry. Complete native
matching is required by the composed safety path; legacy records stay readable
but fail this gate. Member facts bind the existing plan/intent policy hash even
without a revision change. Signed source
`b95f60c873020f632674a2302acb1a4a3a6ca646` passed all 46 cases on both targets,
complete portable review and verified teardown; see the
[execution manifest](../../testing/M2/property-member-enrollment-manifest.json).
Actual ZFS publication and softupdates UFS unchanged-record denial both passed.
Enrollment provisioning/adoption, system-store eligibility, full recovery, atomic
admission, transport and execution remain required before operator workflows.

## Read-only property state checks

The [store contract](../../testing/M2/property-store-readiness.md) adds a separate
composed observer requiring complete private policy/job inspection, exact actual
system backing and retained job holds. The existing safety observer remains
read-only with its accepted scope. All observations keep false eligibility.
Signed source `231b8f0f2c250fcbfc900d95d17ed94de6a24cbb` passed all 56 focused
assertions on both targets, complete portable review and teardown. Actual ZFS
success and softupdates UFS denial passed; see the
[execution manifest](../../testing/M2/property-store-readiness-manifest.json).
Provisioning/adoption, full restored-state/executor
validation, atomic admission, transport and execution remain open.


## Explicit property enrollment review

The [contract](../../testing/M2/property-enrollment-review.md) adds a volatile
immutable five-minute review of an explicit root proposal, complete native
context and normalized full system-use baseline. Fresh typed validation refuses
proposal/native/system drift; no scope is inferred and eligibility stays false.
No runtime provisioning/publication workflow is added. Require all 64 focused
native cases on both targets, complete portable review and owned teardown;
both targets passed at signed source `8f6f0cef7f4321b56f37ae99f05c50696c8cc319` with complete portable review and teardown;
see the [manifest](../../testing/M2/property-enrollment-review-manifest.json).
Accepted in PR 67. Root authorization, current predecessor/backing/capacity/
recovery and atomic publication gates remain open.


## Root-leased enrollment state binding

The [contract](../../testing/M2/property-enrollment-state.md) connects the volatile
proposal review to one held root lease and explicit first/exact predecessor,
complete private record/backing baseline, projected capacity and retained job holds.
Fresh native/context/expiry and full metadata validation remain read-only; no
publication, claims, recovered-state authority or write eligibility is added.
Require all 76 focused native assertions on both releases, complete portable review
and owned teardown; both targets passed at signed source `9329f795185df1a823247d8cba917357d7e614fe`.
See the [manifest](../../testing/M2/property-enrollment-state-manifest.json);
accepted in PR 68 at merged source `52195cf5135f04d09422dd85c337e84329bf6cf9`. Actual provisioning/atomic publication,
restored-state/executor validation and authorized runtime workflows remain open.


## Native enrollment collection prerequisite

The [source contract](../../testing/M2/native-enrollment-source.md) joins root-explicit
owned selections with independent pool/provider/system collectors and the sealed
root-state binding. Fresh validation uses the original lease; all denials return
zero and success remains ineligible. Require all 84 focused assertions on both
releases, complete portable review and verified owned teardown. Both targets passed all 84 assertions at signed source
`d13f4ba489cd659b87f81f2746830aaa3a94bfef` with complete portable review and teardown;
see the [manifest](../../testing/M2/native-enrollment-source-manifest.json).
Accepted in PR 69 at merged source `1b74c3f5e3d15d7049ccd2099f8efd86f1771548`. Authorized runtime provisioning, final fresh publication checks, full
restored-state/executor reconciliation and the remaining M2 lifecycle remain open.


## Reviewed enrollment metadata publication

The [contract](../../testing/M2/enrollment-publication.md) adds one attempt across
copies/rebindings, complete census/predecessor/hold/backing/capacity checks under
the original root lease and clock/context guards around the unchanged sync/rename
engine. Require all 819 state assertions and 88 actual native pool assertions on
both releases, complete portable review and teardown. Corrected source
`0a1f4041fc4cf18ae0919428e303dd2140e63d48` passed complete portable review and both
88-case pool suites. Original source `656e4220a4040669c09bce29311e2fdd74d62992`
passed both 819-assertion state suites; the corrected source state binary is
byte-identical. See the [manifest](../../testing/M2/enrollment-publication-manifest.json).
The superseded 14.5 predecessor fixture failure remains unqualified; all teardown
passed. Accepted in PR 70 at merged source
`1562334b09f3492b81a540459368efc2911a1309`.
Runtime provisioning/adoption, native eligibility/recovery, authorized transport,
backend/executor and remaining M2 lifecycle/operator workflows stay open.


## Independently collected enrollment publication

The [contract](../../testing/M2/native-enrollment-publication.md) connects sealed
root-state reviews to independent final native collection and the accepted
one-attempt publisher. Callers cannot provide observations or substitute selections;
actual metadata acknowledgements survive later cancellation/expiry. Both targets
passed all 88 assertions at signed source `d6fc9eb9dfee6cda75de7ed727db11bcaf00cb8d`,
with focused portable review and verified teardown; see the
[manifest](../../testing/M2/native-enrollment-publication-manifest.json). The initial
invalid drift projection remains unqualified; a failing-first fixture regression
and source-consistent correction passed. Accepted in PR 71 at merged source
`c937a29d202b1714c15509b75ba043c625a8a6c8`. Runtime
provisioning, eligibility/recovery, authorized transport and operator workflows
remain open.


## Foreground retained job-status diagnostic

The [contract](../../testing/M2/storage-job-status.md) connects the root lease's
complete mixed census and bounded restart pages to a foreground OS-root command.
It stages all retained summaries with explicit unverified recovery and unchanged
holds, closes its lease before output and enables no storage write or service.
Both native targets passed all 21 assertions at signed source
`03d4ab5948dc56a41333acb0e4dd20d3191e1bc7`, with actual root/wrong-UID checks, full-pipe
signal cancellation, record/lock preservation, backing and complete teardown.
Full portable review passed 122 tooling tests, 11 race packages and 35 bounded
fuzz targets. See the [manifest](../../testing/M2/storage-job-status-manifest.json).
All 47 historical M2 manifests and native metadata code/tests/codecs are unchanged;
the state-directory public comment now names inspection wiring. Accepted in
[PR 72](https://github.com/gonz0w/horvium/pull/72) at merged source
`d3ca26e03b0f40be38d39dfa226d2da4bc29991c`. Management UI/CLI parity, native reconciliation/transport/execution and the
remaining M2 lifecycle remain open.


## Reviewed property postcondition comparison

The [contract](../../testing/M2/property-postconditions.md) checks exact local
property effects and relevant retained context against independently fresh native
observations, preserving false eligibility and all existing precondition gates.
Both native targets passed all 88 existing assertions plus eleven mandatory
postcondition subtests at signed source `0b828eb38be56708abca17dbfc200fb49d1ddd62`.
Focused portable review passed 122 tooling tests, nine affected race packages
and 30 selected bounded fuzz targets, with full owned teardown; see the
[manifest](../../testing/M2/property-postconditions-manifest.json). Both superseded
fixture failures remain unqualified; the corrections change only fixture reset
semantics/raw-column expectations. The existing pool-source checker/selector and
all 48 historical metadata manifests retain their actual scope. Accepted in [PR 73](https://github.com/gonz0w/horvium/pull/73) at merged source
`37343f66e3c3a95ebc7dfbb0eb965581ed20231b`. No runtime native
writer, job-success publication, attribution, recovered-state clearance or
management mutation transport is added.

## Retained property-job observation assessment

The [contract](../../testing/M2/property-assessment.md) joins independently fresh
baseline/expected-effect facts to immutable retained job history and conservative
restart status. Both/neither matches are explicit and supply no causation or
recovery clearance. Focused review passed at signed source
`2ddc0bb971af47ac6d21f0898deb374cf94ef0d5`: 122 tooling tests, nine affected race
packages and all 31 selected bounded fuzz targets. Both releases passed the 88
existing native assertions, eleven postcondition subtests and eight mandatory
assessment subtests with complete owned teardown. Original 15.1 and corrected 14.5
sources ran the exact same independently rebuilt binary after a portable-only
selector/fixture correction. The failed portable review remains unqualified; see
the [manifest](../../testing/M2/property-assessment-manifest.json). Metadata
codecs/publication, shared runner/checker/selector and all 49 prior manifests are unchanged.
Accepted in [PR 74](https://github.com/gonz0w/horvium/pull/74) at merged source
`4fe812eed9c1ecf4265f9b3c72c9e5c2f0d2a4e0`.
Actual executor/reconciliation/restoration, durable outcomes, runtime transport/
eligibility/writes and operator workflows remain pending.

## Root-leased native property-job assessment

The [source contract](../../testing/M2/native-property-assessment.md) joins an opaque
retained job ID to complete root-leased before/after census fingerprints and
independently collected current policy/native facts. Unsafe/incomplete/drifting
metadata or collection returns zero; history/holds/false eligibility remain intact.
Focused portable review passed at signed source
`b3e5007efbf0c134f149a21538c0b186c8c9d1e6` (baseline checks, 122 tooling tests,
nine race packages and 32 fuzz targets). Both release candidates passed 88
existing pool assertions, eleven postcondition cases, eight retained-assessment
cases and all seventeen mandatory native-source cases, plus full owned
preservation/teardown. Independently rebuilt binaries matched exactly; see the
[manifest](../../testing/M2/native-property-assessment-manifest.json). Two
earlier deadline attempts remain unqualified; the correction removes duplicate
unchanged fixture observations while retaining every assertion and time limit.
Accepted in [PR 75](https://github.com/gonz0w/horvium/pull/75) at merged source
`aa65f5a269a5f3dafd1ae4d64226b5f3bef46768`; all 50 prior M2 manifests remain unchanged.
No metadata codec/mutation engine or shared runner/selector/checker changes are
needed. Actual executor/reconciliation/restoration, durable outcomes, authorized
runtime transport/writes and operator workflows remain pending.

## Foreground native property-job assessment

The [diagnostic contract](../../testing/M2/property-assessment-diagnostic.md)
connects the accepted independent source to an OS-root foreground command with
explicit protected ledger/job selection and bounded output/lifetime. It presents
fresh match facts separately from unchanged durable/restart labels, false recovery/
executor verification and retained holds. Focused portable review passed at
signed source `b8d078d0097ae47f2e9d5f9b035fccbd2f144e9c`: baseline checks, 122
tooling tests and command-package race checks. Both native command fixtures
passed all 21 old and fifteen new cases with complete preservation/teardown;
independently rebuilt binaries matched exactly. See the
[manifest](../../testing/M2/property-assessment-diagnostic-manifest.json). The
initial timestamp fixture failure remains unqualified; its portable
failing-first regression and corrected constructor passed without changing
product code or time limits. Accepted in [PR 76](https://github.com/gonz0w/horvium/pull/76)
at merged source `14c7c7201a987c5c239b5587e29e50f448bd7234`.
The new isolated mirror/provider subgroup retains the existing job-status parent
and its 21 original assertions. Native eligibility/execution, recovery resolution,
restored-state validation, authenticated UI/CLI parity and full M2 remain open.

## Read-only pool health and scan observations

The [contract](../../testing/M2/pool-health.md) binds complete paired status to
fresh permanent/import identity and supported release. Report unhealthy facts
and typed legacy scans without write eligibility or job attribution. Failing-first
portable tests and focused review passed at signed source
`ab2685cdffcb6aa06d14b356b1df6f2787143300` (122 tooling tests, 9 race packages, 33 fuzz targets).
Both pinned releases passed six new health cases and all old status/assessment
checks with identical binaries and complete owned teardown; see the
[manifest](../../testing/M2/pool-health-manifest.json). Accepted in
[PR 77](https://github.com/gonz0w/horvium/pull/77) at merged source
`60b6f254badbf95d3a781c24b99feff632b5009e`.
This progresses slice 4 observation independently of unqualified mutation gates;
attributed start/stop/recovery and complete health/scan acceptance remain open.

## Native scrub-state observation qualification

The [fixture contract](../../testing/M2/scan-lifecycle-observations.md) exercises
actual running/paused/resumed/cancelled/completed scrub observations using a
restored, disposable-guest-only kernel progress control. Preserve existing
product code/gates, all old status/assessment/health cases and deadlines. Native
qualification and focused review passed at signed source
`728e980a8b21755c9b87f33e48405f096bb03b24`: baseline/122 tooling tests, command-package
race checks and eight new scan cases plus all old status/assessment/health checks
on both releases, with identical binaries and complete verified teardown; see the
[manifest](../../testing/M2/scan-lifecycle-observations-manifest.json). Accepted in
[PR 78](https://github.com/gonz0w/horvium/pull/78) at merged source
`0ffc5f4771bb3c5640e4ce04f53deffc8d36d71b`. Advancing progress, faults,
attribution, executor/recovery and full M2 acceptance remain open.

## Native repaired-checksum findings

The [fixture contract](../../testing/M2/checksum-findings.md) targets one owned
mirror leaf with the base kernel read-corruption facility and verifies member
findings separately from scan completion/error count. Preserve content, remove
only the acknowledged injector, restore original properties and retain all old
native checks/deadlines. Product code/gates remain unchanged. Native qualification
and focused review passed at signed source
`8078e1d0c694a0384b5183ae33ea95fddc6a45fb`: baseline/122 tooling tests, command-package
race checks and eight new checksum cases plus all original checks on both
releases, with identical binaries and verified handler/storage/process/base/
runtime teardown; see the [manifest](../../testing/M2/checksum-findings-manifest.json).
Accepted in [PR 79](https://github.com/gonz0w/horvium/pull/79) at merged source
`ef66dcfb6992ce5d898e0598dded86fca5cfdabd`; unrecoverable errors, attribution/executor/
recovery and complete M2 acceptance remain open.

## One-attempt property execution sequence

The [contract](../../testing/M2/property-execution.md) connects queued records,
durable applying intent, fresh rechecks, a single typed control attempt, observed
facts and exact durable outcomes through a trusted root-worker boundary.
Publication uncertainty stops work; missing quiescence and differing observations
retain nonterminal reconciliation holds. Signed source
`cceb615b82aedc98d4e93fb1cdb90545b05916ea` passed baseline/122 tooling tests, ten
behavioral groups, nine affected race packages and all 34 selected fuzz targets.
FreeBSD tests cross-compiled without execution, the artifact was removed and all
owned processes stopped; see the [manifest](../../testing/M2/property-execution-manifest.json).
Accepted in [PR 80](https://github.com/gonz0w/horvium/pull/80) at merged source
`252e07e3cd2412a698b3b0c6c4651ca47db1331f`.
No native backend, runtime launch, authority restoration, scheduler or hold
release is added; these require separate native gates before operator writes.

## Private native property control

The [contract](../../testing/M2/property-control.md) validates one allowlisted
property set against exact applying metadata, immutable selection and fresh GUID
context. Its private callback still needs original live root authority; no runtime
consumer exists. Eleven actual control/denial/restoration cases, including both
normal-exit descendant guard cases, are required on both disposable releases.
Signed source `6ef01cb5fbd3f3015a6fc8c9a5a8ccc4302031aa` passed baseline/122 tooling tests,
eight affected race packages and all nine selected fuzz targets. Both native
releases retained all 88 assertions and old subgroups and passed every new case
in 201.42/193.37 seconds, within the original five-minute deadline.
Actual effects retained applying metadata; kernel UID denial, fresh drift refusal
and descendant waiting passed. All metadata used an explicitly owned mounted ZFS
child; the guest root and softupdates refusal remain independent. Complete
preservation/teardown passed, and the operator-stopped initial attempt stays
unqualified; see the [manifest](../../testing/M2/property-control-manifest.json).
Accepted in [PR 81](https://github.com/gonz0w/horvium/pull/81) at merged source
`e16e130aa502f66202386e3578095d607947b0ba`. Native backend/eligibility, crash/recovery/scheduling
and operator workflows remain separate gates.

## Original admission execution handoff

The [contract](../../testing/M2/property-handoff.md) transfers one fresh admission
from its original coordinator to the accepted execution sequence. It retains the
same combined backend, exact queued acknowledgement and original civil-clock
lineage. Stored bytes, duplicate backend acknowledgements and reconstruction
cannot create the handoff. Signed source `01af1c83f2deb4e1c492aaee1098344b3def8dcc` passed
baseline/122 tooling tests, eight new behavioral groups, 9 affected race packages
and all 36 selected fuzz targets. FreeBSD tests cross-compiled without execution;
the artifact was removed, all owned processes stopped and native code/fixtures/
selectors remained byte-identical. See the
[manifest](../../testing/M2/property-handoff-manifest.json). Accepted in
[PR 82](https://github.com/gonz0w/horvium/pull/82) at merged source
`76774cf1add6479ab6a917beddfb464b869cbd10`; a complete original root backend and all actual native eligibility/
interruption/recovery gates remain.

## Fresh root submission state claim

The [contract](../../testing/M2/property-live-state.md) distinguishes an original
post-sync submission from restored/foreign metadata through a bounded volatile
receipt on the original root lease. One opaque read-only claim permits only its
exact queued record or applying successor through unchanged state/backing gates;
other holds remain blocking. It enables no native executor or eligibility.
Signed source `fe56955fcc2b1803bfa6bf61685b336f8716b420` passed baseline/122 tooling tests,
five new portable groups, nine affected race packages and all 37 selected fuzz
targets. Both serial native state runs passed the 19 new children and every
historical 819 control/67-parent/1,876-codec gate with identical artifacts and
verified teardown. Native policy/physical system-use inputs remain synthetic;
no eligibility or complete backend follows. Superseded native/portable attempts
remain unqualified; see the [manifest](../../testing/M2/property-live-state-manifest.json).
Accepted in [PR 83](https://github.com/gonz0w/horvium/pull/83) at merged source
`36a919d78c03a5572087a7da8f6e6c9467189ef2`. Complete root/native/effect recovery/operator acceptance
remain open.

## Native observation through the original root claim

The [contract](../../testing/M2/property-claim-source.md) binds root metadata readers
and the state/backing check to one original opaque claim. Fresh guarded native
collection is bracketed by exact whole job and full census presentation reads;
all failures discard output and success remains ineligible. Four portable groups
and selector fuzzing plus the unchanged state/pool runners on both releases are
required. Signed source `aa0794996733706c47cce45ad6e0aad3f4155d28` passed baseline/122 tooling
tests, nine affected race packages and all 38 selected fuzz targets. Both state
releases passed every historical gate and all 19 claim children. The two 15.1 pool
attempts passed all eight new actual native cases but exceeded the unchanged
parent deadline in old cases; full pool qualification on both releases remains
pending at that source. PR 84 was draft; see the [partial manifest](../../testing/M2/property-claim-source-manifest.json).
All owned teardown passed, historical/shared inputs remain unchanged and unrelated
workloads were not stopped. No executor, recovery/launch clearance, release,
runtime consumer or operator writer is enabled.

## Separate native claim qualification fixture

PR 84 merged at `41140a8061267a000bd321b5c5f4f536954cfb25` with full pool gates
outstanding. The quiet merged-source retry still exceeded the historical parent
budget. The [fixture contract](../../testing/M2/property-claim-fixtures.md) keeps
all old assertions, explicitly amends the aggregate collector budget to eight
minutes, isolates the new eight cases
in a fresh 135-second fixture, and requires both actual parents plus outer teardown
in the shared checker. Two failing-first checker regressions now pass. No product
bytes change; native and required full portable qualification are pending. Historical
partial results remain source-bound and unqualified attempts are not relabeled.

The first separated retry passed its 27.05-second claim parent, but the old parent
still exhausted five minutes. The fixture contract explicitly amends its aggregate
lab budget to eight minutes while retaining all cases and production operation
limits. This is qualification tooling only; current native results remain pending.

At signed source `03b7dce3f70ea13d972febba8702536785c78fb0`, both serial native pool releases passed
all 88 old assertions, all retained subgroups, eight independent claim cases and
complete teardown. Aggregate parents took 316.89/289.67 seconds; claim parents
took 27.21/23.75 seconds. Fresh full portable review passed 124 tooling tests,
11 race packages and 44 bounded fuzz targets. See the
[manifest](../../testing/M2/property-claim-fixtures-manifest.json). The historical
pending/draft instructions are superseded for this observer subset; all 60 prior
manifests and product code remain byte-identical. Root backend/runtime execution,
native effects/recovery and operator acceptance remain open.


## Original claim applying acknowledgement

PR 85 accepted the [separate fixture qualification](../../testing/M2/property-claim-fixtures.md)
at merged source `1c8afa7ff49052a6db1ef997204eedd55246becd`, closing the
read-only observer pool gates without enabling effects. The
[applying contract](../../testing/M2/property-claim-applying.md) adds one exact
metadata publication/acknowledgement through the original live root claim,
retaining the shared durability pipeline and strict full census through staging
and readback. Portable failing-first tests now pass; both current-source native
state gates and required portable review are pending. No full backend, launch
authority, terminal resolution, runtime write or operator workflow is enabled.

Signed source `80f3588a89abcf8484f45da06f37eca18bcfa6dd` passed baseline/124 tooling tests, 9 affected
race packages and all 38 selected fuzz targets. Both serial native state releases
retained every old gate and passed all 23 applying children plus actual UID denial;
the new groups took 3.92/3.83 seconds with complete teardown. See the
[manifest](../../testing/M2/property-claim-applying-manifest.json). Earlier pending
instructions for this subset are superseded; 61 historical manifests, shared
publisher/readers and pool fixtures remain byte-identical. No native executor,
complete root backend, runtime writer, hold release or M2 acceptance follows.

## Original-claim observing/outcome metadata

PR 87 accepted worker uncertainty propagation and the disposable Git-maintenance
teardown fix at merged source `412cf3a9bd49c435eb77dbd7664ed0ed8981a378`.
The next [successor metadata subset](../../testing/M2/property-claim-successor.md)
continues exact original-claim acknowledgement from applying through observing,
success or reconciliation. Only successful sync and independent exact record/
census readback populate its volatile hash/revision. Raw/restored metadata cannot
replace this lineage; failed valid attempts cannot retry and no hold is released.
Existing queued/applying readers/readiness, generic metadata publishers and native
pool/control sources remain unchanged. The shared claim transaction requires
both disposable native state gates plus focused portable review; see its
[manifest](../../testing/M2/property-claim-successor-manifest.json). Runtime writes,
complete root backend, native effects/interruption/recovery and transport remain
unqualified/disabled; metadata success alone supplies no native outcome authority.
Signed source `7af2f1cda24179f4e9f8ceebc4a20503dc6c9c25` passed all 18 portable
lineage cases, mandatory baseline with 125 tooling tests, 9 affected race packages
and 38 bounded fuzz targets. Both native state releases passed all 38 new children,
23 applying/19 live-state children, actual UID denial and every historical gate,
with identical artifacts and complete teardown. PR 88 accepted this subset;
all 63 prior manifests retain their original sources.

## Original claim fixed-control binding

PR 88 accepted exact original-claim successor metadata at merged source
`c77923a96842b16c6c34199ebdc79279e0757e23`; its evidence retains executed source
`7af2f1cda24179f4e9f8ceebc4a20503dc6c9c25`. The next
[private control binding](../../testing/M2/property-claim-control.md) verifies exact
successful applying acknowledgement and fresh independent native baseline through
that same root claim. Its mandatory live-clearance callback remains a complete
original-backend obligation, with only explicit disposable fixtures supplying it.
Portable failing-first lineage/binding tests pass. Both complete state releases
passed at signed source `277afb1ef7e139d61ae930a720f064e92dc5ed86`; its first
pool attempt remained unqualified after the fixture's quota restoration failed.
The fixture-only correction at `d5ca26c6a814b63eb88057623a33fb70cbc901d3` passed
both complete pool releases, all six new binding cases and exact quota-source/
metadata/sentinel preservation. Every state/build input remains byte-identical,
so that evidence retains its actual source without redundant state reruns.
Focused review passed 125 tooling tests, nine race packages and 38 bounded fuzz
targets in 194.118 seconds. All nineteen new state cases, every historical gate,
64 prior manifests and unchanged tools/readers/publishers/deadlines are retained.
All owned teardown passed; see the
[manifest](../../testing/M2/property-claim-control-manifest.json). No runtime
writer, complete backend/session/recovery/global authority or assembled M2
acceptance is enabled. PR 89 accepted this subset at merged source
`025e496a3d10f935071fae59ba606e0a246e9d62`.

## Original coordinator authority binding

The [next binding](../../testing/M2/property-execution-authority.md) supplies an
opaque link from one original admission handoff to that same backend, allowing
its final prelaunch callback to check exact retained plan/session context, queued
acknowledgement, original civil clock and publication uncertainty. Pending and
failed binding cannot launch; control return and every consumed worker exit
retire copied handles without releasing holds or discarding late verified outcome
duties. Nine new parents and extended existing handoff fuzz actions retain all
old handoff/uncertainty cases. Signed source
`68a6cf6e1794d258be31c42a7b7913574fec7311` passed nine new parents/41 children,
all ten retained handoff parents, baseline/125 tooling tests, nine race packages
and 38 selected fuzz targets in 212.450 seconds. FreeBSD cross-compilation passed
without execution and the artifact was removed; both review supervisors and all
lab sessions stopped with verified teardown. All 65 historical manifests and 364
protected paths remain byte-identical; Actions stay disabled. See the
[manifest](../../testing/M2/property-execution-authority-manifest.json). Native adapters,
transactions and fixtures stay unchanged. Full root backend, authenticated write
transport, native recovery/global clearance and runtime writers remain open.


## Private native property backend integration

PR 90 accepted the original coordinator authority at merged source
`fa7013198b50bb058bb6775cefdad2698aaf270b`; its evidence retains actual source
`68a6cf6e1794d258be31c42a7b7913574fec7311`. The
[private backend](../../testing/M2/property-native-backend.md) now composes one
original review/admission/claim, exact applying publication, one fixed native
control, independent facts and witnessed observing/success acknowledgement.
Require mandatory external authentication/recovery/global clearance, supplied
only by fixtures; no runtime consumer exists. Signed source `649a60b1b6a28f5c791f1d1332e44362d05a98ab`
passed all eight new portable parents/42 children, baseline/125 tooling tests,
nine race packages and all 38 selected fuzz targets in 185.659 seconds. Both complete
native pool suites passed all twelve new backend cases and their backing,
preservation and teardown groups, retaining every old PASS name/count (76/70).
The same executable ran on both guests, with exact native value/source and
applying-before-effect evidence, lost-acknowledgement uncertainty and no replay.
The initial 15.1 attempt remains unqualified after exhausting the old claim
context; new scenarios use separate serial 135-second fixtures. Original deadlines,
shared tools, metadata transactions and all 66 historical manifests stay unchanged.
All owned sessions stopped with verified artifact/guest/supervisor teardown;
Actions remain disabled. See the
[manifest](../../testing/M2/property-native-backend-manifest.json). No hold release,
new schema, authenticated runtime transport, native recovery/global launch fence
or complete root/M2 acceptance follows. Operator writes remain disabled.

## Live application-session authority prerequisite

PR 91 accepted the private native backend at merged source
`304d00df81a98b899c9d6e49d026ed96059a7661`; its native evidence retains actual
source `649a60b1b6a28f5c791f1d1332e44362d05a98ab`. The
[next application binding](../../testing/M2/session-authority.md) supplies an opaque
original-session prerequisite behind explicit POST provenance/cookie/CSRF checks.
Context is credential-free correlation, not root authorization. Exact private
identity/fingerprint/creation-time checks and sticky logout/expiry/clock retirement
prevent reconstruction or renewal through copied strings/records. Ordinary read
responses stay unchanged; no runtime route or helper/native consumer is registered.
Nine new portable parents/51 children and bounded action fuzz cover the contract;
signed source `3ddd368e747ae36813c009cdd10f6aa097cf41d7` passed baseline/125 tooling
tests, all eight affected race packages and eleven selected bounded fuzz targets
in 72.839 seconds. FreeBSD compilation passed without execution and its artifact
was removed. Review supervisors/all owned sessions stopped with verified teardown;
Actions stay disabled. See the
[manifest](../../testing/M2/session-authority-manifest.json). Kernel-peer mutation
negotiation, root recovery/
global launch, enrollment and complete M2/operator gates remain open. Native
facilities, shared tools and all 67 historical M2 manifests retain their actual
bytes/sources; unrelated QEMU suites are not rerun for this portable boundary.

## Application-context negotiation prerequisite

PR 92 accepted original live-session authority at merged source
`2cce0f276c84a61241a971dffddb546414d85682`; its actual portable evidence remains
source-bound. The next [negotiation contract](../../testing/M2/action-negotiation.md)
adds strict bounded version-2 hello/ready codecs and a private original-session
adapter. The helper remains version-1 inventory only; no runtime caller is
registered. Twelve parents/57 children, exhaustive frame truncations and bounded
fuzz cover correlation, malformed/forged claims, downgrade/operation denials,
original session revocation, no credentials/idle renewal and read-only dispatch
isolation. Native facilities/fixtures and shared tools remain unchanged. Actual
kernel peer binding, typed operations, root recovery/global launch and assembled
M2 qualification remain pending. Executed evidence is recorded below.

Executed partial evidence: signed source `e449b8c005f2f4635f471305a6e2cf8ab11a1100` passed
all twelve parents/57 children and three fuzz seeds, baseline/125 tooling tests,
eight affected race packages and all twelve bounded fuzz targets in 67.911
seconds. Both admin/helper FreeBSD test packages compiled without execution;
checksummed artifacts were removed. Review supervisors and all 612 owned
sessions stopped with verified teardown; Actions remain disabled. All 68
historical M2 manifests and 386 existing Go/tooling inputs are byte-identical
to the merged base. See the [manifest](../../testing/M2/action-negotiation-manifest.json).

## Native action-peer prerequisite

PR 93 accepted the bounded negotiation codecs at merged source
`3728211628a270de3f958534d6f37e3374ce6f46`. The next
[peer contract](../../testing/M2/action-peer.md) binds private negotiation to the
actual kernel UID and original connection with bounded cancellation/deadline and
owned close. It remains unregistered, supplies no policy/recovery/launch authority
and preserves version-1 inventory behavior. A focused checksummed prebuilt peer
suite avoids repeated discovery SDK/package builds on the Mac. Shared runner/
cleanup changes require fresh full portable qualification plus both native
targets. Source-bound evidence is recorded below.

Executed partial evidence: signed source `4222e8f098114df8942fb4843f7e683ae9068ee7` passed
four new portable parents/nine children, baseline/131 tooling tests, all 11
portable race packages and all 46 bounded fuzz targets in 223.465 seconds.
The shared-tooling change selected a fresh full review. The same exact executable
passed all eighteen native cases on FreeBSD 15.1-RELEASE-p3 and 14.5-RELEASE,
in 33.303/36.617 seconds including boot/teardown. Wrong/root UID, nonroot helper
despite repaired file ownership, EOF and extra-frame denials used actual kernel
facilities. Every socket/fixture/artifact/guest/supervisor cleaned up; all
618 owned sessions stopped. The earlier native stub run is explicitly
unqualified with verified cleanup. All 69 historical M2 manifests and 383
protected existing inputs retain actual bytes/sources; existing native peer/
socket functions and old suite commands/deadlines are preserved. See the
[manifest](../../testing/M2/action-peer-manifest.json). No runtime operation or full recovery/global
launch/acceptance follows. Actions remain disabled.

## Typed property request prerequisite

PR 94 accepted the original native peer boundary at merged source
`543889037dba05df0906130ddcc720e995afde9f`. The next
[request contract](../../testing/M2/property-requests.md) defines strict version-2
property review/submission selections: canonical pool/filesystem GUIDs and the
planner's literal property allowlist, or only pool/plan/submission identifiers.
Each schema rejects extra privilege fields and replacement intent. Original request
correlation and fixed frame/field/depth limits are syntax checks, never peer/session,
enrollment, original plan, recovery or launch proof. Inventory runtime and original
peer quiet behavior remain unchanged; response/exchange/dispatcher integration
remains pending. This is a T1 subset of M2-PLAN-001/M2-PRIV-001, not native or M2
acceptance. Baseline/focused race/fuzz and unexecuted FreeBSD compilation are
recorded below; preserve historical source-bound native evidence.

Executed partial evidence: signed source `2c5d89680f6be5be790c92c741f8f965f2f5bf2a` passed
five behavioral parents/61 children and three fuzz seeds, baseline/131 tooling
tests, 9 affected race packages and all 41 bounded fuzz targets in 227.424
seconds. Helper/storage FreeBSD test packages compiled without execution and their
checksummed artifacts were removed. All 623 owned sessions stopped with
verified teardown; Actions remain disabled. All 70 historical M2 manifests and
393 protected existing inputs retain actual bytes. Existing planner validation is
unchanged apart from the new bounded syntax entry point. The earlier lint failure
is explicitly unqualified and corrected before this fresh review. See the
[source-bound manifest](../../testing/M2/property-requests-manifest.json). No native typed exchange,
root recovery/global launch or assembled acceptance follows.

## Bounded property reply prerequisite

PR 95 accepted typed property requests at merged source
`43cd4b11b10ff7b155b71a6e0ef910a4b40e5089`. The next
[reply contract](../../testing/M2/property-replies.md) preserves complete review
and lifecycle display views with exact original correlation/selection, plus bounded
refusal codes. Shared converters/validators preserve the existing durable formats;
private session/policy digests and execution proofs stay outside replies. The fixed
64 KiB frame cap includes the envelope and never permits truncated views/history.
Admission/duplicate/refusal labels remain peer assertions, not durability or retry
permission. This is a portable T1 subset of M2-PLAN-001/M2-PRIV-001/M2-JOB-001;
owned socket phases, original session/kernel authorization, enrolled root policy,
recovery/global launch and actual admission/native execution remain pending.
Baseline/focused race/fuzz and unexecuted FreeBSD compilation are recorded below.

Executed partial evidence: signed source `728387ba21d322f5ee42426bdf617d527e2afdda` passed
twelve behavioral parents/36 children and five fuzz seeds, baseline/131 tooling
tests, 9 affected race packages and all 43 bounded fuzz targets in 225.190
seconds. Helper/storage FreeBSD test packages compiled without execution; exact
checksummed artifacts were removed. Independent old/new synthetic intent/property-job
record hashes match merged PR 95 and are guarded by golden regression tests.
All 71 historical M2 manifests and 395 protected existing inputs retain actual
bytes. All 628 owned sessions stopped with verified teardown; Actions
remain disabled. The earlier native-selector naming collision is explicitly
unqualified and corrected before this fresh review. See the
[source-bound manifest](../../testing/M2/property-replies-manifest.json). Owned exchange, root
recovery/global launch, actual admission and full M2 acceptance remain pending.

## Owned property exchange prerequisite

PR 96 accepted bounded replies at merged source
`f36876b0de6b278634ec5718efa0f901c7093e59`. The next
[exchange contract](../../testing/M2/property-exchange.md) owns one original
kernel-authorized connection across negotiation, one strict typed request and one
terminal correlated reply. Fixed existing frame limits bound four-frame aggregate
traffic; original deadlines never renew. Terminal EOF is required before returning
client metadata, without asserting peer liveness after EOF. Root parsed identities
remain snapshots; no application-session, enrolled policy, original plan, recovery,
durability or launch authority is granted. Native synthetic replies qualify only
kernel/transport behavior. Single-operation/overlap/cancellation/partial-stream and
role/identity/correlation cases join the unchanged eighteen old native peer cases
in a separate opted-in parent; strict complete evidence is mandatory on both pinned
releases. Shared checker/selector maintenance requires fresh full portable review.
Signed source `f26e51413b8325e2ffa3d7681f0213564f108c4f` passed all 44 actual
peer/exchange cases on disposable FreeBSD 15.1 and 14.5. Baseline/132 tooling tests,
11 race packages and all 49 bounded fuzz targets passed with verified teardown.
The actual denying-stub native run stays separately unqualified; all 72 historical
M2 manifests retain their bytes. See the
[source-bound evidence](../../testing/M2/property-exchange-manifest.json).
Actual backend admission, authenticated session assembly, recovery/global launch
and M2 acceptance remain open.

## Bounded property connection admission prerequisite

PR 97 accepted the owned typed exchange at merged source
`70d090d9c0ea84f514b1adba846b0368b96a59f9`. The next
[connection contract](../../testing/M2/property-connections.md) bounds original
accepted sockets and their lifetimes before future negotiation/workers. It is a
private network accounting prerequisite, without listener/dispatcher, UID/session,
durable admission or worker/native quiescence authority. Signed implementation
`8273a09538d566982f0bd1d3edc311aa0580e4d9` passed eight behavioral parents/twelve
children, baseline/132 tooling tests, eight affected race packages and fourteen
bounded fuzz targets. Helper tests compiled for FreeBSD without execution; the
artifact was removed and all 637 owned sessions stopped with verified teardown.
All 73 historical M2 manifests retain their bytes; existing native peer/exchange/
fixtures/shared tools remain unchanged. See the
[source-bound manifest](../../testing/M2/property-connections-manifest.json).
Native admission/runtime assembly, authenticated application binding, original
root/backend/recovery/global launch gates and full M2 acceptance remain open.

## Admitted property exchange prerequisite

PR 98 accepted connection accounting at merged source
`e241a9a08cc1c6dc530c5d80c2acec24bca3ea37`. The next
[admitted exchange](../../testing/M2/property-admitted.md) ties one negotiation
attempt to the original admission context/descriptor, automatically releases
capacity after terminal replies/errors and contains shutdown/overlap in both I/O
phases. Signed source `24fd2777481139386f8e824579f22e9579755a8e` passed all 60
transport cases on both disposable FreeBSD releases, four portable behavioral
parents, baseline/133 tooling tests and fresh full portable review (11 race
packages and 49 bounded fuzz targets). Native denying-stub evidence remains
unqualified; earlier passing results retain their actual source. All 647 owned
sessions stopped with verified teardown; Actions remain disabled. All 74 historical
M2 manifests retain their bytes. See the
[source-bound manifest](../../testing/M2/property-admitted-manifest.json).
It registers no listener, worker, session adapter, backend or storage command;
independent root/session/recovery/global launch and complete M2 acceptance remain open.

## Session-bound property client prerequisite

PR 99 accepted admitted exchange ownership at merged source
`50c342f8314e704db7da4c8d1df7e39d7670d1d5`. The next
[application client](../../testing/M2/property-client.md) joins original live-session
checks to an actual kernel-authenticated one-operation client, with typed selection
validation before negotiation. Signed source
`9876bd3d637be43daedef3afe335fbb4b7488ed2` passed five portable parents/nineteen
children, all 74 transport cases on both disposable FreeBSD releases, baseline/134
tooling tests and fresh full portable review (eleven race packages/49 bounded fuzz
targets). Native callbacks model phase revocation; real original authority is
checked in portable admin tests. Earlier successful native results retain their
source; the stopped incomplete deep review stays unqualified. All 657 owned sessions
stopped with verified teardown; Actions remain disabled. All 75 historical manifests
retain their bytes; see the
[source-bound evidence](../../testing/M2/property-client-manifest.json). Runtime/session backend
assembly, root recovery/global launch and complete M2 acceptance remain open.

## Private root property listener prerequisite

PR 100 accepted the original-session property client at merged source
`4ae7c268fa3c939a28c62cdf74852e13ba4da119`. The next
[root listener](../../testing/M2/property-listener.md) owns its root socket and joins
original admission to a separately bounded worker ledger. Signed source
`1245cb4e5c85ca01eca91b96c91131207167593c` passed six portable parents/four children,
all 92 transport cases on both disposable FreeBSD releases, baseline/135 tooling
tests and fresh full portable review (eleven race packages/49 bounded fuzz targets,
210.028 seconds). The signed denying-stub run remains unqualified; its 74
retained transport cases passed. Earlier native passes retain their actual source
before the mutex-copy fixture correction. All 665 owned sessions stopped with
verified teardown; Actions remain disabled and all 76 historical manifests retain
their bytes. See the [source-bound evidence](../../testing/M2/property-listener-manifest.json).
All typed operations remain denied. Backend, original plans/session, recovery/global
launch and assembled runtime/M2 acceptance remain open.

## Configured property client prerequisite

PR 101 accepted the private denial listener at merged source
`acb51f0405ad5d7db22e319b5024e19465d96c6e`. The next
[configured client](../../testing/M2/property-dial.md) joins trusted socket selection,
actual UID, original deadline and session-phase checks without runtime registration.
Signed source `dae7383aaf3d9518535254d281756f96c7c9507c` passed three portable parents/
seventeen children, all 104 transport cases on both disposable FreeBSD releases,
baseline/136 tooling tests and fresh full portable review (eleven race packages/
49 bounded fuzz targets in 220.066 seconds). The first signed denying-stub native
run remains unqualified; it also exposed a startup race in the prior replacement
fixture. That case now completes an actual exchange before replacement; product
listener bytes remain unchanged. The corrected signed red run retained all 92 old
cases and failed only the new configured parent. Both red records retain actual
sources and verified cleanup. All 672 owned sessions stopped with verified teardown;
Actions remain disabled and all 77 historical manifests retain their bytes.
See the [source-bound evidence](../../testing/M2/property-dial-manifest.json).
Root application authentication, backend/policy/plan, recovery/global launch,
durable admission and assembled M2 acceptance remain open.
