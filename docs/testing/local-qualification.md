# Local qualification

Status: the Mac runner executed the existing property suite on both FreeBSD
targets on 2026-10-05, with all 51 assertions, unchanged base/runtime and owned
cleanup. Foundation shell/DCO smoke also passed on 15.1. See the
[sanitized execution manifest](local-qualification-manifest.json) for exact
source commits, image/tool hashes and versions. Local results do not replace
previously accepted manifests or establish new M2 device/runtime eligibility.

The updated lease supervisor was then qualified on source
`9a40e72d7438bbd7bbf8305aa59392ca91de8fd4`: foundation/15.1 passed with
unchanged artifacts and verified group/fixture cleanup. Killing the controller
of a second live QEMU guest stopped its listener and removed key/image/variables;
that interrupted run correctly remained unqualified. A real preview was reachable,
reported active, stopped through `lab-stop`, and had its listener closed; final
`lab-status` verified all owned sessions stopped. The complete portable gate,
47 tooling regressions, race and all twelve fuzz targets passed. See the separate
[sanitized teardown manifest](local-teardown-manifest.json). Earlier property
runs retain their original source evidence; other guest/browser combinations
were not requalified by these host-lifecycle tests.

## Mac first

The maintainer selected the current Mac as the first lab host. Run repository
checks and deep Go checks there before requesting review:

```sh
make tools
PR_TITLE='ci: move qualification to local hosts' make review
```

Attach the checked commit/base/title and results to the PR. The
[local review gate](../engineering/local-review.md) runs baseline checks and
affected-package race/fuzz checks, with a fresh full milestone mode; GitHub Actions are disabled. Automated
Firefox currently has Linux-only artifact pins and needs a local Linux lab.
Safari uses the installed-driver procedure; manual accessibility remains pending.

## Disposable FreeBSD guests

Prerequisites: Python 3.14, Git, OpenSSH, QEMU with x86 UEFI firmware and `unzstd`.
The current Mac already has QEMU and zstd; this runner installs no host packages,
services, GitHub runner or SSH configuration. On a future Linux lab, provision
QEMU/OVMF/zstd first. No nested virtualization is required by the current TCG
runner.

Use a clean committed checkout. The runner transfers `git archive HEAD`, never
the host's ignored files, Git credentials or private configuration. The state
and focused pool-property/peer suites additionally transfer one explicitly built,
checksummed test executable.

```sh
# Describe the selected guest without downloading or starting it.
python3.14 tools/local-qualification.py properties 15.1

# Explicitly start one disposable guest and run only the selected suite.
python3.14 tools/local-qualification.py properties 15.1 --run
python3.14 tools/local-qualification.py properties 14.5 --run
# Focused native property collector, host cross-build; no guest packages required.
python3.14 tools/local-qualification.py pool-properties 15.1 --run
python3.14 tools/local-qualification.py pool-properties 14.5 --run
```

Supported suites: `foundation`, `research`, `discovery`, `state`, `properties`, `pool-properties`,
`continuity`, `performance`, `devices`, `job-status`, `action-peer`, `persistent-backing`. Run the two release candidates **sequentially** on
the Mac. Each guest has two CPUs, 6,144 MiB RAM, qemu64 and snapshot storage.
Reserve additional RAM for macOS; stop other large workloads while testing.
These are lab allocations, not Horvium runtime hardware requirements.

The [focused pool-property collector](M2/native-property-collection.md) passed all
16 assertions on both native targets at signed source
`57ac9dde8c87db90d6cda26c28e0f3ee55ea408e`, with complete owned artifact/guest
teardown and base/runtime preservation. It requires no guest package installation
and transfers the same exact cross-built binary to each guest. Qualification took
43.432/43.554 seconds per guest; these lab timings do not establish NAS performance.
Actual root/system/device-use eligibility, policy binding and writes remain open.

