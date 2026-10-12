# Separate startup deadline provenance fixture and runner

Owner: [issue 345](https://github.com/gonz0w/horvium/issues/345), under
[acceptance owner 305](https://github.com/gonz0w/horvium/issues/305) and
[parent 139](https://github.com/gonz0w/horvium/issues/139), after merged input
binding PR 344. Authority: the [adopted method/Test Design](../../research/M2/policy-startup-timeout-provenance.md),
approved [M2 PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[TDD](../../milestones/M2/TDD.md), retained-policy contract and ADR 0009 linked
there. This implements fixture/build/runner parts of
M2-POLICY-DEADLINE-PROVENANCE-001–006. Native execution remains separate.

## Original operation and exact inventory

The new FreeBSD/native-tag-only parent is
`TestNativeMultiplePolicyStartupProvenanceUFS`. It uses the original fact/data
fixture, original closed two-policy seed and original facts child, plus one
normally waited `TestMultiplePolicyStartupProvenanceChild`. The diagnostic child
makes one original `activateNativeClosedExecutionOwner` call. No native producer
is wrapped. Enablement adds only the existing tag-only value to the original
absolute caller context and retains its original monotonic start. Cancellation
stays deferred until original child exit; no timer or deadline is renewed.

| Inventory | Exact count | Authority |
| --- | ---: | --- |
| Diagnostic child | 13 | Eight original envelope controls plus five below |
| Diagnostic parent | 29 | Thirteen original parent controls, all thirteen carried child controls and three below |
| Original facts child/parent | 12 + 9 | Unchanged original fact/data fixture |
| Parent-visible total | 50 | [Strict consumer contract](deadline-provenance-consumer.md) |
| Normally waited processes | One parent, two children | Actual process success and bounded complete transcripts required |

The five new child controls actually check:

- `trace_enabled`: enablement returned the dedicated tagged context carrier.
- `error_preserved`: a bounded tag-only verifier checks actual annotation
  delegation and original Error/Is/As targets. It rejects unsafe graphs and
  unknown classifier hooks without emitting raw error text or reading contexts.
- `snapshots_before_cancel`: trace capture intervals precede the original
  factory return; repeated extraction preserves those immutable scalar bytes
  before deferred original caller cancellation.
- `trace_inventory`: exact canonical tagged JSON, known fixed identifiers and
  classes, twelve-frame/8 KiB caps and explicit empty states.
- `original_contexts`: original caller and enabled carrier retain the same
  absolute deadline/error, with bounded original parent/child ancestry and
  pre-return captures. Full path/constraint semantics remain independently checked
  by the strict host consumer.

The three new parent controls are `trace_extracted` (exact bounded child trace),
`trace_sanitized` (canonical fixed scalar encoding) and `source_isolated` (the
reviewed host builder's explicit proof acknowledgment). The acknowledgment alone
is not source attestation: the controller separately requires the actual compiler
file selections and ordinary/tagged cross-build receipt before forming the guest
command and validating any evidence.

Original caller 180s, diagnostic child 210s, parent work 480s and process phase
18min are unchanged. The original facts child stays 150s. The lab remains
qemu64/TCG, one CPU, 6 GiB and seven owned devices; original review/activation
90s, collectors 45s, commands 10s, samples 30/75/110s, raw snapshots 64 KiB,
sampler join 5s, cleanup 30s, child output 16 KiB, remote 7200s and supervisor
9000s/grace15s remain unchanged. These are disposable lab bounds, not deployment
hardware assumptions or performance/support claims.

The parent retains original independent journal hash/state/generation readback,
ledger/nonjournal preservation, released leases, raw snapshot removal and
metadata/fact teardown. Actual returned owner release does not erase active
history or stand in for durable idle retirement. Missing observations remain
unverified rather than inferred from host cleanup.

## Reviewed build and transcript boundary

The new development suite is `policy-startup-provenance`, using the existing
local qualification runner. Only the adopted fresh prebuilt 15.1 profile is
accepted; 14.5 and guest source build are refused before execution. Dry-run
description starts no VM. Execute the one diagnostic only after fixture/runner
review; stop after that candidate, retaining failures and incomplete attribution.
There is no automatic retry or alternate CPU/profile/release comparison.

The host verifies clean signed/DCO source and the pinned compiler, asks the real
compiler for ordinary/tagged storage/host file selections, and cross-builds both
FreeBSD/amd64/v1/CGO-disabled test binaries. Ordinary selection excludes tagged
annotations/semantics and this native fixture/support; tagged selection requires
them. The ordinary binary checksum and selections enter the build receipt, and
that temporary executable is removed. Compiler identity must remain unchanged
through both builds; source is checked before/after. The existing build receipt
retains compiler/pins/module/source/artifact identities and arguments.

The prebuilt wrapper verifies its actual binary checksum and CPU count, retains
base version/command metadata outside an explicit BEGIN/END frame, and places
source/compiler/artifact/profile headers plus the entire test-binary transcript
inside it. No unknown parent lines are filtered. The runner refuses extra test or
diagnostic evidence outside the frame, malformed/duplicate frames and missing
post-parent device teardown. Full private-log parsing is capped at 1 MiB;
canonical evidence remains at the consumer's 128 KiB cap.

The runner creates its canonical transcript exclusively with mode 0600, refusing
existing paths and leaf symlinks, and applies the merged
[actual-file input verifier](deadline-provenance-candidate.md). It retains the
validated transcript checksum and checks it again after teardown. A well-formed
incomplete trace is retained in the failed private record, never qualified.
Failed child output may be retained, bounded, in the private full log; raw output
or failed/partial parents cannot qualify canonical evidence. Success means
diagnostic completeness only. Native lifecycle acceptance is always false.

Normal and supervisor interruption cleanup register only
`policy-startup-provenance-native.test` and
`policy-startup-provenance-normal.test` in the owned run directory. Existing
device/base/runtime/key/image/process preservation and teardown remain required.
Private full logs and sanitized failed evidence survive executable cleanup;
unrelated files are preserved.

## Offline verification and limits

Signed failing-first helper/runner tests, tagged portable race checks, bounded
decoder fuzzing, actual normal/tagged FreeBSD cross-build/source selection and
mandatory actual-title portable review are required. Compiler-selected native
tests are not executed on macOS. The [source-bound receipt](deadline-provenance-fixture-manifest.json)
records executed results and owned teardown; final receipt-only changes use fresh
baseline and verified exact-input deep reuse.

All 26 prior policy receipts, production timers/producers/factory/registry/leases,
earlier native fixtures, absolute caller/sampler/sanitizer and strict consumers
remain unchanged. This implementation changes diagnostic observation overhead
and makes no controlled timing comparison or earliest-timer/command-duration
claim. No VM, service, new dependency or runtime interface is added or enabled
by this offline PR. The one fresh diagnostic, both complete lifecycle gates,
worker/backend/global launch, manual browser/client, hardware/performance and
release acceptance stay open; runtime writes remain disabled. Earlier failed,
skipped and unverified evidence retains its original candidate boundaries.

## Recorded offline candidate

Signed source `3e19ce862b4100059859377b435c3e01cfcbae47`, base `673a1f0a304948320c5ac303bdebaab95b8ae5c3`,
passed nine focused runner methods, eleven tagged portable methods under race
checks in two packages, and two compiler-listed tagged decoder fuzz targets.
Signed red `07ac1f37c3559bd0ffec0d86112ed59b306e9ba1` replays three failing Go methods
and eight Python methods with two failures/seven errors. The signed archive and
all disposable build/transcript files were removed. Both actual FreeBSD binaries
cross-built with real compiler selection proof; neither was executed. Synthetic
complete, incomplete and rejected transcripts bind the actual signed source,
pinned compiler and cross-built tagged artifact; synthetic cleanup lines are
not native/device observations.

The actual-title `make review` passed 258 tooling tests, mandatory check/lint/
test/secrets, eleven ordinary race packages and 54 ordinary compiler-listed fuzz
targets under the conservative shared-tooling plan. A preceding portable review
failed on an older serial-profile expectation that omitted this new one-CPU
suite; the signed test correction and successful review are separate evidence.
The first private verifier's constant-name assertion failure is also retained;
its corrected runs passed. All involved owned process groups were cleaned.
The [receipt](deadline-provenance-fixture-manifest.json) preserves those limits,
exact build identities, all 26 preceding receipt checksums and source preservation.
Final receipt-only changes require fresh baseline and verified exact-input deep
reuse. Actual native execution and broader acceptance remain pending.
