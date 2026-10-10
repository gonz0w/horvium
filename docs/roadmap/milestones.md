# Milestone roadmap

Detailed PRD/EDD/Test Design documents are written when a milestone becomes active.
M0 was accepted in [PR 2](https://github.com/gonz0w/horvium/pull/2).
M1 design was approved in [PR 3](https://github.com/gonz0w/horvium/pull/3):
[PRD](../milestones/M1/PRD.md), [EDD](../milestones/M1/EDD.md),
[Test Design](../milestones/M1/TDD.md). See [implementation status](../milestones/M1/implementation.md)
for reviewed slices and remaining acceptance gates.
M1's automated slices and [qualification record](../testing/M1/qualification-status.md)
are merged; M1 is Validating with Safari/manual accessibility and explicit milestone
acceptance pending. M2 design was approved by maintainer merge of
[PR 20](https://github.com/gonz0w/horvium/pull/20) on 2026-10-04: [PRD](../milestones/M2/PRD.md),
[EDD](../milestones/M2/EDD.md), [Test Design](../milestones/M2/TDD.md) and
[implementation sequence](../milestones/M2/implementation.md). M2 is Implementing;
its first scope/planning model does not waive M1's pending gates.
The first deployable release spans M1-M6; intermediate evidence is not a production
claim. Each milestone links requirements, tests, and compatibility evidence.

| Milestone | Outcome | Required proof |
|---|---|---|
| M0 Foundation | Charter, governance, design templates, repository checks, Git/CI policy | Local gates; remote protections and CI when destination is confirmed |
| M1 Discovery and design | Version pins, host discovery, read-only console, management/security design | Native FreeBSD discovery; permissions/security denial tests; UI accessibility; bounded inventory |
| M2 Storage lifecycle | Supported pools/datasets, adoption, quotas, health, scrubs, safe job execution | Disposable ZFS integration; drift/partial failure; replacement/reboot recovery |
| M3 Sharing and identities | Local users/groups, SMB/NFS/SFTP configuration | Real client access/denial, ACL/locking compatibility, validated config/reload |
| M4 Protection | Snapshots, peer registration, incremental replication, retention, restore | Interrupted/resumed sends, identity mismatch, receiver full/offline, independent history, restored file evidence |
| M5 Port/package | Native build/install/service integration and upgrade/removal | Clean poudriere builds; staging; install/upgrade/deinstall preserve operator data/config |
| M6 Release qualification | Complete NAS workflow, hardware/performance/security runbooks | Small reference machine, representative larger configuration, restore and upgrade gates |
| M7 Appliance image | Image using the same tested package and behavior | Boot/install/update/recovery on supported hardware |

AD, fleet management, and automatic failover need demonstrated demand and separate
designs after the first release. Packaging research begins in M1 and build metadata
is introduced when a real source artifact exists; M5 is its release qualification,
not permission to ignore port constraints until then.

The [original fresh journal activation prerequisite](../testing/M2/fresh-journal-activation.md)
was accepted in PR 122. It joins exclusive fresh domains to an initial durable
journal receipt. The subsequent [fixed native composition](../testing/M2/native-journal-activation.md)
was accepted in PR 123 after both pinned host and affected state safety gates.
The [original unused activation close](../testing/M2/initial-activation-retirement.md)
was accepted in PR 124 after both pinned native targets and local review.
The [registered fresh commissioning owner](../testing/M2/commissioned-execution-owner.md)
was accepted in PR 125 after both pinned host gates and local review.
It permits one commissioning attempt per helper process before provisioning;
failure or close cannot start another ledger lifetime in that process.
The [original closed-incarnation startup review](../testing/M2/execution-startup-review.md)
was accepted in PR 126 after both pinned native targets passed 186 controls each.
It retains original journal and directory/lock/mount observations under the same
process registration, without publishing a new active incarnation.
The [original closed-startup active publisher](../testing/M2/closed-startup-activation.md)
passed both pinned native targets: 268 durable publication/cut controls and 362
fixed-root controls, including all 186 retained startup controls; accepted in
PR 127. Only a confirmed empty ledger can receive this new original
active lifetime; the old review transfers its leases once and cannot release it.
The [original unused restarted-lifetime close](../testing/M2/restart-activation-retirement.md)
passed all 938 affected publication/root/startup controls on both pinned targets,
with clean portable review and teardown; it awaits maintainer review. This closes
only the unused original incarnation; general worker drain remains open.
Nonempty history, configured startup/restart, journaled publication integration,
global native launch, worker draining, backend/supervisor assembly and remaining
property runtime
acceptance precede write enablement; M2 remains Implementing.