The subsequent [root policy/native binding](M2/property-policy-binding.md)
extended this focused suite to 22 assertions. Both targets passed at signed source
`cd1991c6a1cb36375f0fca7e7dd90228338390cc`, with complete owned teardown and
base/runtime preservation; see the [manifest](M2/property-policy-binding-manifest.json).
It uses test-owned private metadata under trusted guest `/root`, retaining the
actual root lease's writable-ancestor refusal. The same checksummed binary ran
on both targets without guest packages. System/device clearance, actual enrollment,
atomic admission and runtime writes remain open.

The subsequent [native system exclusion](M2/property-system-exclusions.md) suite
attaches the existing five declared snapshot disks and requires all 32 assertions,
plus the mandatory host device-cleanup marker. Both native targets passed at
signed source `c0f26eeef6e09e955f1edeb8d6fe15c59481b3cb`, using the same executable;
see the [manifest](M2/property-system-exclusions-manifest.json). Complete workloads
took 73.607/71.764 seconds. Actual root backing, own data/system mount checks,
active disjoint swap, md refusal and full preservation/cleanup passed. The initial
ZFS-root fixture omission remains unqualified. Broader provider/enrollment and
runtime write eligibility remain open.

Foundation/15.1, properties/15.1/14.5 and devices/15.1/14.5 have actual execution evidence
through this runner. The other selectable suites and Linux host remain unqualified
through this runner until executed and reviewed; existing hosted evidence is
separate. The property runs took approximately six to eight minutes on this Mac,
including boot, package/SDK preparation, cold compilation and complete cleanup.

The runner uses TCG on both hosts. On an ARM Mac, amd64 executes through software
emulation and can be substantially slower. Performance procedures retain their
original assertions: slow results are failures, never adjusted to pass. Record
TCG/host/QEMU/firmware independently; results cannot be compared directly with
hosted/KVM measurements or physical performance. KVM acceleration and macOS ARM
FreeBSD qualification require a separately reviewed extension.

The [device fixture](M2/devices.md) now uses this supervisor and appends only its
five newly created snapshot raw disks. Both releases passed all eight device
observations and complete host teardown; see its
[execution manifest](M2/devices-local-manifest.json). Property-suite success remains
separate from device/provider safety proof. Teardown rehashes/removes declared
backings after stopping QEMU, with supervisor fallback for controller death.
The [disk correlation extension](M2/disk-correlation.md) adds four mandatory
assertions to the current devices suite; previous eight-assertion manifests remain
bound to their earlier source and do not qualify the new parser. The
[attachment extension](M2/disk-attachments.md) adds five mandatory assertions
and a second executed native test, for seventeen assertions. It partitions only
one independently verified owned snapshot fixture disk and removes its table
without force. The [system-use extension](M2/system-device-use.md) adds six
assertions and a third native test, for twenty-three assertions. Only two owned
snapshot disks are used for UFS/mount/swap controls; cleanup removes owned
mount/swap and the VM is discarded. Previous manifests retain their actual source
and case sets. The [imported-pool extension](M2/imported-pool-use.md) adds six
assertions and a fourth native test, for twenty-nine assertions. It runs before
the UFS/swap test, creates one unmounted pool on the unique 4K snapshot disk and
destroys only its verified owned pool without force. The modified guest is
discarded; this supplies no product create/destroy or signature-clearance authority.
The [signature extension](M2/filesystem-signatures.md) adds seven mandatory probe
assertions to those same four tests, for thirty-six assertions. It introduces no
fixture writes; it binds actual UFS/ZFS recognition, probes removed-pool metadata
and verifies whole bytes of its probed fixture before/after. Unknown output never
establishes absence. Prior manifests retain their original case sets.

## Isolation, cache and evidence

Only a randomly allocated **127.0.0.1 SSH port** is forwarded; there is no VNC,
public listener, host NFS export or host disk attachment. OpenSSH uses `-F
/dev/null`, a checksum-verified public builder fixture key, disabled SSH agent,
and a private per-run known-hosts file. Fixture credentials never authorize the
guest to log into the host. The key is deleted after execution and is never
cached or committed. Image/key preparation and guest source-build package/SDK downloads require internet access;
local execution means no GitHub-hosted runner minutes, not an air-gapped test.

