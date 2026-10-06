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
suite additionally transfers one explicitly built, checksummed test executable.

```sh
# Describe the selected guest without downloading or starting it.
python3.14 tools/local-qualification.py properties 15.1

# Explicitly start one disposable guest and run only the selected suite.
python3.14 tools/local-qualification.py properties 15.1 --run
python3.14 tools/local-qualification.py properties 14.5 --run
```

Supported suites: `foundation`, `research`, `discovery`, `state`, `properties`,
`continuity`, `performance`, `devices`. Run the two release candidates **sequentially** on
the Mac. Each guest has two CPUs, 6,144 MiB RAM, qemu64 and snapshot storage.
Reserve additional RAM for macOS; stop other large workloads while testing.
These are lab allocations, not Horvium runtime hardware requirements.

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


The [job publication extension](M2/job-publication.md) expands state to 429 named
assertions, thirty-five parents (ten native, twenty-five unit) and 176 schema
subtests. Both release/root candidates and full teardown remain required; native
qualification is pending. Actual UFS ENOSPC/read-only job controls extend the shared
pipeline/ACL/capacity suite. Shared evidence/selector changes require a complete
portable review; evidence-only followups may reuse exact matching deep execution.
