# M2 review and implementation sequence

Date: 2026-10-04. Status: Draft design; implementation not authorized by this record.
Owner: Cam (`@gonz0w`). [Issue 19](https://github.com/gonz0w/horvium/issues/19).
[PRD](PRD.md), [EDD](EDD.md), [Test Design](TDD.md).

M1 automated work and the qualification record are merged. M1 is Validating;
Safari/manual accessibility and explicit milestone acceptance remain pending.
M2 design can be reviewed while those observations are pending. M2 product work
requires approval of the linked documents and proposed ADRs under
[the repository instructions](../../../AGENTS.md).

## Proposed slices and acceptance boundaries

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

## Current evidence and required approval

- Primary source inspection and SHA-256 capture: complete for the documented M2 questions.
- M2 native mutation/state/failure/browser/resource execution: not started.
- Physical MicroServer/commercial/controller/client qualification: not supplied.
- M2 PRD/EDD/Test Design and ADRs 0006/0007: proposed, approval pending.

Approval must identify the maintainer/date/merged design source. It adopts the
documented scope and gates, not passing native behavior or a NAS release. Record
actual slice results here and in the compatibility matrix; skipped fixtures are
never acceptance evidence. Full M1 acceptance remains a separate pending decision.
