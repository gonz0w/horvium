# Host binding of the original dashboard guest preflight

Date: 2026-10-12. Status: development prerequisite awaiting maintainer review.
Owner: [issue 374](https://github.com/gonz0w/horvium/issues/374), parent
[Capacity 290](https://github.com/gonz0w/horvium/issues/290), coordinated by
[M2 131](https://github.com/gonz0w/horvium/issues/131) and
[Overview 289](https://github.com/gonz0w/horvium/issues/289).
Authority: DASH-REQ-006 / DASH-PERF-001 and the approved
[resource Test Design](../../design/dashboards/resource-qualification.md).
Prerequisites: [host candidate preparation](host-candidate-build.md) and
[separate guest artifact observation](guest-artifact-observations.md).

## Independently checked candidate

[The inactive host module](../../../tools/dashboard-host-preflight.py) has no CLI,
runner registration, transfer operation or native execution. Its trusted future
caller supplies the original source/base/title, review directory, retained candidate
directory, owned lab and supervisor directories, expected release and loopback port.
None of these expectations comes from guest stdout. The module does not read
credentials, create a process group, boot QEMU, write files or open an execution gate.
Existing Git/compiler inspection commands are read-only host verification.

`verify_build` requires the original private `.tools/dashboard-candidates/` child
with retained outputs and successful preparation. It repeats existing clean signed
source/contributor-DCO, actual compiler pin/version and completed original review
verification. The actual opened build report must agree with those observations,
caller source/base/title, original controlled role/target/arguments, six completed
clean build supervisors with original command/group identity, normal exit, no
escalation/cleanup failure and finite ordered execution times, and original build/module logs. Report booleans alone
cannot replace opened files or source/review/compiler verification.

The retained Git archive is checked against the exact signed commit's recursive
`git ls-tree -r -z` projection. Every original regular member must have the matching
Git blob ID and executable mode; the full file set and commit PAX header must agree.
Directory entries must belong to that tree. Links, unknown paths, duplicates,
missing files, altered source bytes, unsupported source modes and oversized input
fail closed. This inspection never extracts archive members. Input ceilings are
128 MiB of archive/files, 20,000 entries and a 2 MiB tree projection. Git blob SHA-1
IDs match this repository's existing object format; they do not replace its signed
source or add an external attestation scheme. Future intentional export-ignore or
export-subst archives would require a reviewed contract change rather than silently
weakening the full-tree check.

The three original executable byte hashes, archive digest, compiler identity and
review identity reconstruct the existing strict candidate contract independently.
Both declared releases must identify the same actual prepared set. Each build and
module log is opened and hashed. Source/review/compiler and every original file are
rechecked before a build binding can return. Failure returns no successful result;
existing private failures stay intact. `verify_build` runs before artifacts are
removed; a later caller must preserve its original binding and independently record
teardown rather than trying to recreate a build after removal.

## Separate original preflight and owned command

`preflight_command` describes the existing loopback SSH argv with the original
private lab's fixture-key/known-host paths and validated integer port. It uses the
existing SSH options, fixed `/root/work` cwd and existing Go test flags to select
only `TestNativeDashboardArtifactProvenance`, once, with a 45-second test timeout
and the existing disposable opt-in. It never executes that command. A future
reviewed runner remains responsible for guest identity/authentication, transfer,
outer deadline and ownership; no caller-supplied shell content is interpolated.

`verify_preflight` reads the fixed private `dashboard-preflight.log` and original
supervisor `state.json`, each through existing bounded stable opened-file checks.
The preflight must contain exactly one RUN, one original guest observation, one
matching PASS and final complete-suite PASS in order. Missing, extra, duplicate,
failed/skipped, truncated, unknown or malformed records are rejected. Duplicate
JSON fields, non-finite values, copied candidate/acceptance flags and unsupported
fields cannot qualify. Its three hashes and compiler must exactly match the
verified host candidate. Actual running-kernel/userland versions must match the
caller's expected declared release; ZFS text and physical bytes are bounded and
validated. These fields do not establish host/QEMU/CPU procedure or performance.

The original owned supervisor must be a private child of this checkout's process
registry, with the exact independently constructed command digest, successful
non-boolean zero exit, completed clean group, normal exit reason, no escalation,
positive original group identity and finite ordered start/completion timestamps.
Preflight start must follow all candidate build completions. A remaining control
socket or any cleanup failure/unknown supervisor field denies binding. This reads
metadata only; it never signals a recorded PID. Local wall-clock regression fails
closed rather than fabricating an ordering.

After parsing, the candidate, transcript and original supervisor are rechecked.
Only then does the module return a normalized private source/candidate/release,
actual guest observations and original build/transcript/supervisor/command digests.
`native_acceptance` and `resource_execution` remain false. It emits neither the
strict resource provenance marker nor a full acceptance receipt.

## Evidence boundary and remaining assembly

Behavioral tests use explicitly synthetic guest, archive/compiler and supervisor
inputs; no SSH or FreeBSD executable is run. They exercise report/file mismatch,
wrong candidate/compiler/release, archive-to-tree forgery, missing/duplicate/link
members, malformed or incomplete stdout, command/completion/time disagreement,
unsafe paths and changes during rechecks. The source/archive revalidation is also
exercised against actual reviewed unexecuted cross-build artifacts on the Mac.
That exercise covers `verify_build` only, never a fabricated native preflight pass.

These developer-controlled records are evidence for maintainer review, not external
attestation against malicious local/root writers. Successful parsing cannot prove
how artifacts were transferred, which later management/helper children executed,
original fixture/control completion or guest/VM teardown. Trusted runner wiring,
actual launches, independent fixture/control and guest/host teardown bindings must
be reviewed before fresh sequential both-release resource execution.

Runner guards, strict/legacy validators, native resource/crash/boundary procedures,
runtime, dependencies, workflows and historical receipts remain unchanged. Broader
#131/#289/#290, lifecycle #305/#139, manual browser, hardware/commercial and release
gates remain open; writes stay disabled. No service, paid automation or tracking
synchronization is added. Use `make lab-stop` / `make lab-status` to verify owned
teardown and remove cross-build outputs after inspection. Rollback removes this
inactive module, its tests and documentation without altering product behavior.

## Recorded portable source and actual build revalidation

[Receipt](host-preflight-binding-manifest.json): signed behavioral source
`77a7298ff4e9c000c3d4616b356cf781ea9ba24c`, base `026395aa035a08a80cf0f5abe65954b82e3a4742`, actual title
`test(dashboards): bind original guest preflight receipts`. Required checks passed with
294 tooling tests, including ten new behavioral groups, and ordinary Go tests.
The conservative shared-tool plan executed fresh races for all 11 portable packages
and all 54 compiler-listed portable fuzz targets. Guest/supervisor inputs in the
new behavioral tests are explicitly synthetic; no native preflight is qualified.

The reviewed host builder produced all three original FreeBSD/amd64/v1 artifacts
from the signed archive with pinned Go and before/after module verification. The
new `verify_build` reverified actual source/contributor-DCO, original review and
compiler, full signed-tree/archive blob/mode coverage, actual artifact bytes,
original module/build logs and six normal successful owned build completions.
Its whole original candidate matched. Native executables were never run and
`verify_preflight` was not exercised against a fabricated native record.
The archive, executables and extraction were removed; owned teardown was verified.
All 145 historical manifests and native/runtime/validator/runner guard inputs
remained unchanged.

Initial missing-module, archive-digest rewrite and original-build-supervisor
regressions retain their red logs. The preparation review of initial signed source
`757bd2f5773d9df8dba165045d21bc1f8293abf0` was deliberately stopped while adding
the supervisor refusal regression. Its unqualified record and verified cleanup
remain intact. It is not native evidence and is not relabelled as a pass.
Current-head documentation review reruns baseline checks and may reuse only this
corrected original exact-input deep evidence. Actual guest preflight, transfer/
launch, fixture/control and complete guest/host teardown remain prerequisites
before fresh both-release measurements. Broad gates stay open and writes disabled.
