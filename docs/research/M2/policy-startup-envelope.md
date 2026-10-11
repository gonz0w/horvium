# Separate startup caller-envelope diagnostic

Status: implementation candidate; native execution/completeness unqualified until
a source-bound receipt records the result. Owner: Cam (`@gonz0w`), 2026-10-11.
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
