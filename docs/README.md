# Documentation index

## Product and design

- [Charter](project/charter.md)
- [Architecture baseline](architecture/baseline.md)
- [Architecture decisions](adr/README.md)
- [Roadmap](roadmap/milestones.md)
- [uPlot dashboard roadmap and build pickup](roadmap/dashboards.md)
- [UI baseline](ui/baseline.md)
- [Compatibility matrix](compatibility/matrix.md)
- [Ports distribution plan](../packaging/ports/README.md)
- M1 approved: [PRD](milestones/M1/PRD.md), [EDD](milestones/M1/EDD.md),
  [Test Design](milestones/M1/TDD.md), [implementation status](milestones/M1/implementation.md)
- M2 approved design / implementing: [PRD](milestones/M2/PRD.md), [EDD](milestones/M2/EDD.md),
  [Test Design](milestones/M2/TDD.md), [review/implementation sequence](milestones/M2/implementation.md)

## Engineering and evidence

- [Bootstrap status](engineering/bootstrap-status.md)
- [Git workflow](engineering/git-workflow.md)
- [CI/CD](engineering/ci-cd.md)
- [Complete local review gate](engineering/local-review.md)
- [CI cost controls](engineering/ci-costs.md)
- [Local Mac/FreeBSD qualification](testing/local-qualification.md)
- [Local runner execution manifest](testing/local-qualification-manifest.json)
- [Supervised teardown execution manifest](testing/local-teardown-manifest.json)
- [Dependency policy](engineering/dependency-policy.md)
- [Release strategy](engineering/release-strategy.md)
- [Document lifecycle](process/document-lifecycle.md)
- [Research process](process/research-process.md)
- [Testing strategy](testing/strategy.md)
- [Lab gates](testing/lab-gate.md)
- [M1 native discovery execution evidence](testing/M1/discovery.md)
- [M1 helper transport execution evidence](testing/M1/helper.md)
- [M1 collector/cache execution evidence](testing/M1/collector.md)
- [M1 administration security contracts/evidence](testing/M1/security.md)
- [M1 console and CLI contracts/evidence](testing/M1/console.md)
- [M1 foreground development runbook](operations/M1-development.md)
- [Security and supply chain](security/supply-chain.md)
- [Design templates](templates/README.md)
- [M0 foundation acceptance](milestones/M0/acceptance.md)
- [M1 research manifest](research/M1/manifest.md)
- M2 source evidence: [findings](research/M2/findings.md),
  [immutable source manifest](research/M2/source-manifest.json); its pre-approval
  status is historical. Current [native property evidence](testing/M2/properties.md)
  covers a narrow subset only
- M1 evidence: [findings](research/M1/findings.md),
  [native QEMU spike](research/M1/native-spike.md),
  [immutable source/version manifest](research/M1/source-manifest.json)
- [M2 planning model contracts and partial test evidence](testing/M2/planning.md)
- [M2 native property projection and qualification](testing/M2/properties.md)
- [M2 dataset identity binding and version drift](testing/M2/identities.md)
- [M2 filesystem mount and encryption constraints](testing/M2/dataset-state.md)
- [M2 vdev topology contracts and qualification](testing/M2/topology.md)
- [M2 feature/profile contracts and qualification](testing/M2/features.md)
- [M2 pool identity, import incarnation and policy observations](testing/M2/pool-state.md)
- [Operational runbook plan](operations/README.md)

- [M1 runtime lifecycle and observability evidence](testing/M1/runtime.md).

- [M1 combined-process performance qualification](testing/M1/performance.md).

- [M1 helper denial events and transport abuse evidence](testing/M1/helper-events.md).

- [M1 native probe process cleanup evidence](testing/M1/probe-cleanup.md).

- [M1 helper crash-containment procedure/evidence](testing/M1/crash-containment.md).

- [M1 native reader resources and integrated crash procedure](testing/M1/native-resources.md)

- [M1 assembled HTTP abuse qualification](testing/M1/http-abuse.md)

- [M1 independent SMB/NFS continuity qualification](testing/M1/file-continuity.md)

- [M1 browser qualification and manual accessibility checklist](testing/M1/browser-checklist.md)

- [Consolidated M1 qualification review and open acceptance gates](testing/M1/qualification-status.md)

- [M2 disposable virtual device identity fixture](testing/M2/devices.md).

- [M2 local device fixture qualification manifest](testing/M2/devices-local-manifest.json)

