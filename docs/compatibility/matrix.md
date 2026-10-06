# Compatibility matrix

**No production support is claimed.** Architecture targets and observed
validation are separate. Refresh exact versions in each active research manifest.

| Area | Target | Evidence/status |
|---|---|---|
| Runtime | Approved M1 FreeBSD 15.1 primary / 14.5 compatibility | Native reader/helper/cache/admin fixture tests passed; foreground runtime/console/CLI owned-fixture tests passed on both releases; full NAS runtime pending |
| Architecture | amd64 first candidate; other FreeBSD architectures as validated | Hardware-neutral design, no universal support claim |
| Storage | ZFS pools/datasets on supported direct-access devices | Observed by M1; M2 ownership/creation/property/maintenance contracts approved in PR 20; no M2 product mutation evidence |
| File clients | macOS and Linux/BSD first; SMB/NFS/SFTP as designed in M3 | Priorities confirmed; version/permission matrix and actual tests pending |
| Backup peer | Compatible ZFS receiver over authenticated SSH | Planned; feature negotiation and restoration pending |
| Management | Go 1.27.1 / HTMX 2.0.11; Safari/macOS and Firefox/Linux first | Read-only semantic console and authenticated CLI implemented in accepted slice 5; Firefox fixture evidence below; Safari/manual acceptance pending |
| Distribution | FreeBSD port/package, then image | Packaging plan only; not in official Ports |
| Small hardware | 16 GB MicroServer Gen8 / four 3 TB SATA disks | Reference profile supplied by maintainer, unmeasured |
| Larger hardware | Representative commercial FreeBSD deployments | Synthetic management budgets passed; physical/commercial lab evidence pending |
| Foundation smoke | FreeBSD 15.1 amd64 | Shell/hooks passed; not native NAS evidence |
| Native research | FreeBSD 15.1-RELEASE-p3 / 14.5-RELEASE amd64; Go 1.27.1 | Narrow TSV/GEOM/credential fixture assertions passed; see spike |
| Product discovery reader | Same two observed release/patch levels, Go 1.27.1 amd64 | Actual Go reader/parser and isolated pool fixture passed; see product evidence |
| Product helper | Same two observed release/patch levels, Go 1.27.1 + x/sys v0.48.0 amd64 | Actual kernel UID, socket ownership/lifecycle, admission/timeout and reader-over-transport tests passed |
| Collector/cache | Hardware-neutral Go library; FreeBSD helper integration | Local generation/ownership/concurrency/boundary tests and actual unprivileged native helper outage/recovery passed |
| Administration security | Hardware-neutral auth/config parser; FreeBSD descriptor ownership and loopback HTTP | Local denial/expiry/capacity tests and native credential ownership/atomicity/dropped-UID real HTTP fixtures passed on both releases; UI/CLI and deployable auth pending |
| Console/CLI | Go 1.27.1 with locally embedded HTMX 2.0.11 | Local auth/parity/escaping/strict-query/expiry tests pass; limited in-app browser login/filter/keyboard/refresh/logout and 390-pixel layout verified; actual root-helper/unprivileged foreground runtime/CLI/HTML with owned GUID/property/teardown assertions passed on both releases |
| Runtime lifecycle / budgets | FreeBSD foreground processes; hardware-neutral response admission/logging | Local admission/logging/lifecycle tests pass; actual built-process kill/restart/outage/recovery and independent fixture-only SFTP reads passed on both FreeBSD targets; 100/10,000-row handler diagnostics recorded; synthetic combined RSS/CPU/latency thresholds passed on both targets; later native-reader resource and SMB/NFS continuity fixtures passed as recorded below; full M1 acceptance pending |
| Performance qualification | Checksum-pinned FreeBSD 15.1/14.5 QEMU images; qemu64, two vCPUs, 6144 MiB | Actual suite and evidence validation passed on both targets (run 37218753397); small idle CPU 0.020–0.023%, large p95 66–90 ms and combined RSS 43–44 MiB; later native-reader resource results are recorded separately below; hardware/commercial support remains unqualified |
| Helper denial events | FreeBSD kernel authorization and fixed-schema bounded logging | Local denial/redaction/blocked-sink/cancellation tests and actual dropped-UID, admission recovery and built-helper stderr assertions passed on both targets (run 37221158595); sampled logs and full abuse review remain open |
| Native probe cleanup | FreeBSD 15.1-RELEASE-p3 / 14.5-RELEASE amd64, Go 1.27.1 | Local synchronized regressions and all eight actual executor/collection cases passed on both native targets (run 37225094799); unrelated fixture group survived; abrupt helper SIGKILL and escaped groups remain outside this result |
| Helper crash containment | FreeBSD 15.1-RELEASE-p3 / 14.5-RELEASE, Go 1.27.1, base timeout reaper | Actual executor owner-death and guarded bounds cases passed on both targets (37227688891), including separate-session descendant; PR 13 accepted by maintainer merge; utility hashes recorded; built-helper captured-command death/recovery and native resources additionally passed in 37233391796 and were accepted with PR 14 |
| Native reader resource procedure | Pinned FreeBSD 15.1/14.5 QEMU guests; actual small native collection and test-only 10,000-row TSV projection | Both targets passed native-small/10,000-row projection budgets and integrated captured-command death/recovery (37233391796); small RSS 32.8–36.5 MiB, idle CPU 0.057–0.087%; projection p95 44–82 ms; no hardware/commercial qualification |
| Assembled HTTP abuse | Built FreeBSD root helper/unprivileged runtime over real loopback TCP | Actual denial/flood/slow deadline/admission/recovery/redaction/unchanged-state tests passed on both targets (37230688573); portable connection overflow/recovery/shutdown race checks pass; accepted with PR 14 |
| Independent SMB/NFS continuity | Fresh FreeBSD 15.1/14.5 disposable guests; fixture-only Samba and base NFSv3 TCP | Both protocols passed seven fresh reads through helper/management outages, session restart and full teardown on both targets (37233530125); accepted with PR 16; no external client/ACL/throughput claim |
| Tooling | Python 3.14.7 + checksum-pinned tools on Linux/macOS | See bootstrap status for checks actually run |
| Local qualification tooling | Mac arm64 / QEMU 11.1.1 TCG; disposable FreeBSD 15.1-RELEASE-p3 and 14.5-RELEASE amd64 guests | [Local runner](../testing/local-qualification.md) executed all 51 property assertions on both targets and foundation shell/DCO smoke on 15.1, with base/runtime preservation and owned process/key/image cleanup; see [execution manifest](../testing/local-qualification-manifest.json); [device fixtures](../testing/M2/devices-local-manifest.json) passed all eight assertions and full teardown on both releases; other suites/Linux host through this runner, macOS Firefox and performance equivalence remain unqualified; no new runtime support claim |
| Development teardown | Local macOS foreground process groups | Real descendant/owner-SIGKILL/TERM-escalation/timeout/operator-stop regressions pass; unrelated sentinel survives; leased supervisor used by local guests, review, preview and browser fixture/driver; [executed teardown manifest](../testing/local-teardown-manifest.json) records actual foundation/15.1 success, live QEMU controller SIGKILL cleanup and real preview operator-stop/listener closure; Linux host, detached external applications, supervisor death and host power loss remain outside this result |
| M2 ownership/planning model | Internal Go GUID scopes and immutable property reviews; runtime stays read-only | D1 model accepted in PR 21; scope/impact/expiry/drift/bounds and received-value context; D2 bounded property parser/source/GUID/received-value assertions and teardown passed on both pinned FreeBSD guests (37242881791); see [native projection](../testing/M2/properties.md); D2 identity/type coverage, snapshot/bookmark owner/GUID, replicated GUID, rename/name-reuse, volume refusal and stable enumeration assertions passed on both guests (37244653409); see [identity binding](../testing/M2/identities.md); D2 [filesystem mount/readonly/crypto constraints](../testing/M2/dataset-state.md), identity/source binding, loaded/unloaded refusal, drift and preservation assertions passed on both guests (37246040343); D2 [pool identity/import/policy observations](../testing/M2/pool-state.md), value/source correlation, bootfs/readonly refusal, import-incarnation/review drift, preservation and pool name-reuse assertions passed on both guests (37247751495); accepted in PR 25; D2 [feature observations and exact common profile checks](../testing/M2/features.md), complete 39/47-feature GUID/source binding, read preservation, installed profile/hash, resulting common feature set and feature/version drift passed on both guests (37249224834); accepted in PR 26; [vdev topology/review drift](../testing/M2/topology.md), paired GUID/path binding, mirror/RAIDZ2 recognition, independent membership, degraded/stripe refusal, read preservation and vdev GUID reuse passed with all 51 assertions and teardown on both guests (37252047398); accepted in PR 27; full scan/JSON status and provider/system/device-use proofs remain open; cross-release transfer/import remains open; no runtime native adapter, persisted enrollment or submitted jobs. ADRs 0006/0007 and designs approved in PR 20; complete slice 1 and native/state/mutation gates pending; see [planning evidence](../testing/M2/planning.md) |