Compressed images and the pinned upstream AnyVM artifact are cached by SHA-256
under ignored `.tools/lab-cache`. Every cache hit is authenticated; corrupt
entries fail closed. Fresh image copies are decompressed per run, with snapshot
mode and before/after image verification. Cached images are never writable
guest disks. The local runner drives QEMU directly; it does not execute AnyVM or
allow it to change the host's SSH configuration.

Private logs and `result.json` are retained under ignored `.tools/lab-runs`.
The record names the exact source commit, suite/release, image/runtime hashes,
QEMU/firmware, host architecture, emulation mode, assertions and cleanup result.
Qualification requires a successful guest command, suite-specific assertion
validation, unchanged base/runtime and verified owned-process termination.
Missing/skipped/failed assertions and cleanup/preservation failures remain
unqualified. Normal teardown removes the fresh image, private key and variable
store and transferred host executable; logs and the known-hosts file remain for inspection. Preservation failure remains unqualified; the supervisor still removes disposable
image/key/variable-store files after stopping the owned group.

## Teardown

```sh
make lab-status
make lab-stop
make lab-status
```

`lab-status` exits nonzero for active or unverified sessions. `lab-stop` requests
termination through each private owned session socket and waits for a completed
teardown record. It never kills by process name or signals a stale stored PID.
These commands cover supervised local guests, review gates, previews and browser
fixture/driver groups; they do not control unrelated operator processes.

A short-lived supervisor owns each foreground worker group and watches an open
lease pipe to its controller. Normal exit, failure, timeout, Ctrl-C, SIGTERM and
controller SIGKILL all close that lease or trigger explicit cleanup. It sends
TERM, waits a bounded grace period, escalates to KILL if needed, verifies no live
group member remains, then reaps its child. Holding the leader unreaped reserves
its PID while signalling, avoiding accidental kills after PID reuse. On guest
interruption it also deletes the fixed fixture key/image/variable-store/executable paths
inside the private lab directory and marks the run unqualified. Successful
qualification requires guest/base/runtime checks and the supervisor's verified
teardown. Status records and socket directories are private and ignored.

The supervisor installs no service and exits after cleanup. A running Mac preview
expires through the Go fixture's twenty-minute timer as well. Temporary lab/cache
and evidence files are inert: the compressed verified cache can remain for reuse.
Inspect retained logs, then delete only stopped session/evidence directories or
the cache if disk space is needed. Never remove active session sockets to hide a
running job; `lab-stop` first, then require a successful `lab-status`.

Host power loss, killing the supervisor itself as well as its controller, or a
host process deliberately detaching into another group cannot be covered by this
lease. External macOS browser applications may also be outside the group. Such
runs remain unqualified: stale sockets/metadata are reported as UNVERIFIED and
are never permission to signal an old PID. Investigate the specific owned process
identity manually before recovery; no broad process-name kill or production
storage cleanup belongs in this procedure.

Review/sanitize selected evidence before putting it in a PR; never commit raw
guest/host inventories, key material or credentials. A local record is input to
the milestone evidence review, not automatic milestone acceptance. Preserve
both release candidates and requirement/test traceability.

## Optional later home lab

A dedicated x86-64 Linux host/VM can run the same sequential fixtures. Suggested
starting allocation: four vCPUs, 16 GB RAM, 40 GB scratch space and internet
access. These are unmeasured provisioning suggestions. A later KVM runner needs
virtualization or nested virtualization, controlled `/dev/kvm` access and its own
recorded execution evidence. This host stays separate from live NAS storage.

A GitHub runner is unnecessary: run local scripts on the isolated lab when those
gates are needed. This work registers no persistent execution service. Native
replication/restore will require two isolated FreeBSD peers; hardware power-loss
and controller failures remain physical-lab gates.

