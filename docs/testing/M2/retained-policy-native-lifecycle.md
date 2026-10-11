# Original root lifecycle with two retained native policies

Status: fixture merged in [PR 315](https://github.com/gonz0w/horvium/pull/315); all three candidates failed and native qualification remains pending. Live status is in issue 305.
Date: 2026-10-11. Owner: Cam (`@gonz0w`).
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305) and
[parent M2-RUNTIME-002](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted design](../../milestones/M2/retained-policy-startup.md),
[PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md),
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
Requirements: M2-RUNTIME-002, M2-STATE-001, M2-OWN-001, M2-DRIFT-001,
M2-PRIV-001 and M2-NATIVE-001.

## Bounded outcome and seed authority

Extend the [original two-policy review fixture](retained-policy-native-review.md)
with two actual fixed-factory lifecycle parents, on UFS without softupdates and
ZFS with distinct journal/ledger datasets. Reuse the declared seven-device
profile, original excluded metadata spare, independent actual mirrors and exact
member/scope baselines. Extract only the shared disposable metadata setup/cleanup
harness; preserve every existing review and fact control. Qualify all affected
parents with one new checksum-bound artifact on both pinned releases.

The existing tagged `SeedExecutionPolicyFixture` creates one original closed
policy-only seed per lifecycle parent through original commissioning, pending
journaled publication of both policies, confirmed census, strict closed successor
and original release. No JSON editing, receipt reconstruction, registry reset,
synthetic positive native facts or guard relaxation. The initial closed seed is
controlled fixture setup, not worker shutdown or admitted execution provenance.

A normally waited child calls the actual existing
`activateNativeClosedExecutionOwner`, with its original process registry, actual
root credentials and fixed backing/system/boot/policy producers. No alternative
factory or wrapped producer. Require exact active generation+1/new owner/current
boot and unchanged ledger/other journal files; independently observe, refuse
copied/decoded/nil/cancelled authority, then publish the exact unused generation+2
closed successor. Entered finish retires observation/finish; release retains
idempotence, releases both actual locks and cannot reset registration.

A second normally waited child starts from that same first child's original
closed successor, without reseeding or replacing either namespace. It must perform
the same full cycle with a new owner and two additional generations. The parent
independently checks each complete successor, original ledger bytes/names and
released locks. This is controlled unused metadata retirement, not worker draining.

## Initial inventory and limits declared before native execution

The new focused `policy-startup-lifecycle` runner uses one tagged host binary in
three serial, normally completed eighteen-minute phases:

1. Facts: `^TestNativeMultiplePolicyFacts$`.
2. Review: `^TestNativeMultiplePolicyReview(UFS|ZFS)$`.
3. Lifecycle: `^TestNativeMultiplePolicyLifecycle(UFS|ZFS)$`.

Every parent retains eight minutes, every child 150 seconds and GOMAXPROCS=1.
Construction uses the accepted two-minute caller budget across review/activation;
observation and finish each use an independent 90-second caller budget. Every
product operation retains its own existing 90-second bound, shared across all
policies/collectors/publication checks. No per-policy renewal or retry. The
whole-child cumulative timeout remains 150 seconds; no deadline is enlarged.
Separate denial probes use fresh unexpired contexts. Output remains bounded to
16 KiB; cleanup retains an independent 30-second budget. The existing two-hour
host/controller bound remains unchanged. Run releases serially, separate from
deep portable checks. A timeout remains an unqualified candidate.

| Required evidence | Controls |
|---|---|
| Retained standalone fact and UFS/ZFS review parents | Same three parents, 135 controls and seven children |
| Each lifecycle child, `first` and `restart` | registered, active_successor, active_private_record, current_boot, leases_held, nonjournal_unchanged, original_observation, observation_preserves_history, copied_denied, decoded_denied, cancelled_no_effect, closed_successor, ledger_unchanged, close_retains_leases, retired_observation, retired_finish, release_once, locks_released, registration_consumed (19 each) |
| Lifecycle parent | seed_closed (1); for each child: child_waited, child_inventory, exact_closed, ledger_preserved, leases_released (10); domain_teardown and mounts_restored (2) |
| Each lifecycle parent's retained fact/data fixture | Same 12 child/nine parent controls (21) |

