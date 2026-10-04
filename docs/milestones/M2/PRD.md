# PRD: safe storage lifecycle

Status: Implementing; design approved. Owner: Cam (`@gonz0w`).
Date: 2026-10-04. Work: [issue 19](https://github.com/gonz0w/horvium/issues/19).
[EDD](EDD.md), [Test Design](TDD.md), [research](../../research/M2/findings.md).
Approved by Cam (`@gonz0w`) on 2026-10-04 through maintainer merge of
[PR 20](https://github.com/gonz0w/horvium/pull/20), source
`753f601960529cd17596d0ffc628afe38d041c82`. Design approval authorizes the
reviewed slices; it is not M2 acceptance or native qualification.

## Problem, users and outcomes

An operator needs to turn discovered storage into explicitly managed storage
without losing existing data or mistaking a submitted command for a completed
operation. The same application must handle a small home server and multiple
pools on larger FreeBSD hosts. Four bays, disk sizes, controller names and 16 GB
memory are reference facts, not product assumptions.

M2 adds a foreground development workflow for reviewed ZFS changes. The operator
can register ownership of eligible existing resources, create supported redundant
pools and filesystems, manage capacity limits, inspect health, scrub and replace
a redundant pool member. Every change has a review, fresh preconditions, durable
job status and a recovery outcome. Existing file daemons remain independent.

M1's automated implementation and qualification records are merged. Its
[Safari/manual accessibility gate](../../testing/M1/browser-checklist.md) and
explicit milestone acceptance remain pending. Implementing M2 does not waive them.
M2 approval authorizes its test-first slices; M1 acceptance and a NAS release
remain separate. Packaging/service installation and release qualification remain M5/M6.

## Approved operation scope

| Operation | Approved boundary |
|---|---|
| Adoption | Register an already imported eligible pool by GUID and individually selected filesystems; preserve data, properties, mount/import policy and unrelated descendants |
| Pool creation | Explicitly cleared-for-use direct devices with unique verified identity; homogeneous mirror groups or homogeneous RAIDZ2 groups; operator chooses layout and sector policy |
| Filesystem creation | Under an owned parent; separately reviewed mount beneath a locally approved namespace; no volumes or implicit missing-parent creation |
| Property changes | Quota, reference quota and compression (`off`/`lz4`) only; explicit byte limits or no-limit choice, current effective/local/inherited values and descendant impact shown |
| Health | Pool/vdev identity, health, error counters, scrub/resilver state and honest unavailable/unsupported observations; SMART integration is a separate future design |
| Scrub | Start and explicitly stop an attributable scrub on an owned pool; completion and checksum findings are separate results |
| Replacement | A particular leaf vdev GUID in a supported redundant layout, using a distinct verified unused device; track resilver through observation |
| Reimport | Explicit recovery of a previously Horvium-created pool by registered GUID and verified devices, without force or mounts; adopted pools retain their OS/operator import policy |

Unsupported existing layouts remain observable. Encryption/key loading, automatic
feature upgrades, vdev expansion/add/remove/attach/detach, log/cache/special/spare
management, dRAID, shared block devices, arbitrary properties, disk partitioning,
wiping, force/rewind import and pool/dataset destruction are outside this scope.
Deletion needs a separate reviewed extension covering protection dependencies and
irreversible outcomes. M4 remains responsible for snapshots/replication/restore;
M3 for users, ACLs and shares. M2 must not change their host configuration.

## Requirements and acceptance

| Requirement | Operator-visible outcome | Acceptance tests |
|---|---|---|
| M2-REQ-001 Ownership | Observation grants no authority; explicit pool/dataset GUID scopes distinguish owned, external, missing and drifted resources | M2-OWN-001, M2-DRIFT-001 |
| M2-REQ-002 Device safety | Deny uncertain/duplicate identity, alias overlap, boot/swap/mounted/partitioned/in-use devices and unsupported sector geometry; a reused device path is never identity | M2-DEVICE-001 |
| M2-REQ-003 Review and authorization | UI/CLI show the same immutable plan, affected identities and consequences; helper independently checks local write scope and fresh state; changed/expired plans require a new review | M2-PLAN-001, M2-PRIV-001, M2-HTTP-001 |
| M2-REQ-004 Durable jobs | Persist intent before mutation; duplicates return the same job; restart, ambiguous outcome and cancellation never silently replay a write | M2-JOB-001, M2-RECOVERY-001, M2-STATE-001 |
| M2-REQ-005 Creation and properties | Create the selected supported pool/filesystem; preserve unrelated storage; verify GUID/topology/properties/mount results; capacity semantics and partial application are explicit | M2-POOL-001, M2-DATASET-001, M2-PROPERTY-001, M2-FEATURE-001 |
| M2-REQ-006 Health and maintenance | Show actual pool/vdev errors and scan progress; scrub and replacement remain running until independently verified terminal outcomes; no invented progress/ETA or cancellation | M2-HEALTH-001, M2-SCRUB-001, M2-REPLACE-001 |
| M2-REQ-007 Drift and recovery | External rename/property/topology changes suspend affected writes; reboot/missing pool/store failure preserve recoverable intent; never repair or take ownership automatically | M2-DRIFT-001, M2-RECOVERY-001, M2-STATE-001 |
| M2-REQ-008 Independent file serving | Management/helper restart and maintenance do not stop/reconfigure unrelated file daemons; supported target maintenance preserves fixture file content | M2-CONTINUITY-001 |
| M2-REQ-009 Usable bounded operation | Semantic HTML5/CSS/local HTMX and CLI retain parity, focus and honest state; bounded plans/history/concurrency keep cached views responsive | M2-UI-001, M2-SCALE-001, M2-PERF-001 |
| M2-REQ-010 Reproducible evidence | Test real FreeBSD facilities on disposable fixtures; document exact versions, operation limits, evidence, restoration of job metadata and incompatible state versions | M2-NATIVE-001, M2-STATE-001 |

## Compatibility and budgets

Candidates remain FreeBSD 15.1/14.5 amd64, Go 1.27.1, x/sys v0.48.0 and embedded
HTMX 2.0.11. [Source inspection](../../research/M2/source-manifest.json) records
bundled OpenZFS differences. Refresh actual kernel/userland/module versions and
capabilities when tests execute. No additional product dependency is proposed.

For new pools, qualify the common `openzfs-2.2` feature profile after installed
file/hash and cross-release native qualification. This limits new feature
activation; it is not proof of backup-peer compatibility or universal portability.
No upgrade of adopted pools is allowed. Sector alignment is reviewed, derived
from actual geometry and checked against native limits; do not copy a lab value
into all hardware.

Keep M1's measured workload definitions and proposed 128 MiB small / 256 MiB large
combined Go RSS, less than 1% of one core small idle CPU and cached-page p95 below
250 ms for ten readers. Measure again with job/store activity. These budgets
exclude ZFS ARC, native commands, file daemons and disk throughput. Report their
separate measurements/limitations; neither QEMU nor synthetic scale qualifies
the MicroServer or commercial hardware.

Proposed bounded defaults: one native mutation launch at a time, four outstanding
kernel-maintenance jobs on distinct pools, 64 admitted nonterminal jobs, 64 KiB
plan/job records, 1,000 retained terminal jobs and 64 MiB durable metadata. One
pool serializes its changes; reads continue. The EDD defines safe exhaustion and
validated tuning. Larger deployments can tune tested safety budgets rather than
receiving hardcoded hardware assumptions or unbounded queues.

WCAG 2.2 AA remains the UI target, with Safari/macOS and Firefox/Linux selected.
New review/job workflows need actual keyboard/screen-reader/native-zoom findings.
M1's pending manual observations cannot be counted as M2 passes.

## Review and dependencies

The linked EDD/Test Design were approved together with the
[ownership/authorization ADR](../../adr/0006-storage-ownership-and-plans.md) and
[durable job ADR](../../adr/0007-durable-storage-jobs.md). The conservative operation
scope, common creation feature profile, local write enrollment and bounded JSON
ledger are approved choices. Native operations and the ledger remain unimplemented;
see the [slice record](implementation.md) for current product behavior.

Native gates must settle actual status parsing, device-use checks, directory sync,
mount collision handling, feature profile installation and interruption recovery
before the corresponding slice can be accepted. A failing capability blocks that
operation and requires a reviewed design change; it never causes a force fallback.
Exact CLI flags, HTTP routes, wire fields and on-disk field names are finalized
with the approved contracts and their implementation tests, not invented here.