The combined-use extension adds fifteen assertions to the same four tests, for
fifty-one assertions. It adds read-only native collection and pure immutable
baseline comparisons to existing owned fixtures, with no additional fixture writes.
Both complete `DEVICE_USE_POOL_RESULT` and `DEVICE_USE_SYSTEM_RESULT` marker sets
are required; thirty-six-case signature evidence cannot qualify the extension.
See [combined device-use review](M2/device-use-review.md).

The [pool-label extension](M2/pool-labels.md) adds eight required cases to the
same four native tests, for fifty-nine assertions. It exports and reimports only
the verified owned fixture GUID without force, checks unchanged exported disk
bytes across probes and preserves prior foreign/root/host teardown controls.
Both targets passed all fifty-nine assertions and complete preservation/teardown;
see the [native manifest](M2/pool-labels-manifest.json). The initial utility-path
and native-format failures remain unqualified with verified cleanup. Prior
fifty-one-case evidence remains separate.

The M2 [disk-access extension](M2/disk-access.md) adds nine mandatory assertions
for held direct/alias raw handles and access review drift, extending the same four
native tests to sixty-eight assertions. Successful logs must include the complete
`DEVICE_ACCESS_RESULT` group. All handles close before the existing filesystem
fixtures start; no bytes are written through them. Both declared releases and
full preservation/teardown must pass before qualification is recorded.

The M2 [state-directory prerequisite](M2/state-directory.md) uses the `state`
suite on disposable system-storage fixtures under `/root`, with owned copied
executables and temporary mounts. The [ownership-read](M2/ownership-read.md) and
[ACL validation](M2/state-acl.md) extensions require all 108 named assertions,
three executed native tests, seven unit tests and fifty-seven codec subtests
at the accepted ACL source;
missing/skipped/incomplete records fail. ACL controls format only new owned 32 MiB
memory devices and mount them as UFS with POSIX or NFSv4 ACLs inside the guest.
No operator disk, host mount or persistent runner/service is used. The ordinary
supervisor verifies base/runtime preservation, image/key/variable removal and
process teardown before qualification.


The [ownership publication extension](M2/ownership-publication.md) expands `state`
to 161 mandatory assertions, five native tests, nine unit tests and fifty-seven
codec subtests. Both targets passed at signed source `eabf4691fc55b825db74ba73501d758126fda4e7`
with complete teardown; see its [manifest](M2/ownership-publication-manifest.json).
Maintainer review is pending. It adds one new owned 32 MiB
UFS memory device for real ENOSPC/read-only/ACL controls; all fixtures remain
inside the disposable guest. Phase-error seams qualify accounting separately
from actual I/O failure or power-cut durability. Prior manifests keep their exact
sources and case sets; the supervisor still requires full owned cleanup.

## Prebuilt state tests

The `state` suite defaults to a host cross-build with the checksum-pinned Go SDK,
verified module cache, `GOOS=freebsd`, `GOARCH=amd64`, `GOAMD64=v1`, cgo disabled
and `-trimpath`. It builds the actual storage test package from clean committed
source, transfers its executable into the owned guest and verifies its SHA-256
before executing. Host records identify source, compiler, module/pin hashes,
target and artifact size/checksum; guest logs identify actual kernel/userland,
ZFS and base utilities. Cross-compilation alone is never native qualification:
the unchanged evidence checker requires every current assertion, parent test
and codec subtest to execute successfully in the guest.

```sh
python3.14 tools/local-qualification.py state 15.1 --run
python3.14 tools/local-qualification.py state 14.5 --run
# Explicit SDK/package/native compilation qualification when needed:
python3.14 tools/local-qualification.py state 15.1 --source-build --run
```