Evidence levels: D0 designed; D1 unit/component verified; D2 native/client verified;
D3 workflow/restore verified; D4 release-qualified with failure, upgrade, security,
and performance gates. Mark unavailable/skipped/failed separately from passed.
Commercial deployment size alone is not a D4 claim.

See [M1 native spike](../research/M1/native-spike.md) for source/run IDs, bundled
ZFS version strings and fixture limits, and [M1 Test Design](../milestones/M1/TDD.md)
for product tests still required and [product reader evidence](../testing/M1/discovery.md)
for the completed reader slice and [helper evidence](../testing/M1/helper.md)
for actual Go transport tests; [collector evidence](../testing/M1/collector.md)
records the unprivileged cache/outage/recovery path;
[security evidence](../testing/M1/security.md) tracks configuration, credentials
and the administration boundary; [console evidence](../testing/M1/console.md)
records native foreground/CLI/HTML and limited synthetic browser execution;
[runtime evidence](../testing/M1/runtime.md) records built-process restart,
independent SFTP continuity and bounded response/logging behavior; [performance qualification](../testing/M1/performance.md)
tracks the combined-process synthetic workload separately. The earlier C spike is ABI
evidence only. No
physical identity, file-client, scale or throughput claim follows.

Actual Firefox 157.0/geckodriver 0.37.1 passed the synthetic console's four
JavaScript enabled/disabled, light/dark and verified 1280/390/320 CSS-pixel flows
on Ubuntu 24.04.5 (37233600130); automated qualification was accepted with PR 16.
Safari 27.0 automation was unavailable; manual
accessibility/native zoom remains pending by maintainer choice. See
[browser evidence](../testing/M1/browser-checklist.md). These fixtures do not
establish FreeBSD runtime or SMB/NFS client compatibility.