Require five completed parents, **279 controls and thirteen normally waited
children per release**: each lifecycle parent has 51 root controls plus 21
fact/data controls. All original phases, controls and independent cleanup remain
required; no stale standalone receipt substitutes for an affected parent. Missing,
extra, duplicate, skipped, failed or misplaced phases/parents/control vectors or
incomplete host cleanup reject the whole gate. Require metadata backing/directory
teardown, exact data sentinels, original-GUID pool removal, input removal, restored
mounts/swap and device/guest/key/artifact/supervisor cleanup, preserving base/runtime.

## Review and remaining acceptance

Write failing-first strict evidence-consumer tests. These new native tests qualify
existing product behavior; no new product implementation or fabricated behavioral
red is claimed. Run signed clean-source `make review` for required
`make check lint test secrets` and the conservative full portable fallback for
shared qualification tooling. Record actual source/base/title, input fingerprint,
compiler/artifact/log/record hashes and any failed/skipped attempt. Preserve all
eight earlier retained-policy manifests byte-for-byte. Normal builds exclude
these fixture files and the existing seed bridge.

This covers valid original fixed-root two-policy activation, observation and unused
retirement, plus one subsequent clean process restart. The retained review phase
keeps its own last-member refusal scope. Remaining multi-policy drift/credential/
lineage/concurrency/census/publication/context/capacity families, mixed-history
fixed-root composition, configured startup, workers/backend/global launch and
browser/client/hardware/performance/release gates remain separate. No commercial
capacity, physical crash, worker-shutdown or anti-rollback claim. Product behavior
and runtime write denial remain unchanged; issues 305 and 139 remain open.

## Failed initial run and bounded affinity candidate

Signed source `c4591db14c44a61c33eb5e3de0f9624cf332b988` failed its FreeBSD
15.1 UFS lifecycle child during construction: `context deadline exceeded` at
120.02 seconds, before returning an owner. Its UFS lifecycle parent failed at
211.28 seconds. Standalone facts and both retained reviews completed at 25.16,
255.88 and 333.14 seconds respectively, but the whole gate remains unqualified.
No ZFS lifecycle or 14.5 attempt completed; do not infer their result. Host/device/
artifact/supervisor teardown and base/runtime preservation passed. No incomplete
in-guest parent teardown is accepted. Portable review was not run for this failed
native candidate; earlier receipts remain unchanged.