The default state path installs no guest package/SDK. Other suites keep their
existing guest source-build procedures until individually qualified for prebuilt
execution. `--source-build` is restricted to state. Each run still uses a fresh
verified snapshot and requires complete base/runtime/fixture/process teardown.
The host test executable is deleted after completion or interruption, including
supervisor fallback after controller death. Caches/logs are inert. This changes
development execution only; no NAS runtime, service or compatibility scope grows.
See [focused workflow qualification](focused-review.md) for executed evidence.


The [bounded catalog extension](M2/ownership-catalog.md) expands state to 202
assertions and seventeen parent tests, with the same 57 codec subtests. It adds
complete ownership namespace/record inspection and actual reduced-budget/drift
controls; existing owned UFS ACL and inherited-UID fixtures also test inspection.
No additional disks, mounts or runtime services are introduced. Both declared
native targets passed all 202 at signed source `5ea125fdc815fd7c6c54aba36824e50ea876b9f6`,
with full teardown; see the [manifest](M2/ownership-catalog-manifest.json).
Inspection-only proof leaves full-store admission/recovery and resource gates open.


The [ownership capacity extension](M2/ownership-capacity.md) expands state to 234
assertions, nineteen parents and the same 57 codec subtests. Native fixtures cover
exact reduced reservation limits, unrelated state denial, orphan preservation/
uncertainty, real pre-rename namespace changes and concurrent last-slot refusal.
There are no extra host services or live storage fixtures. Both targets and complete
owned guest/host teardown passed at signed source `bf80cb9f2e1c196eae330ae0d681d86e0c466033`;
see the [manifest](M2/ownership-capacity-manifest.json).


The [native job-reader extension](M2/job-read.md) expands state to 275 assertions,
twenty-eight parents and 176 codec/schema subtests. It requalifies the shared native
ownership/job byte reader, job ID binding and actual private-file/ACL/UID controls.
The existing owned POSIX/NFSv4 UFS fixture includes job records and proves effective
native job access cannot bypass the product's kernel UID guard. All added files
are owned disposable fixtures; no persistent services or operator storage are used.
Both targets passed at signed source `5241ae7f0bccde0423613af1dfcde9aed951ac34` with full
teardown; see the [manifest](M2/job-read-manifest.json).


The [mixed state census](M2/state-catalog.md) expands the state suite to 336 named
assertions, thirty-two parents (nine native, twenty-three unit) and 176 codec/schema
subtests. Both declared targets and complete teardown are required. Actual owned
64/1,000 job populations test fixed census limits. Both passed at signed source
`b79ff0f03a85c719882b93e8080f562704cc6bac` with full teardown; see the
[manifest](M2/state-catalog-manifest.json).
Shared evidence/selector changes select one complete portable review. Ownership-only
publication stays strict and no job admission/recovery follows.


The [job publication extension](M2/job-publication.md) expands state to 432 named
assertions, thirty-five parents (ten native, twenty-five unit) and 176 schema
subtests. Both candidates passed at signed source `dc789c4e3672f61e670e6c4352e1eae89b3fb13a` with
full teardown; see the [manifest](M2/job-publication-manifest.json). Actual UFS ENOSPC/read-only job controls extend the shared
pipeline/ACL/capacity suite. Shared evidence/selector changes require a complete
portable review; evidence-only followups may reuse exact matching deep execution.


The [property-job store extension](M2/property-job-storage.md) raises the state
gate to 555 assertions, fifty parents and 1,804 codec/model subtests. Both job
formats share the existing owned namespace/ACL/UFS/capacity fixtures. The same
prebuilt execution and complete teardown rules apply; both passed at signed source
 `5a0e67cd7f469cf398b1cd831284d8a44a3b67ce`
with full teardown; see the [manifest](M2/property-job-storage-manifest.json).
Review is pending.