The final combined assembled-source run 37234167265 passed all eight jobs,
including both pinned resource guests, both native suites/HTTP/SMB/NFS fixtures
and actual Firefox. Native-small combined Go RSS 33.65–35.45 MiB and idle CPU
0.080–0.083%; projection RSS 40.20–40.79 MiB, p95 57.877–58.634ms. All exclusion,
manual-browser and hardware/client/release limits above still apply; M1 remains
unaccepted and Validating. See [consolidated qualification](../testing/M1/qualification-status.md)
and the separate approved [M2 Test Design](../milestones/M2/TDD.md).

M2 virtual device fixture: [identity/geometry procedure](../testing/M2/devices.md)
and [local execution manifest](../testing/M2/devices-local-manifest.json) record all
eight actual assertions on both pinned releases through the Mac supervisor, with
backing/base/runtime preservation and disk/image/key/variable/group teardown.
Actual controller SIGKILL cleanup also passed, leaving the interrupted run
unqualified. Original failed/unstarted hosted jobs and both repaired Mac
failures remain separate historical evidence. PR 28 accepted this fixture; complete
provider/use/physical safety and product write eligibility remain unqualified.

M2 disk correlation: [bounded parser/contract](../testing/M2/disk-correlation.md)
all twelve fixture/correlation assertions and complete preservation/teardown passed
on both targets through the Mac supervisor; see the [native manifest](../testing/M2/disk-correlation-manifest.json).
PR 31 accepted this subset at merged source `ef60c33489a2ddd30c99e2a8e865a76272473f2b`.
It is not wired into collection or storage planning
and grants no device-use/write eligibility.

