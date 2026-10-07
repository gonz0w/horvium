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

## Traceability and required gates

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
