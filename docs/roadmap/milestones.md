# Milestone roadmap

Detailed PRD/EDD/Test Design documents are written when a milestone becomes active.
M0 was accepted in [PR 2](https://github.com/gonz0w/horvium/pull/2).
M1 design was approved in [PR 3](https://github.com/gonz0w/horvium/pull/3):
[PRD](../milestones/M1/PRD.md), [EDD](../milestones/M1/EDD.md),
[Test Design](../milestones/M1/TDD.md). See [implementation status](../milestones/M1/implementation.md)
for reviewed slices and remaining acceptance gates.
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
