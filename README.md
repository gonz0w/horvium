# Horvium

**Dependable storage on FreeBSD.**

Horvium is a dedicated NAS management system for FreeBSD and ZFS, designed to
remain lightweight on a 16 GB home server and grow to larger commercial storage
systems. The HP MicroServer Gen8 is a reference configuration, not a hardware
requirement or a limit on disks, memory, pools, or datasets.

**Status:** M0 foundation and the main M1 read-only discovery/management slices
and crash containment, native resources, HTTP abuse recovery, independent SMB/NFS
continuity and automated Firefox qualification are accepted. The consolidated
qualification record was accepted in PR 18. M1 is Validating; Safari/manual
accessibility acceptance remains pending. See the
[M1 qualification review](docs/testing/M1/qualification-status.md) for evidence and
remaining gates. There is no installable port, appliance image or production
support claim.

The next M2 work follows the accepted
[property runtime delivery contract](docs/milestones/M2/property-runtime.md):
qualify executor recovery, integrate startup and global launch gates, then assemble
root admission/workers and the shared HTML/CLI workflow. PR 105 accepted the
contract and bounded experiment; the current runtime remains read-only. The
[executor recovery observations](docs/research/M2/executor-recovery.md) passed on
both disposable targets and demonstrated that singleton lock acquisition can
coexist with a surviving original executor. The
[accepted recovery mechanism](docs/milestones/M2/property-recovery-mechanism.md)
specifies a dedicated native supervisor, independent execution journal and
conservative restart/restore gates. PR 107 accepted its design and test plan;
the [private supervisor](docs/testing/M2/property-supervisor.md) is the first
implementation prerequisite. Its 42 focused cases passed on both disposable
FreeBSD targets with verified teardown; the runtime remains denied.
PR 108 accepted that prerequisite. The [journal record/census core](docs/testing/M2/execution-journal.md)
implements bounded portable metadata and single-step checks; native persistence,
provisioning/startup/global ownership and integrated publishers remain required.
PR 109 accepted the portable core. The [native ledger census](docs/testing/M2/native-execution-census.md)
connects its digest to actual descriptor-validated records, preserving publication
uncertainty. Its focused assertions and affected state safety corpus passed on both
disposable FreeBSD/amd64 targets with verified teardown. Independent journal
persistence/bindings and startup/global ownership
remain required; this inspection cannot enable a writer.
PR 110 accepted that native observation prerequisite. The next
[private journal persistence component](docs/testing/M2/execution-journal-persistence.md)
adds a pre-existing journal lease and original durable file receipts. Both disposable FreeBSD targets passed the file and process-death gates with
verified teardown. Persistent bindings, startup/global ownership and integrated
runtime ordering remain required; PR 111 accepted this prerequisite.
The [binding representation and native object observer](docs/testing/M2/execution-bindings.md)
adds private persistent-identity framing and original descriptor observations.
Guarded persistent backing discovery, fresh commissioning, dual-lease ownership
and startup/global integration remain required. Both pinned disposable FreeBSD
targets passed the object and UFS remount gates with verified teardown;
PR 112 accepted this prerequisite.
The [guarded persistent backing collector](docs/testing/M2/persistent-backing.md)
adds private native UFS/ZFS identity discovery. Both disposable FreeBSD targets
passed the focused identity/remount/rename and complete teardown gates;
PR 113 accepted this prerequisite.
Registered dual-lease ownership, system exclusion, restore independence and commissioning
remain required.
The [original-lease backing observation](docs/testing/M2/original-backing.md)
connects the fixed native producer to the original protected state lease under its
mutex. Both pinned native targets and the affected state safety corpus passed;
PR 114 accepted this prerequisite. Dual-domain ownership, complete system
exclusion, restore independence and commissioning/startup/global clearance remain
required.
The [joint journal/ledger lease owner](docs/testing/M2/joint-execution-leases.md)
adds journal-first acquisition and simultaneous original protected observations.
Both pinned native targets and the expanded state safety corpus passed with
verified teardown; PR 115 accepted this prerequisite. Registered writer ownership,
complete system/owned-pool exclusion, restore independence, commissioning,
publication and startup/global clearance remain required; runtime writes stay denied.
The [execution-domain system backing path](docs/testing/M2/execution-system-backing.md)
adds complete repeated system/owned-pool checks under both original leases.
Both pinned native targets and the affected state safety corpus passed with
verified teardown; PR 116 accepted this prerequisite. Unresolved owned backing denies,
and no commissioning, independent restore or launch authority is added.
The [publication snapshot correction](docs/testing/M2/publication-preservation.md)
resolves a reproduced test allocation-accounting failure while retaining exact
object/content checks. Both native state gates passed; PR 117 accepted the correction.
Production behavior and runtime write denial remain unchanged.
The [independent ledger restore fixtures](docs/testing/M2/independent-ledger-restore.md)
exercise UFS file-level restoration and distinct ZFS dataset rollback against a
preserved disarmed journal. Both pinned native targets passed with complete
teardown; PR 118 accepted the fixture prerequisite. Commissioning,
startup/global ownership and the operator restore workflow remain required.
The [original journaled metadata publisher](docs/testing/M2/journaled-publication.md)
connects the private original journal receipt to the original ledger under both
leases. Both pinned FreeBSD releases passed all native gates, with local portable
review and verified teardown; PR 120 accepted the prerequisite. Commissioning,
registered startup/global ownership and runtime writes remain pending.
The [exclusive fresh-domain provisioning component](docs/testing/M2/fresh-execution-domains.md)
creates both protected namespaces without adopting existing empty directories.
Both pinned native state gates and local portable review passed with verified
teardown; journal activation and registered startup/global ownership
remain required before runtime writes.
Completing this sequence covers the property workflow rather
than the entire NAS release.

