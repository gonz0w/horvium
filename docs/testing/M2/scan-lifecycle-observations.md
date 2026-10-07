# M2 native scrub-state observation qualification

Date: 2026-10-07. Status: accepted subset in [PR 78](https://github.com/gonz0w/horvium/pull/78).
Authority: [approved PRD](../../milestones/M2/PRD.md),
[EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).
Partial M2-HEALTH-001, M2-SCRUB-001, M2-DRIFT-001 and M2-NATIVE-001 evidence.

## Scope and fixture

Qualify the accepted [read-only health/scan observer](pool-health.md) against
actual kernel running, paused, resumed and cancelled scrub states on both pinned
disposable FreeBSD releases. No product implementation, executor, admission,
command, policy, ledger or runtime storage-write path is added. The new assertions
exercise existing code, so there is no new implementation to drive with a failing
portable test. Retain every actual native attempt at its tested source; an
unavailable observation is not a successful scan result.

Reuse the existing checksum-bound `job-status` suite. Keep all 21 original
status assertions, fifteen assessment children and six health children intact.
After the health subgroup and before GUID/backing-verified teardown, the same
owned unmounted mirror adds `native-scan-lifecycle`. Require all eight children
and the group exactly once, without FAIL/SKIP:

- `preflight`: kernel root and both disposable VM opt-ins; stable collected
  owned pool/import/healthy identity and unchanged property projection; no active
  guest scans; expected zero scan-progress control before changing it.
- `running`: start only the GUID-verified owned pool; independently collect actual
  running scrub text, its start label and reported bounded progress; no fabricated
  final error count or completion.
- `paused`: actual `zpool scrub -p`; independently observe paused state, native
  start/pause labels and progress, with no rates or estimate.
- `paused-preserved`: repeat the actual collector; complete facts and raw property
  projection remain identical while the kernel scrub is paused.
- `resumed`: actual `zpool scrub`; independently observe running again with the
  retained native start label, no paused label and no fabricated completion.
- `cancelled`: actual `zpool scrub -s`; independently observe native cancelled
  state/end label without inventing progress, final errors or successful completion.
- `control-restored-completed`: restore and reread the original kernel control;
  start a fresh real scrub and independently verify native completion/error count.
- `preserved`: permanent/import identity, raw properties, healthy zero-error
  topology and original kernel control remain intact.

Abort dependent fixture steps after a failed child. Each convergence wait has a
20-second child context within the unchanged two-minute native parent. Never
extend product budgets or weaken unknown-format refusal to make a fixture pass.

## Pinned fixture control and restoration

The small mirror may finish before a running/paused observation can be collected.
Inside the opted-in disposable guest only, read `vfs.zfs.scan_suspend_progress`,
require its expected zero baseline and set it to one before starting the fixture
scrub. Both pinned kernel sources describe this debug switch as preventing scan
progress for testing. FreeBSD's module macro exports it under `vfs.zfs`.

- [FreeBSD 14.5 kernel scan implementation](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/module/zfs/dsl_scan.c).
- [FreeBSD 15.1 kernel scan implementation](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/module/zfs/dsl_scan.c).
- [FreeBSD module parameter export](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/contrib/openzfs/include/os/freebsd/spl/sys/mod.h).
- [FreeBSD scrub procedure](https://docs.freebsd.org/en/books/handbook/zfs/):
  native pause, resume and stop behavior; the accepted formatter provenance
  remains in the health manifest.

This switch holds real kernel progress; it supplies no synthetic status text,
percentage, event, success or property result. An observed zero percentage remains
zero. This fixture does not qualify advancing progress, sustained throughput,
checksum fault injection, attributable ownership, generic cancellation or an
executor. Native labels and matching starts do not identify an exclusive episode.

Arm restoration before the control write, including uncertain command failure.
On every exit, restore with an independent ten-second cleanup context, reread and
require the original value. A failure marks the native run unqualified and requires
discarding the guest. Explicit restoration is also a mandatory positive case
before completion. The outer fixture independently verifies the pool GUID and
exact vnode backing before destruction/detach, verifies absence, and removes its
files. Require all original ledger/executable/base/runtime/host/process teardown.
No control or command is applied to the development Mac or an operator's NAS.

```sh
PR_TITLE='test(storage): qualify native scrub state observations' make review
.tools/venv/bin/python3 tools/local-qualification.py job-status 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py job-status 14.5 --run
make lab-status
```

Keep all existing storage/native-source product code and tests, shared tools,
checkers/selectors/supervisor/Makefile and 53 historical M2 manifests unchanged.
Focused review selects the command package; there is no compiler-listed portable
fuzz target in that package. Full milestone/release, attributed lifecycle,
management UI/CLI, native executor/recovery and hardware/client/scale acceptance
remain separate gates. GitHub Actions and persistent services remain disabled.

## Source-bound results

Signed source `728e980a8b21755c9b87f33e48405f096bb03b24` passed focused portable review: `make check lint test secrets`,
122 tooling tests and command-package race checks. That selected package has no
compiler-listed portable fuzz target; unrelated prior fuzz evidence is not rerun
or relabeled. Both releases independently rebuilt the identical binary
`1ebf41b3ae1f0c0e805f04d7c8ae8e990bd31f0298ca1b43f5176c630179991c`
and passed all eight mandatory scan children/group, six existing health children,
fifteen assessment children and 21 original status assertions.

- FreeBSD 15.1: native parent 29.51 seconds; new scan group 8.77 seconds.
- FreeBSD 14.5: native parent 24.87 seconds; new scan group 7.55 seconds.

The [manifest](scan-lifecycle-observations-manifest.json) records actual source,
control provenance, binary/tool/image/report/log hashes and preservation/teardown.
The guest kernel control was restored/reread before fresh completion and verified
again by independent cleanup. All owned storage/metadata/executable/VM files and
process groups were removed; base/runtime preservation and host teardown passed.
No active or unverified owned session remains. Product/source code, all 53 prior
M2 manifests, existing health fixture and shared tools remain byte-identical.
Every original assertion/deadline is preserved. No Actions, persistent service or
operator scrub capability was enabled. Advancing progress, injected errors, full
formats, attributed jobs/executor/recovery and complete M2 acceptance remain open.
