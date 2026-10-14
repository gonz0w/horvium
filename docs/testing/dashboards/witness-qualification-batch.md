# Batched inactive witness qualification support

Date: 2026-10-14. Status: implementation candidate; native results pending.
Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[accepted EDD](../../design/dashboards/native-witness-binding-EDD.md) and
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-008 / DASH-WIT-009 and native 005 / 006.
The earlier [rejection custody record](witness-rejection-custody.md) retains its
original native candidate boundaries. Its PR 389 is merged; this is one continuation
batch of related fault, instrumentation, coverage and calibration work.

## Resource accounting and original samples

The supplemental portable arithmetic keeps the original product-pair values and
adds observer high-water plus the **largest** connector incarnation to a conservative
sum. Sequential connectors are not summed as simultaneous RSS; every connector's
CPU is included. Pending/unreaped/unknown owners, missing scheduled samples,
negative or overflowing measurements, altered modes and unchanged 128/256 MiB
RSS or <1% idle CPU failures deny the record. A small-mode idle record still requires
300 seconds. This arithmetic contract is not a native measurement by itself.

The native observer records self high-water/CPU and accounts every started connector
when it is waited, including a failed connector. An admitted connector cannot leave
its usage out of a final original receipt. A controller-side 100 ms monitor separately
records product, observer, active connector and controller samples. Initial verified
executable/UID/process-start identities are rechecked by kernel lifetime while sampling.
Controller/decoder and tmpfs costs remain separate from the measured product pair;
missing samples/gaps stay visible. Reduced diagnostic records do not invoke or
accept the full resource builder and do not prove a 300-second idle budget.

## Versioned supplemental response coverage

Version 1 binds a source/base/title/release/mode, compiler/source archive, three
original artifact roles and actual ABI probe to each observed complete response.
The candidate context comes from the independently verified host build/review and
actual guest artifacts; a report's own context never supplies trusted expectations.
Each response has its original duration/body hash and generation, original raw and
forward hashes, and hash of an actual publication metadata response. Names are
generated controller slots, never paths derived from generation IDs.

The independent validator reopens immutable helper originals, actual metadata bytes
and complete HTML bodies. It independently decodes original frames, compares only
publication metadata, and reapplies exact HTML predicates. Management accounting
values never enter numerical expectations. Wrong source/release/mode/probe, missing
or duplicate samples, malformed/expired/unconfirmed generation, mutated originals,
repaired delivery, missing metadata and forged HTML fail. Version/scope distinctions
prevent old coverage or a ten-response short probe from becoming a 2,000-observation
resource record. Only existing safe replacement Overview expiry is allowed.

The still-inactive native short suite now retains ten bodies per mode and bounded
metadata originals in root-only private directories, and verifies them against the
actual live original owner before teardown. A later offline verifier must independently
reopen the transferred artifacts too. The full resource caller is not activated by
this implementation; legacy resources and controls remain separately necessary.
HTTP durations stop at complete body read, before predicates, metadata, archive I/O
or decoder work. These outside-duration costs still affect controller pacing.

## Additional native negatives and calibration

The batch runs descriptor wrong-tag/extra/truncated/non-socket/closed-channel cases,
CLOEXEC/forced-GC ownership and endpoint preservation after sender closure. The
ordinary connector handoff separately observes an actual reaped sender process.
A same-object raw mutation fails its root receipt, and restoring original bytes and
timestamp cannot clear the observer's first failure.

Physical ENOSPC uses a **separate owned 1 MiB tmpfs mount**, preventing exhaustion
from changing the live observer archive. The normal wire/archive caps remain
unchanged. Filling this mount until actual ENOSPC and denying a subsequent capture
is an observer I/O control, not a producer model-limit success or a full normal
320 MiB occupancy measurement. Best-effort prefix/index uncertainty remains rejected.
The private stage `capture-io` identifies the read/spool phase without claiming
whether a underlying read or write failed; prior source-bound labels are preserved.

A direct native helper/management pair and a brokered pair each have a 65-second
diagnostic window with a 30-second configured schedule. Record actual complete
metadata response times, observed publication generations/timestamps and usage;
all diagnostic metadata reads are explicitly counted separately. Three observed
generations are required, with identical product artifacts and disposable storage.
These are observed publication cadence, not an exhaustive producer collection census
or an assumption that broker overhead is conservative. The probe reports path
differences; it does not qualify idle, latency, schedule completeness or resources.

## Remaining activation gates

Actual controller/observer/connector interruption at every capture/seal/forward/
metadata stage, all ancillary/cap/ownership adverse controls and calibrated complete
sampling/cadence still need the adopted native matrix. Supplemental coverage must
be independently reopened after transfer for the same reviewed candidate on both
pinned releases. Keep failed/refused/incomplete attempts and original cleanup proof.
The earlier `6525631` recovery rejection remains unexplained. Complete integration
and native short acceptance precede activating the unchanged full resource procedure;
large-inventory latency, wider M2/startup/browser/hardware/release gates remain open.
Runtime writes stay disabled; no product API/wire/config change, dependency, paid
Actions, installed service or runtime C component is introduced.