M2 storage lifecycle design is approved in [PR 20](https://github.com/gonz0w/horvium/pull/20): [PRD](docs/milestones/M2/PRD.md),
[EDD](docs/milestones/M2/EDD.md), [Test Design](docs/milestones/M2/TDD.md) and
[implementation sequence](docs/milestones/M2/implementation.md). The first
model subset was accepted in PR 21; bounded property-source parsing passed on
both disposable FreeBSD targets and was accepted in PR 22. Dataset identity/type
binding and version drift were accepted in PR 23. Filesystem mount/encryption
constraints were accepted in PR 24. Pool identity/incarnation and policy
observations were accepted in PR 25. Complete native feature observations and
exact common profile checks were accepted in PR 26. Native vdev membership,
layout refusal and topology drift checks were accepted in
[PR 27](https://github.com/gonz0w/horvium/pull/27). See the
[topology qualification](docs/testing/M2/topology.md).
The [virtual device fixture](docs/testing/M2/devices.md) passed its eight native
identity/geometry assertions and complete teardown on both targets;
[PR 28](https://github.com/gonz0w/horvium/pull/28) accepted the fixture.
The [native disk correlator](docs/testing/M2/disk-correlation.md) passed all twelve
assertions and complete teardown on both targets and was accepted in
[PR 31](https://github.com/gonz0w/horvium/pull/31). The
[attachment classifier](docs/testing/M2/disk-attachments.md) distinguishes exact
DEV/disk-ident correspondence and refuses PART/unsupported attachments; native
partition-denial qualification passed all seventeen assertions and full teardown
on both targets and was accepted in [PR 32](https://github.com/gonz0w/horvium/pull/32).
The [system device-use subset](docs/testing/M2/system-device-use.md) maps
root, mount and swap observations to their direct GEOM disks; native UFS/swap
qualification passed all twenty-three assertions and complete teardown on both
targets and was accepted in [PR 33](https://github.com/gonz0w/horvium/pull/33).
The [imported-pool use subset](docs/testing/M2/imported-pool-use.md) maps
all imported pools to their direct GEOM disks, including pools with no mounted
filesystems; native qualification passed all twenty-nine assertions and complete
teardown on both targets and was accepted in [PR 34](https://github.com/gonz0w/horvium/pull/34).
The [filesystem signature subset](docs/testing/M2/filesystem-signatures.md)
binds positive base-probe UFS/ZFS observations to freshly correlated disks;
native qualification passed all thirty-six assertions and complete teardown on
both targets and was accepted in [PR 35](https://github.com/gonz0w/horvium/pull/35).
The [combined device-use subset](docs/testing/M2/device-use-review.md) retains
selected positive use facts in an immutable baseline and requires new review on
identity/geometry/mount/swap/pool/signature drift. All fifty-one assertions and full
teardown passed on both targets; PR 36 accepted this subset.
The accepted [pool-label subset](docs/testing/M2/pool-labels.md) binds positive exported
pool identities and recorded state to the same disk; all fifty-nine assertions
and full teardown passed on both targets; accepted in PR 37.
The accepted [disk-access subset](docs/testing/M2/disk-access.md) binds aggregate
GEOM access counters to disk identity and invalidates review on access drift;
all sixty-eight native assertions and full preservation/teardown passed on both
targets; accepted in PR 38.
The accepted [state-directory subset](docs/testing/M2/state-directory.md) establishes
the root-owned single-writer boundary for future durable metadata. All thirty-five
native assertions and complete preservation/teardown passed on both targets;
accepted in PR 39.
The accepted [ownership record codec](docs/testing/M2/ownership-records.md) validates
bounded versioned metadata and checksums without granting enrollment or writing
files. Five behavioral tests, fifty-seven subtests and all twenty-one bounded fuzz
targets passed; accepted in PR 40.
The [native ownership reader](docs/testing/M2/ownership-read.md) binds those
records to safe opened file objects and the held state directory. All sixty-three
lease/read assertions and complete teardown passed on both targets; accepted in
PR 41. Default-ACL disposable fixtures define the qualified scope.
The accepted [state ACL validation](docs/testing/M2/state-acl.md) checks actual opened
objects before trusting mode bits. All 108 expanded assertions and complete
teardown passed on both targets; accepted in PR 42.
The [ownership publication prerequisite](docs/testing/M2/ownership-publication.md)
adds private metadata publication and explicit uncertainty after failed durability.
All 161 assertions and complete teardown passed on both targets; accepted in PR 43.
The [bounded ownership catalog](docs/testing/M2/ownership-catalog.md) passed all 202 assertions
and complete teardown on both targets; accepted in PR 46. The
[ownership capacity guard](docs/testing/M2/ownership-capacity.md) passed all 234 assertions
and complete teardown on both targets; accepted in PR 47. The
[job lifecycle codec](docs/testing/M2/job-records.md) passed portable qualification;
accepted in PR 48. The [native job reader](docs/testing/M2/job-read.md) and
immutable successor guard passed all 275 assertions and full teardown on both
targets; accepted in PR 49. The [mixed state catalog](docs/testing/M2/state-catalog.md)
passed all 336 state assertions and complete teardown on both targets;
accepted in PR 50. The [durable job publisher](docs/testing/M2/job-publication.md)
passed all 432 state assertions and full teardown on both targets; accepted in
PR 51. The [reviewed property intent](docs/testing/M2/property-intent.md) capture/codec
passed portable qualification; accepted in PR 52. The
[property job binding](docs/testing/M2/property-jobs.md) combines that baseline
with immutable lifecycle metadata and passed portable qualification; accepted in
PR 53. The [native property-job store](docs/testing/M2/property-job-storage.md)
passed all 555 state assertions and complete teardown on both targets; accepted
in PR 54. The [job reservation checks](docs/testing/M2/job-reservations.md) add
bounded duplicate/plan/pool metadata checks; accepted in PR 55. The
[native reservation inspector](docs/testing/M2/native-job-reservations.md) connects
that view to the held root state lease; both targets passed all 622 assertions
and full teardown; accepted in PR 56. Runtime admission
remains disabled.
Runtime enrollment and full store/recovery gates remain open.
Other native gates, runtime job admission and storage writes remain pending; the runtime is
still read-only.

The [reserved submission publisher](docs/testing/M2/property-submission.md)
combines complete retry/pool checks and initial record publication under one
writer lock. Both targets passed all 710 assertions and teardown; accepted in
[PR 57](https://github.com/gonz0w/horvium/pull/57). Executable admission remains disabled.

The [bounded restart projection](docs/testing/M2/job-restart.md) passed portable qualification and was accepted in
[PR 58](https://github.com/gonz0w/horvium/pull/58); it presents interrupted work as requiring reconciliation while retaining durable
history and pool holds. Actual native recovery remains pending.

The [metadata process-death qualification](docs/testing/M2/property-publication-cuts.md)
passed all 777 state/backing assertions on both native targets. It tests interrupted
publication and conservative restart metadata on disposable fixtures; runtime writes remain disabled.

The [property review/submission coordinator](docs/testing/M2/property-coordinator.md)
connects volatile reviews, fresh baseline checks, single claims and immutable
metadata acknowledgements through a trusted backend contract. Portable baseline,
storage race/all 21 fuzz checks and a FreeBSD cross-build passed; no native backend
or runtime storage writer is enabled.

The [native property observer](docs/testing/M2/native-property-collection.md)
assembles fixed FreeBSD projections in two bounded passes and refuses ordinary
drift. All 16 collector assertions passed on both native targets, with complete
portable review and teardown; successful collection still grants no write
eligibility or runtime submission authority.

The [root policy/native binding](docs/testing/M2/property-policy-binding.md)
derives pool addresses from the held root metadata reader, refuses policy drift
and checks every explicit scope against fresh native observations. Both native
targets passed all 22 assertions, complete portable review and teardown;
successful binding still grants no write eligibility.

The [native system exclusions](docs/testing/M2/property-system-exclusions.md)
join policy/property observations with full imported-pool, mount and swap backing.
The new safety path rejects system mounts and physical-disk conflicts while
retaining false write eligibility. Both targets passed all 32 focused assertions,
complete portable review and verified teardown.

The [native provider binding](docs/testing/M2/property-provider-binding.md)
requires fresh direct-DISK serial/geometry and attachment proof for every pool
leaf in the safety path. Both native targets passed all 37 focused assertions,
complete portable review and verified teardown; write eligibility remains false.

The [explicit property member baseline](docs/testing/M2/property-member-enrollment.md)
extends ownership records to version 2 and requires every leaf's enrolled identity
and geometry to match fresh native reads. Version 1 remains readable but cannot
satisfy this gate. Both targets passed all 46 focused assertions, complete portable
review and verified teardown; runtime writes remain disabled.

The [property state check](docs/testing/M2/property-store-readiness.md) adds
complete private policy/job inspection, actual system-backing capability and
conservative recovery holds to a separate read-only native source. Both targets
passed all 56 focused assertions, complete portable review and teardown; it grants
no write eligibility.

The [explicit property enrollment review](docs/testing/M2/property-enrollment-review.md)
retains individually selected scopes, member identities and complete native context
in an immutable five-minute plan. Fresh validation detects proposal/native/system
drift without publishing or granting write authority. Both native targets passed
all 64 focused assertions, complete portable review and teardown.

The [root-leased enrollment binding](docs/testing/M2/property-enrollment-state.md)
checks explicit first/exact predecessors, complete private state, backing and
projected capacity. Fresh validation keeps job holds and detects metadata drift;
both native targets passed all 76 focused assertions, complete portable review
and teardown; accepted in PR 68. Runtime publication remains disabled.

The [native enrollment source](docs/testing/M2/native-enrollment-source.md) connects
explicit root proposals to fresh native collectors and sealed state checks.
Both targets passed all 84 focused assertions, complete portable review and
verified teardown; runtime publication remains disabled.

The [reviewed enrollment publisher](docs/testing/M2/enrollment-publication.md) adds
one-attempt root ownership metadata publication with complete original-lease
state and clock/context checks. Both targets passed 819 state/backing and 88 native
pool assertions with complete portable review and teardown; runtime storage writes
remain disabled; accepted in PR 70.

The [independently collected enrollment publisher](docs/testing/M2/native-enrollment-publication.md)
connects that sealed review to fresh native evidence without caller-supplied
observations. Both targets passed all 88 native assertions with verified teardown;
focused portable review passed. Operator workflows remain disabled.

The [foreground storage job-status diagnostic](docs/operations/M2-job-status.md)
reads the complete protected ledger as OS root and shows retained events separately
from conservative restart status. Both disposable FreeBSD targets passed all 21
assertions with verified teardown; full portable review passed. No write or
recovery authority is added.

The [reviewed property postcondition comparison](docs/testing/M2/property-postconditions.md)
checks selected local effects and retained inheritance/received context from fresh
observations. Both FreeBSD targets passed the eleven native postcondition
subtests and 88 existing assertions, with focused portable review and verified
teardown. Matching does not authorize writes, attribute an executor or
acknowledge durable job success.

The [retained property-job assessment](docs/testing/M2/property-assessment.md)
compares fresh observed baseline and expected-effect facts separately from unchanged
durable history and conservative restart status. Eight new native assessment cases
passed on both FreeBSD releases, with focused portable review and verified teardown.
No reconciliation resolution, replay, write or reservation release is enabled.

The [root-leased native assessment](docs/testing/M2/native-property-assessment.md)
connects retained property jobs to independently collected root policy/native facts
and complete before/after ledger checks. Both FreeBSD targets passed all seventeen
new source cases, eleven postcondition cases, eight retained-assessment cases and
88 existing assertions, with focused portable review and verified teardown. Runtime
writes, job resolution and operator workflows remain disabled.

The [foreground property-job assessment](docs/operations/M2-property-assessment.md)
selects one protected retained job and presents fresh native match facts separately
from history and recovery status. Both FreeBSD releases passed the 21 existing
status assertions and fifteen new assessment cases, with focused portable review
and verified teardown. No writes or recovery resolution are enabled.

The [pool health/scan observer](docs/testing/M2/pool-health.md) binds fresh pool
identity to paired status and reports unhealthy facts and typed legacy scans.
Both pinned FreeBSD releases passed six native health cases, all existing status/
assessment checks and verified teardown, with focused portable review. It supplies
no write eligibility or job outcome.

The [native scrub-state fixture](docs/testing/M2/scan-lifecycle-observations.md)
adds running, paused, resumed and cancelled observations under an explicitly
restored disposable-guest control. Both FreeBSD targets passed all eight new
cases and existing checks with verified teardown; product code and write
eligibility remain unchanged.

The [checksum-finding fixture](docs/testing/M2/checksum-findings.md) checks real
member checksum findings, separate completed-scan facts and preserved mirror
content with the base fault-injection tool. Both FreeBSD targets passed all eight
new cases and existing checks with verified teardown; product code and write
eligibility remain unchanged.

The [property execution sequence](docs/testing/M2/property-execution.md) connects
queued jobs to durable applying intent, one control attempt, independent facts
and durable outcomes through a trusted worker contract. Ten behavioral groups,
affected race/fuzz checks and FreeBSD compilation passed; no native executor or
operator write workflow is enabled.

The [private native property control](docs/testing/M2/property-control.md) binds
fixed allowlisted arguments to applying metadata and fresh GUID context.
Both disposable FreeBSD targets passed the actual controls, denials, retained
applying records, descendant waiting and full teardown; see the
[manifest](docs/testing/M2/property-control-manifest.json). No root backend or
operator writer is installed.

The [original admission execution handoff](docs/testing/M2/property-handoff.md)
binds one worker to the same coordinator/backend and exact acknowledged queued
record. Eight behavioral groups and affected race/fuzz checks passed; see the
[manifest](docs/testing/M2/property-handoff-manifest.json). Full native backend
and runtime execution remain open.

Worker publication uncertainty now also suspends the original admission
coordinator: failed, lost or mismatched acknowledgements block subsequent
confirmations and execution handoffs. Read-only review remains available; no
recovery or native execution authority is added. See the
[handoff contract](docs/testing/M2/property-handoff.md#worker-publication-uncertainty).

The [fresh root submission state claim](docs/testing/M2/property-live-state.md)
adds an original-lease receipt for a read-only readiness exception for its own
queued/applying job. Restored records cannot reconstruct the claim; other holds
remain blocking. Focused portable checks and all 19 new cases plus historical
state gates passed on both disposable releases with full teardown; see the
[manifest](docs/testing/M2/property-live-state-manifest.json). Native eligibility
and the complete root backend remain open; runtime writes stay disabled.

The [claim-bound native observer](docs/testing/M2/property-claim-source.md) joins
that original claim to fresh properties, providers, system use and metadata backing.
Portable, native state-reader and complete pool gates passed. The
[separate native fixture qualification](docs/testing/M2/property-claim-fixtures.md)
retains every old case and explicitly gives the aggregate lab corpus eight minutes.
Both FreeBSD targets passed with complete teardown. All observations remain
ineligible; the complete root backend and runtime writes remain disabled. Earlier
[partial evidence](docs/testing/M2/property-claim-source-manifest.json) remains source-bound.

The [original-claim applying acknowledgement](docs/testing/M2/property-claim-applying.md)
passed focused portable review and all 23 native metadata cases on both FreeBSD
targets, with complete teardown. It permits one exact applying acknowledgement
through the original root claim, without native effects. Complete root backend,
runtime writes and recovery workflows remain disabled/pending; see the
[manifest](docs/testing/M2/property-claim-applying-manifest.json).

The [original-claim successor acknowledgement](docs/testing/M2/property-claim-successor.md)
adds exact observing/outcome metadata after the original applying acknowledgement.
It retains volatile publication lineage, strict census checks and holds. Focused
portable review and all 38 new native metadata cases passed on both FreeBSD targets
with complete teardown. Native outcome authority and runtime storage writes remain
disabled; see its [manifest](docs/testing/M2/property-claim-successor-manifest.json).

The [private original-claim control binding](docs/testing/M2/property-claim-control.md)
requires an exact applying acknowledgement and fresh native checks through that
same root claim. Both native releases and focused portable review passed with
verified teardown; see the [evidence](docs/testing/M2/property-claim-control-manifest.json).
The complete backend must still supply
live session/clock/recovery/global clearance; runtime storage writes stay disabled.

The [original coordinator execution authority](docs/testing/M2/property-execution-authority.md)
adds a mandatory opaque binding to the same admission/execution backend. It checks
the retained whole plan/session context, acknowledgement and civil clock, and
revokes launch authority when the attempt ends. Focused portable qualification
passed with verified teardown; see the
[evidence](docs/testing/M2/property-execution-authority-manifest.json).
Authentication, complete root
backend and native recovery/global clearance remain pending; runtime writes stay disabled.

The [private native property backend](docs/testing/M2/property-native-backend.md)
composes review/admission, the original root claim/coordinator authority, one fixed
native control and independent durable outcome verification. It has no runtime
consumer; external authentication/recovery/global clearance is still supplied only
by disposable tests. Focused portable review and both complete disposable native
pool suites passed with verified teardown; see the
[evidence](docs/testing/M2/property-native-backend-manifest.json). Operator writes
remain disabled.

The [configured property client](docs/testing/M2/property-dial.md) joins trusted
socket policy, bounded dialing and original session checks. Three portable parents/
seventeen children, all 104 transport cases on both disposable FreeBSD releases
and full portable review passed with verified teardown. See the
[evidence](docs/testing/M2/property-dial-manifest.json). No runtime route, backend
or storage write is registered.

The [private root property listener](docs/testing/M2/property-listener.md) joins
root socket ownership, connection admission and bounded workers. Six portable
parents/four children, all 92 transport cases on both disposable FreeBSD releases
and full portable review passed with verified teardown. See the
[evidence](docs/testing/M2/property-listener-manifest.json). Typed operations remain
denied and no runtime consumer is registered.

The [session-bound property client](docs/testing/M2/property-client.md) connects
original application authority to the kernel-authenticated typed client. Five portable
behavioral parents/nineteen children, all 74 transport cases on both FreeBSD releases
and full portable review passed with verified teardown. See the
[evidence](docs/testing/M2/property-client-manifest.json). No runtime route, backend
dispatch or storage write is registered.

The [admitted property exchange](docs/testing/M2/property-admitted.md) composes
original connection capacity/deadlines with one kernel-authorized typed exchange.
Both disposable FreeBSD releases passed all 60 transport cases; baseline and full
portable race/fuzz checks passed with verified teardown. See the
[evidence](docs/testing/M2/property-admitted-manifest.json). Runtime/session/backend
and root recovery/global launch integration remain disabled.

The [bounded property connection admission](docs/testing/M2/property-connections.md)
is a private network accounting prerequisite for the future typed transport.
Original slots/deadlines, immediate overflow denial and socket retirement passed
portable behavioral/focused race/fuzz checks. Helper tests compiled for FreeBSD
without execution; native admission/runtime assembly remains unqualified. See the
[evidence](docs/testing/M2/property-connections-manifest.json).

The [owned property exchange](docs/testing/M2/property-exchange.md) carries one
bounded typed request and terminal reply over the original kernel-authorized
connection. Both disposable FreeBSD releases passed all 44 peer/exchange cases;
baseline and full race/fuzz checks passed with verified teardown. See the
[evidence](docs/testing/M2/property-exchange-manifest.json). Backend dispatch and
root recovery/global launch remain disabled.

The [bounded property replies](docs/testing/M2/property-replies.md) preserve complete
review and job display views, enforce original selection/correlation and use bounded
refusal codes. Private session/policy digests stay outside the transport. The codecs
remain unregistered. Twelve behavioral parents/36 children, baseline/focused race/fuzz
and unexecuted FreeBSD compilation passed. See the
[source-bound evidence](docs/testing/M2/property-replies-manifest.json).

The [typed property requests](docs/testing/M2/property-requests.md) define strict
review and submission payloads with the existing planner's literal allowlist.
They are unregistered codecs; actual exchange, replies and recovery/global launch
integration remain pending. Five behavioral parents/61 children, focused race/fuzz
and baseline checks passed; helper/storage compiled for FreeBSD without execution.
See the [source-bound evidence](docs/testing/M2/property-requests-manifest.json).

The [native action-peer boundary](docs/testing/M2/action-peer.md) passed all
eighteen cases on both disposable FreeBSD targets plus full portable review.
It remains a private, unregistered kernel UID/connection-lifetime prerequisite;
runtime operations are disabled. See the
[source-bound evidence](docs/testing/M2/action-peer-manifest.json).

The [application-context negotiation](docs/testing/M2/action-negotiation.md)
adds bounded version-2 hello/ready codecs with exact request/context correlation
and original live-session checks around the exchange. Focused local review and
unexecuted FreeBSD compilation passed with verified teardown; see the
[evidence](docs/testing/M2/action-negotiation-manifest.json). They grant no operation
permission and are unregistered; the helper remains inventory-only. Kernel-peer,
recovery/global launch and typed mutation operations still precede runtime writes.

The [live application-session authority](docs/testing/M2/session-authority.md)
adds an explicit authenticated POST wrapper and an opaque original-session
binding, with logout/expiry/clock revocation and credential-free correlation.
It registers no runtime route and supplies no root/native clearance. Focused
portable qualification and unexecuted FreeBSD compilation passed with verified
teardown; see the [evidence](docs/testing/M2/session-authority-manifest.json).
Operator writes remain disabled.

## Agreed direction

- FreeBSD and ZFS; BSD-3-Clause license.
- SMB, NFS, and SFTP file serving; local users/groups initially.
- Snapshots, backup-peer registration, incremental replication, retention, and
  tested restoration in the first release.
- Native FreeBSD port/package first; an appliance image later.
- Go, server-rendered HTML5, hand-authored CSS, and HTMX when needed.
- BSD Classic interface, following the BSD Cloud design guidance.
- File serving continues when the management interface is unavailable.
- Fleet management, automatic failover, AD, VMs, and containers are outside the
  first release. Horvium itself does not host application workloads.

## Start here

1. [Project charter](docs/project/charter.md): users, outcomes, and boundaries.
2. [Architecture baseline](docs/architecture/baseline.md): components and state.
3. [Roadmap](docs/roadmap/milestones.md): foundation through the first release.
4. [Bootstrap status](docs/engineering/bootstrap-status.md): evidence and open gates.
5. [Contributing](CONTRIBUTING.md) and [agent instructions](AGENTS.md).
6. [M1 requirements](docs/milestones/M1/PRD.md),
   [engineering design](docs/milestones/M1/EDD.md), and
   [Test Design](docs/milestones/M1/TDD.md): approved implementation contracts.
7. [M1 implementation status](docs/milestones/M1/implementation.md): completed
   portions, evidence and remaining acceptance gates.

## Foundation checks

Development-only tools are checksum-pinned and installed under ignored `.tools/`.
Use Python 3.14.7 on a Linux/macOS development host for this tool installation.
Python is not a planned runtime dependency of the NAS.

```sh
make tools
make hooks
# Commit signed changes first; use the actual proposed PR title.
PR_TITLE='docs: describe the proposed change' make review
# Fresh complete portable qualification at milestones/releases:
# PR_TITLE='type: actual proposed PR title' make review-full
make lab-status
```

Native FreeBSD checks and physical hardware tests have separate gates. Passing
these foundation checks does not demonstrate functioning storage or file shares.
Use the [local qualification runner](docs/testing/local-qualification.md) for
supported disposable FreeBSD suites on the development Mac; see
[CI cost controls](docs/engineering/ci-costs.md) for the zero Actions policy. The [focused testing workflow](docs/engineering/local-review.md) keeps everyday
review proportional and reserves fresh complete portable runs for milestones,
releases and broad changes. All checks run locally; no persistent runner is installed. `make lab-stop` stops supervised development sessions and verifies
teardown. Maintainers inspect source-bound evidence before merging.

The Go module provides discovery, cache, administration, console and authenticated
CLI adapters. `make build` creates a cgo-free foreground development executable;
see the [M1 development runbook](docs/operations/M1-development.md). `make tools` also
installs checksum-verified Go 1.27.1 under `.tools/`; it leaves the system toolchain
alone. See [CI/CD](docs/engineering/ci-cd.md) for opt-in disposable native tests.

See [documentation index](docs/README.md), [testing strategy](docs/testing/strategy.md),
and [ports plan](packaging/ports/README.md). The selected remote is the private
`gonz0w/horvium` repository. See bootstrap status for creation and CI evidence.

## License and provenance

[BSD-3-Clause](LICENSE). Adapted BSD Cloud foundation tooling retains upstream
attribution; see [NOTICE](NOTICE.md).

The [native application property client](docs/testing/M2/property-application.md)
joins actual original Auth/action authority to the configured Unix connection in
an unprivileged FreeBSD fixture. Both pinned releases passed all 115 cases,
including synchronized original-store logout/Close, with full local portable
review and verified teardown. Root typed operations still receive denial;
independent root/backend/recovery/global admission and runtime workflows remain
open. See the [source-bound evidence](docs/testing/M2/property-application-manifest.json).

The [original root request binding](docs/testing/M2/property-binding.md) ties a
received typed request to its original admitted kernel peer for repeated live
checks. Both pinned releases passed all 131 transport/application/binding cases
with local portable review and verified teardown. The denial listener checks the
binding before its fixed refusal; root/backend/recovery/global admission and
operator storage workflows remain open. See the
[source-bound evidence](docs/testing/M2/property-binding-manifest.json).