The [native job reservation extension](M2/native-job-reservations.md) raises the
current state gate to 622 assertions, 58 parents and 1,837 codec/model children.
It streams the complete private ownership/job census into an immutable full job
view, includes ownership bytes in candidate capacity checks and preserves all
terminal/uncertain holds. Both accepted job formats use the existing root/UFS/ACL
fixtures; the new API must also deny the actual inherited-descriptor UID-65534
child. Both targets passed at signed source
`cca3a07eed591ed3bab757fc530eada0a40686be` with full teardown. Prior 555-case
evidence remains bound to its original source and cannot qualify this extension.
The same prebuilt execution and complete guest/host teardown rules apply; no
runtime admission is enabled.


## Reserved property-submission state extension

The [contract/procedure](M2/property-submission.md) combines strict complete retained
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

See the [execution manifest](M2/property-submission-manifest.json); no runtime admission/release follows.


## Property metadata process-death qualification

The [process-cut contract](M2/property-publication-cuts.md) kills/reaps the actual built metadata writer
at eight synchronized transaction boundaries, then reacquires the same lock and
independently verifies unchanged objects. Orphan temporaries remain preserved and
block publication; visible queued jobs project reconciliation and retain pool
holds. Matching retries are metadata readback without fresh sync/durability claims.
No native storage command, startup consumer or product behavior/schema changes.
Require all 777 state/backing assertions, 66 parents and 1,876 codec/model children
on both targets, including native restart-page execution and complete backing/
child/host teardown. Both targets passed at signed source
`a3a33d5929f7fba32fd9ab6364320984b49d9162` with full teardown; maintainer
review is pending. Shared evidence/selector changes required one complete portable
review. Reboot/power-cut and actual native
executor/reconciliation/restoration/release gates remain open.

See the [execution manifest](M2/property-publication-cuts-manifest.json); no native executor/recovery authority follows.


The property-provider extension retains the existing owned five-disk fixture and
requires all 37 focused assertions, including actual duplicate/missing serial pool
refusal and extra GUID-checked pool teardown. See the
[provider qualification](M2/property-provider-binding.md). No additional guest,
background service or metadata publication run is needed.

The [member enrollment extension](M2/property-member-enrollment.md) requires all
46 focused cases on the same declared disks. It tests version 1/2 private reads,
mixed catalog accounting, exact member matching, legacy/drift/newer-version
refusal and restoration. Kernel statfs selects actual version 2 publication or
unchanged-record softupdates UFS denial on system backing; no mount settings or
extra services change. Historical state-suite evidence retains its original
source; this focused run does not requalify unrelated metadata crash cuts.

The [property store readiness extension](M2/property-store-readiness.md) requires
all 56 focused cases on the same declared disks. It adds complete private-record
and canonical-policy checks, actual system-backing capability, retained job holds,
orphan uncertainty and unchanged-record assertions. No backing settings, services
or extra devices change; historical metadata crash/ACL/capacity evidence retains
its original source.


## Focused retained job-status suite

`job-status` cross-builds a checksum-bound `cmd/horvium` test executable and runs
only `TestNativeStorageJobStatus` on the selected disposable guest. It re-executes
the compiled public command path for root/actual wrong-UID controls. No guest SDK,
package, disk attachment, pool, mount or service is required. Both worker and
supervisor fallback remove `storage-job-status-native.test` on every exit; children
are waited/reaped and owned metadata/executable directories removed. See the
[contract](M2/storage-job-status.md) and [runbook](../operations/M2-job-status.md).

```sh
.tools/venv/bin/python3 tools/local-qualification.py job-status 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py job-status 14.5 --run
```

This suite covers retained metadata presentation. Native recovery, actual storage
operations, resources/hardware and M2 acceptance remain separate gates.

The focused job-status suite passed all 21 assertions on both pinned guests at
signed source `03d4ab5948dc56a41333acb0e4dd20d3191e1bc7`; actual UID, full-output-pipe
cancellation, backing/preservation and complete teardown are recorded in the
[source-bound manifest](M2/storage-job-status-manifest.json). This is diagnostic
qualification, not native mutation/recovery or full application acceptance.

## Focused action-peer boundary