- [M2 native disk correlation](testing/M2/disk-correlation.md)
- [M2 disk correlation native execution manifest](testing/M2/disk-correlation-manifest.json)
- [M2 disk attachments and partition denial](testing/M2/disk-attachments.md)
- [M2 disk attachment native execution manifest](testing/M2/disk-attachments-manifest.json)
- [M2 system device-use observations](testing/M2/system-device-use.md)
- [M2 system device-use native execution manifest](testing/M2/system-device-use-manifest.json)
- [M2 imported-pool use observations](testing/M2/imported-pool-use.md)
- [M2 imported-pool use native execution manifest](testing/M2/imported-pool-use-manifest.json)
- [M2 filesystem signature observations](testing/M2/filesystem-signatures.md)
- [M2 filesystem signature native execution manifest](testing/M2/filesystem-signatures-manifest.json)
- [M2 combined device-use review](testing/M2/device-use-review.md)
- [M2 combined device-use native execution manifest](testing/M2/device-use-review-manifest.json)

- [M2 positive pool-label identity](testing/M2/pool-labels.md).

- [M2 disk access](testing/M2/disk-access.md): identity-bound native aggregate access and review drift; both releases and full teardown qualified, accepted in PR 38.

- [M2 state directory](testing/M2/state-directory.md): private directory and stable single-writer lock prerequisite; thirty-five native assertions and full teardown passed on both targets; accepted in PR 39.
- [M2 ownership records](testing/M2/ownership-records.md): strict bounded metadata codec; accepted in PR 40; no file store or durable admission.
- [M2 native ownership reads](testing/M2/ownership-read.md): bounded descriptor reads, object/identity refusal and unchanged bytes; both native targets qualified, accepted in PR 41.

- [M2 state ACL validation](testing/M2/state-acl.md): descriptor-based permission checks and disposable POSIX/NFSv4 controls; both native targets qualified; accepted in PR 42.

- [M2 ownership publication](testing/M2/ownership-publication.md): ordered private metadata publication, revision checks and uncertainty; both native targets passed 161 assertions and complete teardown, accepted in PR 43.

- [M2 bounded ownership catalog](testing/M2/ownership-catalog.md): complete bounded read-only record inspection, orphan refusal and ordinary drift; both targets passed all 202 assertions and complete teardown, accepted in PR 46.

- [M2 ownership publication capacity](testing/M2/ownership-capacity.md): complete namespace validation, transient-byte/count reservation and final rechecks; both targets passed all 234 assertions and complete teardown, accepted in PR 47.

- [M2 job lifecycle metadata](testing/M2/job-records.md): bounded lifecycle/strict codec and restart projections without replay; seven tests/119 schema subtests and focused storage race/fuzz passed, accepted in PR 48.

- [M2 native job reader](testing/M2/job-read.md): private ID-selected reads, real ACL/UID gates and immutable successor guard; both targets passed all 275 assertions and full teardown, accepted in PR 49.

- [M2 mixed state catalog](testing/M2/state-catalog.md): complete ownership/job census, shared byte and class ceilings; both targets passed all 336 assertions and complete teardown, accepted in PR 50.

- [M2 durable job publication](testing/M2/job-publication.md): current predecessor and immutable history checks, shared publication capacity and uncertainty; both targets passed all 432 assertions and teardown; accepted in PR 51.

- [M2 reviewed property intent](testing/M2/property-intent.md): immutable complete current property-review capture and strict bounded codec; seven parents/769 subtests and focused storage race/fuzz passed; accepted in PR 52.

- [M2 property job binding](testing/M2/property-jobs.md): single bounded baseline/lifecycle record and immutable successors; six parents/859 subtests and focused storage race/fuzz passed; accepted in PR 53.

- [M2 native property-job store](testing/M2/property-job-storage.md): full baseline reads/publication, mixed capacity and format-preserving successors; both targets passed all 555 assertions and teardown; accepted in PR 54.

- [M2 job reservation checks](testing/M2/job-reservations.md): immutable complete job-record view, exact retry meaning, reused-plan refusal and conservative whole-pool holds; portable qualification passed, accepted in PR 55.

- [M2 native job reservations](testing/M2/native-job-reservations.md): complete root-leased mixed census and bound property retry/reservation views; both targets passed all 622 assertions and teardown; accepted in PR 56.

- [M2 reserved property submissions](testing/M2/property-submission.md): one lease/mutex across complete retry/pool checks and synced initial publication; both targets passed all 710 assertions and teardown; accepted in PR 57; runtime writes disabled.

- [M2 bounded job restart projections](testing/M2/job-restart.md): immutable mixed-job pages, conservative interrupted-work presentation and unchanged holds; portable baseline/storage race/all twenty storage fuzz targets passed; accepted in PR 58; no startup replay/recovery consumer.

