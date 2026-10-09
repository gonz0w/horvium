# M2 private native property backend

Date: 2026-10-08. Status: partially qualified private prerequisite; runtime writes disabled.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-PLAN-001, M2-PROPERTY-001, M2-JOB-001, M2-PRIV-001,
M2-DRIFT-001, M2-STATE-001 and M2-RECOVERY-001 evidence.

## One original native sequence

The private `newPropertyBackend` composes the existing review/submission
coordinator, root metadata lease, native state/claim sources, opaque original
coordinator authority, fixed property control and exact claim publishers. Require
actual FreeBSD/kernel EUID zero, an explicit validated pool GUID, the original
held root lease and a mandatory trusted clearance callback. It has no public
constructor, runtime consumer, HTTP/CLI operation or installed service.

That callback must independently establish authorized kernel peer/application
context, recovered-executor/store clearance and global/pool launch reservations.
Only declared disposable fixtures supply it today. Returning nil is not proof of
those gates; this integration cannot enable operator writes until their real
implementation and combined native safety/recovery qualification pass.

Each instance permits one admission attempt, one original authority binding,
one control object/attempt and one post-effect observation. It borrows the lease
without closing it. Use a nonblocking backend mutex: refuse concurrent work rather
than wait while a coordinator can hold its own mutex. Native callbacks consult
the original coordinator; there is no lock recursion or replacement backend.
This is a conservative single-job prerequisite, not a scheduler or an implementation
of queued multi-pool tuning. Keep every terminal and reconciliation hold.

Review and confirmation independently collect complete native policy/provider/
system/backing/ledger facts through the same state source. Require false observer
eligibility and trusted clearance around collection. Only private trusted-backend
outputs may become eligible; all public observers remain read-only/ineligible.
Confirmation checks the complete captured baseline and actual civil clock/expiry,
then publishes one reserved queued record. Exact acknowledgement and a fresh
original-lease claim precede authority binding. Duplicate stored acknowledgements
cannot install that claim. Failed/lost/foreign acknowledgements latch the backend
and original coordinator; decoded records cannot reconstruct an executable job.

Preparation and refresh bracket independent original-claim collection with live
coordinator/clearance/root/clock/context checks, exact whole current record and
applying acknowledgement verification. Require the full native baseline before
privately supplying worker eligibility. Applying publication uses the original
claim's existing sync/rename/census transaction. Construct exactly one fixed
control after its actual acknowledgement; preserve that acknowledgement even if
control setup fails, allowing durable reconciliation without a native launch.

The fixed control repeats original live authority, trusted clearance, exact
applying metadata and independent native baseline at its final callback. All
existing allowlist, GUID/address, expiry, ten-second/4 KiB guard and descendant
quiescence rules are retained. The current private control additionally consumes
the [original boot/reaper supervisor](property-supervisor.md); its evidence is separate
from the historical backend run below. An error never proves quiescence or authorizes
replay. The handoff retires original launch authority when the control returns.

After a normal quiescent return, independently recollect through the same original
claim without requiring renewed launch/session/expiry clearance. Preserve false
eligibility and hash the actual bounded relevant projection/full policy using
`PropertyExecutionObservationDigest`, the unchanged worker digest algorithm.
This read-only helper authenticates nothing. The backend retains its own digest
and postcondition match. Observing publication requires that digest and actual
quiescence/observation; success also requires a matching independently observed
effect. Arbitrary matching metadata or digest strings cannot substitute for those
private witnesses. Exact successor acknowledgement precedes each returned record.
Forward expiry cannot erase a verified late outcome. Rollback/uncertain publication
remain blocking. No hold release, retry, repair, destructive rollback or recovered
execution is implemented.

## Required tests and native fixture

Eight portable parents cover the complete durable sequence, eight admission
denials, ten phase cuts, nine original-binding/outcome refusals, 32 concurrent
worker attempts, unsupported-host refusal, three final clearance/clock/root cuts and twelve lost/foreign/invalid
acknowledgements. Preserve existing coordinator, execution, handoff and control
cases and all selected bounded fuzz targets. The initial complete-sequence test
must fail against the denying contract stub. Its initial unexecuted fixture-field
compile error is separate from the behavioral red result.