`python3.14 tools/local-qualification.py action-peer 15.1 --run` (then `14.5`)
uses one host-cross-built FreeBSD helper test executable with an exact checksum.
It installs no guest packages and attaches no storage fixture disks. The selected
native parent has a 90-second outer/80-second Go deadline; every kernel UID,
negotiation, cancellation/deadline/disconnection and owned fixture case must pass
once. The [contract](M2/action-peer.md) and strict host checker distinguish actual
execution from portable tests and prohibit missing/skipped/duplicate evidence.
Both normal runner cleanup and controller-interruption fallback remove the new
executable along with guest image/key/variables. Existing suite commands, cases
and deadlines remain unchanged. Signed source `4222e8f098114df8942fb4843f7e683ae9068ee7` passed all eighteen cases
on both pinned guests in 33.303/36.617 seconds using one exact executable, with
complete owned cleanup and base/runtime preservation; see the
[manifest](M2/action-peer-manifest.json). Fresh full portable review passed 131
tooling tests, all 11 race packages and 46 fuzz targets. The interruption
regression removes the new executable; it supplies no new actual controller-death
claim. Runtime operations and full recovery/global launch remain unavailable.

## Owned property exchange extension

The `action-peer` suite additionally selects the separate actual
`TestNativePropertyExchange` parent. Preserve all eighteen original peer cases and
80/90-second deadlines. Require complete new exchange children/result and both
parents; historical peer-only evidence cannot qualify this extension. It adds no
artifact, package, storage fixture disk or service; existing owned executable/guest
cleanup covers both parents. See the [contract](M2/property-exchange.md).

## Admitted property exchange extension

The `action-peer` suite also selects the separate actual
`TestNativeAdmittedPropertyExchange` parent. Require all three parents and their
complete original/composed inventories; retain existing 80/90-second deadlines.
This adds no artifact/package/storage disk or service; existing owned guest/binary
cleanup applies. Previous peer/exchange proofs stay bound to their original sources.
See the [contract](M2/property-admitted.md).

The session-bound property client extends `action-peer` with a fourth separate
native parent and fourteen callback phase/typed-client cases. These callbacks model
application revocation; actual root/configured service kernel UIDs remain mandatory.
All 74 transport cases and unchanged 80/90-second bounds must pass; old three-parent
proofs retain their original sources but cannot qualify this extension. No new
persistent service, operator storage, guest disk or artifact type is introduced.

The private root property listener extends `action-peer` with a fifth separate
native parent and eighteen actual socket/worker/denial cases. All 92 transport cases
must pass within the unchanged 80/90-second limits. The listener defaults to denied
operations and calls no backend; no operator storage or persistent service is used.
Old four-parent proofs retain their actual sources but do not qualify this extension.

Configured property clients extend action-peer qualification with twelve actual
cases and a sixth exact parent; all 92 prior cases remain mandatory (104 total).
Original deadlines/artifact/supervisor checks remain unchanged. See the
[configured client contract](M2/property-dial.md); older five-parent proof cannot
qualify this extension. No backend or runtime write is enabled.


The focused `persistent-backing` suite cross-builds the existing FreeBSD host
package and selects only the two native UFS/ZFS backing parents. It installs no
guest development packages and requires the exact artifact checksum, all identity
controls and both owned filesystem/device/directory teardowns. Run 15.1 then 14.5
serially, as for the other Mac qualifications. See the [scope](M2/persistent-backing.md).

The focused `original-backing` suite runs actual UFS/ZFS observations through the
original protected lease, including replacement/drift, kernel UID denial and Close
exclusion. Run both pinned guests serially and also retain the affected `state`
safety gates; see [original-lease qualification](M2/original-backing.md). It uses
the existing prebuilt checksum and supervised complete teardown policy.

The focused `joint-leases` suite retains both `original-backing` parents and adds
actual paired journal/ledger ownership on UFS/ZFS. It requires all four fixtures
and their filesystem/device/directory teardown. Run both pinned releases serially
and the affected expanded `state` corpus; see [joint-domain qualification](M2/joint-execution-leases.md).

