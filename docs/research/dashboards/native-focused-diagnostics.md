# Focused native dashboard diagnosis

Status: Bounded research; no changed resource acceptance contract.
Date: 2026-10-13. Owner: Cam (`@gonz0w`).
Refs [issue 382](https://github.com/gonz0w/horvium/issues/382),
[Overview 289](https://github.com/gonz0w/horvium/issues/289) and
[Capacity 290](https://github.com/gonz0w/horvium/issues/290).
Authority: adopted [PRD](../../design/dashboards/PRD.md),
[EDD](../../design/dashboards/EDD.md), [Test Design](../../design/dashboards/TDD.md)
and [resource supplement](../../design/dashboards/resource-qualification.md),
DASH-REQ-006 / DASH-PERF-001. Preserve the
[original failed candidate](../../testing/dashboards/native-resource-qualification.md).

## Question and time box

Can fixed predicate classifications and an independent original-to-cache pool
comparison distinguish the native table mismatch, and do complete JSON/HTML
responses expose different costs in the same paired-process guest?
One source-bound disposable FreeBSD 15.1 diagnostic attempt per reviewed candidate, with an eight-minute
Go-test deadline and ten-minute owned command deadline, follows signed clean local
review. Preparation/boot retain their existing bounded owned deadlines. Do not
retry the same failed diagnostic to obtain a passing result. A reproduced setup
defect requires corrected signed source and fresh local review before new execution. FreeBSD 14.5 and the full
resource procedure remain unexecuted in this spike. Research observations require
review before any change to the fixture contract or runtime behavior.

The completed short probe did not reproduce the mismatch. One additional,
separately source-bound `TestNativeReaderResourceBudgets/native-small` diagnostic
uses the original 100-owned-filesystem fixture, 60-second warm-up, 300-second idle
window, ten readers and original predicates. Keep the eight-minute Go and
ten-minute owned command deadlines. It deliberately selects one original subcase;
the omitted projection means the complete parent's declaration/cleanup acceptance
cannot pass. Preserve that failure and original controls; never call this a
qualified parent, both-release receipt or full resource run. This targeted
execution is a distinct timing/workload investigation, not a repeat of the short
probe or a retry to obtain a qualifying result. No budgets or predicates change.

## Test-only observations

`TestNativeDashboardFocusedDiagnosis` uses the existing disposable root process
owner, byte-verified build artifacts, actual management service and existing
native/projection/synthetic helper fixtures. Its modes are native-reduced (two
owned filesystems), projection-10,000 (native host/pool facts, generated datasets)
and synthetic-10,000. Native-reduced cannot qualify native-small's 100 owned
filesystems. Independently collected/generated original facts are fixed before
predicates; expectations never come from management responses. A portable
regression reproduces changed pool accounting rejection without mutating the
original oracle. It does not prove the cause of the historical native failure.

A separate bounded cached JSON pool page is compared with original facts by GUID.
Only differing-field counts and original snapshot SHA-256 are emitted; no values,
identities, response bodies, errors or session data. Each mode schedules 50 HTML
requests across the five existing cases and 50 JSON requests across corresponding
paths, sequentially. Timings include the complete bounded body and exclude HTML
predicate parsing. JSON observations check status/body only, not full equivalence.
No ten-reader contention, mandatory warm-up, idle window, replacements, crash/
recovery or resource budget measurement occurs. Low timings cannot accept a view;
case-to-case differences identify hypotheses, not a causal CPU profile.

The request driver adds fixed failure classes for transport, body, status,
document, element, table, comparisons, storage metadata, Overview metadata,
generation, table count and unknown predicates. Unknown error text is withheld.
Successful `original-inventory` and replacement `safe-expiry` labels are unchanged.
Failure samples remain failures and strict resource receipt construction rejects
them. No predicate, retry policy, runtime route, budget or public configuration
changes. Native execution of the actual classifier is evidence distinct from its
portable regression.

## Source binding and teardown

Use existing candidate build, byte-verified transfer, guest artifact preflight and
owned process facilities. The private one-off host controller is a scoped copy of
the existing `release_worker`: its fixed command invokes only this Go diagnostic;
full resource assembly/original sidecar retrieval is omitted. After a five-second
SSH banner timeout during preparation, only this research transfer's connection
window is fixed at 20 seconds. No actual Go diagnostic ran in that preparation;
it remains a failed preparation record. The canonical qualification runner,
resource command deadline and budgets are unchanged. Preserve its actual
source/hash with private evidence and disclose this scope. It is development
research orchestration, not a new qualification suite or production tool. Record
signed source/base/title/review, build/archive/artifact hashes, actual original
stdout and before/after guest preflight. Never set `qualified` or
`native_acceptance` true because the diagnostic Go test completed.

Observe each pair's temporary directory/socket removal, original process reaper
emptiness, owned pool/md cleanup and unchanged executable artifacts. Independently
observe host QEMU exit, listener refusal, key/image/variables removal and preserved
base/QEMU/firmware bytes. On failure retain missing cleanup evidence explicitly
and discard the guest. Run `make lab-stop` and `make lab-status` after completion.
Historical manifests and primary operator edits remain unchanged.

## Findings and remaining work

Preparation candidate `f44918155eaab266b0e207e00e7045d17753cf2d` exposed an
initial-publication setup defect: its strict Overview reader encountered HTTP 503
before native/projection inventory was ready. This failed execution stays retained.
The correction uses the existing complete/fresh readiness wait, with an independent
native count or the declared projection count; it changes no product or acceptance
contract. The corrected source `7dc434dcb9dd807e7d3b3482ae3a2376424ccdad` passed local
review but its guest stopped during management artifact transfer with an SSH banner
timeout; guest preflight and the Go diagnostic were not reached. This preparation
failure and verified host teardown remain retained. Only the private research
transfer window changes for the next preparation. New preparation/execution is
bound to its reviewed source and actual driver hashes. A followup receipt retains
the exact executed candidate and actual findings. This spike cannot close issue
382 or broader dashboard/native/browser/release gates. An adopted correction needs
new reviewed source and both unchanged full native resource procedures. Runtime
storage writes remain disabled; manual accessibility and startup recovery stay open.

The short execution at signed source `f4a8c1b03f35d695ac2c215f28ad7120dcbeabfb`
completed in 37.27 seconds with 150 successful original-inventory HTML predicates
and 150 status/body-only JSON responses. All three modes reported zero differing
pool fields between independently collected originals and the cached pages.
Guest cleanup, before/after preflight and independent host teardown/preservation
passed. This does not reproduce the historical native mismatch or explain the
ten-reader latency failure; the targeted original native-small case retains its
actual warm-up/idle and contention while omitting the other resource modes.
An earlier scoped-controller dispatch refusal also occurred before transfer or
Go-test execution and remains retained with verified host teardown.

## Original-timing result and interpretation

The [source-bound research manifest](native-focused-diagnostics-manifest.json)
retains every executed/refused preparation, signed reviewed candidate, original
transcript hash, actual artifact/compiler/environment binding and cleanup scope.
The short probe had three modes, each with 50 sequential HTML and 50 JSON
responses. Its observed nearest-rank p95 over only ten maximum-page samples was
77.207 ms HTML / 9.762 ms JSON for projection-10,000 and 124.943 / 8.673 ms for
synthetic-10,000. This is a scoped cost difference, not a controlled causal CPU
profile, ten-reader measurement, budget pass or native performance qualification.

The separate original native-small case at signed source
`99274977fc326c32cff4ddeae57d8bdffa0678a0` retained the original 60-second warm-up
and full 300-second idle window. Its 115 total / 100 independently owned datasets
reproduced **600 HTTP-200 `failure:table` observations**: 200 each for Overview,
pools/default and pools/maximum. Both dataset cases passed all 200 predicates,
for 400 successes out of 1,000 initial scheduled requests. The parent failed in
394.73 seconds, with its native-small subcase failing in 390.54 seconds. No
replacement phase, native crash/recovery or resource-budget publication was
reached. Omitted projection also leaves the complete parent scope incomplete.
The first failed check compares the exact original table. It may reject table
structure, headers, row/link identity or cell content; this vocabulary does not
identify the specific field. Later comparison checks are not reached. The old
candidate's empty classifications remain unchanged and cannot be backfilled from
this new candidate. No pool-accounting or runtime defect is yet established.

The targeted failed command and outer worker exited nonzero with completed clean
owned groups and no escalation. Host listener/key/image/variables absence and
base/QEMU/firmware preservation were independently reopened and verified for all
five actual preparations/attempts. The short successful diagnostic additionally
observed owned pool/md removal, pair directory/socket removal, empty original
reaper and unchanged artifacts before/after the guest. The failed original case
has no complete parent cleanup receipt or after-preflight; host discard does not
upgrade that boundary. `make lab-stop` and `make lab-status` report stopped,
verified owned sessions. All 150 existing manifests and primary operator edits
remain unchanged. Required checks and affected `cmd/horvium` race review passed;
prose followups reuse the original exact-input deep evidence and rerun baseline
checks. No eligible fuzz targets belong to that affected package.

Next work under #382 is to distinguish the exact table field/shape/link mismatch
at the original timing and inspect the maximum-page rendering/query cost under
contention. Keep expectations independent and fixed; do not refresh them from
responses, relax predicates or increase budgets. This PR ends at a reviewable
research diagnosis because neither a particular runtime correction nor a changed
fixture contract is justified by these observations. Future corrections and full
both-release qualification remain in the same owning issue. No broader gate closes.
