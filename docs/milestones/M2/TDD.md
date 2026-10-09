# Test Design: safe storage lifecycle

Status: Approved design; M2 implementation/acceptance evidence is partial. Owner: Cam (`@gonz0w`).
Date: 2026-10-04. [PRD](PRD.md), [EDD](EDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).
Approved by Cam (`@gonz0w`) on 2026-10-04 through maintainer merge of
[PR 20](https://github.com/gonz0w/horvium/pull/20), source
`753f601960529cd17596d0ffc628afe38d041c82`. Design approval authorizes the
reviewed slices; it is not M2 acceptance or native qualification.

The first [planning model record](../../testing/M2/planning.md) covers a T1
subset; [identity binding](../../testing/M2/identities.md) tracks independent type
coverage and version drift as a further partial subset; [property projection](../../testing/M2/properties.md) adds a narrow native
parser/source/GUID fixture on both releases. [Filesystem state](../../testing/M2/dataset-state.md)
tracks further partial mount/readonly/crypto coverage. [Pool observations](../../testing/M2/pool-state.md)
track permanent/import identity and coarse pool policy refusal checks.
[Feature observations](../../testing/M2/features.md) track complete native feature
tables, feature drift and partial installed/common-profile checks.
[Topology observations](../../testing/M2/topology.md) track paired legacy
GUID/path membership, redundant layout refusal and review drift; full scan/JSON
status and native provider/use eligibility remain pending. None of the complete acceptance rows
below is satisfied yet. The virtual-device identity/geometry prerequisite passed
on both pinned guests through the Mac supervisor; see [fixture evidence](../../testing/M2/devices.md).
TDD means Test Design Document; implementation uses failing behavioral tests first.

The [portable journal foundation](../../testing/M2/execution-journal.md) adds
failing-first fixed-depth JSON and single-successor/census tests under
M2-RUNTIME-001/M2-STATE-001/M2-RECOVERY-001. Require every top-level/nested field,
zero-on-error/damage/version/overflow denials, separate pending domains and
unchanged bindings; failed/incomplete/copy consumers must never yield a partial
hash. Exact record bytes and original lock binding must affect the framed digest.
Bound both record/census fuzz targets, run affected-package review, and report
FreeBSD compilation as unexecuted. Native persistence/restore/bootstrap/global
owner and real publisher/backend consumer gates are still required together.

The [native census qualification](../../testing/M2/native-execution-census.md)
adds failing-first T2/T3 observation checks under M2-STATE-001 and the journal
portion of M2-RUNTIME-001. Require exact native bytes after actual publication,
complete omission/drift detection, original handle isolation, both ACL models,
credential-dropped inherited-descriptor refusal and preserved uncertainty.
Run the affected state publication/crash corpus on both disposable releases;
partial digest evidence cannot satisfy bootstrap/global/recovery acceptance.

The [private persistence qualification](../../testing/M2/execution-journal-persistence.md)
adds failing-first namespace/initial-state tests and actual file/ACL/durability
controls. Require all 65 store, 74 process-death, eighteen added ACL, six full/
read-only and one softupdates assertions alongside the affected state corpus.
Original receipt loss/drift must retire writes, copied handles must leave the
original intact and reopen must never reconstruct authority. These file gates
cannot satisfy persistent binding, startup/global or active kernel-cut acceptance.

## Traceability and required gates

The accepted [property runtime delivery gates](property-runtime.md) map
M2-RUNTIME-001 through M2-RUNTIME-006 to the existing acceptance IDs below.
They specify an executor-recovery experiment before choosing the missing native
clearance mechanism, then integrated startup/launch/dispatch/recovery/UI tests.
PR 105 accepted the contract and bounded experiment. The
[executed research](../../research/M2/executor-recovery.md) passed seventeen cases
and actual reboot/readback on each target. This is partial observation evidence:
active property-command/kernel interruption remains untested, and the concrete
recovery mechanism and assembled runtime gates remain required. No complete
acceptance row below is satisfied by this experiment. The accepted
[mechanism Test Design amendment](property-recovery-mechanism.md) enumerates
failing-first and actual native boot-ID/reaper/journal/restore gates; PR 107 accepted that design/test plan at source
`02a995e5681f38b4a40141402dd2d0a9f0244933`. The
[private supervisor](../../testing/M2/property-supervisor.md) implements the first
boot/reaper consumer; 42 focused cases across five parents passed on both pinned
FreeBSD/amd64 fixtures. This includes retained actual controls, raw boot change,
clock independence and verified teardown; it does not satisfy journal/startup/global
or active kernel-interruption gates. See its [manifest](../../testing/M2/property-supervisor-manifest.json).

| Requirement | Test ID / observable assertion | Level/environment |
|---|---|---|
| M2-REQ-001 | M2-OWN-001: observation/markers never grant ownership; exact selected GUID scopes survive rename/restart; adoption leaves data/properties/mounts/cache/import state unchanged | T1-T3 / E1-E2 |
| M2-REQ-001, M2-REQ-007 | M2-DRIFT-001: external rename, changed properties/inheritance, reused name, missing pool, foreign child, topology or policy revision suspends the affected write without changing unrelated resources | T2-T3,T6 / E1-E2 |
| M2-REQ-002 | M2-DEVICE-001: prove stable identity/provider correspondence; deny duplicate/missing identity, aliases, overlap, partition/signature/boot/swap/mount/pool use, wrong geometry and reused paths at review and launch | T1-T3 / E1-E2; E3 for hardware claims |
| M2-REQ-003 | M2-PLAN-001: UI/CLI share immutable plan; expiry/edit/session/policy/drift forces re-review; duplicate confirm yields one durable job; contradictory reuse denied | T1-T3 / E1-E2 |
| M2-REQ-003 | M2-PRIV-001: actual kernel UID and helper-peer policy; root/wrong UID/forged actor/unknown operation/arbitrary path/property/scope denied; observe-only defaults remain read-only | T1-T3 / E1-E2 |
| M2-REQ-003 | M2-HTTP-001: assembled mutation/plan/job/cancel endpoints enforce auth, CSRF, Host/Origin, methods, strict body bounds and no fragment/CLI bypass; logs redact secrets | T2-T3 / E1-E2 |
| M2-REQ-004 | M2-JOB-001: durable admission precedes launch, applying intent precedes effect, result durability precedes success; no duplicate writes under disconnect/flood/restart/expired replay | T1-T3,T6 / E1-E2 |
| M2-REQ-004, M2-REQ-007 | M2-RECOVERY-001: abrupt death/reboot/timeout at every acknowledged boundary preserves intent and reservations; reconcile by identity/postconditions and never automatically replay an ambiguous write | T2-T3,T6 / E1-E2 |
| M2-REQ-004, M2-REQ-007, M2-REQ-010 | M2-STATE-001: native ownership/modes/locks/sync/rename, torn/corrupt/newer/full/read-only store, data-pool absence and older metadata restoration fail safely | T1-T3,T6-T7 / E1-E2 |
| M2-REQ-005 | M2-POOL-001: actual mirror and RAIDZ2 layouts, reviewed alignment and unused identities match independent native GUID/membership; denial/partial result never destroys an existing pool | T3,T6 / E2 |
| M2-REQ-005 | M2-DATASET-001: exact owned parent, absent direct child, initial no-mount and nonce/GUID correlation; validated mount namespace; no implicit parents, shadowing or symlink escape | T1-T3,T6 / E1-E2 |
| M2-REQ-005 | M2-PROPERTY-001: quota/reference-quota/compression actual effects, explicit no-limit versus invalid zero/overflow, ancestor limits and inherited/local sources; partial steps preserve honest state and unrelated properties | T1-T3,T6 / E1-E2 |
| M2-REQ-005 | M2-FEATURE-001: installed common profile/hash, creation feature set and clean cross-release export/import; no force, feature upgrade or fallback when incompatible | T3,T6 / E2 |
| M2-REQ-006 | M2-HEALTH-001: bounded real status output on both bundled ZFS versions maps GUID/topology/errors/scans; unknown/truncated/contradictory format is unavailable, never healthy | T1-T3 / E1-E2 |
| M2-REQ-006 | M2-SCRUB-001: start, progress, native completion, checksum findings, attributable stop and duplicate/conflicting/external scan handling; no success from launch alone | T3,T6 / E2 |
| M2-REQ-006 | M2-REPLACE-001: exact old leaf GUID/new identity, redundancy/size/alignment denial, actual resilver/completion/errors and helper-death recovery; no detach/force/cancel illusion | T3,T6 / E2; E3 for physical resilience |
| M2-REQ-008 | M2-CONTINUITY-001: independent fresh SFTP/SMB/NFS fixture reads survive management/helper restart and unrelated storage work; supported scrub/resilver preserves target fixture bytes | T3-T4,T6 / E2 |
| M2-REQ-009 | M2-UI-001: plan/confirmation/job/detail/recovery pages and CLI parity, JS on/off, actual selected browsers, keyboard/focus/status/light-dark/reduced-motion/narrow/native zoom and screen-reader findings | T2 + actual browser/manual / E1-E2 |
| M2-REQ-009 | M2-SCALE-001: bounded plan/store/history/queue/device reservations and one launch by default; per-pool serialization with unrelated readers/pools progressing | T1-T3 / E1-E2 |
| M2-REQ-009 | M2-PERF-001: combined Go RSS/idle CPU/cached latency budgets with real store/job activity and declared large synthetic state; native child and ZFS work reported separately | T8 / E1-E2 |
| M2-REQ-010 | M2-NATIVE-001: real built helper/unprivileged service/CLI/HTML exercise supported operations on both release candidates, with independent assertions, exact pins and complete fixture teardown | T3 / E2 |

Every row is an M2 behavior acceptance gate. Unit/cross-build results, source
inspection and old M1 tests do not satisfy M2 native tests. A skipped/unavailable
capability is unqualified; amend scope/design explicitly rather than count it as
a pass. Manual browser findings remain required. Physical and external client
claims need their separate labs; passing E2 does not imply all hardware support.

## Fixture isolation and independent assertions

Use fresh disposable FreeBSD 15.1/14.5 amd64 QEMU guests. Pin the image/runtime
hash, QEMU/CPU/memory, source/suite/toolchain/modules/assets and actual kernel,
userland and OpenZFS strings. Inherit M1's verified image/toolchain machinery;
[Virtual disk attachment/identity fixture](../../testing/M2/devices.md) tooling
passed its eight prerequisite assertions and owned host teardown on both targets;
PR 28 accepted the fixture at merged source
`27dafaba7a5d04a4896f27776ef79f4b1f4c2bbc`. The [disk correlation extension](../../testing/M2/disk-correlation.md)
adds four assertions; all twelve native assertions and host teardown passed on both
targets; PR 31 accepted this partial subset. The [attachment extension](../../testing/M2/disk-attachments.md)
adds five assertions including actual empty/populated GPT refusal and owned
cleanup. All seventeen assertions and both executed native tests are required;
all seventeen and host teardown passed on both targets; see its
[native manifest](../../testing/M2/disk-attachments-manifest.json). PR 32 accepted
this subset. The [system-use extension](../../testing/M2/system-device-use.md)
adds six assertions for actual UFS mount, active swap, root backing, denial,
foreign preservation and teardown. Twenty-three assertions and all three tests
are required; all twenty-three and full teardown passed on both targets; see
the [native manifest](../../testing/M2/system-device-use-manifest.json). PR 33
accepted this subset. The [imported-pool extension](../../testing/M2/imported-pool-use.md)
adds six assertions for an actual unmounted imported pool, catalogue binding,
missing-status denial, foreign/root preservation and teardown. Twenty-nine
assertions and all four tests are required; all twenty-nine and complete teardown
passed on both targets; see the [native manifest](../../testing/M2/imported-pool-use-manifest.json).
PR 34 accepted this subset. The [signature extension](../../testing/M2/filesystem-signatures.md)
adds seven assertions for UFS and active/removed-pool ZFS recognition, exact probe
address binding, unavailable-output refusal and whole fixture probe preservation.
Thirty-six assertions and the same four tests are required; all thirty-six and
complete preservation/teardown passed on both targets; see the
[native manifest](../../testing/M2/filesystem-signatures-manifest.json).
PR 35 accepted this subset. The [combined-use extension](../../testing/M2/device-use-review.md)
adds fifteen assertions for actual use joins, independently recollected unchanged
pool use, new/removed mount/swap/pool drift, retained residual signatures, missing
status denial, immutable copies and unavailable signature evidence. Fifty-one
assertions and the same four tests are required; all fifty-one and complete host
preservation/teardown passed on both targets; see the
[native manifest](../../testing/M2/device-use-review-manifest.json). PR 36 accepted this subset. These partial results do not
satisfy the complete device-safety row. The [pool-label extension](../../testing/M2/pool-labels.md) adds eight required
assertions for actual export/reimport identity, missing imported membership, drift,
probe preservation and denial, for fifty-nine assertions in four tests. Both
targets and complete preservation/teardown passed; see the
[native manifest](../../testing/M2/pool-labels-manifest.json). PR 37 accepted
this subset. The [disk-access extension](../../testing/M2/disk-access.md) adds nine
required assertions for held direct/alias read/write handles, review drift,
unchanged access, close/restoration, unavailable mode refusal and byte preservation.
Sixty-eight assertions in the same four tests and full host preservation/teardown
are required; both targets passed with complete preservation/teardown; see the
[native manifest](../../testing/M2/disk-access-manifest.json). PR 38 accepted
this subset. Further fault control and complete device safety still
need implementation and qualification. Never accept an arbitrary host device supplied to a test.

Pool creation/device-identity tests use new guest-attached virtual block devices
with distinct configured identifiers and independently verified native identity.
The guest's boot disk and foreign sentinel pool are explicit denial controls.
No special product switch bypasses missing serial/lunid checks to permit md tests.
File-backed md may support adoption/property/scrub fixtures after native creation,
but it cannot qualify the real creation identity path. Logical/physical sector
variants are qualified only when the guest actually reports the intended geometry.

Declare ownership of every backing file, provider, pool, directory, socket, mount,
daemon and loopback alias before tests create them. Teardown verifies absence of
all owned resources and preservation of foreign sentinels. Native fixture cleanup
may destroy only test-owned pools; the product does not gain a destroy operation
from this test privilege. Discard the guest even after success. A teardown leak
is failure, not an accepted warning.

For every mutating case, capture independent before/after native GUIDs, membership,
properties and property sources, feature flags, mount state and fixture-file hashes.
Inspect actual native state in addition to the product response/job. Capture only
sanitized assertions and version/hash records, never host inventory or credentials.
Golden parsers originate in these release-specific observations, including both
JSON and legacy status formats. Fuzz hostile/oversized/control-byte/duplicate and
unknown structures; bounded status text does not become a general command parser.

## Failure matrix and deterministic synchronization

Synchronize each cut point with test-owned barriers rather than sleeps. Exercise
death/disconnect before admission, after durable admission, after applying sync
but before launch, while the actual native control command is active, after its
effect but before outcome durability, during registry update, while a scrub or
resilver is active, and during a metadata export. Kill the actual built process,
verify descendant/quiescence behavior and check state through an independent
native observer before restarting.

Repeat with graceful management stop, helper SIGKILL, guest reboot and a disposable
VM power-cut experiment. These qualify specified recovery logic, not real disk
controller/cache honesty. A failpoint may coordinate a phase but must not replace
the actual native write with a fake when recording native acceptance. Record any
uncertain effect as reconcile-required and prove a retry cannot duplicate it.

Explicit cases include:

- Dataset name reused with a different GUID, external parent rename, changed
  property source and foreign descendant between review and execution.
  Parent quota/compression changes affecting an already unadopted descendant are
  denied before launch; partial impact collection never narrows the required scope.
- Device disappearing, changing identity at the same path, aliasing a selected
  member, new signature/use between review and launch, and unsupported geometry.
- Missing/changed installed feature profile; unavailable boot/data pool; clean
  reimport of a registered created pool; foreign or force-required import denied.
  Reboot after create but before GUID durability must not guess ownership or
  import an unregistered candidate from a matching name/device set.
- Second helper writer, PID reuse, an orphan guard/native process, damaged store,
  bad file owner/mode/symlink, failed file/directory sync/rename, quota/space/full
  filesystem and missing/newer state revision. No new native write follows denial.
- Root-enrolled scope revoked while queued; duplicate request after lost response;
  expired replay/tombstone; filled queue/history/metadata budget; uncertain jobs
  remain visible and are never removed to make capacity.
- Actual scan cancelled/completed with errors, external scan ambiguity, failed
  replacement command with possible effect, and interruption during resilver.
  No automatic force/repair/destroy/detach or claim of generic cancellation.

Restore a consistent older root metadata backup into a fresh fixture containing
newer ZFS state. Require policy re-enrollment and full identity/job reconciliation;
the first startup performs no write or automatic adoption. Refuse newer schema
without rewriting it. This tests state recovery only; file/key/ACL restoration
belongs to M4 and package migration/removal to M5.

## Continuity, browser and resource procedure

Reuse the independent fixture-only file-daemon separation approach from
[M1 continuity](../../testing/M1/file-continuity.md), with M2 operations actually
running. Every protocol performs fresh exact-byte reads at the meaningful job and
restart phases. Do not reuse cached mounts/files as proof of a new request. Tests
must verify daemon/config preservation, target content and cleanup independently.
No macOS/Linux/BSD external client/ACL/locking claim follows from loopback fixtures.

Browser cases cover both successful and refused plans, expired confirmation,
duplicate submit, queued/running/partial/reconcile-required outcomes, unavailable
cancel and restored sessions/jobs. Match CLI/HTML/fragment identities and outcomes.
Run actual pinned Firefox JS on/off, plus Safari and manual screen-reader/keyboard/
native 200%/400% zoom tests. The pending M1 checklist is not automatically waived
or transferred as evidence. Record manual defects before M2 acceptance.

Use M1's immutable measured fixture and warm-up/five-minute idle/1,000-page ten-reader
definitions. Add a small real pool/100-filesystem ownership fixture, admitted jobs,
reconciliation and native launch/scan polling; large synthetic fixture has 10,000
filesystem/ownership records, 1,000 retained terminal jobs and 64 admitted jobs.
Fill byte/count limits and eight eligible independent pool simulations; prove the
configured launch and four-maintenance defaults instead of starting 64 native
scans on the small guest. A separate eight-pool native fixture verifies actual
serialization/concurrency. Report time waiting for a pool reservation separately
from launch duration and scan time.

Measure paired management/helper RSS (128 MiB small / 256 MiB large targets),
small idle CPU below 1% of a core and cached p95 below 250 ms. Include generation
replacement, job polling, admission denial and three controlled restarts. Record
p50/p95/p99, peaks, sampling gaps, allocated metadata bytes and native child CPU/RSS;
ARC/daemon/disk work are separate, not hidden in the Go budget or called throughput
qualification. Evidence validators must reject skipped/empty/incomplete suites.

## Result and acceptance record

Each record names requirement/test IDs, exact tested functional/suite commits,
input hashes, native commands/capabilities, versions, fixture ownership/topology,
workload, fault synchronization, independent assertions, result and limitations.
Maintain a per-release/per-operation pass/fail/skip/unavailable matrix and link the
approved revision. Keep initial failures/repairs in the record. Update behavior
docs and compatibility alongside each slice, run repository gates, and keep PRs
open for review. M2 acceptance requires the approved design, these actual gates
and an explicit maintainer acceptance record. The native property projection
and identity/state projection subsets passed as linked above. Complete acceptance
rows and the remaining native gates are still pending.

## State-directory prerequisite

The [state-directory contract](../../testing/M2/state-directory.md) implements an
independent partial M2-STATE-001 boundary, without using unqualified device
eligibility or admitting storage work. Its thirty-five required assertions cover
root ownership/modes, descriptor anchoring, safe-path/ancestor/symlink refusal,
lock-file owner/mode/link/type/content refusal, native close-on-exec flags,
competing first writers, successful file/directory sync, dropped kernel UID,
actual separate-process contention/death/reacquisition, stable lock inode,
foreign preservation and complete fixture teardown. Both declared releases and
full host base/runtime/group teardown passed at signed functional/suite source
`123044324e75f6df27d43eee085c23435ddbcce1`; see the
[native manifest](../../testing/M2/state-directory-manifest.json). PR 39 accepted
this partial boundary.
No full record/durable job/recovery/power-cut/ACL/hardware claim follows.

## Ownership record prerequisite

The [strict ownership codec](../../testing/M2/ownership-records.md) covers partial
M2-OWN-001/M2-STATE-001 at T1: required exact fields/types/version, duplicate/null/
unknown refusal, identity/parent/property consistency, canonical damage checksums,
byte/capacity limits, immutable round trips and no implicit scope. Independently
recomputed checksums on invalid policies must still refuse; fuzzing cannot return
authority on denial. No filesystem operations or admission are implemented here.
Five behavioral tests and fifty-seven subtests passed without skips at signed
source `e9f685c2f4460b7b4545bfa761c9089a2743439a`; all twenty-one bounded fuzz
targets and complete portable review passed. See the
[source-bound manifest](../../testing/M2/ownership-records-manifest.json).
PR 40 accepted the codec; actual file/store/recovery gates remain open.

## Native ownership read prerequisite

The [native reader](../../testing/M2/ownership-read.md) covers partial M2-OWN-001/
M2-STATE-001 at T1/T3: actual descriptor-relative private-file reads, UID/owner/
mode/link/type/byte/version/checksum/identity refusal, kernel-dropped UID with
inherited descriptors, unchanged valid/control/foreign bytes, stable anchoring
after directory rename, directory/lock drift denial and complete teardown. The
state suite requires all sixty-three lease/read assertions, two native tests, six
unit tests and fifty-seven codec subtests on both targets. Both targets passed at
signed source `94cf66e9c3a34184c76b84858cb7ef998e4bd2dc` with complete teardown; see
the [manifest](../../testing/M2/ownership-read-manifest.json). PR 41 accepted this subset;
ACL provisioning, coordinated races, full store/publication/admission/recovery and
M2 acceptance remain open.

## State ACL validation prerequisite

The [ACL validation](../../testing/M2/state-acl.md) covers partial M2-STATE-001/
M2-OWN-001 at T1/T3: actual-descriptor capability queries and bounded ACL reads,
mode-only versus ACL-enabled distinction, conservative permission refusal,
ancestor denial before lock creation, retained private modes/UIDs/inodes/ACLs/bytes,
real NFSv4 named-user access and kernel UID denial, and owned mount/device/child
teardown. Require all 108 lease/read/ACL assertions, ten parent tests and 57
codec subtests on both releases. Both targets passed at signed source

`33f4d91bc57adb71f27ce80bbe0552fb1e7685d2`, including complete teardown; see the
[manifest](../../testing/M2/state-acl-manifest.json). PR 42 accepted this subset; complete
provisioning, coordinated races, publication/admission/recovery and M2 acceptance
remain open.

## Ownership publication prerequisite

The [publisher contract/procedure](../../testing/M2/ownership-publication.md)
adds partial M2-STATE-001/M2-OWN-001 at T1/T3. Failing-first portable phase/revision
and evidence-checker tests precede implementation. Native tests require real
private file creation/replacement, canonical bytes, preserved old opened inode,
strict existing-record refusal, stale/concurrent revision denial, stable lock,
anchoring/drift, inherited-descriptor kernel UID denial, actual UFS ENOSPC and
read-only mounts, ACL preservation and complete teardown. Package-private seams
inject phase errors around native operations to qualify ordering/accounting only;
they do not qualify real device EIO, abrupt process/reboot/power-cut durability.
Require all 161 named assertions, fourteen parent tests and 57 codec subtests on
both releases. Both targets passed at signed source
 `eabf4691fc55b825db74ba73501d758126fda4e7`
with complete teardown; see the [manifest](../../testing/M2/ownership-publication-manifest.json).
Review is pending. Full store budgets, temporary
record discovery after death, durable admission and recovery remain open.


## Bounded ownership inspection prerequisite

The [catalog contract/procedure](../../testing/M2/ownership-catalog.md) covers a
further partial M2-STATE-001/M2-OWN-001 at T1/T3: complete descriptor-anchored
namespace enumeration, independent sorted summaries, strict record checks,
orphan/unknown refusal, bounded accounting, repeated offset independence and
ordinary final drift checks. Three catalog ACL controls extend the existing
owned UFS fixture; inherited kernel-UID denial also covers inspection. Require
all 202 named assertions, seventeen parents and 57 codec subtests on both
releases, with full preservation/teardown. Reduced-limit native fixtures are
budget-code proof, not full populated large-workload/resource qualification.
Inspection never repairs/deletes state, clears publication uncertainty or grants
write authority. Both targets passed all 202 at signed source `5ea125fdc815fd7c6c54aba36824e50ea876b9f6`
with full teardown; see the [manifest](../../testing/M2/ownership-catalog-manifest.json).
Accepted in PR 46. Job/admission/recovery,
full store allocation and complete M2 acceptance remain open.


## Ownership publication capacity prerequisite

The [capacity procedure](../../testing/M2/ownership-capacity.md) adds partial
M2-STATE-001/M2-OWN-001 at T1/T3: validate the complete namespace before writing,
reserve old-plus-new transient bytes and published count, recheck before rename,
refuse unrelated unsafe/corrupt/newer state, retain unknown/orphan objects, latch
orphan uncertainty, and serialize last-slot writers. Require 234 named assertions,
nineteen parents and 57 codec subtests on both releases with preservation/teardown.
Reduced native bounds prove accounting, not populated M2-SCALE-001/resource gates.
Both targets passed at signed source
 `bf80cb9f2e1c196eae330ae0d681d86e0c466033` with
full teardown; see the [manifest](../../testing/M2/ownership-capacity-manifest.json).
Accepted in PR 47; mixed job budgets, durable admission/restoration/
recovery, abrupt death/power loss and full M2 acceptance remain open.


## Job lifecycle metadata prerequisite

The [job lifecycle procedure](../../testing/M2/job-records.md) covers partial
M2-JOB-001/M2-STATE-001 and M2-RECOVERY-001 (projection only) at T1. Require strict
bounded shape/version/checksum/intent/history validation, independent round trips,
all required fields at each object level, checksummed invalid lifecycle refusal,
full uint64 precision, no terminal replay/post-launch cancellation, retained
uncertainty and restart projections with no execution. Fuzz every accepted hostile
input for exact semantic round trips and zero metadata on denial. Portable
qualification passed at signed source `a3791513f6423fe5860064e93f89399c21dc1445`: seven tests,
119 schema subtests, baseline and focused storage race/all 15 fuzz checks; see the
[manifest](../../testing/M2/job-records-manifest.json). Accepted in PR 48; no native
file/acknowledgement/recovery claim follows.
Full durable baseline/attribution data, mixed store/admission/retention, actual
restarts/restoration/quiescence and complete M2 acceptance remain open.


## Native job reader and successor prerequisite

The [reader procedure](../../testing/M2/job-read.md) adds partial M2-STATE-001/
M2-JOB-001/M2-PRIV-001 at T1/T3: actual ID-selected descriptor reads, strict private
file/ACL/size/schema/history validation, filename binding, zero metadata on error,
real inherited-UID denial despite effective NFSv4 access, preserved unsafe objects
and root-filesystem anchoring. Pure successor tests refuse independently valid
identity/intent/history rewrites and stale/skipped/truncated updates. Require all
275 state assertions, twenty-eight parents and 176 codec/schema subtests on both
releases with complete fixture/base/runtime/process teardown. Both passed at
signed source `5241ae7f0bccde0423613af1dfcde9aed951ac34`; see the
[manifest](../../testing/M2/job-read-manifest.json). Accepted in PR 49. Mixed store
publication/admission, complete executable intent, native
attribution/quiescence, acknowledgement cuts/restoration and M2 acceptance remain
open; the job reader cannot grant storage authority.


## Mixed state census prerequisite

The [census procedure](../../testing/M2/state-catalog.md) adds partial M2-STATE-001/
M2-JOB-001/M2-PRIV-001 at T1/T3. Require whole-namespace strict validation and zero
catalog on error, shared ownership/job byte totals, exact sorted summaries,
nonterminal reconciliation, actual 64/1,000 class ceilings and exhaustion, submission
collision refusal, unsafe files/ACLs/UID denial, ordinary drift, preserved uncertainty
and complete owned teardown. Require 336 named assertions, thirty-two parent tests
and 176 codec/schema subtests on both releases. Both passed at signed source
`b79ff0f03a85c719882b93e8080f562704cc6bac` with complete teardown; see the
[manifest](../../testing/M2/state-catalog-manifest.json). Accepted in PR 50.
This census grants no publication/admission/recovery authority or M2-SCALE-001
resource acceptance; ownership-only publication remains strict.


## Durable job metadata publication prerequisite

The [publication procedure](../../testing/M2/job-publication.md) adds partial
M2-STATE-001/M2-JOB-001/M2-PRIV-001 at T1/T3: current-predecessor immutable history,
actual private file publication and every durability/refusal phase, complete mixed
capacity preflight/final rechecks, terminal-slot exhaustion, preserved uncertainty,
unsafe/ACL/UID denials, concurrent duplicate/last-slot serialization and actual
UFS ENOSPC/read-only job controls. Require 432 named assertions, thirty-five parents
and 176 schema subtests on both releases with full owned teardown. Both passed
at signed source `dc789c4e3672f61e670e6c4352e1eae89b3fb13a`; see the
[manifest](../../testing/M2/job-publication-manifest.json). Accepted in PR 51. No runtime admission/native effect, executable intent,
reservation/pruning/recovery or acknowledgement/power-loss acceptance follows.


## Reviewed property intent capture prerequisite

The [procedure](../../testing/M2/property-intent.md) adds partial M2-PLAN-001,
M2-DRIFT-001 and M2-STATE-001 at T1: fresh plan/session/expiry capture, complete
current review/model binding and independent snapshots, source/identity/impact/
policy drift, exact timestamps/uint64, strict required shapes, valid-checksum
semantic refusals and bounded hostile codec input. Existing ownership/job parser
regressions remain mandatory. Storage-only race/all fuzz and baseline checks are
required; FreeBSD cross-compilation proves portability only. No filesystem/syscall,
job publication or native eligibility adapter changes. Seven parents/769 subtests,
storage race/all 17 storage fuzz targets passed at signed source `e66d59eec2e05137a11d8533b327206c3682e423`;
see the [manifest](../../testing/M2/property-intent-manifest.json). Accepted in PR 52. Native execution, job embedding/admission, attribution/postconditions,
recovery/cuts and M2 acceptance remain unqualified.


## Property job embedding prerequisite

The [procedure](../../testing/M2/property-jobs.md) adds partial M2-JOB-001,
M2-PLAN-001 and M2-STATE-001 at T1: matching target/property/policy/plan binding,
initial timestamp against nanosecond capture and expiry, immutable owned copies,
complete baseline/history successor refusal, canonical whole-record checksum,
required shape checks at all levels, valid-checksum semantic denials, exact numeric
and byte limits and hostile-input round trips. Require baseline and storage-only
race/all compiler-listed fuzz checks. Existing metadata regressions remain required.
Cross-compilation proves portability only. No native operations/file formats in
accepted readers/publishers change; native persistence/admission/recovery and
full M2 acceptance remain separate gates. Six parents/859 subtests, required baseline and storage race/all 18 fuzz targets
passed at signed source `ddec32450bfc533eca98c2e39d6a68ce09951404`;
see the [manifest](../../testing/M2/property-jobs-manifest.json). Accepted in PR 53.


## Native property-job store prerequisite

The [procedure](../../testing/M2/property-job-storage.md) adds partial M2-STATE-001,
M2-JOB-001 and M2-PRIV-001 at T1/T3: complete private baseline reads and ID binding,
immutable durable predecessors, format conversion denial, combined byte/class/
submission accounting, actual shared publication/phase failures and uncertainty,
late namespace/valid-baseline drift, duplicate serialization, safe-file refusal,
POSIX/NFSv4 ACL/kernel UID controls and real UFS ENOSPC/read-only preservation.
Require all 555 named assertions, fifty parents and 1,804 distinct codec subtests
on both declared targets with complete guest/host teardown. Shared evidence changes
require a complete portable review. Both targets passed at signed source
`5a0e67cd7f469cf398b1cd831284d8a44a3b67ce` with complete teardown; see the
[manifest](../../testing/M2/property-job-storage-manifest.json). Accepted in PR 54; no admission/native
effect, recovery/acknowledgement cut or M2 acceptance claim follows.


## Job reservation metadata prerequisite

The [procedure](../../testing/M2/job-reservations.md) adds partial M2-JOB-001,
M2-PLAN-001 and M2-SCALE-001 at T1: failing-first complete-set/collision/zero-return
tests, exact retry binding/current history, every lifecycle pool hold, unrelated
pools, actual 64/1,000 class ceilings, byte preflight/exhaustion, copy isolation,
concurrent read-only checks and bounded hostile-input fuzzing. Require baseline
and storage-only race/all compiler-listed fuzz checks; cross-compilation is
portability only. Portable qualification passed. Actual native admission, root caller/
policy/eligibility provenance, recovery/release/cuts and M2 acceptance remain open.

Reservation qualification: signed source `eec1ad7c34485bc2b548f8a1c88f68cdaaaae647`, six parents/thirty
subtests, required baseline, storage race/all nineteen fuzz targets and FreeBSD
cross-compilation; see the [manifest](../../testing/M2/job-reservations-manifest.json). Accepted in PR 55; no native admission/release claim follows.


## Native job reservation inspection prerequisite

The [procedure](../../testing/M2/native-job-reservations.md) adds partial
M2-STATE-001, M2-JOB-001, M2-PRIV-001 and M2-SCALE-001 at T1/T3. Require complete
private mixed inspection with actual UID/ACL/name/object binding, full property
duplicate/current history, ownership/presentation/capacity accounting, actual
class ceilings, unsafe/ unknown/orphan preservation, late record/namespace drift,
uncertainty retained and complete owned teardown. A sealed portable streaming
builder must not change its prior state on failure or mutate a returned view.
Require all 622 named assertions, 58 parents and 1,837 distinct codec/model
children on both releases. Shared evidence changes require a complete portable
review. Both targets passed at signed source
`cca3a07eed591ed3bab757fc530eada0a40686be` with full teardown. No admission,
recovery/release/cuts, execution or M2 acceptance follows from this inspection.

Native reservation qualification: `cca3a07eed591ed3bab757fc530eada0a40686be`; see
the [manifest](../../testing/M2/native-job-reservations-manifest.json). Accepted
in PR 56; no runtime admission/release follows.


## Reserved property-submission publication prerequisite

The [contract/procedure](../../testing/M2/property-submission.md) combines strict complete retained
plan/submission/pool checks and initial queued property publication under one
root lease/mutex. Repeat full baseline/capacity checks before staging and rename;
new success follows final directory sync, while retries return existing current
metadata without rewriting or a fresh durability acknowledgement. Uncertain
publication denies even duplicate returns. Terminal holds remain; runtime native
admission, caller/policy/eligibility/session/clock, executor/recovery/release and
M2 acceptance stay open. Require 710 state assertions, 60 parents and 1,837
codec/model children on both releases with full owned teardown. Both native
targets passed at signed source
`1dc80019f5d68430ad49845ecdb666192922806f` with full teardown; accepted in PR 57 at merged source
`d79a55779f713e1209a930343fdb482d403c85ef`. Shared pipeline/evidence/supervisor changes required one
complete portable review; startup failure before worker creation must remain a
failure and clean prepared owned fixtures without inventing/signalling a PID.

Publication validates its exact renamed inode/bytes before final directory
sync. Native concurrency uses 32 callers per scenario. UFS softupdates publication
and retry calls are refused; supported metadata backing is ZFS or UFS without
soft updates. Twelve exact native backing/teardown controls and ten actual
softupdates refusal/read/preservation controls are required. Root filesystem and
metadata fixture backing are recorded separately. Strict census timestamps remain;
no product mount/tunefs change or root-fencing claim is added.

See the [execution manifest](../../testing/M2/property-submission-manifest.json); no runtime admission/release follows.


## Bounded job restart projection prerequisite

The [restart projection contract](../../testing/M2/job-restart.md) provides ID-sorted pages of 1..64
summaries from the complete immutable mixed-job view. Queued work projects
review-expired; applying/observing work projects executor-unverified, separately
from the unchanged last durable event. Both record formats, all terminal/uncertain
holds and full retry history remain retained. Unavailable/malformed cursors and
invalid page bounds expose no partial page; cursors require the same retained
view. No native observation, publication, startup consumer, replay or release is
added. Portable behavioral/race/fuzz and FreeBSD cross-compilation passed at signed
source `60db363d5b0f615dbdb7fd2f247b01e8ab12b788`; accepted in PR 58 at merged source
`caf21958b3d89dc0cde3f2ca0ad05787d63d5247`.
Native executor attribution/recovery and complete M2 gates remain open.

See the [execution manifest](../../testing/M2/job-restart-manifest.json); no native recovery/admission follows.


## Property metadata process-death qualification

The [process-cut contract](../../testing/M2/property-publication-cuts.md) kills/reaps the actual built metadata writer
at eight synchronized transaction boundaries, then reacquires the same lock and
independently verifies unchanged objects. Orphan temporaries remain preserved and
block publication; visible queued jobs project reconciliation and retain pool
holds. Matching retries are metadata readback without fresh sync/durability claims.
No native storage command, startup consumer or product behavior/schema changes.
Require all 777 state/backing assertions, 66 parents and 1,876 codec/model children
on both targets, including native restart-page execution and complete backing/
child/host teardown. Both targets passed at signed source
`a3a33d5929f7fba32fd9ab6364320984b49d9162` with full teardown; maintainer
review was accepted in PR 59. Shared evidence/selector changes required one complete portable
review. Reboot/power-cut and actual native
executor/reconciliation/restoration/release gates remain open.

See the [execution manifest](../../testing/M2/property-publication-cuts-manifest.json); no native executor/recovery authority follows.

## Property review and submission integration

M2-PLAN-001, M2-DRIFT-001, M2-STATE-001 and M2-SCALE-001 at T1 (partial):
require independently collected policy/observations for review and submission,
owned immutable baseline/output, 64-plan exhaustion before collection, conservative
five-minute expiry, civil clock rollback, actual cancellation before publication,
restart invalidation and exact session/submission binding. Thirty-two concurrent
confirms must invoke the publisher once. Require zero results and no replay after
publication failure or invalid/foreign acknowledgement, preserving review access.
Existing duplicate history and a successful acknowledgement after the publication
deadline must survive without a second sync claim. Fuzz bounded action sequences.

See the [coordinator procedure](../../testing/M2/property-coordinator.md).
Signed source `a8868fa61bb60b80d3b83378a4f0e964e13e8028` passed the required baseline,
nine behavioral parents, storage race/all 21 fuzz targets and an unexecuted FreeBSD
cross-build; see the [execution manifest](../../testing/M2/property-coordinator-manifest.json).
Mocks and FreeBSD cross-compilation do not qualify a native
backend, kernel caller/session authorization, durable retry lookup, storage launch,
native recovery/release or the assembled workflow.

## Native property observation integration

M2-PLAN-001, M2-DRIFT-001 and M2-SCALE-001 at T1/T3 (partial): require fixed typed
pool selection, bounded success/failure accounting, actual format selection,
complete dataset/type/property/state agreement and two full independent passes.
Any malformed/unavailable/mixed/drifting projection exposes zero observation.
Require cancellation, record/byte ceilings and unsupported-host/format denials;
fuzz hostile captures and selectors. Complete output must still fail eligibility.

On both native targets, the actual reader must match independently read pool and
dataset identities, properties, inherited state, snapshot ownership, complete
features and mirror topology. Verify wrong-GUID, readonly and volume denials,
fresh rename with stable GUID, unchanged properties/sentinel and complete owned
provider/backing/pool teardown. Require all 16 assertions, one executed test and
the exact cross-built executable checksum; reject missing/duplicate/skipped/failed
evidence. Supervisor interruption must remove the new owned artifact. See the
[collector procedure](../../testing/M2/native-property-collection.md). Signed source
`57ac9dde8c87db90d6cda26c28e0f3ee55ea408e` passed all 16 assertions on both targets,
the baseline/109 tooling tests, all 11 race packages and 30 fuzz targets with full
teardown; see the [execution manifest](../../testing/M2/native-property-collection-manifest.json).
No native root/policy eligibility, mutation/executor/recovery or complete
M2 acceptance follows.

## Root policy and native property binding

M2-OWN-001, M2-PLAN-001 and M2-DRIFT-001 at T1/T3 (partial): require GUID-only
selection with names/scopes independently read through the held root lease.
Canonical policy reads must bracket complete native collection; refuse changed
revision/properties, failed reads, foreign pool, missing/renamed scopes,
inconsistent unselected descendants, empty scope and cancellation with zero
output. Require independent canonical results, harmless ordering equivalence and
bounded hostile policy fuzz. The original planner eligibility refusal must remain.

On both native targets, extend the focused collector fixture from 16 to 22
assertions with actual private-file/lease binding, absent policy, actual revision
change between reads, wrong filesystem scope, external GUID-preserving rename and
closed lease refusals. Test-only metadata setup grants no enrollment/publication
claim. Require the exact executable, executed pass, all assertions and complete
owned teardown. Shared checker changes require one complete portable review.
See the [procedure](../../testing/M2/property-policy-binding.md). Signed source
`cd1991c6a1cb36375f0fca7e7dd90228338390cc` passed all 22 assertions on both native
targets, the baseline/109 tooling tests, all 11 race packages and 31 fuzz targets
with full teardown; see the [execution manifest](../../testing/M2/property-policy-binding-manifest.json).
System/device eligibility, native admission and M2 acceptance remain open.

## Native property system-use exclusions

M2-DEVICE-001, M2-OWN-001 and M2-DRIFT-001 at T1/T3 (partial): require fixed bounded
all-imported-pool/mount/swap/GEOM reads, qualified formats, safe enumerated pool
addresses and two equal normalized full views. Bind selected topology exactly;
refuse root/system mounts, other mount/swap/pool backing overlap, unsupported
backing, failures/partial data and ordinary drift. Permit own ordinary data mounts
while preserving the planner eligibility refusal. Check private proof/copy/record
budgets and fuzz hostile collector/joined captures.

On both targets extend the focused suite to all 32 mandatory assertions using
the existing five declared snapshot disks. Require actual root/pool backing,
policy/property/safety binding, owned mount relocation beneath `/root` and refusal,
fresh restoration, active disjoint swap, sentinel preservation, direct fixture
teardown and md backing refusal. Verify exact binary and executed test, unchanged
base/runtime/device backing and complete guest/host cleanup. Require one appended
host device-cleanup marker; zero or duplicate markers deny qualification. Shared
runner/checker changes require one complete portable review. See the
[procedure](../../testing/M2/property-system-exclusions.md). Signed source
`c0f26eeef6e09e955f1edeb8d6fe15c59481b3cb` passed all 32 assertions on both targets,
the baseline/111 tooling tests, all 11 race packages and 33 fuzz targets with full
teardown and unchanged snapshot backing; see the
[execution manifest](../../testing/M2/property-system-exclusions-manifest.json).
Full eligibility, admission/execution/recovery and M2 acceptance remain open.


## Native pool-provider binding

M2-DEVICE-001, M2-OWN-001 and M2-DRIFT-001 at T1/T3 (partial): require every native
leaf to bind a direct uniquely identified DISK, independent serial/geometry and
qualified attachment graph, including exact terminal ZFS::VDEV consumers. Keep
the unused-device classifier's ZFS denial. Validate before disk commands; refuse partitions,
unsupported attachments, missing/duplicate identities, mismatches, partial reads,
limits/cancellation and ordinary drift. Re-read topology in both passes and compare
against the property observation. Require provider identity/geometry agreement with the later system graph and evidence in the composed path,
zero results on failure and false eligibility on success, with a final root policy
read after all collectors. Fuzz bounded hostile projections. See the
[procedure](../../testing/M2/property-provider-binding.md).

On both targets retain all 32 previous assertions and require five additional
actual native checks: two unique mirror leaves/geometry, duplicate-serial mirror
refusal, missing-serial pool refusal and GUID-verified teardown of both additional
fixtures. Require exact executable/source, all 37 assertions, actual executed PASS,
unchanged snapshot/base/runtime backings and all owned guest/host cleanup. Shared
evidence-checker changes require complete portable review. Signed source
`1b6fef0cc54f304a88badffabeb5929ce7edd6d6` passed all 37 assertions on both
targets, the baseline/112 tooling tests, all 11 race packages and 35 fuzz targets
with full teardown; see the
[execution manifest](../../testing/M2/property-provider-binding-manifest.json).
Root enrollment, admission, execution/recovery and M2 acceptance remain open.

## Explicit property member enrollment prerequisite

Partial M2-OWN-001, M2-DEVICE-001, M2-DRIFT-001 and M2-STATE-001 at T1/T3:
require unchanged version 1 canonical bytes/read semantics, strict required version
2 members, unique GUID/serial/provenance and bounded valid observed geometry.
Reject checksummed semantic damage and unknown/missing/null/duplicate fields.
Require complete exact native correspondence and final root policy reread, with
zero results on legacy/partial/member drift and false eligibility on success.
Persisted property intents must reject same-revision member drift. Retain all 37
native cases and add nine enrollment/read/catalog/publication/denial/restoration
cases: all 46 exact assertions and complete owned teardown on both targets.
Observe actual state backing via kernel statfs: qualify version 2 publication on
accepted backing and unchanged-record denial on softupdates UFS. No mount/tunefs
changes or additional devices. The shared checker requires complete portable
review; unchanged metadata durability/cut behavior retains historical evidence.
See the [procedure](../../testing/M2/property-member-enrollment.md). Signed source
`b95f60c873020f632674a2302acb1a4a3a6ca646` passed all 46 cases on both targets,
the baseline/112 tooling tests, all 11 race packages and 35 fuzz targets with full
teardown; see the
[execution manifest](../../testing/M2/property-member-enrollment-manifest.json).
Full runtime enrollment/admission/recovery and acceptance remain open.

## Read-only property store readiness prerequisite

Partial M2-STATE-001, M2-RECOVERY-001, M2-OWN-001 and M2-DRIFT-001 at T1/T3:
check the complete mixed private namespace and selected canonical policy under
one held lease/mutex. Require kernel-observed supported writable/non-softupdates
system backing matched to complete fresh mount/provider facts; refuse enrolled
pool or shared physical backing. Retain all selected-pool terminal/nonterminal
holds and deny global nonterminal work. Orphans latch uncertainty without repair.
Require final backing identity/capability and root policy/context checks, zero
composed results on failures and false eligibility on success. Preserve every
record/hold. Non-FreeBSD stays unavailable. The focused native fixture retains
46 cases and adds ten store/backing/policy/namespace/hold/uncertainty/preservation
cases: all 56 assertions and actual PASS/teardown on both targets are required.
Observe actual ZFS success and softupdates UFS denial; no filesystem settings or
additional disks/services change. Complete portable review is required for the
shared checker. See the [procedure](../../testing/M2/property-store-readiness.md).
Signed source `231b8f0f2c250fcbfc900d95d17ed94de6a24cbb` passed all 56 assertions
on both targets, the baseline/112 tooling tests, all 11 race packages and 35 fuzz
targets with full teardown; see the
[execution manifest](../../testing/M2/property-store-readiness-manifest.json).
Full recovery/admission/execution and acceptance remain open.


## Explicit property enrollment review prerequisite

The [contract/procedure](../../testing/M2/property-enrollment-review.md) adds partial
M2-OWN-001/M2-PLAN-001/M2-DRIFT-001 proof. Test explicit child-only selection,
complete unselected context, member/property scope denials, input/accessor
isolation, random IDs, context/expiry, fresh graph/property/member/system drift,
canonical ordering and hostile size/count bounds. Success must remain ineligible.
Retain the focused 56 native cases and add eight review/refusal/preservation cases,
using two independent native property/provider/system collections on the same
owned mirror. Require all 64 assertions and actual PASS on both targets, exact
policy/content/native preservation and full teardown. Shared checker changes
require complete portable review. Prior native metadata engines and historical
manifests retain their sources; do not rerun or relabel the 777-case state suite.
Both targets passed all 64 cases at signed source `8f6f0cef7f4321b56f37ae99f05c50696c8cc319`, with complete
portable review and teardown; see the
[manifest](../../testing/M2/property-enrollment-review-manifest.json). Accepted in PR 67; no runtime enrollment/admission/recovery or M2 acceptance.


## Root-leased enrollment state binding prerequisite

The [contract/procedure](../../testing/M2/property-enrollment-state.md) adds partial
M2-OWN-001/M2-PLAN-001/M2-STATE-001/M2-DRIFT-001 proof. Test explicit first/legacy
predecessors, exact successors, partial-zero/overflow/same-revision drift, projected
whole-temporary capacity, proposed-owner backing exclusions, retained holds, zero
bindings and unsupported hosts. Retain the 64 native cases and add twelve actual
first/successor/backing/policy/unrelated-record/namespace/job/orphan/expiry/context/
preservation/teardown controls on a separate owned private lease. Actual ZFS permits
binding and revalidation; softupdates UFS must refuse with zero binding and unchanged
records. Require all 76 assertions, actual PASS on both releases, complete portable
review and full owned teardown. No metadata engine/schema change or historical
777-case state-suite rerun. Both targets passed all 76 assertions at signed source
`9329f795185df1a823247d8cba917357d7e614fe` with complete portable review and teardown; see the
[manifest](../../testing/M2/property-enrollment-state-manifest.json). Accepted
in PR 68. Runtime provisioning/publication,
restored-state/executor validation and complete M2 acceptance remain open.


## Native enrollment source prerequisite

The [contract/procedure](../../testing/M2/native-enrollment-source.md) adds partial
M2-OWN-001/M2-PLAN-001/M2-STATE-001/M2-DRIFT-001 proof. Test owned explicit proposal
and predecessor copies, denial before native I/O, cancellation/final expiry and
backwards clocks, native/system drift and zero/hydrated binding refusal. Retain all
76 focused native cases and add eight successor review/fresh revalidation,
predecessor/member/context/cancellation/uncertainty/preservation controls. Actual
root-backed ZFS success and softupdates UFS zero-result denial must remain distinct.
Require all 84 exact assertions, actual PASS on both targets, complete portable
review and full teardown. Historical metadata engines/manifests stay unchanged;
no 777-case state rerun or runtime enrollment/publication/recovery acceptance.

Both targets passed all 84 assertions at signed source
`d13f4ba489cd659b87f81f2746830aaa3a94bfef`, including actual successor review/fresh
revalidation on root ZFS and preserved-record softupdates UFS denial. Complete
portable review and teardown passed; see the
[manifest](../../testing/M2/native-enrollment-source-manifest.json). Accepted in PR 69; first enrollment through this source and full M2 acceptance stay open.


## Reviewed enrollment metadata publication

The [contract/procedure](../../testing/M2/enrollment-publication.md) qualifies partial
M2-OWN-001/M2-PLAN-001/M2-STATE-001/M2-DRIFT-001 behavior. Require portable concurrent
single-attempt/typed-denial/late-acknowledgement tests, all 819 native state assertions,
67 parents and 1,876 codec children on both releases, plus all 88 actual native pool
controls and complete owned teardown. Root ZFS real-input publication, synthetic
metadata-input non-softupdates UFS publication and actual softupdates UFS refusal
are distinct claims. Unchanged historical manifests retain their original sources.

Corrected source `0a1f4041fc4cf18ae0919428e303dd2140e63d48` passed complete portable
review and both 88-case native pool suites. Both 819-assertion/67-parent/1,876-child
state runs passed at original source `656e4220a4040669c09bce29311e2fdd74d62992`;
a corrected-source cross-build exactly matched the executed state binary. Native
results remain bound to those actual sources; see the
[manifest](../../testing/M2/enrollment-publication-manifest.json). All teardown
passed. Maintainer review and runtime/M2 acceptance remain pending.


## Independently collected enrollment publication integration

The [procedure](../../testing/M2/native-enrollment-publication.md) adds partial
M2-OWN-001, M2-PLAN-001, M2-DRIFT-001 and M2-STATE-001 integration coverage.
Portable orchestration must recollect complete bounded inputs, reject pre-attempt
failures, preserve publisher outcomes and refuse zero public bindings before I/O.
Both releases must pass the unchanged 88-marker native pool suite with its four
publication controls using the composed source. Supported root ZFS additionally
runs six mandatory refusal/absence subtests on the same first review and then
valid publication; softupdates UFS retains explicit zero-binding denial and
independently provisioned predecessor preservation. No shared evidence/selector
or storage-package behavior changes; retain PR 70's actual 819-case state/backing
evidence without relabeling it. Require focused portable review and complete
teardown; runtime provisioning/recovery/transport and M2 acceptance remain open.


Both releases passed all 88 assertions at signed source
`d6fc9eb9dfee6cda75de7ed727db11bcaf00cb8d`, with the six supported-backing refusal
subtests, required softupdates denial, focused portable review and full teardown.
See the [manifest](../../testing/M2/native-enrollment-publication-manifest.json).
The superseded invalid default-quota projection remains unqualified at its original
source; its portable regression failed first and the source-consistent local-quota
fixture passed. No historical metadata evidence was rewritten.


## Foreground retained job-status integration

The [procedure](../../testing/M2/storage-job-status.md) adds partial M2-STATE-001,
M2-JOB-001, M2-RECOVERY-001, M2-PRIV-001 and M2-NATIVE-001 diagnostic coverage.
Portable tests must cover complete multi-page and mixed-format summaries without
private intent, conservative restart state, validation/argument/cancellation and
output failure. Both releases must pass the twenty-one exact compiled public-main
command, actual UID, full-record/preservation/lease/backing and teardown assertions.
Use only owned private metadata and a checksummed child test executable; no disk
or storage mutation fixture. Shared runner/supervisor changes require fresh full
portable review and exact new-artifact cleanup after interruption. No native
metadata implementation/codec or prior manifest changes; full state evidence keeps
its actual sources. Management transport/browser parity, native reconciliation,
restoration, execution and complete M2 acceptance remain open.

Both targets passed all 21 assertions at signed source
`03d4ab5948dc56a41333acb0e4dd20d3191e1bc7`, with full portable review (122 tooling tests,
11 race packages, all 35 bounded fuzz targets) and verified teardown; see the
[manifest](../../testing/M2/storage-job-status-manifest.json). Retain original
passed and superseded unqualified fixture scopes without relabeling.


## Reviewed property postcondition comparison

The [contract](../../testing/M2/property-postconditions.md) adds partial
M2-PROPERTY-001 and M2-DRIFT-001 matching coverage. Require failing-first portable
exact local/literal/no-limit/inherited/override/received and relevant context/
policy drift tests, immutability and bounded arbitrary-value fuzzing. Both releases
must pass all existing 88 pool-source assertions and the eleven mandatory actual
property-postcondition subtests, with independent native columns and original
projection restoration plus full owned teardown. Native eligibility remains
false; comparison is not attribution, durable success, admission or recovery.
Runtime native writes, assembled workflows and complete M2 acceptance stay open.

Both targets passed the 88 existing assertions and eleven mandatory subtests at
signed source `0b828eb38be56708abca17dbfc200fb49d1ddd62`, with 122 tooling tests,
nine affected race packages, 30 bounded fuzz targets and full owned teardown.
See the [manifest](../../testing/M2/property-postconditions-manifest.json); both
superseded fixture failures remain unqualified and historical evidence unchanged.

## Retained property-job observation assessment

The [contract](../../testing/M2/property-assessment.md) adds partial M2-RECOVERY-001,
M2-JOB-001, M2-DRIFT-001 and M2-PROPERTY-001 evidence. Require independent
baseline/expected/both/neither facts across nine retained histories, unchanged
restart projection/bytes/holds, incomplete/invalid zero-result refusal, decoded
binding, relevant-context/policy drift and preserved plan eligibility refusal.
Run bounded value fuzzing and focused review, then both actual disposable native
pool suites: 88 existing assertions, eleven existing postcondition subtests and
eight new mandatory assessment subtests. Actual graphs are native; histories are
encoded/decoded test metadata. No publication, process-death, reboot, executor
attribution, restoration, resolution, release or full M2 acceptance follows.
Focused portable review passed at signed source
`2ddc0bb971af47ac6d21f0898deb374cf94ef0d5`: 122 tooling tests, nine affected race
packages and 31 bounded fuzz targets. Both declared FreeBSD releases passed the
required native inventory and complete preservation/teardown; 15.1 retained its
original source and 14.5 independently rebuilt the exact same binary at corrected
source. The portable selector failure remains unqualified. See the
[manifest](../../testing/M2/property-assessment-manifest.json). All 49 prior M2
manifests and native metadata/shared tooling retain their existing scope.

## Root-leased native property-job assessment

The [contract](../../testing/M2/native-property-assessment.md) adds partial
M2-RECOVERY-001, M2-STATE-001, M2-JOB-001, M2-DRIFT-001 and M2-PROPERTY-001 evidence.
Require failing-first source denial/binding tests, unsupported-host refusal, focused
review and both actual native pool suites: 88 existing assertions, eleven
postcondition cases, eight retained-assessment cases and all seventeen mandatory
native-source cases. Test actual private root records and wrong UID, complete ledger
and raw presentation drift, missing/legacy/unsafe/oversized/duplicate state, retained
applying history/holds and original lock/bytes/native/fixture preservation/teardown.
Metadata are directly provisioned test fixtures; no publication/acknowledgement,
executor/restoration/resolution/replay/release or full M2 acceptance follows.
Focused portable review and both native release candidates passed at signed source
`b3e5007efbf0c134f149a21538c0b186c8c9d1e6`; all mandatory old/new cases and
preservation/teardown gates passed. The
[manifest](../../testing/M2/native-property-assessment-manifest.json) retains
exact source/binary/check hashes and both earlier unqualified deadline attempts.
Only redundant unchanged fixture observations were removed; all assertions and
time limits remain unchanged. All 50 prior M2 manifests are preserved.
Maintainer review and complete recovery/runtime/M2 acceptance remain pending.

## Foreground native property-job assessment

The [contract](../../testing/M2/property-assessment-diagnostic.md) adds partial
M2-STATE-001, M2-JOB-001, M2-RECOVERY-001, M2-PROPERTY-001 and M2-DRIFT-001
operator evidence. Require failing-first projection/denial/lifetime tests, focused
review and both disposable native job-status suites, retaining all 21 old
assertions and fifteen new actual command/UID/native/preservation/teardown cases.
Copied public main re-exec is test-only; native root records/intents are synthetic
fixtures, not executed jobs. Both native releases passed all original and new
mandatory cases with complete preservation/teardown at signed source
`b8d078d0097ae47f2e9d5f9b035fccbd2f144e9c`. Focused portable review passed
baseline checks, 122 tooling tests and command-package race checks. See the
[manifest](../../testing/M2/property-assessment-diagnostic-manifest.json),
including the earlier unqualified timing fixture and actual failing-first
regression. All 51 historical M2 manifests remain byte-identical. Accepted in
[PR 76](https://github.com/gonz0w/horvium/pull/76). Execution attribution, recovered-state clearance,
resolution/replay/release and full M2 acceptance remain open.

## Read-only pool health and scan observations

The [contract](../../testing/M2/pool-health.md) adds partial M2-HEALTH-001,
M2-SCRUB-001, M2-DRIFT-001 and M2-NATIVE-001 observation evidence. Require
failing-first typed scan/state/denial/bounds/drift tests and focused race/fuzz
review. Both disposable release suites must retain 21 status assertions, fifteen
assessment children and six new native health children with full preservation
and verified teardown. Native progress/pause/cancellation/error injection and
attributed lifecycle are not inferred from source-format/portable tests. Native
qualification passed at signed source `ab2685cdffcb6aa06d14b356b1df6f2787143300`: six mandatory
health children, fifteen assessment cases and all 21 original status assertions
on both releases, with identical binaries and complete teardown. Focused review
passed baseline/122 tooling tests, 9 race packages and 33 fuzz targets; see the
[manifest](../../testing/M2/pool-health-manifest.json). Accepted in [PR 77](https://github.com/gonz0w/horvium/pull/77);
complete health/scan lifecycle acceptance remains pending. All 52 prior manifests
retain their actual scope.

## Native scrub-state observation fixture

The [contract](../../testing/M2/scan-lifecycle-observations.md) adds partial
M2-HEALTH-001, M2-SCRUB-001, M2-DRIFT-001 and M2-NATIVE-001 observation
evidence. Require eight actual running/pause/resume/cancel/completion/preservation
children on each release, retaining all old status/assessment/health checks and
deadlines. A pinned disposable-guest-only progress control holds actual kernel
scanning; independently restore/reread it on every exit. No synthetic output
or job attribution qualifies a native state. Existing product code needs no new
implementation. Signed source `728e980a8b21755c9b87f33e48405f096bb03b24` passed
baseline/122 tooling tests and command-package race checks; both releases passed
all eight new cases, six health/fifteen assessment children and 21 status assertions
with identical binaries and verified control/storage/process/base/runtime teardown.
See the [manifest](../../testing/M2/scan-lifecycle-observations-manifest.json).
Accepted in [PR 78](https://github.com/gonz0w/horvium/pull/78). Advancing
progress, injected errors, attributed lifecycle and full M2 remain open.

## Native repaired-checksum finding fixture

The [contract](../../testing/M2/checksum-findings.md) adds partial M2-HEALTH-001,
M2-SCRUB-001, M2-DRIFT-001 and M2-NATIVE-001 evidence. Require eight actual
GUID-bound injection/scrub/finding/content/restoration cases on both releases,
retaining all old status/assessment/health/scan checks and deadlines. A native
read-buffer bit flip on one mirror leaf must produce nonzero member checksum
findings while repaired scan completion/error count remain separate; independently
verify content and clear only the owned acknowledged handler. Missing/unknown
acknowledgement cannot broaden cleanup. No new product implementation; native
qualification and focused review passed at signed source
`8078e1d0c694a0384b5183ae33ea95fddc6a45fb`: baseline/122 tooling tests, command-package
race checks and all eight checksum cases plus original checks on both releases
with identical binaries and verified handler/storage/process/base/runtime
teardown; see the [manifest](../../testing/M2/checksum-findings-manifest.json).
Maintainer review is pending. Unrecoverable errors, advancing
progress, attribution/executor/recovery and full acceptance remain open.

## Property execution ordering subset

The [contract](../../testing/M2/property-execution.md) adds partial M2-JOB-001,
M2-PROPERTY-001, M2-DRIFT-001 and M2-RECOVERY-001 behavioral evidence. Require
failing-first durable ordering/owned result, publication/lost-ack cuts, fresh
pre/post-applying launch denials, quiescence/observation ambiguity, exact whole
record acknowledgement, concurrent single attempt, retained-state refusal,
clock/context/late-outcome and invalid-input cases. Bounded cut-point fuzzing
refuses duplicate control and success without all independent durable gates.
Signed source `cceb615b82aedc98d4e93fb1cdb90545b05916ea` passed baseline/122 tooling
tests, all ten behavioral groups, nine affected race packages and all 34 selected
fuzz targets, including cut-point fuzzing. FreeBSD tests compiled without execution
and the artifact was removed; complete owned-process teardown passed. See the
[manifest](../../testing/M2/property-execution-manifest.json). Accepted in
[PR 80](https://github.com/gonz0w/horvium/pull/80). Native live-claim/root/eligibility/transport,
global/pool serialization, actual commands/child containment, interrupted effect/
metadata/reboot/recovery and complete acceptance are separate required gates.

## Private native property-control subset

The [contract](../../testing/M2/property-control.md) adds partial M2-PROPERTY-001,
M2-PRIV-001, M2-DRIFT-001 and M2-NATIVE-001 evidence. Require failing-first binding/
argument/denial/error/ownership/concurrency tests and bounded selection fuzzing.
Both opted-in releases must retain all 88 assertions and old subgroups and pass
all eleven new control children/group plus same-group/escaped guard children and
actual dropped-UID denial. Actual property effects and fresh postconditions must
retain applying metadata; denied controls change no storage. Exact properties/
sources/child/metadata/pool/provider/sentinel/base/runtime/process teardown and all
original deadlines remain mandatory. Signed source `6ef01cb5fbd3f3015a6fc8c9a5a8ccc4302031aa`
passed baseline/122 tooling tests, eight affected race packages and all nine
selected fuzz targets. Both releases passed every required new case, all retained
assertions/subgroups and full teardown. Metadata publication uses an actual owned
mounted ZFS child, preserving the UFS softupdates refusal; the operator-stopped
initial attempt remains unqualified. See the
[manifest](../../testing/M2/property-control-manifest.json). Accepted in
[PR 81](https://github.com/gonz0w/horvium/pull/81); native root backend/live eligibility and crash/recovery/assembled
acceptance remain open.

## Original admission execution handoff subset

The [contract](../../testing/M2/property-handoff.md) adds partial M2-PLAN-001,
M2-JOB-001, M2-DRIFT-001 and M2-RECOVERY-001 evidence. Require original backend/
admission binding, exact whole current queued record, identifier-only handoff,
32 concurrent handoffs and 32 calls through the issued object, duplicate/restored/
unsubmitted/missing-adapter/input denials, final clock/context refusal, original
coordinator uncertainty/rollback propagation, retained acknowledgement ownership,
late outcomes and lost applying/observing/success acknowledgement cuts. Bounded
action fuzzing must refuse a second publication, handoff or effect. Signed source
`01af1c83f2deb4e1c492aaee1098344b3def8dcc` passed baseline/122 tooling tests, all eight
new groups, 9 affected race packages and all 36 selected fuzz targets.
FreeBSD tests cross-compiled without execution and the artifact was removed;
complete preservation and owned-process teardown passed. See the
[manifest](../../testing/M2/property-handoff-manifest.json). Accepted in
[PR 82](https://github.com/gonz0w/horvium/pull/82) at merged source
`76774cf1add6479ab6a917beddfb464b869cbd10`; no root lease, native backend/eligibility/scheduling, actual durable
interruption/recovery, transport or assembled acceptance follows from synthetic
adapters or an unexecuted FreeBSD cross-build.

## Worker publication uncertainty regression

The [worker publication uncertainty repair](../../testing/M2/property-handoff.md#worker-publication-uncertainty)
adds portable regression evidence for M2-JOB-001 and M2-RECOVERY-001. Require
all four publication boundaries crossed with ordinary errors, lost acknowledgements,
invalid replies, foreign replies and earlier valid records (20 cases). Each must
retain the last acknowledged state, suspend the original coordinator, deny an
already-issued second worker and new handoffs/duplicate confirmations, preserve
read-only review, and refuse 16 concurrent new confirmations without backend I/O.
A separately verified reconciliation acknowledgement must not latch uncertainty.
Retain the prior late-success/clock/context/cut cases. Native implementations and
fixtures are unchanged; portable checks and an unexecuted FreeBSD cross-build
cannot qualify native launch, interruption/recovery or the assembled backend.

## Fresh property submission state claim amendment

Require the [state-claim contract](../../testing/M2/property-live-state.md) at T1
and T2 for partial M2-JOB-001, M2-STATE-001, M2-DRIFT-001 and M2-RECOVERY-001.
Five behavioral parents and bounded action fuzzing must prove exact original
receipt/current record binding, stage/refused restored and foreign metadata,
retained independent holds, bounded nonreplacement, one claimant and invalidation
on close. Add all 19 live-state children to the existing native submission parent;
require 32 native concurrent callers, actual post-sync/no-receipt failure cuts,
root/closed/orphan/unsafe-record denials and a waited UID/GID-65534 child.
Preserve every historical state/backing/codec assertion and all deadlines on both
pinned releases with serial disposable QEMU runs and full owned teardown.
Kernel metadata behavior is actual; fixture policy/physical system-use facts are
synthetic, and positive readiness must not be claimed as full native eligibility.
No changes to shared tools/selectors/checkers/old manifests or runtime consumers.
Signed source `fe56955fcc2b1803bfa6bf61685b336f8716b420` passed all new portable/race/fuzz gates,
the mandatory baseline with 122 tooling tests, and all 19 new children plus every
historical native state/backing/codec gate on both pinned releases. Identical
artifacts, same supported metadata backing rules and complete owned teardown
were verified; superseded attempts remain unqualified. See the
[manifest](../../testing/M2/property-live-state-manifest.json). Accepted in
[PR 83](https://github.com/gonz0w/horvium/pull/83);
original backend/native eligibility, effect interruption/recovery, authenticated
workflows, hold release and complete M2 remain open.

## Original claim native observation amendment

Require all four [claim-source](../../testing/M2/property-claim-source.md) portable
parents and bounded selector fuzzing, plus actual claim-bound metadata readers
inside all existing 19 live-state cases. Retain every old state/backing/codec gate.
Require all eight actual native claim-source children on both pinned releases,
using a separate declared fixture disk for GUID-checked independent ZFS metadata
backing. Verify actual GEOM/system/kernel agreement, original queued/applying
collection with false eligibility, foreign/closed/restored denials, unchanged
metadata/data and exact mount restoration. Preserve all old pool assertions and
subgroups, the original five-minute deadline and shared tools. Qualified sources
and artifacts must be explicit, with full owned teardown; native execution and
recovery clearance remain separate.

Signed source `aa0794996733706c47cce45ad6e0aad3f4155d28` passed all four new portable
parents, nine affected race packages, all 38 selected fuzz targets and baseline/
122 tooling tests. Both native state releases passed all required gates and new
readers. Both 15.1 pool attempts passed the new eight children but failed the
unchanged parent deadline in old cases. Require full current-source pool gates on
both releases before accepting this observer; 14.5 is unattempted. Every attempted
owned guest/process/fixture was cleaned. See the explicitly incomplete
[manifest](../../testing/M2/property-claim-source-manifest.json). Historical proofs
are unchanged and cannot substitute for these pending gates.

## Native claim fixture isolation amendment

The [fixture contract](../../testing/M2/property-claim-fixtures.md) moves the new
claim group to a separate opted-in native parent with a 135-second context,
retaining the historical collector's five-minute context and every old assertion/
subgroup. Select both actual parents serially in the same disposable guest.
Require all eight claim children once, no failures/skips/unknown cases, plus exact
outer data-mirror/private-state teardown and restored mount/swap presentations.
Keep all 88 old named assertions and host snapshot/base/runtime/process cleanup.
Checker regressions must reject legacy-only, partial, duplicated, failed/skipped
claim execution and missing or incorrect teardown. Run both pinned releases.
Product bytes and historical manifests remain unchanged; shared checker/selector
maintenance requires current full portable review under the existing planner.
Native qualification is pending; no eligibility or backend acceptance follows.

The separated-fixture retry still exhausted the historical collector's five-minute
aggregate context while its independent claim parent passed. Amend the aggregate
collector fixture budget to eight minutes for the complete accumulated corpus,
explicitly superseding its old aggregate limit. Retain all assertions and individual
production observer (135-second) and property command (ten-second) limits. The
separate claim parent remains 135 seconds. Require both full parents and all cases,
source-bound artifacts and complete teardown on both releases. This allocation
is a lab-fixture limit, with no runtime behavior or performance/support claim.

At signed source `03b7dce3f70ea13d972febba8702536785c78fb0`, both serial native releases passed all 88
historical assertions, every retained subgroup, all eight independent claim
children and complete fixture/host teardown, within the explicit amended lab
budgets. Fresh full portable review passed 124 tooling tests, 11 race packages and
44 bounded fuzz targets. The [manifest](../../testing/M2/property-claim-fixtures-manifest.json)
closes these observer pool gates only; historical partial/failed results retain
their actual sources. Full root backend, native eligibility/effect interruption/
recovery, operator workflow and M2 acceptance remain separate.

## Original claim applying acknowledgement amendment

Require portable exact-successor/receipt/hold/immutable-record denials and
non-FreeBSD zero/unavailable behavior. The
[contract](../../testing/M2/property-claim-applying.md) adds 23 actual metadata
children to the existing submission parent, with actual credential-child denial.
Require exact durable applying bytes and owned acknowledgement, one publication
across copied/concurrent claims, missing/unsafe/foreign/stage/hold denials,
whole-temporary capacity, late policy/job presentation/hold drift, actual
sync/rename/cleanup cuts and uncertain changed readback after sync. Preserve
queued or visible applying bytes as appropriate, stable lock inode, absence of
unowned cleanup and complete lease/directory/backing/guest/process teardown.
Retain every historical native state/backing/codec gate and all 19 live-state
cases on both serial pinned releases; shared selectors/checkers/deadlines and
all pool observer/control fixtures remain unchanged. Native input selection
remains synthetic; actual metadata operations grant no native eligibility.
Source-bound artifacts, baseline and affected race/fuzz evidence are required.
Complete backend, actual effects/interruption/recovery and M2 remain separate.

Signed source `80f3588a89abcf8484f45da06f37eca18bcfa6dd` passed both new portable parents, all 15
exact-successor cases, mandatory baseline/124 tooling tests, 9 affected race
packages and all 38 selected fuzz targets. Both native releases passed every
required historical gate, all 19 live-state and 23 applying children plus actual
credential denial and complete teardown. All shared pipelines/readers/runners,
old limits and 61 prior manifests remain unchanged; see the
[manifest](../../testing/M2/property-claim-applying-manifest.json). This qualifies
metadata acknowledgement only; full backend, native effects/recovery, operator
workflows and M2 acceptance remain open.

## Original claim successor acknowledgement amendment

Require the [successor contract](../../testing/M2/property-claim-successor.md) for
partial M2-JOB-001, M2-STATE-001 and M2-RECOVERY-001 evidence. Portable tests must
cover exact private acknowledged predecessor lineage, the four permitted edges,
changed/missing/foreign records, complete holds and terminal/reconciliation replay.
Both actual disposable FreeBSD targets must pass all 38 new children, the real
non-root credential child, all 23 applying/19 live-state children and every old
state/backing/codec assertion under unchanged deadlines and shared checkers.
Native phase cuts must preserve exact bytes, private receipt/revision, stable lock,
owned output and no-retry behavior; verify complete group/fixture/backing teardown.
Matching metadata/digests grant no native outcome/quiescence/eligibility authority.
Existing claim readiness/readers and native pool/control sources are unchanged;
retain their historical source-bound evidence without relabeling. Full backend,
actual effects/interruption/recovery, transport and M2 acceptance remain separate.

## Original claim control binding amendment

The [contract](../../testing/M2/property-claim-control.md) requires eleven portable
applying-lineage cases and eighteen binding denial cases plus exact record/native/
baseline/false-eligibility and repeated live/metadata checks. Both disposable state
releases must retain all historical gates and add nineteen acknowledged-applying
read checks, actual non-root denial, unchanged metadata/lineage/lock and teardown.
Both complete pool suites retain the original eight claim children; six nested
control-binding cases add raw/clearance/ledger denial, one actual quota effect,
repeat refusal, independent false-eligibility postcondition and unchanged applying
records. Use only declared separate metadata/data fixtures, explicit synthetic
capture/live clearance and armed exact quota-source restoration. The actual
non-root control child must refuse the factory. Shared checkers/selectors, old
controls/guard cases, all deadlines and historical manifests stay unchanged.
Require fresh affected native suites plus focused portable review and verified
teardown. This does not qualify full backend session/recovery/global authority,
effect-death/reboot/recovery, operator workflows or M2 acceptance.

The [manifest](../../testing/M2/property-claim-control-manifest.json) records both
state releases at their actual implementation source and both repaired pool runs
at the fixture correction, with unchanged state inputs and no relabeling. All
nineteen/six new native cases, every old gate and mandatory/focused portable review
passed with verified teardown. The initial failed quota-restoration attempt remains
unqualified; only its fixture was corrected and rerun. Full authority/M2 gates
remain open.

## Original coordinator execution authority amendment

Require the [binding contract](../../testing/M2/property-execution-authority.md)
for partial M2-PLAN-001, M2-JOB-001, M2-DRIFT-001 and M2-RECOVERY-001 at T1/T2.
Nine new parents cover owned exact admission evidence, inactive/bounded binding
outside the original mutex, missing-binder refusal, ten binding cuts, seventeen
live denials, three final clock/context cuts, eleven attempt retirement cases,
late verified success, request-independent worker context and 32 concurrent
pending/active checks. Keep all ten old handoff/uncertainty parents; extend the
existing bounded handoff fuzz actions with copied authority and failed binding.
No second admission, handoff or effect is permitted. Require failing-first
behavioral evidence, mandatory baseline, affected race/all selected fuzz and an
unexecuted FreeBSD compilation record with artifact removal. Native adapters,
transactions, fixtures, shared tools and historical manifests are unchanged;
retain their actual sources without redundant native reruns or relabeling.
Full root backend/authentication/eligibility/recovery/global launch and runtime
writer gates remain open; retained session context is not authentication.

Signed source `68a6cf6e1794d258be31c42a7b7913574fec7311` passed all required
nine new parents/41 children, ten retained handoff/uncertainty parents, baseline/
125 tooling tests, nine race packages and all 38 selected fuzz targets in 212.450
seconds, with both supervisors cleaned. FreeBSD compilation passed without
execution; the artifact was removed. The initial sandbox socket-denial baseline
remains unqualified. All 65 historical manifests and 364 protected paths retain
exact base bytes, with verified lab teardown and disabled Actions; see the
[manifest](../../testing/M2/property-execution-authority-manifest.json). Complete
native/root backend, authentication, recovery/global and M2 gates remain open.

## Private native property backend amendment

Require the [integration contract](../../testing/M2/property-native-backend.md)
for partial plan/property/job/privilege/drift/state/recovery evidence. Eight portable
parents require the complete sequence, eight admission denials, ten phase cuts,
nine original-binding/outcome refusals, twelve lost/foreign/invalid acknowledgements, three final clearance cuts
and 32 concurrent worker calls. Preserve all old coordinator/worker/control tests
and selected fuzz targets; record the denying-stub behavioral red separately from
an unexecuted fixture-field compile error. Both complete pinned pool suites must
retain every old PASS name/count and add twelve nested backend children exactly
once, including real applying-before-effect, exact durable success/independent
quota-source facts, holds/no replay, native clearance/closed-lease denial and lost
applying/success acks after actual publication. Require actual non-root factory
refusal, unchanged fixed runner/guard, test-only clearance explicitly named, exact
GUID-checked quota restoration, all metadata/policy/lock/sentinel/mount/backing
preservation and complete owned teardown. Use separate serial success/denial and
response-gap fixtures, each with its own 135-second context after the original
claim fixture closes; retain all historical PASS names and require both new groups,
backing/preservation/teardown names and twelve new children once. Keep shared
runner/checkers and existing deadlines
unchanged; old-only runner success cannot qualify the new group. Native metadata
transactions/fixtures outside the pool suite are unchanged and retain historical
source-bound evidence. Full authenticators/executor-death/reboot/recovery/global
proof, operator workflows, reservation release and assembled M2 remain open.

Executed partial integration evidence: signed source `649a60b1b6a28f5c791f1d1332e44362d05a98ab`
passed the eight new portable parents/42 children, baseline/125 tooling tests,
nine race packages and all 38 selected fuzz targets. Both native targets retained
every prior PASS name/count and passed twelve new cases plus their separate
bounded fixture/backing/preservation/teardown groups. The initial native timeout
remains unqualified; the repair changes only fixture setup, preserving old
deadlines and shared tooling. All guest/artifact/supervisor cleanup was verified;
see the [manifest](../../testing/M2/property-native-backend-manifest.json).
External authenticators/recovery/global launch and assembled M2 gates remain open.

## Live application-session authority amendment

Require the [application contract](../../testing/M2/session-authority.md) for
partial M2-PLAN-001/M2-PRIV-001/M2-HTTP-001 T1/T2 evidence. Nine parents/51 children
cover original credential-free stable context, seventeen request denials, eleven
revocation/invalid-context cases, ten original-entry denials, ten entropy/final
cuts, 32 concurrent requests, no idle renewal, read isolation and sticky retirement.
Preserve old authentication/HTTP/consumer tests. Record denying-stub and changed-
CSRF/creation-time/retirement behavioral red results; require bounded 32-action
fuzz, affected race/fuzz review, explicit unexecuted FreeBSD compile and artifact/
supervisor teardown. Native facilities/fixtures/shared tooling stay unchanged;
no kernel-peer/root/runtime/browser or full M2 acceptance is inferred.

Executed partial evidence: signed source `3ddd368e747ae36813c009cdd10f6aa097cf41d7` passed
all nine new parents/51 children, retained authentication/HTTP/consumer tests,
baseline/125 tooling tests, eight race packages and all eleven selected bounded
fuzz targets. FreeBSD compilation passed without execution; the artifact was
removed. All supervised/lab teardown passed; native/shared facilities and 67
historical M2 manifests retain actual sources. See the
[manifest](../../testing/M2/session-authority-manifest.json); full privilege,
transport/recovery/runtime/M2 acceptance remains open.

## Application-context negotiation amendment

The [negotiation contract](../../testing/M2/action-negotiation.md) supplies partial
M2-PRIV-001/M2-HTTP-001 T1/T2 evidence. Require denying-stub red tests followed by
exact hello/ready bytes, strict field/type/version/operation/claim denials,
request/context mismatch refusal, every frame truncation, fixed preallocation
bounds, short/error write refusal and no retry. Preserve version-1 behavior; an
owned temporary Unix socket must reject negotiation without collector dispatch
(the injected UID is portable evidence only). Use real original authenticated
POST authority for successful exchanges and reject nil/zero/decoded/revoked
capabilities, logout/expiry/cancellation during the exchange. No credentials or
idle renewal; copied capability retains the same original binding. Twelve parents/
57 children and bounded 516-byte request/response fuzz, all affected race/fuzz
checks, unexecuted FreeBSD compilation and artifact/supervisor teardown are
required. Existing native facilities and shared tools stay unchanged; no kernel,
root recovery, operation, browser or complete M2 acceptance is inferred.

Executed partial evidence: signed source `e449b8c005f2f4635f471305a6e2cf8ab11a1100` passed
all twelve parents/57 children and three fuzz seeds, baseline/125 tooling tests,
eight affected race packages and all twelve bounded fuzz targets in 67.911
seconds. Both admin/helper FreeBSD test packages compiled without execution;
checksummed artifacts were removed. Review supervisors and all 612 owned
sessions stopped with verified teardown; Actions remain disabled. All 68
historical M2 manifests and 386 existing Go/tooling inputs are byte-identical
to the merged base. See the [manifest](../../testing/M2/action-negotiation-manifest.json).

## Native action-peer amendment

The [peer contract](../../testing/M2/action-peer.md) supplies partial M2-PRIV-001/
M2-NATIVE-001 T1/T3 evidence. Require actual reexec UID drop before connecting,
actual root/kernel nonroot UID binding, root/wrong-UID rejection before reads,
nonroot helper rejection before writes despite repaired file ownership, local
UID denial, schema/version/operation refusal, idle/truncated cuts and retained
connection cancellation/deadline/disconnect/extra-frame/close denials. A canonical
service-asserted context is correlation, not independently verified human/session
identity. All eighteen named native children and unique complete result, exact
artifact hash and owned socket/binary/guest/host/supervisor teardown must pass on
both release candidates. Reject unissued/decoded private peers portably; preserve
existing inventory/negotiation tests. Shared runner/cleanup changes require fresh
full portable review; old suites retain their actual source-bound evidence and
unchanged commands/deadlines. Record failing-first native stub evidence without
qualifying it. Runtime operations/recovery/global launch/M2 acceptance remain open.

Executed partial evidence: signed source `4222e8f098114df8942fb4843f7e683ae9068ee7` passed
four new portable parents/nine children, baseline/131 tooling tests, all 11
portable race packages and all 46 bounded fuzz targets in 223.465 seconds.
The shared-tooling change selected a fresh full review. The same exact executable
passed all eighteen native cases on FreeBSD 15.1-RELEASE-p3 and 14.5-RELEASE,
in 33.303/36.617 seconds including boot/teardown. Wrong/root UID, nonroot helper
despite repaired file ownership, EOF and extra-frame denials used actual kernel
facilities. Every socket/fixture/artifact/guest/supervisor cleaned up; all
618 owned sessions stopped. The earlier native stub run is explicitly
unqualified with verified cleanup. All 69 historical M2 manifests and 383
protected existing inputs retain actual bytes/sources; existing native peer/
socket functions and old suite commands/deadlines are preserved. See the
[manifest](../../testing/M2/action-peer-manifest.json). No runtime operation or full recovery/global
launch/acceptance follows. Actions remain disabled.

## Typed property request prerequisite

PR 94 accepted the original native peer boundary at merged source
`543889037dba05df0906130ddcc720e995afde9f`. The next
[request contract](../../testing/M2/property-requests.md) defines strict version-2
property review/submission selections: canonical pool/filesystem GUIDs and the
planner's literal property allowlist, or only pool/plan/submission identifiers.
Each schema rejects extra privilege fields and replacement intent. Original request
correlation and fixed frame/field/depth limits are syntax checks, never peer/session,
enrollment, original plan, recovery or launch proof. Inventory runtime and original
peer quiet behavior remain unchanged; response/exchange/dispatcher integration
remains pending. This is a T1 subset of M2-PLAN-001/M2-PRIV-001, not native or M2
acceptance. Baseline/focused race/fuzz and unexecuted FreeBSD compilation are
recorded below; preserve historical source-bound native evidence.

Executed partial evidence: signed source `2c5d89680f6be5be790c92c741f8f965f2f5bf2a` passed
five behavioral parents/61 children and three fuzz seeds, baseline/131 tooling
tests, 9 affected race packages and all 41 bounded fuzz targets in 227.424
seconds. Helper/storage FreeBSD test packages compiled without execution and their
checksummed artifacts were removed. All 623 owned sessions stopped with
verified teardown; Actions remain disabled. All 70 historical M2 manifests and
393 protected existing inputs retain actual bytes. Existing planner validation is
unchanged apart from the new bounded syntax entry point. The earlier lint failure
is explicitly unqualified and corrected before this fresh review. See the
[source-bound manifest](../../testing/M2/property-requests-manifest.json). No native typed exchange,
root recovery/global launch or assembled acceptance follows.

## Bounded property reply prerequisite

PR 95 accepted typed property requests at merged source
`43cd4b11b10ff7b155b71a6e0ef910a4b40e5089`. The next
[reply contract](../../testing/M2/property-replies.md) preserves complete review
and lifecycle display views with exact original correlation/selection, plus bounded
refusal codes. Shared converters/validators preserve the existing durable formats;
private session/policy digests and execution proofs stay outside replies. The fixed
64 KiB frame cap includes the envelope and never permits truncated views/history.
Admission/duplicate/refusal labels remain peer assertions, not durability or retry
permission. This is a portable T1 subset of M2-PLAN-001/M2-PRIV-001/M2-JOB-001;
owned socket phases, original session/kernel authorization, enrolled root policy,
recovery/global launch and actual admission/native execution remain pending.
Baseline/focused race/fuzz and unexecuted FreeBSD compilation are recorded below.

Executed partial evidence: signed source `728387ba21d322f5ee42426bdf617d527e2afdda` passed
twelve behavioral parents/36 children and five fuzz seeds, baseline/131 tooling
tests, 9 affected race packages and all 43 bounded fuzz targets in 225.190
seconds. Helper/storage FreeBSD test packages compiled without execution; exact
checksummed artifacts were removed. Independent old/new synthetic intent/property-job
record hashes match merged PR 95 and are guarded by golden regression tests.
All 71 historical M2 manifests and 395 protected existing inputs retain actual
bytes. All 628 owned sessions stopped with verified teardown; Actions
remain disabled. The earlier native-selector naming collision is explicitly
unqualified and corrected before this fresh review. See the
[source-bound manifest](../../testing/M2/property-replies-manifest.json). Owned exchange, root
recovery/global launch, actual admission and full M2 acceptance remain pending.

## Owned property exchange prerequisite

PR 96 accepted bounded replies at merged source
`f36876b0de6b278634ec5718efa0f901c7093e59`. The next
[exchange contract](../../testing/M2/property-exchange.md) owns one original
kernel-authorized connection across negotiation, one strict typed request and one
terminal correlated reply. Fixed existing frame limits bound four-frame aggregate
traffic; original deadlines never renew. Terminal EOF is required before returning
client metadata, without asserting peer liveness after EOF. Root parsed identities
remain snapshots; no application-session, enrolled policy, original plan, recovery,
durability or launch authority is granted. Native synthetic replies qualify only
kernel/transport behavior. Single-operation/overlap/cancellation/partial-stream and
role/identity/correlation cases join the unchanged eighteen old native peer cases
in a separate opted-in parent; strict complete evidence is mandatory on both pinned
releases. Shared checker/selector maintenance requires fresh full portable review.
Test-first native and full portable qualification are pending. Actual backend
admission, authenticated session assembly, recovery/global launch and M2 acceptance
remain open.

## Bounded property connection admission prerequisite

The [connection contract](../../testing/M2/property-connections.md) is a portable
T1 subset of M2-PRIV-001/M2-JOB-001. Use disposable actual Unix sockets to verify
configuration bounds, exact original descriptor/context and no input consumption,
zero/decoded/reconstructed/copied lease denial, overflow without bytes/waiting,
closed/nil/duplicate descriptor denial, unrelated socket preservation and slot
reuse. Concurrent last-slot attempts admit exactly one owner; concurrent release/
shutdown closes all owned sockets. Parent cancellation, fixed deadline and an
earlier parent deadline interrupt blocked I/O without phase/traffic renewal.
Retired leases return no descriptor/context. Compile the helper tests for FreeBSD
without execution, run baseline/affected race/fuzz and verify process teardown.
Preserve existing kernel peer primitives, native fixtures/commands/deadlines and
historical source-bound evidence. Socket closure supplies no worker/executor
quiescence or storage authorization; actual native assembled safety/recovery/
continuity/resources and full M2 acceptance remain open.

## Admitted property exchange prerequisite

The [admitted exchange contract](../../testing/M2/property-admitted.md) is a
T1/T2 subset of M2-PRIV-001/M2-JOB-001. Portably deny unissued/copy/retired composed
owners and release failed UID/unsupported-host negotiation capacity without touching
unrelated originals. A new separate native parent qualifies actual configured/root/
wrong UIDs, complete synthetic review/submission/refusal exchanges, automatic
terminal/malformed slot reuse, overflow before bytes, original negotiation/phase/
earlier-parent deadline equality, shutdown during both I/O phases, overlapping
negotiation, duplicate lease retirement and full teardown. Child credentials apply
before exec; no live storage is used. Require all three original/composed native
parents and exact case/result inventories with unchanged 80/90-second deadlines.
Shared tooling changes require fresh full portable review; execute failing-stub
native proof before implementation and retain it as unqualified. Full authenticated
session/backend/recovery/global launch and assembled M2 qualification remain open.

## Session-bound property client prerequisite

Require the [client contract](../../testing/M2/property-client.md) for partial
M2-PLAN-001/M2-PRIV-001/M2-HTTP-001 T1/T2 evidence. Start with denying stubs and
real original-session tests; deny reconstructed/mismatched/revoked/expired/rollback/
cancelled authorities without idle renewal. Retire invalid/unsupported factories.
A separate native parent uses actual configured nonroot reexec credentials and
root Unix peers, synthetic display replies and callback phase-cut fixtures (never
kernel UID hooks): review/submission/duplicate/refusal, every guarded phase,
copy/invalid selection denial, fixed missing-reply deadline and fixture removal.
Strictly retain every old 60 case and fourteen new client cases on both pinned
releases, original 80/90-second bounds, source/hash provenance and verified cleanup.
Shared tools require fresh full portable review. Full native authenticated runtime,
backend/recovery/global launch and M2 acceptance remain separate.

## Private root property listener prerequisite

Require the [listener contract](../../testing/M2/property-listener.md) for partial
M2-PRIV-001/M2-SCALE-001/M2-NATIVE-001 evidence. Portable failing-first tests check
separate worker/socket retirement, original/copy/duplicate lease identity, 32-way
last-slot exclusivity, config bounds and unsupported-host denial. Actual root/native
fixtures must qualify all eighteen new cases with configured/wrong/root reexec UIDs,
typed default refusals, malformed/legacy denial, immediate overflow/recovery, idle
expiry, both shutdown phases, original socket/foreign-entry preservation, unsafe
directory/owner/alias and nonroot constructor denial, single-serve and full removal.
Retain all 74 original cases, five exact native parents and unchanged 80/90-second
bounds on both pinned releases. Shared tools require fresh full portable review;
store signed denying-stub failures separately. No callback/native write or runtime
registration is added. Session/backend/recovery/global launch/assembled M2 gates
remain mandatory.

## Configured property client prerequisite

Require the [configured connection contract](../../testing/M2/property-dial.md)
after the accepted private listener. Reuse trusted helper configuration and actual
kernel identity, original safe directory/socket policy and session checks. Repeat
namespace identity/policy after dialing, before protocol bytes; own one original
bounded dial/negotiation/operation lifetime and retire it on every terminal path.
Keep raw-socket APIs and all 92 earlier native cases. Add twelve actual configured
cases, six exact native parents, unchanged 80/90-second bounds and signed failing-
first evidence. Shared checker/selector changes require fresh full portable review.
Private configured adapters register no route/service/backend or native write;
independent root application/recovery/global launch gates remain mandatory.

## Native original application client integration amendment

Require the [native application contract](../../testing/M2/property-application.md)
for partial M2-PRIV-001/M2-PLAN-001/M2-NATIVE-001 evidence. Signed denying stubs
must fail the existing configured portable contract and new native assembly.
Retain all 104 original transport cases and add eleven actual Auth/action/client
cases: seven parents/115 exact cases on both pinned releases, under unchanged
80/90-second bounds. Synchronize real original-store logout/Close before root
continues; require the exact revocation error, zero metadata and terminal denial.
No credential/context output or fake UID/session/phase hook is permitted. Preserve
source-bound red/green and complete cleanup; shared checker changes require fresh
full portable review. Independent root/backend/recovery/global gates remain open.

Signed source `ea63b809858c2edd9561197ad33d473879fa09ca` passed all 115 actual cases on both disposable FreeBSD
releases, three portable parents/seventeen children, baseline/137 tooling tests
and fresh full portable review (eleven race packages/all 49 bounded fuzz targets,
218.33 seconds). All 678 owned sessions stopped with verified teardown;
Actions remain disabled and all 78 historical manifests retain their bytes.
Signed red evidence remains unqualified; see the
[source-bound record](../../testing/M2/property-application-manifest.json).
This is native application/client integration, with root operations still denied;
independent root/backend/recovery/global/runtime gates remain open.

## Original root request binding amendment

Require the [binding contract](../../testing/M2/property-binding.md) for partial
M2-PRIV-001/M2-PLAN-001/M2-NATIVE-001. Signed stubs must fail concrete portable
retirement and native original binding. Preserve all 115 existing native cases;
add sixteen actual root peer/request/retirement cases, including isolated snapshots,
forged lineage, synchronized disconnect/extra frame, original deadline, 32 live
checks and nil/cancelled context. Require eight parents/131 exact cases on both
pinned releases with unchanged 80/90-second bounds, fresh full portable review
for shared checker edits and complete source-bound preservation/teardown. No
human/session, policy, backend, recovery, native launch or quiescence follows.

Signed revised fixture source `ee4a997fd17eba420e6231b02c506fe90a304dde` passed all 131 actual cases on both
pinned disposable releases, two portable parents/six children, baseline/138
tooling tests and fresh full portable review (eleven race packages/all 49 bounded
fuzz targets in 222.215 seconds). The first implementation's native run exposed a
prior shutdown readiness race; its new binding cases passed. The fixture now
observes actual completed child negotiation, with unchanged product/deadlines.
Initial native and interrupted deep records stay unqualified. All 687 owned
sessions stopped with verified teardown; Actions remain disabled and all 79
historical manifests retain their bytes. See the
[source-bound record](../../testing/M2/property-binding-manifest.json).
This is live peer/request provenance only; all other clearance/runtime gates stay open.
