# Milestone roadmap

Detailed PRD/EDD/Test Design documents are written when a milestone becomes active.
M0 was accepted in [PR 2](https://github.com/gonz0w/horvium/pull/2).
M1 design was approved in [PR 3](https://github.com/gonz0w/horvium/pull/3):
[PRD](../milestones/M1/PRD.md), [EDD](../milestones/M1/EDD.md),
[Test Design](../milestones/M1/TDD.md). See [implementation status](../milestones/M1/implementation.md)
for reviewed slices and remaining acceptance gates.
Live status, priority and dependencies are in the [shared private Project](https://github.com/users/gonz0w/projects/2)
and [stable roadmap index](github-roadmap-index.md).
M2 design was approved by maintainer merge of
[PR 20](https://github.com/gonz0w/horvium/pull/20): [PRD](../milestones/M2/PRD.md),
[EDD](../milestones/M2/EDD.md), [Test Design](../milestones/M2/TDD.md) and
[historical implementation evidence](../milestones/M2/implementation.md).
That authorization did not waive M1's manual acceptance criteria.
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

## Planned dashboard work

The maintainer selected uPlot on 2026-10-10 for progressive charting. The
[dashboard roadmap](dashboards.md) defines the operator data story, eight views,
BSD Classic visual/color direction, drill-down, data prerequisites and build-chat
pickup sequence. Delivery spans existing milestones as qualified sources become
available. The initial focus is Overview and Storage capacity; richer metrics,
sharing and protection follow their approved contracts. Implementation is deferred
until a reviewed PRD/EDD/Test Design packet and dependency/integration ADR exist.
This roadmap does not change current M1/M2 acceptance or authorize new telemetry.

## Proposed second release security work

The [second release security roadmap](v2-security.md) proposes stronger authority,
independent backup protection, encryption/key recovery, service confinement and
recovery evidence against external attackers, including AI-enabled attackers.
It also proposes a hardened FreeBSD image within M7 using the tested package and
a [deployment security guide](../security/deployment-security.md) for software,
image and operator responsibilities. Basic M3-M6 security, backup independence
and restore obligations remain first-release work. This proposal assigns no release
tag/date, changes no acceptance gate and requires reviewed implementation packets.