M2 disk attachments: [contract and procedure](../testing/M2/disk-attachments.md)
recognizes exact DEV/disk-ident correspondence and refuses PART/unsupported
attachments. Actual empty/populated GPT denial, foreign preservation and full
teardown passed with all seventeen assertions and both native tests on both
targets through the Mac supervisor; see the [native manifest](../testing/M2/disk-attachments-manifest.json).
PR 32 accepted this subset at merged source `f53688f29d6ba85e47b7062bbada73a857f02378`;
local behavioral/cross-build coverage is separate.
No complete device-use, physical/locality or write eligibility follows.

M2 system device use: [contract and procedure](../testing/M2/system-device-use.md)
correlates positive root/mount/swap backing through complete GEOM and mounted
ZFS pool GUID/status captures. Actual UFS/swap/root preservation and full teardown
passed with all twenty-three assertions and three tests on both targets through
the Mac supervisor; see the [native manifest](../testing/M2/system-device-use-manifest.json).
PR 33 accepted this subset. No absent-signature/pool-use, physical/locality,
complete device clearance or product mutation claim follows.

M2 imported-pool use: [contract and procedure](../testing/M2/imported-pool-use.md)
binds the all-imported-pool catalogue, independent GUID/status and GEOM captures
to positive disk membership, including pools with no mounts. Portable behavioral
tests and a FreeBSD cross-build pass; actual unmounted-pool/root/foreign
preservation and full teardown passed all twenty-nine assertions and four tests
on both targets; see the [native manifest](../testing/M2/imported-pool-use-manifest.json).
PR 34 accepted this subset. No exported-pool or
signature absence, complete device clearance, collector/planner integration or
product mutation authority follows.

M2 filesystem signatures: [contract and procedure](../testing/M2/filesystem-signatures.md)
binds successful base `fstyp -u` UFS/ZFS scalar captures to independently
correlated direct disks. Portable tests and a FreeBSD cross-build pass; actual
UFS/ZFS/removed-pool metadata recognition, probe preservation and full teardown
passed all thirty-six assertions, four executed tests and complete host cleanup
on both targets; see the [native manifest](../testing/M2/filesystem-signatures-manifest.json).
PR 35 accepted this subset. First-label ZFS recognition is not checksum-verified
pool identity, all-label coverage, exported-pool membership or blank-disk clearance.
No collector/planner integration or product mutation authority follows.

M2 combined device use: [contract and procedure](../testing/M2/device-use-review.md)
retains selected disk correspondence, positive mount/swap/imported-pool backing and
optional signature recognition in an immutable baseline. Portable tests and a
FreeBSD cross-build pass; actual joined facts, fresh unchanged pool comparison,
new/removed-use drift and complete teardown passed all fifty-one assertions and
four tests on both targets; see the [native manifest](../testing/M2/device-use-review-manifest.json).
PR 36 accepted the combined device-use subset. No collector/planner integration, complete signature absence, unused flag, locality,
reservation, enrollment or mutation authority follows.

