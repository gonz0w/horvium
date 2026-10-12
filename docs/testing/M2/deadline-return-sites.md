# Original startup deadline return sites

Status: bounded return-site component; native placement and qualification pending.
Date: 2026-10-11. Owner: Cam (`@gonz0w`).
Refs [issue 339](https://github.com/gonz0w/horvium/issues/339),
[parent 305](https://github.com/gonz0w/horvium/issues/305) and
[broader gate 139](https://github.com/gonz0w/horvium/issues/139).
Authority: PR 336 adopted the [method and Test Design](../../research/M2/policy-startup-timeout-provenance.md),
refining the approved [M2 Test Design](../../milestones/M2/TDD.md),
[PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[retained-policy contract](../../milestones/M2/retained-policy-startup.md) and
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
Requirements: return-site parts of M2-POLICY-DEADLINE-PROVENANCE-001–003/006;
M2-RUNTIME-002, M2-STATE-001, M2-RECOVERY-001, M2-OWN-001, M2-DRIFT-001,
M2-PRIV-001. No complete native test ID is accepted by this component.

## Behavior and boundaries

The [portable adapter](deadline-provenance-core.md), merged in PR 338, now observes
errors at the existing nine timeout-constructor scopes. Every original
`context.WithTimeout` parent, allowance and deferred cancellation remains intact.
Collectors retain their actual parent immediately before their original timeout;
commands use their actual collector parent. Explicit return expressions capture
errors before that scope's cancellation, keeping immutable inner observations
when an outer scope adds its frame. Ordinary and tagged-disabled builds preserve
exact error identity; nondeadline errors remain unchanged when enabled.

| Original scope | Allowance | Return phase |
| --- | ---: | --- |
| Review inspection | 90s | inspect |
| Activation's shared context | 90s | first-inspection, second-inspection, publication, final-inspection |
| Backing, pool, provider, system collectors | 45s each | collection |
| Backing, pool, system commands | 10s each | capture |

Provider collection reuses the original system-command capture, without another
command timer. Entry guards before review/activation children, both portable
activation guards and command guards produce parent-only frames with no child
and zero allowance. Activation's publication guard already has its original
90-second child and uses that child's publication phase. No phase is assigned
artificially to entropy, census or metadata errors between the three inspections.
Domain-opening failures before the review child remain unannotated; the existing
caller summary must preserve an unattributed result. Original joined failures
retain competing branches rather than suppressing them to manufacture attribution.

The public joint/single-domain observer's 90/45-second timers are outside this
factory path and remain untouched. Producers, output/record budgets, leases,
mutex order, registry, publication, fail-closed retirement, retry and release
remain unchanged. No fixture, consumer, runner, service or runtime interface is
introduced. Only explicit tagged fixture enablement can collect diagnostic frames.

## Failing-first and verification

Signed red source `d74e9035d9c89d314066dda8662507980fb27059` compiles deterministic tests against the original
unannotated sites and fails on missing collector/command and activation-guard
frames. An initial pre-commit attempt also found an unused test import, which
was removed before the signed behavioral red. Ordinary identity cases passed
already and remain regression coverage, rather than fabricated product failures.

Existing reader seams test actual original contexts and configured deadlines,
inner-to-outer immutable snapshots, capture before deferred cancellation,
Error/Is/As, reused provider command scope, bounded scalar-only serialization and
unchanged reader counts. Already-expired real contexts and a deterministic locked
activation guard exercise parent-only returns without slow sleeps. These injected
reader errors establish portable annotation placement, not actual command expiry.
The existing failed-result and budget corpus remains part of affected-package tests.

Source review compares the seven changed production files against the merge base
after removing only annotations, saved-parent locals and the review return tuple.
It verifies identical original timeout/cancel sites, guards and control flow,
including all three activation inspections and publication guards. Both excluded
observer sources, native fixtures, consumers, runners and 23 preceding policy
receipts remain byte-identical. Normal/tagged FreeBSD storage and host cross-builds
verify compilation and file selection; cross-builds are not native execution.

The signed clean actual-title focused `make review` runs check/lint/test/secrets
and affected-package race/fuzz checks. Supplemental tagged tests/race cover both
changed packages. A receipt-only followup requires fresh baseline checks and only
verified exact-input deep reuse. Owned processes must be stopped and verified.

## Remaining acceptance

No new VM or native run is required for this bounded integration. Actual native
return placement, the separate strict one-parent/two-child/50-control fixture and
consumer, and the one adopted fresh 15.1 diagnostic remain under 305. That next
slice must declare and test its exact control inventory before execution. All
complete lifecycle, native safety/recovery, worker/backend/global launch, manual
browser/client, hardware/performance and release gates remain open. Runtime writes
remain disabled. Earlier failed, skipped and unverified candidate limits remain
unchanged; portable passes and source review establish no native support claim.

## Recorded candidate

Signed implementation `770dbe35c9d78b13be660c3418f75fa550bb8d7d`, merge base `36b00d992610c25fb53c869af1135649720740f5`,
passed all seven supplemental checks: ordinary identity, all tagged tests in both
changed packages, focused tagged race and four normal/tagged FreeBSD cross-builds.
Signed red archive replay reproduced the expected missing annotations; its archive
and all temporary cross-build artifacts were removed. The actual-title focused
portable review passed 228 tooling tests, nine affected race packages and 48
compiler-listed ordinary fuzz targets. All 23 preceding policy receipts remain
byte-identical. The [source-bound receipt](deadline-return-sites-manifest.json)
records these executed results and supervised teardown. A final receipt-only head
requires fresh baseline checks and verified exact-input deep reuse; native
placement and the separate diagnostic remain pending.