The existing independent claim fixture supplies the already declared data mirror
and separate nonboot ZFS metadata disk under `/root`. After the original claim
fixture closes, run two new serial groups: success/launch denials and response
gaps. Each recreates its own owned data/system/metadata fixtures under a fresh
135-second context. Preserve the original 135-second claim fixture, all eight
original children, six prior control-binding children, every historical pool/guard
assertion and existing time limits; do not append the extra work to its context.
Create only owned private metadata subdirectories with the explicit fixture policy;
do not claim enrollment provisioning or restored-state clearance from this setup.
Use test-only external clearance, actual original coordinator/handoff, actual fresh
state/claim readers and actual metadata transactions throughout.

Require twelve nested native cases on both pinned releases: clearance denial,
exact original admission/duplicate, durable success, applying-before-effect,
independent quota/value/source postcondition, no replay, retained terminal hold,
preserved policy/lock/no temporaries, revoked-clearance and closed-lease launch
denials, and lost applying/success acknowledgements after actual durable writes.
A wrapper independently reads exact applying bytes and live original authority
before delegating once to the unchanged fixed native runner. Gap wrappers discard
only an actual completed publisher acknowledgement; they fake no native write.
Closed lease/clearance cuts perform no effect, returning actual applying or durable
reconciliation. A lost applying ack returns the last queued ack; a lost success
ack returns observing even when newer terminal bytes are visible. Both block
replay and retain the actual records. These synchronized response gaps are not
process death, reboot or power-loss proof.

The actual inherited-UID control child also refuses the backend factory. Arm
cleanup before every possible effect; restore exact original default quota/source
only after verifying the fixture filesystem GUID. Close/remove every owned state
subdirectory and preserve policy bytes, stable lock inode, complete original raw
properties/sentinel/mount/backing. Both complete pool suites, portable review and
all guest/artifact/supervisor cleanup must pass. Shared checker/selector/supervisor
bytes stay unchanged; the source-bound record additionally requires all twelve new
children and both groups/backing/preservation/teardown cases exactly once, plus
every prior PASS name/count, without treating old-only
runner success as this integration's qualification.

## Remaining gates

Public/native observer eligibility remains false. Private backend eligibility is
conditional on external clearance which only tests provide. No deployed root
backend, live application/peer authenticator, process-death/reboot/recovery proof,
global native fence, operator workflow, reservation release, NAS hardware/client
claim or M2 acceptance follows. Existing metadata transaction/backing/ACL/cut
facilities remain unchanged and keep their historical source-bound state evidence;
this PR reruns the affected complete pool suites rather than unrelated native
state fixtures. Complete mutation/recovery/resource/continuity and UI gates remain
required before operator exposure.

## Initial native attempt

Signed source `601fcfe8ad621c70bf10575603345f34a953e448` passed portable review,
but its first 15.1 native run remains unqualified. Real success, applying-before-
effect, independent postcondition, holds and both launch-denial cases passed;
appending all new scenarios to the original claim context exhausted its unchanged
135-second deadline before the response gaps. The owned guest, disks, metadata,
artifacts and supervisor were cleaned with verified base/runtime preservation.
The repair separates newly added scenarios into their own bounded fixtures,
without changing product behavior, old test deadlines or shared tooling. Subsequent
native evidence must identify its actual revised fixture source.

## Executed qualification

Signed revised fixture source `649a60b1b6a28f5c791f1d1332e44362d05a98ab` passed fresh
focused portable review in 185.659 seconds: baseline, 125 tooling tests, nine
affected race packages and all 38 selected fuzz targets. Both review supervisors
verified cleanup. The eight new portable parents and 42 children passed; product
and portable test inputs remain identical to the initial implementation source.

The same checksummed FreeBSD/amd64 executable ran in both disposable guests:
15.1-RELEASE-p3 (ZFS root, OpenZFS 2.4.2) and 14.5-RELEASE (UFS root, OpenZFS
2.2.9), with metadata on its separately owned nonboot ZFS disk. Complete
qualification took 496.728/392.975 seconds including guest preparation and teardown;
these timings make no NAS performance claim. Both runs retained every prior PASS
name/count (76/70 respectively) and added 24 names: twelve cases and two groups'
backing/preservation/teardown checks. Actual kernel non-root factory refusal,
fixed control, independent quota/source, response gaps and original holds passed.
Exact raw properties, policy bytes, lock inode, sentinel, mounts, physical backing
and all guest/host/artifact/supervisor teardown were preserved/verified.

All 66 historical M2 manifests and shared tools retain their actual bytes/sources.
GitHub Actions stay disabled. See the source-bound
[qualification manifest](property-native-backend-manifest.json), which also keeps
the first failed native attempt separate. This evidence does not establish the
external authentication/recovery/global gates or enable runtime writes.