M2 positive pool labels: [contract and procedure](../testing/M2/pool-labels.md)
binds complete default zdb -l configurations to independently correlated disks,
retaining native pool state and requiring review on export/label drift. Portable
tests/cross-build pass; all fifty-nine assertions, four tests, actual export/reimport
and full preservation/teardown passed on both targets; see the
[native manifest](../testing/M2/pool-labels-manifest.json). Accepted in PR 37.
No all-label checksum integrity, signature absence, collector or write authority follows.

M2 aggregate disk access: [contract and procedure](../testing/M2/disk-access.md)
binds native GEOM access counts to independently correlated direct disks and
requires fresh review on count changes. Provider/consumer consistency and missing
mode refusal have portable coverage; all sixty-eight assertions, four native tests
and complete preservation/teardown passed on both targets. Actual direct/alias
read/write opens and close/restoration qualified on the owned virtual disk; see
the [native manifest](../testing/M2/disk-access-manifest.json). Accepted in PR 38. Zero counters
are not complete device clearance. No runtime collector or product writes follow.

M2 state directory: [contract and procedure](../testing/M2/state-directory.md)
implements a root-owned, descriptor-anchored UFS/ZFS directory lease and stable
single-writer advisory lock. Native ownership/permission/symlink/hardlink/FIFO
refusal, competing writers, successful sync, process-death release, foreign
preservation and teardown passed all thirty-five assertions on both declared
releases (15.1 ZFS root, 14.5 UFS root); see the
[source-bound manifest](../testing/M2/state-directory-manifest.json). Maintainer
review accepted in PR 39. Only default-ACL disposable fixtures are qualified. This is a
partial M2-STATE-001 prerequisite, not a record store or durable job admission.
M1 collection and the management runtime remain read-only.

M2 ownership records: [strict codec](../testing/M2/ownership-records.md) implements
version 1 ownership metadata using the existing policy model, required exact JSON
fields, duplicate/null/version refusal, canonical checksums and 64 KiB bounds.
Five behavioral tests and fifty-seven subtests passed without skips; complete
portable review passed with twenty-one bounded fuzz targets. Maintainer review
accepted in PR 40; see the [source-bound manifest](../testing/M2/ownership-records-manifest.json).
FreeBSD/amd64 compilation alone is not native file
qualification. No file operations, enrollment, durable jobs or runtime wiring exist.

M2 native ownership reader: [contract/procedure](../testing/M2/ownership-read.md)
adds bounded record reads through the held root directory, exact owner/mode/link/
type checks, file/name metadata comparisons and pool-GUID binding to the accepted
codec. Sixty-three assertions, two native tests, six unit tests and fifty-seven
codec subtests passed on FreeBSD 15.1-RELEASE-p3/ZFS root and 14.5-RELEASE/UFS root
at signed source `94cf66e9c3a34184c76b84858cb7ef998e4bd2dc`; see the
[source-bound manifest](../testing/M2/ownership-read-manifest.json). Complete
preservation/teardown passed; accepted in PR 41.
Default-ACL fixtures only; no enrollment, publisher or runtime write authority.

M2 state ACL validation: [contract/procedure](../testing/M2/state-acl.md) adds
actual-descriptor capability/ACL queries to ancestors, private directories, locks
and ownership records. The conservative policy rejects named principals,
inheritance and nonowner grants beyond its allowlist. The expanded state suite
requires 108 named assertions, ten parent tests and 57 codec subtests on both
releases; all passed at signed source `33f4d91bc57adb71f27ce80bbe0552fb1e7685d2`
with complete teardown. See the [manifest](../testing/M2/state-acl-manifest.json)
for 15.1-RELEASE-p3/ZFS-root and 14.5-RELEASE/UFS-root observations, including
owned UFS POSIX/NFSv4 controls and retained initial failure/repair. Maintainer
review accepted in PR 42. That accepted source provides no publisher, runtime
state provisioning or storage writes. The later publisher has separate evidence below.

