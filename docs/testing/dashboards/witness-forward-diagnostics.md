# Original forwarding diagnostics

Status: inactive test-only fixture. Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-007 / DASH-WIT-009 / DASH-WIT-NATIVE-005.

The [previous 14.5 capacity failure](witness-broker-admission.md) retained a complete
fifth projection original but lost the underlying forwarding error. Its cause
remains unknown. These observations do not retrospectively explain or qualify it.

Each forward admitted through original custody checks retains its copy/verification/half-close stage, bytes
accepted by the writer, SHA-256 of exactly that prefix, elapsed duration, bounded
error class and wrapped OS errno. Hash bytes even when a write returns both a
positive count and an error; `io.MultiWriter` omits that partial chunk. Do not retain
raw error strings, paths or producer values in metadata. A second denied forward
cannot overwrite the first attempt's observation.

Rejection before custody admission does not create a byte-copy receipt. The
original observer failure stage remains authoritative; missing observations
cannot be reconstructed as an OS error or a successful forward.

The observer separately records request ID, original helper and management
incarnations, connection start and its original absolute deadline, capture
completion, forward entry/completion elapsed times, deadline crossing and its own
context cancellation/expiry observations. Elapsed times use the original monotonic
connection clock; the wall timestamp identifies its deadline. The observer cannot
inspect the manager's internal context or prove why a peer closed. A controller's
actual manager wait remains separate evidence.

Accepted socket bytes prove neither peer consumption nor publication. Complete
original, successful full forwarding plus write-half EOF, and confirmed manager
publication remain separate. Copy, verification and half-close failures still
latch permanent denial and retain originals. Diagnostics grant no eligibility.

## Focused controls

Portable regressions first reproduce missing partial-forward and half-close
observations, then require exact prefixes/hashes for partial error, short write,
cancellation, deadline and wrapped broken pipe. A failed second attempt cannot
rewrite the receipt or become eligible. Existing custody/grammar tests remain.

The native selector is `TestNativeDashboardWitnessForwardDiagnostics`:

- Native-small and projection 10,000 each confirm an actual publication and reopen
  its delivered original, sidecar and diagnostic receipt.
- At a separate native-small sealed-original barrier, SIGKILL and reap the actual
  manager before releasing forwarding. Require actual broken-pipe errno, failed
  store, complete/undelivered original, unchanged helper, actor/deadline observations
  and observer/connector cleanup. This deliberate process-loss control establishes
  no ordinary cancellation acceptance.
- An isolated expired Unix-socket fixture requires actual timeout classification
  and zero accepted bytes. Its already-expired socket deadline is an explicit
  classification fault, without changing any broker or manager deadline.

Export root originals/object/hash and framed metadata before unmount. Independent
host readback checks source/base/title/artifact/ABI/preflight and guest/host teardown.
Failed early cases retain failure-only diagnostics and fallback originals without
manufacturing publication or completed-case receipts.

Normal interval/deadline (30/45 seconds), 32 attempts, 64 KiB relay/control,
64 MiB/100,000 wire, 67,508,864 raw bytes, 256 MiB archive and 320 MiB tmpfs remain
unchanged. These short controls establish no full-capacity, performance or overhead
claim. The earlier 14.5 full projection-capacity failure and broader native/resource/
browser/hardware/release gates remain open. The canonical resource runner stays
inactive; runtime writes remain denied. No service or Action is added.

## Candidate evidence

Candidate-specific outcomes follow signed clean review; this document alone
establishes no native result.

### Incomplete host-wrapper predecessor

Source `c2ddc4c898e1c0e9a028a976184c045bbe0961e9` completed all four
15.1 native diagnostic assertions (258.75s), including both normal publications,
manager-loss forwarding and expired socket. Its host wrapper then raised
`NameError: name 'resource_status' is not defined` before exporting originals or
collecting post-run preflight. The raw native process exited zero and reports
guest cleanup; independent host teardown passed. Actual wire/index/control bytes
were not exported and are unavailable after teardown. No independent native
component acceptance follows from those passing assertions. 14.5 was not run.

The private wrapper rewrites the canonical resource worker for a fixed short
selector. Its input match accidentally used the diagnostic's 800-second outer
allowance instead of the canonical source's 1,900-second invocation, leaving the
original invocation in place and the new status variable undefined. Match the
original input, replace it with the fixed diagnostic invocation, and assert exactly
one replacement and absence of the original before launch. Preserve this failed
wrapper and transcript; a corrected, separately hashed controller needs a fresh
signed-source review and both-release execution/readback. This changes no trusted
collection deadline, native assertion, fixture count or acceptance bound, and
cannot upgrade the incomplete predecessor.
