# Property runtime delivery and recovery contract

Status: Accepted implementation amendment; experiment/implementation gates pending.
Date: 2026-10-08. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19), [PRD](PRD.md),
[approved EDD](EDD.md), [Test Design](TDD.md),
[ADR 0008](../../adr/0008-property-runtime-recovery-gates.md).

Authority: maintainer merge of [PR 105](https://github.com/gonz0w/horvium/pull/105),
source `6854b5554f1104c4fc2f2ee0b4e36dd80b962a9f`, on 2026-10-08. The accepted
contract authorizes the bounded [executor recovery experiment](../../research/M2/executor-recovery.md);
its findings still need review before selecting a runtime mechanism.

This contract brings the accepted private prerequisites toward one actual property
workflow. It adds no runtime behavior or qualification claim. The first assembled
operation remains one reviewed quota, reference-quota or compression step on an
explicitly enrolled filesystem GUID. Pool creation, filesystem creation, scans,
replacement and recovery import retain their separate M2 gates.

## Starting point

At merged source `914465b055a6b10a9363bbc30ba0594b7f3e87bf`, PR 104 completes
original root request binding. The following qualified components already exist:

| Boundary | Existing component/evidence | Missing runtime obligation |
|---|---|---|
| Application | [Actual session/action client](../../testing/M2/property-application.md) | Register shared authenticated UI/CLI operations; enforce all original action gates |
| Transport | [Root listener](../../testing/M2/property-listener.md), [request binding](../../testing/M2/property-binding.md) | Dispatch into helper-owned plans and durable admission; all current operations are denied |
| Review/admission | [Coordinator](../../testing/M2/property-coordinator.md), [original handoff](../../testing/M2/property-handoff.md) | One retained helper owner and bounded plans/workers across requests |
| Native effects | [Private backend](../../testing/M2/property-native-backend.md) | Replace fixture-only clearance with independently established runtime gates |
| Restart | [Restart projection](../../testing/M2/job-restart.md), [native assessment](../../testing/M2/native-property-assessment.md) | Executor attribution/quiescence, restored-state review and explicit recovery resolution |
| Scheduling | [Retained reservations](../../testing/M2/job-reservations.md) | Original global launch owner; bounded independent worker lifetime |
| Enrollment | [Native enrollment publisher](../../testing/M2/native-enrollment-publication.md) | Explicit OS-root provisioning/adoption workflow with startup and recovery gates |

Successful transport tests do not count as durable admission or runtime execution.
Native backend fixtures establish their documented original source only. No
production installation, new package dependency or persistent lab service is
needed to qualify the missing path.

## Startup and recovery

1. Hold the original root state-directory lease; validate the complete protected
   namespace, policy, record versions, capacity and actual system backing. A
   failed, interrupted or incomplete inspection returns observation-only status.
2. Inspect retained job history without replaying any effect. Queued jobs from a
   prior incarnation require a new review; applying/observing/uncertain jobs retain
   their conservative recovery status and pool/resource holds. Do not rewrite
   historical success to make an executor check appear satisfied.
3. Establish the provenance of the state being used, including possible restoration
   from older metadata. Empty state, terminal records, singleton acquisition and a
   prior clean-exit assertion cannot individually establish startup clearance.
4. Establish prior execution absence through independently collected boot/process
   identity and qualified containment evidence. Reused PIDs, missing attribution,
   unsupported inspection, a surviving guard/descendant or conflicting evidence
   deny clearance. Reboot may exclude an old userspace process; it does not by
   itself establish the resulting storage state or clear a retained reservation.
5. Bind any resulting live clearance to this original helper/state lease. Closing
   or replacing that lease, failed durability or new uncertainty retires clearance.
   A JSON record or caller-supplied boolean cannot mint it.

Exact durable identity/restore provenance and native inspection mechanisms need
the experiment below and a reviewed followup design. Until implemented and
qualified, startup remains closed to writes. Fresh observations and the existing
foreground diagnostics remain usable without pretending recovery is complete.
The implementation must explain how a new, explicitly provisioned ledger becomes
eligible; it must not treat absence of files as proof of fresh provisioning.

The operator enrollment path must use the accepted reviewed native publisher with
explicit individually selected scopes and original predecessor checks. A fixture
that seeds ownership metadata is not an operator provisioning workflow. Hand-edited
JSON or an observed pool cannot bypass bootstrap/recovery clearance. Qualify the
root provisioning/adoption path before enabling the assembled property workflow.

## Admission and independent worker lifetime

A root request binding is valid only for its original received operation, kernel
peer, correlation context, socket and deadline. Review and submission recheck it
before admitting work. Root checks actual service UID and enrolled policy;
application context is correlation with the authenticated service, not independent
verification of a human identity.

The original helper retains the coordinator and plans across connections. An
accepted job owns a bounded queue/worker entry before admission is acknowledged;
reserve capacity before the irreversible publication attempt. If the publication
acknowledgement is lost, retain uncertainty and the actual durable record rather
than releasing capacity on an assumption that nothing was admitted. A duplicate
confirmation returns existing metadata and starts no second worker. A foreign,
expired or reconstructed plan cannot hand off execution.

Accepted workers use a helper lifetime context rather than the HTTP/socket request
context. Disconnect after durable admission affects delivery of the reply, not
the admitted job. Logout prevents new authenticated operations without implicitly
cancelling accepted work. Admission must still fail if authority was revoked
before the durable admission boundary. Explicit cancellation is a separate typed
operation and remains unavailable until its own durable/native gates qualify.

The existing five-minute launch window remains a pre-effect check. Expiry after
an effect cannot erase the job or skip independent observation and durable outcome.
Helper shutdown waits for owned workers; if quiescence cannot be established,
leave durable intent and holds for restart instead of reporting clean completion.

## Global launch and resource holds

The first runtime property path allows one active native launch for the helper.
Use an original helper owner shared by all coordinators/backends; a mutex on an
individual backend or connection is insufficient. Admission/queue capacity and
native launch capacity are separate. A busy launch gate must leave the job safely
queued or durably denied under the reviewed queue contract; it must not start
another process while waiting for the first acknowledgement.

Acquire the global permit before publishing applying intent. Under that same
owner, repeat original admission authority, actual root, original state/recovery
clearance, fresh policy/native baseline and pool/resource reservations immediately
before the fixed control. No successful callback that merely checks a socket or
metadata lock can substitute for these gates.

Release the global permit only after actual control/guard/descendant completion
and an exact durable outcome acknowledgement. Unknown quiescence or publication
uncertainty suspends further launches for the helper. Do not equate Go worker exit,
descriptor closure, timeout, utility exit alone or matching properties with this
proof. Ordinary read-only views remain independent of the launch slot.

The global permit does not release a pool hold. Retained job reservations remain
conservative under current behavior, including terminal jobs. Repeat changes on a
previously used pool require an explicit, separately qualified hold-resolution
transition; do not silently remove terminal records. Until that transition exists,
a disposable single-operation fixture proves only that restricted workflow.

## Native recovery experiment

Approved time box: two development days after maintainer acceptance of this
contract. The question is which independently collected FreeBSD facts can
attribute and exclude this helper's prior property executor after helper death,
without relying on a PID or the released metadata lock alone.

Use the existing Mac local runner and serial disposable FreeBSD 15.1/14.5 guests.
Start a built root test harness owning the real private backend/control against
an explicitly owned nonboot pool and separately owned metadata disk. No deployed
mutation helper exists yet. Capture bounded sanitized boot/process
and containment facts, cross-check them against independent observations, and
kill that actual owning process while its guard/control is active. Observe surviving
processes and unchanged versus changed native property state after acquiring the
same singleton lock. Also test orderly completion, guard/helper death order and
guest reboot. Do not substitute a mocked launcher for the actual command.

Use controlled witnesses for cuts before/after applying acknowledgement and
control start/return. If a property command finishes too quickly to cut reliably,
record the gap and qualify the real containment primitive separately with an owned
blocking descendant. That descendant test is not evidence that a kernel property
operation was interrupted. PID-reuse/model cases do not claim actual kernel reuse.

Evidence records exact source, image/kernel/userland/toolchain versions, executable
hash, actual boundaries reached, boot/process observation source, independently
observed job/property facts, failed/unsupported cases and complete teardown. Remove
only owned guests, processes, overlays, fixture binaries/keys and disks; preserve
base images/runtime caches and foreign sentinels. Failed or incomplete experiments
remain unqualified. Findings propose a mechanism and durable-format change for
review; spike code does not automatically become a runtime adapter.

## Delivery order and observable gates

The following are deliverables, not estimates of PR count or completion percentage.
Implement a cohesive gate with its consumers where feasible, rather than adding
unregistered wrappers that do not close an obligation.

| Order / ID | Reviewable result | Required evidence and existing requirement |
|---|---|---|
| 1 / M2-RUNTIME-001 | Native executor/recovery experiment and concrete followup EDD/ADR decision | Actual helper death/reboot, independently observed facts, attribution limits and teardown; M2-RECOVERY-001, M2-STATE-001 |
| 2 / M2-RUNTIME-002 | Explicit root provisioning, original startup clearance and shared global launch owner, integrated with private backend | Test-first ownership/copy/revocation/contention denials; enrollment leaves data unchanged; complete store/restore and actual native prelaunch/post-effect uncertainty gates on both releases; M2-OWN-001, M2-SCALE-001, M2-RECOVERY-001 |
| 3 / M2-RUNTIME-003 | Root review/submission dispatcher and independent bounded workers | Actual dropped-UID client and root backend/state/policy/coordinator; disconnect before/after admission, duplicate confirmation, expired plans, wrong actor, full queue, shutdown and lost acknowledgements; M2-PRIV-001, M2-JOB-001 |
| 4 / M2-RUNTIME-004 | Explicit property recovery and retained-hold resolution | Fresh GUID/policy/postcondition attribution; ambiguous effects remain unresolved; no replay; durable decisions and restored-state/operator review; M2-RECOVERY-001, M2-STATE-001 |
| 5 / M2-RUNTIME-005 | Shared authenticated HTML/CLI property review, confirmation and job views | Full action gates, JS on/off, typed literal values, honest outcome/recovery text, repeated property change only after qualified hold resolution; M2-HTTP-001, M2-UI-001 |
| 6 / M2-RUNTIME-006 | Assembled property workflow qualification | Both real releases, crash/reboot/durability cuts, independent SMB/NFS/SFTP continuity and measured resources; affected native gates plus milestone portable review; M2-NATIVE-001, M2-CONTINUITY-001, M2-PERF-001 |

Keep `make review` for focused daily changes and `make review-full` for the
assembled milestone. Run native suites affected by actual behavior; shared
qualification-tool changes retain the conservative full portable plan. Evidence
must identify new cases, all affected retained cases and cleanup; an old-only
suite success cannot qualify a new dispatcher or recovery adapter. This design-only
change needs baseline/local review, not a fresh native suite.

Completing this property sequence does not complete M2: the other operation rows,
full recovery/export/restore, browser acceptance and explicit milestone acceptance
remain. M3 file services, M4 backups/peer management and M5 port/image distribution
also remain separate release work. M1 Safari/manual accessibility stays pending
at the maintainer's request.