- [M2 metadata process-death cuts](testing/M2/property-publication-cuts.md): eight real writer SIGKILL boundaries, preserved uncertain state and native restart pages; both targets passed all 777 state/backing assertions and full teardown; accepted in PR 59; no native mutation/replay.
- [M2 property review/submission coordinator](testing/M2/property-coordinator.md): bounded volatile reviews, fresh baseline checks and one metadata publication attempt; portable baseline/storage race/all 21 fuzz targets passed; accepted in PR 60; no native backend/runtime writes.
- [M2 native property observation collection](testing/M2/native-property-collection.md): fixed FreeBSD projections, complete dual-pass drift checks and zero write eligibility; both targets passed all 16 assertions, full portable review and teardown; accepted in PR 61.
- [M2 root policy/native binding](testing/M2/property-policy-binding.md): GUID-only collection through the held ownership reader, repeated canonical policy checks and complete scope binding; both targets passed all 22 assertions, complete portable review and teardown; accepted in PR 62; eligibility remains false.
- [M2 native property system exclusions](testing/M2/property-system-exclusions.md): complete imported-pool/mount/swap backing and property topology binding, with system/conflict denials; both targets passed all 32 assertions, complete portable review and snapshot-disk teardown; accepted in PR 63; eligibility remains false.

- [M2 native provider binding](testing/M2/property-provider-binding.md): complete leaf serial/geometry and qualified attachment correspondence; both targets passed all 37 assertions, complete portable review and teardown; accepted in PR 64; no enrollment or write eligibility.

- [M2 explicit property member enrollment](testing/M2/property-member-enrollment.md): version 2 member baselines, legacy read compatibility and exact native matching; both targets passed all 46 assertions, complete portable review and teardown; accepted in PR 65; eligibility remains false.

- [M2 property store readiness](testing/M2/property-store-readiness.md): complete private policy/job census, actual system backing and retained recovery holds; both targets passed all 56 assertions, complete portable review and teardown; accepted in PR 66; no write eligibility.

- [M2 explicit enrollment review](testing/M2/property-enrollment-review.md): immutable expiring explicit root proposal and fresh native/system drift checks; both targets passed all 64 assertions, complete portable review and teardown; accepted in PR 67; no publication or write eligibility.

- [M2 root enrollment state binding](testing/M2/property-enrollment-state.md): explicit first/exact predecessor, complete metadata/backing baseline, projected capacity and retained holds; both targets passed all 76 assertions, complete portable review and teardown; accepted in PR 68; no publication or write eligibility.

- [M2 native enrollment source](testing/M2/native-enrollment-source.md): owned explicit selections, fresh pool/provider/system collection and original-lease revalidation; both targets passed all 84 assertions, complete portable review and teardown; accepted in PR 69; no publication or write eligibility.

- [M2 reviewed enrollment metadata publication](testing/M2/enrollment-publication.md): shared one-attempt claims, complete original-lease rechecks and actual clock/context guards around atomic metadata publication; both targets passed 819 state/backing and 88 native pool assertions with complete portable review and teardown; accepted in PR 70; runtime writes disabled.

- [M2 independently collected enrollment publication](testing/M2/native-enrollment-publication.md): sealed root reviews, independent final native inputs and unchanged metadata acknowledgements; both targets passed all 88 assertions, focused portable review and teardown; accepted in PR 71.

- [M2 foreground storage job status](testing/M2/storage-job-status.md): root-leased complete retained summaries, conservative restart status and explicit unverified recovery; accepted in PR 72. [Runbook](operations/M2-job-status.md).

- [M2 reviewed property postconditions](testing/M2/property-postconditions.md):
  exact local effects, inheritance/received context and conservative matching.

- [M2 retained property-job observation assessment](testing/M2/property-assessment.md)

- [M2 root-leased native property-job assessment](testing/M2/native-property-assessment.md)

- [M2 property-job assessment runbook](operations/M2-property-assessment.md).
- [M2 property-job assessment diagnostic qualification](testing/M2/property-assessment-diagnostic.md).
- [M2 pool health and scan observation qualification](testing/M2/pool-health.md).
- [M2 native scrub-state observation qualification](testing/M2/scan-lifecycle-observations.md).
- [M2 private native property control](testing/M2/property-control.md)
- [M2 native property-control execution manifest](testing/M2/property-control-manifest.json)
- [M2 original admission execution handoff](testing/M2/property-handoff.md)
- [M2 original admission handoff execution manifest](testing/M2/property-handoff-manifest.json)
- [M2 property execution sequence](testing/M2/property-execution.md)
- [M2 native repaired-checksum finding qualification](testing/M2/checksum-findings.md).

- [M2 original journaled publication](testing/M2/journaled-publication.md): private
  acknowledged dual-lease metadata transaction qualified on both pinned FreeBSD
  releases with full local portable review and verified teardown;
  commissioning, registered startup/global admission and runtime writes remain pending.