Declare a fixture scheduling experiment before its execution: use FreeBSD base
[cpuset(1)](https://man.freebsd.org/cgi/man.cgi?query=cpuset&sektion=1) to restrict
only the new lifecycle phase's test processes and inherited native command
children to emulated CPU 0. The two-vCPU QEMU profile, retained facts/review phases,
GOMAXPROCS=1 root children, original fixed factory/producers and every existing
time/byte/record bound remain unchanged. Do not modify system-wide CPU sets or
product scheduling. Each lifecycle parent and each normally waited root child
queries its own actual kernel affinity; any mask other than exactly CPU 0 denies.
This does not establish causality for the failed run, production performance,
parallel execution or commercial capacity. Qualification requires complete actual
passes of the corrected gate, not an assumption that affinity improves timing.

The corrected lifecycle child inventory adds `cpu_affinity` to all original
nineteen controls (20 per child); each lifecycle parent retains all old controls
and adds the two child affinity assertions (53 root controls plus 21 fact/data
controls = 74). Corrected whole-gate inventory: **five parents, 283 controls and
thirteen normally waited children per release**. Require exactly one affinity
phase marker inside the normally completed lifecycle phase, checksum-bound
`/usr/bin/cpuset` provenance, actual kernel affinity and independent teardown.
Every existing case, assertion and bound stays required. The failed initial
source and skipped second-release/ZFS lifecycle remain in the final receipt.

## Failed affinity run and serial-guest candidate

Signed affinity source `5336a7be00c59e55cd35faeb373889ab64b9ebea` also remains unqualified on FreeBSD 15.1:
construction succeeded in 91.04 seconds and observation in 28.02 seconds; the
first UFS child then exhausted its unchanged 150-second whole-process timeout
during finish. Its parent failed at 200.18 seconds. Retained facts/UFS review/ZFS
review completed at 19.04/257.80/185.99 seconds, but no lifecycle control vector
qualifies. ZFS lifecycle, 14.5 and portable review were not attempted for this
source. Independent host/device/artifact/supervisor teardown passed; incomplete
in-guest parent cleanup is not accepted. These observations do not isolate affinity
or another scheduling factor's contribution.

Declare one final serial lab experiment before execution: this new lifecycle
suite alone boots its whole disposable guest with **one emulated CPU**, retaining
qemu64/TCG, 6 GiB RAM, seven owned devices, cached base image/firmware and every
case/deadline. Existing suites and `qemu_command`'s default keep two CPUs. Strict
suite selection refuses unknown CPU profiles; the controller records its actual
selected count and the guest checks `hw.ncpu=1` before running any phase. Both
lifecycle parents and four children independently require the actual one-CPU
kernel and CPU 0 affinity; the existing phase affinity wrapper remains required.
No system-wide CPU set, host tuning or product scheduling changes.

The whole serial-guest gate still requires five parents, 283 controls and thirteen
children per release, all original facts/review/lifecycle cases and independent
cleanup. Require exactly one kernel CPU-profile marker before the facts phase.
Actual passes on both pinned releases are mandatory; a timeout remains failed
evidence. This is a serial emulated qualification profile, with no SMP/concurrency,
throughput or commercial capacity claim. Preserve both failed candidates and all
eight earlier manifests. No production or fixture time limit is enlarged.

## Final serial candidate failed; native gate remains blocked

Signed source `b9fa2deebbe88aa383f8d3cf29aab7496d17238c` ran the final
one-CPU experiment on disposable FreeBSD/amd64 15.1. Retained facts/UFS review/
ZFS review completed at 16.93/172.90/199.79 seconds. The first UFS lifecycle
child returned `context deadline exceeded` from construction after 118.72
seconds, before returning an owner; child failure completed at 118.76 seconds,
parent failure at 176.25 seconds. No completed lifecycle vector qualifies.
No ZFS lifecycle or 14.5 attempt followed. Returning no owner does not establish
that no active publication occurred. Do not infer unpublished state or accepted
in-guest parent cleanup from host teardown.

The [unqualified receipt](retained-policy-native-lifecycle-manifest.json) binds
all three actual source/compiler/artifact/log/record hashes and partial diagnostics.
Every failed attempt's independent host/device/guest/key/artifact/supervisor
cleanup and base/runtime preservation passed. Incomplete in-guest parent control/
cleanup vectors remain unaccepted. All eight earlier manifests remain byte-identical;
these failures do not alter their original scope or qualify this gate.

The signed final source passed required `make check lint test secrets` and shared-
tooling full portable review: 204 tooling tests, ordinary Go tests, all eleven
race packages and 54 fuzz targets. Four synthetic evidence-consumer/tooling tests
pass after three signed failing-first declarations. These are portable/tooling
results, not native lifecycle acceptance. Normal native-target builds exclude
the three tagged fixtures and the existing seed bridge.

The original candidate was submitted as a draft and subsequently merged as fixture
work in PR 315; merge does not qualify the failed native gate. Issues 305/139
remain open with broader acceptance unchecked.
Next investigate actual fixture/native timing within the adopted bounds, separating
emulation overhead from product behavior on a suitable disposable native lab;
then require complete same-candidate passes on both pinned releases. No further
fixture tuning, deadline increase or assumption of scheduling causality is accepted
by this receipt. Product code and runtime write denial remain unchanged. Remaining
multi-policy failure families, mixed history, configured startup/workers/backend/
global launch and browser/client/hardware/performance/release gates stay open.

The [independent collection diagnostic](../../research/M2/policy-startup-costs.md)
measures native primitives and preserves these failures. It cannot substitute for
the required actual fixed-factory lifecycle acceptance.

The [offline historical stack analysis](../../research/M2/policy-lifecycle-timeout.md)
identifies the second candidate's whole-child timeout during post-publication
revalidation. Its sanitized source-bound witness preserves this failed receipt
unchanged. No successful close/control vector or new native qualification follows;
this contract's selectors, inventory and bounds are unchanged.

The [cumulative fixture budget amendment](../../milestones/M2/retained-policy-fixture-budgets.md)
was separately adopted through PR 319 at merge
`aebfb2c8b85040154f6e70ffb3296bba543a9db9`; design issue 318 is completed.
The [implementation contract](retained-policy-lifecycle-budget.md) uses only its
330-second lifecycle children, 18-minute lifecycle parent work contexts and a
40-minute lifecycle process phase, retaining every caller/product/native/cleanup
bound and all 283 existing controls plus six actual profile assertions (289).
These supersede the historical fixture limits above for new candidates only;
no existing failed receipt is rewritten or promoted to acceptance.
