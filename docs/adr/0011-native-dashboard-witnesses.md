# ADR 0011: original native dashboard witnesses

Status: Accepted for test-only implementation in [PR 387](https://github.com/gonz0w/horvium/pull/387); no implementation/resource acceptance. Date: 2026-10-13.
Owner: Cam (`@gonz0w`), [issue 382](https://github.com/gonz0w/horvium/issues/382).
[Requirement correction](../design/dashboards/native-invariance-correction.md),
[EDD](../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../design/dashboards/native-witness-binding-TDD.md).
Maintainer merge of this complete packet is requested to adopt its test-only
mechanism for implementation. It does not approve an implementation or accept
resource/native/browser/release results. Until adoption, the existing approved
resource procedure remains authoritative.

## Context

The [native invariance experiment](../research/dashboards/native-value-invariance.md)
observed changed other-guest accounting after fixture activity while the owned
pool/100 datasets stayed unchanged and cached values matched fresh independent
native observations. A later independent collection is not the original input
for an earlier cached generation. The HTML oracle needs the producer's original
values for that particular generation, rather than mutable current expectations.

The existing manager chooses its generation ID before collection, passes it to
the helper client, and publishes that same ID only after successful copying and
publication. The existing inventory protocol echoes the request ID in every
response record. This is an existing correlation, not a new public interface.
Original producer framing plus observed publication can therefore bind exact
expectations without modifying either product executable.

## Alternatives and proposed decision

| Approach | Assessment |
| --- | --- |
| Freeze all native accounting or move only the receipt off ZFS | Does not establish that all other guest accounting stays fixed; can hide ordinary native changes |
| Use management API values or a later native collection as expectations | Circular presentation oracle or wrong collection; rejected |
| Add runtime witness routes, persistence or collector hooks | No runtime requirement justifies that interface or custody change |
| Instrument product helper/management test replacements | Changes measured artifacts and requires separate role/overhead claims; unnecessary for the existing request-ID link |
| Kernel I/O tracing | Adds release-specific trace completeness/truncation/pressure controls; not selected |
| Root-owned development broker recording unchanged original helper frames | Selected: preserves product bytes, independently parses original wire values and binds them to the existing generation/request ID |

Use a root-owned observer only in the disposable qualification fixture. A
short-lived connector drops to the configured management UID before connecting
to the actual helper, then passes that established descriptor to the observer.
The observer independently checks original peer UID/PID/lifetime, records the
complete bounded helper reply in private tmpfs, seals the original, and forwards
its unchanged bytes to the original management process. A separate independent
controller decoder supplies per-generation exact expectations. Publication
metadata selects an original witness; management numbers never supply it.

Both product binaries remain the candidate's real artifacts. The extra root
listener/descriptor broker changes the **development transport path**, so it must
not inherit direct-peer transport evidence. Existing direct native security and
command/crash gates remain required. The EDD specifies custody, resource accounting,
publication confirmation, native ABI/handoff qualification and fail-closed limits.
No runtime route, flag, schema, configuration field, source mode, package or
persistent service is introduced.

## Consequences and revisit triggers

The observer adds IPC, temporary originals and development process work. Keep the
original two-process budgets and add conservative combined instrumentation gates;
never subtract estimated overhead. Disclose the instrumented transport in receipts
and measure a separate direct-path calibration without calling it exact-value
acceptance. No inference that added work necessarily gives a conservative latency
result is permitted. Witness loss, actor/ABI uncertainty or changed byte forwarding
fails qualification. Native parser correctness remains a separate requirement:
independent decoding of producer frames validates presentation, not the producer's
native-command parser by itself.

Adoption authorizes one coherent implementation PR containing broker, original
custody/decoder, generation selection, controls, supplemental evidence/validator,
runbook and compatibility. Internal helpers are incremental commits. Both pinned
release short native gates precede activation; complete both-release resource runs
follow review of the integrated candidate. Historical failed/skipped evidence stays
unchanged and issue 382 stays open. Revisit if native credentials/handoff cannot be
qualified, the broker changes observable failure semantics, bounded originals or
resource accounting cannot fit, or calibration finds unexplained path effects.
