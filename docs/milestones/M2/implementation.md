# M2 review and implementation sequence

Date: 2026-10-06. Status: Implementing; approved design, accepted model subset; accepted native property subset; identity subset accepted; filesystem state subset accepted; pool observation subset accepted; feature subset accepted; topology subset accepted; virtual device fixture qualified on both targets through the Mac supervisor; fixture accepted in PR 28; native disk correlation accepted in PR 31; attachment classifier accepted in PR 32; system device-use subset accepted in PR 33; imported-pool use subset accepted in PR 34; filesystem signature subset accepted in PR 35; combined device-use subset accepted in PR 36; positive pool-label subset accepted in PR 37; disk-access subset accepted in PR 38; state-directory subset accepted in PR 39; ownership codec accepted in PR 40; native ownership reader accepted in PR 41; state ACL validation accepted in PR 42; ownership publication accepted in PR 43; bounded ownership catalog accepted in PR 46; ownership capacity accepted in PR 47; job lifecycle metadata accepted in PR 48; native job reader qualified, review pending; Actions disabled.
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
  merged source `4011e3d9848b2430c7ab30d7d70981a3e0fdd88c`: binds full dataset types/GUIDs to independent property
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

## Independent slice 2 prerequisite

The approved durable-state boundary can be implemented and qualified while
slice 1's device/impact/status gates remain open; this work consumes none of
those unqualified gates and cannot admit or launch storage jobs. The
[state-directory lease](../../testing/M2/state-directory.md) opens only an
existing private root-provisioned local UFS/ZFS directory, establishes a stable
single-writer lock and synchronizes its empty lock file and parent directory.
Thirty-five native assertions, separate process death/reacquisition and full
preservation/teardown passed on both releases at signed functional/suite source
`123044304e75f6df27d43eee085c23435ddbcce1`; see the
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
PR 41 accepted the reader at merged source `a6406670cb7804a1d15582a615114fefdbeac337`. It performs no file publication or runtime enrollment and
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
[manifest](../../testing/M2/job-read-manifest.json). Review is pending. Job-file
publication, complete durable
intent, mixed namespaces/budgets, admission/deduplication and actual recovery remain
open. Ownership publication still refuses all job files; no runtime caller exists.