The focused `execution-system` suite retains the four original/joint parents and
adds native UFS/ZFS domain system checks using the five declared snapshot disks.
It requires complete guest fixture and host device preservation/removal evidence.
Run both pinned guests serially and retain both affected `state` corpora; see
[execution-domain system qualification](M2/execution-system-backing.md).


The separate focused `execution-startup` suite cross-builds the same FreeBSD host
package and selects exactly two new UFS/ZFS original closed-startup parents.
Run both pinned releases serially. Require all 93 controls per filesystem, nine
normally waited exact-binary children per filesystem, exact compiled artifact
checksum and independent domain/data/device/host/supervisor teardown. Each parent
has an eight-minute deadline; the two-parent process has eighteen minutes. No
guest development packages or host services are installed. The retained
`execution-system` and `state` suites/checkers/scripts are unchanged and remain
bound to their prior actual source; they are not selected again for this isolated
read-only path. Old system evidence cannot qualify the new gate. See the
[contract and executed scope](M2/execution-startup-review.md); new-incarnation
publication, nonempty history and unclean recovery remain separate requirements.


Two separate focused suites qualify closed-startup active publication. Run
`closed-startup-publication` then `closed-startup-owner` on each pinned release,
serially. The storage suite selects three exact parents and all 268 publication/
process-cut controls, requiring three actual backing teardowns. It uses actual
UFS/ZFS durability with explicitly synthetic storage physical-use/boot transfers;
its four cut children are killed and waited only after the declared real sync,
acknowledgement or credential-loss boundary. The process deadline is eighteen
minutes and each cut has a forty-second parent deadline.

The fixed-root suite selects four exact host parents and all 362 controls,
including all 186 retained original startup controls and 88 new controls per
filesystem. Its new children use actual fixed native producers and separate
production registries without resets. Retain every original eight-minute host
parent deadline; the four-parent process has 34 minutes. Require exact compiled
artifact checksum, child waits, original history/locks, disjoint declared backing
and independent domain/data/device/guest/host/supervisor teardown. No development
package or service is installed. Old state/system/startup suite records, scripts,
checkers and validation branches remain unchanged. See the
[contract and source-bound execution](M2/closed-startup-activation.md); whole-store
state suites retain their actual historical evidence rather than rerunning
unrelated cases. Empty warm history and userspace cuts do not qualify reboot,
power loss, active kernel work or full recovery/runtime behavior.


The focused `policy-startup-facts` suite builds the tagged host fixture package
and selects only `^TestNativeMultiplePolicyFacts$`. It uses the scoped owned seven-device profile (preserving
the historical five-device default) and checks both actual independent data pools in one normally
waited child. Require all 12 child and nine parent controls, checksum-bound
artifact verification and independent pool/input/mount/swap/device/guest/host/
supervisor teardown on each pinned release. Run releases serially. The eight-
minute parent, 150-second child, shared 90-second collection context and eighteen-
minute process bounds are declared before execution in the [two-pool fact
contract](M2/retained-policy-native-facts.md). This is a read-only collector
prerequisite, not original root startup/activation/retirement qualification; the
existing owner and publication suites retain their separate evidence and limits.


The focused `policy-startup-review` suite builds one tagged host binary and runs
serial standalone facts and original UFS/ZFS review phases. Require three completed
parents, 135 controls and seven normally waited children per release, exact binary
checksum and independent metadata/data/device/guest/host/supervisor teardown.
Run releases serially:

```sh
.tools/venv/bin/python3 tools/local-qualification.py policy-startup-review 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py policy-startup-review 14.5 --run
```

The [pre-execution contract](M2/retained-policy-native-review.md) declares the scoped
seven-device profile, original closed policy-only seeds and unchanged parent,
child and shared operation bounds. It qualifies private original review and
observation, plus last-member denial. Activation/retirement, other failure
families and broader native acceptance remain separate.
