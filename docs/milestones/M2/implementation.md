# M2 review and implementation sequence

Date: 2026-10-04. Status: Implementing; approved design, accepted model subset; accepted native property subset; identity subset accepted; filesystem state subset accepted; pool observation subset accepted; feature subset implementing.
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
- M2 native mutation/state/failure/browser/resource execution: not started.
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
- Slice 1 feature subset requires a complete GUID-bound native feature table and
  immutable drift review; [feature contracts](../../testing/M2/features.md).
  Exact common profile bytes are checked without a product file reader or creation
  authority. Native execution pending; cross-release transfer/import remains open.
- Slice 1 remains incomplete: native eligibility/provider/topology/status
  observations, complete-impact collection and device-use drift gates have not run. No later slice may treat
  model input booleans or old M1 inventory as qualified native write authority.

The approval adopts scope and gates, not passing native behavior or a NAS release.
Record actual slice results here and in the compatibility matrix; skipped fixtures
are never acceptance evidence. Full M1 acceptance remains a separate pending decision.
