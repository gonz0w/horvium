# Dashboard guest artifact and environment preflight

Date: 2026-10-12. Status: development prerequisite awaiting maintainer review.
Owner: [issue 372](https://github.com/gonz0w/horvium/issues/372), parent
[Capacity 290](https://github.com/gonz0w/horvium/issues/290), coordinated by
[M2 131](https://github.com/gonz0w/horvium/issues/131) and
[Overview 289](https://github.com/gonz0w/horvium/issues/289).
Authority: DASH-REQ-006 / DASH-PERF-001 and the approved
[resource Test Design](../../design/dashboards/resource-qualification.md).
Prerequisite: [reviewed host candidate artifacts](host-candidate-build.md).

## Original files and compiler observations

The test-only [observer](../../../cmd/horvium/dashboard_artifact_helpers_test.go)
opens the fixed three outputs through an `os.Root`: `horvium`,
`reader-fixture.test` and `dashboard-native.test`. Their contract roles remain
management, projection helper and resource controller. It accepts no paths from
HTTP requests, storage operators, resource stdout or a candidate JSON record.
It records independently read hashes and compiler metadata, rather than echoing
expected hashes. The future host caller must compare those observations with the
actual reviewed candidate builder's expected identities.

Each original leaf must be a nonempty regular executable, at most 128 MiB, with
no group/world write permission. Missing files, symlinks, directories, oversized
files and ordinary replacement races deny the observation. The observer opens
and hashes the entire original set before decoding metadata. `debug/buildinfo.Read`
reads those same descriptors through a section reader limited to their original
sizes; appended bytes cannot extend the metadata decoder's input. all three require FreeBSD/amd64/v1, cgo disabled
and one observed release compiler version. That version is an observation;
matching the host's pinned compiler is the subsequent trusted caller's job.

The controller's actual `os.Executable()` must identify the same original file as
`dashboard-native.test`. After metadata inspection, every descriptor and leaf is
rechecked for identity, size, mode and modification time, and original bytes are
hashed again. Any mismatch or close error returns an error and an empty result,
so incomplete artifacts cannot become a partial successful receipt. Executables
are inspected, never launched by this observer.

## Separate read-only native preflight

[The FreeBSD test](../../../cmd/horvium/dashboard_artifacts_freebsd_test.go),
`TestNativeDashboardArtifactProvenance`, requires the existing explicit disposable
`HORVIUM_NATIVE_VM=1` opt-in, root and amd64. It runs from the original guest work
root and inspects `.tools` artifacts, including its own prebuilt controller.
There is no new runtime flag, runner suite or registration. The resource parents
and current qualification runner do not select this preflight.

The preflight collects actual facts with fixed FreeBSD base argv:

| Fact | Read-only command |
| --- | --- |
| Running kernel release | `/bin/freebsd-version -r` |
| Installed userland release | `/bin/freebsd-version -u` |
| ZFS versions | `/sbin/zfs version` |
| Physical memory bytes | `/sbin/sysctl -n hw.physmem` |

Each command has a five-second context deadline, one-second pipe wait bound, fixed locale/PATH, discarded
stderr and a 512-byte stdout ceiling. Failure, oversized or malformed output denies
publication. Kernel/userland must identify the same declared 15.1 or 14.5 RELEASE
line, optionally at different patch levels. ZFS version lines are normalized to
one bounded printable field; physical memory must be a positive canonical uint64.
These are actual running-kernel/installed-userland observations; CPU/QEMU
profile and trusted host-to-guest comparison remain the final runner's job. The preflight reobserves the artifact set after commands
and requires identical hashes/compiler before emitting one sanitized
`DASHBOARD_GUEST_ARTIFACTS` JSON record.

This marker belongs to a separate preflight transcript. The unchanged strict
resource validator rejects it in resource stdout as unknown coverage. The future
host caller must validate original preflight completion/owned supervisor and
compare exact artifact/compiler/release facts, then construct the existing strict
provenance contract using its own source and host/QEMU observations. Neither this
record nor a copied candidate can provide that authority.

## Evidence, limitations and remaining work

Portable tests use disposable synthetic files and an injected metadata decoder
for malformed/mutation cases, plus real compiler metadata from the current test
executable. They exercise original byte digests, file/controller refusal, whole-set
mutation, compiler/target disagreement, bounded output, and both declared release
parsers. They do not execute FreeBSD facilities. The native decoder is always
`debug/buildinfo.Read`; the injection stays inside test-only code.

This is developer-controlled preflight evidence, not attestation against a malicious
root guest. File observations do not authenticate executable memory, prevent a
concurrent root writer restoring bytes, prove which later management/helper child
was executed, or authenticate the kernel against a malicious guest. The later reviewed host runner must
retain ownership, compare trusted expected hashes and bind actual launches and
original completions. No acceptance/cleanup/candidate booleans are emitted here.

No storage, native helper, listener, service, VM or runtime writer is created by
this slice. The performance guards, validators, existing warm-up/idle/workload,
resource/crash/boundary controls and historical receipts remain unchanged.
Independent host comparison, original fixture expectations, native control
completion and guest/host teardown binding precede fresh sequential both-release
execution. Broader #131/#289/#290, lifecycle #305/#139, browser/manual, physical
hardware/commercial and release gates stay open. Rollback removes the three
new test-only files and their documentation without changing product behavior.
