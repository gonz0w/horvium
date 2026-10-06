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
`123044304e75f6df27d43eee085c23435ddbcce1`; see the
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
codec subtests on both releases. Native qualification is pending; complete
provisioning, coordinated races, publication/admission/recovery and M2 acceptance
remain open.
