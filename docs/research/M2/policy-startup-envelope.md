# Separate startup caller-envelope diagnostic

Status: completed diagnostic candidate; maintainer review pending. Operation
succeeded; native lifecycle acceptance remains false. Owner: Cam (`@gonz0w`), 2026-10-11.
One development session; owning [issue 328](https://github.com/gonz0w/horvium/issues/328),
Refs [305](https://github.com/gonz0w/horvium/issues/305) and
[parent 139](https://github.com/gonz0w/horvium/issues/139). Authority:
[adopted inspection review/Test Design](policy-startup-inspection.md), M2-POLICY-STARTUP-ENVELOPE-001–006,
approved M2 PRD/EDD/TDD, retained-policy contract and ADR0009 linked there.
PR 327 merge `38b69d77c1f5c4fcbb8c8a79e074ff14f76815dc` adopted the fixture-only
experiment; its offline manifest and all preceding receipts remain immutable.

## Implementation and inventory

The separate fixed suite is `policy-startup-envelope`; its marker prefix is
`POLICY_STARTUP_ENVELOPE` and executable is `policy-startup-envelope-native.test`.
These are development fixture identifiers, not product APIs/configuration.
The new tagged native parent/child uses exactly one original factory call with
one absolute 180s caller, a 210s child and unchanged 480s parent/18min process phase.
Keep product 90s review/90s activation, all native bounds, one CPU, 6 GiB qemu64/TCG
and seven owned devices. Keep cleanup 30s, remote 7200s, supervisor 9000s/grace15s,
child output 16KiB, sample 30/75/110s, buffer 64KiB and stop/join 5s unchanged.

The original trace 120/150 fixture/prebuilt/consumer and full lifecycle profiles
remain unchanged. The separate consumer rejects their evidence and vice versa.
Checks bind actual test timeout/context/profile, executable checksum and clean
source. Test-only caller creation retains the original start; renewal or an old
120s deadline fails its check. Original fixed native producers and registration
are used without wrappers, caching, dropped drift checks or registry resets.
The native fixture is excluded from ordinary builds; portable support is test-only.

Require exactly one fresh closed two-policy UFS parent, two normally waited
children (original factory and original facts) and **all 42 controls**: the original
21 fact/data controls and 21 diagnostic controls. Child inventory remains
`cpu_affinity child_budget caller_deadline original_call sampler_joined
samples_bounded owner_release deadline_unchanged`. Parent retains all eight as
`child_<name>` and adds 13 controls:
`cpu_affinity parent_budget seed_closed child_waited child_inventory
snapshots_verified final_journal_observed ledger_preserved nonjournal_preserved
child_locks_released raw_removed domain_teardown mounts_restored`.

Reuse unchanged test-only sampler/sanitizer/original snapshot verifier: three
private 0600 files in the child's 0700 guest directory; exclusive creation, strict
bounds/permissions/hashes, source-relative sanitized frames and raw deletion.
No raw stacks/arguments are copied to host or Git. Skipped early-return samples
remain explicit; incomplete sampling/normal child waits/raw removal/cleanup
invalidates completeness. Original actual owner Close releases leases only;
never substitute durable idle retirement, erase active history or retry that seed.
Independently record final journal hash/decoded state (or invalid/uncertain),
ledger/nonjournal equality and released child locks.

## Failing-first portable evidence

Two context behavior tests first failed under a 120s fixture stub, then passed
with the absolute 180s caller. They test original start/renewal rejection and
parent90s clamping without slow sleeps. The first eight separate parser/runner tests failed before the separate consumer
existed. Ten now pass, including later source-provenance and interruption-cleanup
regressions, retaining
all existing trace tests. They reject cross-profile/malformed/unknown evidence,
missing/duplicate controls, falsely claimed native acceptance, wrong artifact
checksum/marker and changed/dirty source, and require independent device teardown.
Registered worker cleanup and supervisor interruption fallback remove the new
owned executable, preserving unrelated files. The fallback regression first failed
with the artifact retained, then passed after the fixed-name cleanup addition. Shared-tooling changes require the existing conservative full portable
review fallback; no hosted automation is introduced. Cross-building the tagged
FreeBSD/amd64 executable is compilation evidence, not native execution.

## Declared native procedure and limits

After clean signed-source local review, execute once on the disposable 15.1 guest:

```sh
.tools/venv/bin/python3 tools/local-qualification.py policy-startup-envelope 15.1 --run
```

No automatic 14.5 run, CPU comparison, changed-profile retry or full lifecycle
rerun in this experiment. 180s is a hypothesis envelope: independent product or
nested limits can still expire. Separate operation outcome, diagnostic completeness
and native lifecycle acceptance; the consumer/controller always record
**native acceptance false**, including an operation success.

Require original metadata/fact cleanup plus independent guest/host/device/key/
artifact/supervisor teardown, base/runtime preservation and `make lab-stop` /
`make lab-status`. Preserve all 16 preceding policy receipts byte-identical,
including the original unqualified setup failure and its unverified guest cleanup.
Point snapshots do not measure stage/probe durations, select a failing command
or prove deadlock/emulation/performance/capacity. Both complete five-parent/
289-control/thirteen-child lifecycle gates, second clean cycle and broader native/
recovery/browser/client/hardware/performance/release acceptance remain pending.
No runtime writes, product limits, host services, dependencies or Actions changes.

## Executed candidate and observed limits

The [source-bound receipt](policy-startup-envelope-manifest.json) records signed
implementation/native source `0c8fabbb3341b40e34e89478b273d5ebc716a1c8`, base
`38b69d77c1f5c4fcbb8c8a79e074ff14f76815dc` and the exact tagged executable/pinned
compiler. One disposable 15.1 UFS parent passed at **138.50s**, with two normally
waited children and all 42 controls, original metadata/fact cleanup and independent
host/device/image/key/artifact/supervisor teardown and base/runtime preservation.
All 16 preceding policy receipts and original trace/lifecycle fixtures/consumers/
prebuilt scripts are byte-identical. Existing suite registrations/CPU/budgets are
unchanged; no product Go file changed. Normal runtime build exclusion is verified.

Startup returned successfully at **84.436750s**, with caller/startup errors `none`
and an actual returned owner. Its original Close released leases without durable
clean retirement. Final journal bytes changed from closed generation 6 to active
generation 7, with exact hashes recorded independently; original ledger/nonjournal
bytes were preserved and child locks released. An active journal after release
provides no retry/closed-history/launch authority. No idle retirement, second
cycle or broader recovery was exercised.

| Sample | Capture start/end after caller creation | Bytes | Original located path |
| --- | --- | --- | --- |
| 1 | 30.002178–30.005476s | 9,535 | Activation first pre-publication inspection (`execution_restart_freebsd.go:50`), per-policy system version reader (`system_use_freebsd.go:9`). |
| 2 | 75.005247–75.006118s | 7,810 | Activation post-publication inspection (`execution_restart_freebsd.go:100`), per-policy fixed pool native reader (`pool_properties_freebsd.go:38`); its shared reader line does not select a specific probe. |
| 3 | Skipped | 0 | Actual return preceded the 110s offset; no raw file/capture/frames manufactured. |

The two snapshots map 68 frames (46 Horvium/22 compiler-matched Go) across 24
source files. The unique factory goroutine was waiting on the original native
command path at each point; source orders publication at line 84 before the final
inspection at line 100. Point capture intervals exclude subsequent snapshot writes
and do not measure phase/collector/command duration or total observer overhead.
Raw guest files were sanitized/deleted, not copied to the host; independent
post-deletion raw re-extraction is unavailable by design.

This return was **earlier than the original 120s ceiling**. It does not demonstrate
that increasing the caller caused success, establish stable timings or qualify
production capacity. Earlier unchanged-product candidates ran differently; no
controlled paired experiment or causal scheduling/emulation diagnosis is claimed.
The wider envelope remains a diagnostic profile, not a product recommendation.

Initial signed-source mandatory review passed all 225 tooling tests, ordinary Go
tests and full portable fallback: all 11 race packages/all 54 fuzz targets, with
supervised teardown. Final evidence-only head requires its own current title/
signature/DCO/baseline check and can reuse only that exact-input successful deep
receipt; provenance stays bound to the original executed source. The manifest
records failed/setup/red and passing log hashes, including the corrected macOS
path fixture and reproduced retained-artifact cleanup failure.

**Diagnostic completeness true; startup success true; native lifecycle acceptance
false.** Scope permits closing only issue 328 upon maintainer acceptance. The
original full lifecycle suite on both releases, durable unused retirement, second
clean cycle and broader 305 / 139 gates remain unqualified. 14.5 was not attempted.
After this bounded evidence is accepted, the next qualification can revisit the
original full lifecycle gate on a fresh disposable fixture under its unchanged
profiles; this receipt cannot substitute for those required controls.