M2 ownership publication: [contract/procedure](../testing/M2/ownership-publication.md)
adds descriptor-anchored bounded ownership metadata publication only. The expanded
state suite requires 161 assertions, fourteen parent tests and 57 codec subtests.
All passed on both targets at signed source `eabf4691fc55b825db74ba73501d758126fda4e7`,
with complete teardown; see the [manifest](../testing/M2/ownership-publication-manifest.json).
The 15.1 ZFS root and 14.5 UFS root passed actual publication; new owned UFS
fixtures passed zero-free-space ENOSPC/read-only/ACL controls. Review is pending.
Failure seams establish error accounting/order, not actual device EIO or power-loss recovery. No runtime enrollment, full store,
storage admission or hardware/commercial support follows.

Focused developer review: [policy and runner](../engineering/local-review.md)
select package consumers from host/FreeBSD/test imports and retain cheap baseline
checks on every PR. Fresh complete portable mode remains a milestone/release gate.
The prebuilt state path passed all 161 current assertions, 14 parent tests and
57 codec subtests on both declared native targets, with identical artifact hashes
and verified teardown; see [workflow qualification](../testing/focused-review.md). This tooling change
adds no OS/architecture, client, hardware or runtime feature support claim.


M2 ownership catalog: [contract/procedure](../testing/M2/ownership-catalog.md)
adds bounded ownership-only record inspection, complete namespace refusal and
ordinary drift checks. The expanded state gate requires 202 assertions,
seventeen parent tests and 57 codec subtests on both releases. Both targets passed all 202 at signed source `5ea125fdc815fd7c6c54aba36824e50ea876b9f6`,
with full teardown; see the [manifest](../testing/M2/ownership-catalog-manifest.json).
Accepted in PR 46. Inspection alone grants no write-admission, recovery,
scale, hardware or client support claim.


M2 ownership capacity: [contract/procedure](../testing/M2/ownership-capacity.md)
connects complete catalog validation and old-plus-new logical-byte/count reservation
to private publication. The expanded state gate requires 234 assertions, nineteen
parents and 57 codec subtests on both existing targets. Both passed at signed source
`bf80cb9f2e1c196eae330ae0d681d86e0c466033` with full teardown; see the
[manifest](../testing/M2/ownership-capacity-manifest.json). Accepted in PR 47. Reduced fixture limits add no populated scale, commercial hardware,
client, mixed-store admission or recovery support claim.


M2 job lifecycle codec: [contract/procedure](../testing/M2/job-records.md) adds
bounded in-memory property-intent lifecycle metadata and strict checksums/schema,
with no native/file/runtime wiring. Signed source `a3791513f6423fe5860064e93f89399c21dc1445`
passed seven tests/119 schema subtests and focused storage race/all 15 fuzz checks;
see the [manifest](../testing/M2/job-records-manifest.json). Accepted in PR 48. Existing
FreeBSD support candidates and their immutable native sources are unchanged;
compilation alone grants no native durable job/admission/recovery or scale claim.


M2 native job reader: [contract/procedure](../testing/M2/job-read.md) adds private
ID-selected native metadata reads and immutable successor validation. Require all
275 state assertions, twenty-eight parents and 176 codec/schema subtests on both
existing targets. Both passed at signed source `5241ae7f0bccde0423613af1dfcde9aed951ac34` with
full teardown; see the [manifest](../testing/M2/job-read-manifest.json). Accepted in
PR 49. No runtime job publication/admission,
actual recovery, commercial scale or new hardware/client support follows.


M2 mixed state catalog: [contract/procedure](../testing/M2/state-catalog.md) requires
336 assertions, thirty-two parents and 176 codec/schema subtests on both existing
targets. Complete private ownership/job census and shared byte/class ceilings;
both passed at signed source `b79ff0f03a85c719882b93e8080f562704cc6bac` with full teardown; see the
[manifest](../testing/M2/state-catalog-manifest.json). Review is pending. Actual record counts do not qualify commercial
resources/scale, mixed publication, admission or recovery; runtime remains read-only.
