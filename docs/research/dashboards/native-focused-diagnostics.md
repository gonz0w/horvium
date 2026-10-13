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
One source-bound disposable FreeBSD 15.1 diagnostic attempt, with an eight-minute
Go-test deadline and ten-minute owned command deadline, follows signed clean local
review. Preparation/boot retain their existing bounded owned deadlines. Do not
retry a failed diagnostic to obtain a passing result. FreeBSD 14.5 and the full
resource procedure remain unexecuted in this spike. Research observations require
review before any change to the fixture contract or runtime behavior.

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
full resource assembly/original sidecar retrieval is omitted. Preserve its actual
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

Execution has not occurred at this preparation source. A followup receipt retains
the exact executed candidate and actual findings. This spike cannot close issue
382 or broader dashboard/native/browser/release gates. An adopted correction needs
new reviewed source and both unchanged full native resource procedures. Runtime
storage writes remain disabled; manual accessibility and startup recovery stay open.
