# Batched inactive witness qualification support

Date: 2026-10-14. Status: draft implementation; reduced native results mixed, acceptance open.
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
actual live original owner before teardown. A private offline driver also reopens the transferred artifacts and uses this
same reviewed validator with independently supplied build/probe expectations. The full resource caller is not activated by
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

## Retained sampler failure and correction

Source `8dc1461` failed all three native sampler paths on 15.1. Diagnostic source
`996fa7e` retained the actual bounded private command output and rejected immediately
at `usage-shape`: FreeBSD interpreted `pid=,ppid=,rss=,time=` as a custom header
and emitted one PID column, rather than four columns. The direct/observer runs had
53/50 single-field rows including that header. A Mac self-process probe emitted four
columns for both forms, so it could not reproduce the native format semantics.
Separate `-o pid= -o ppid= -o rss= -o time=` arguments correct the collector using
the existing native sampler convention. Failed original runs/inputs remain retained;
a new reviewed source and actual native row/lifetime observations supply regression
coverage for the native-only format difference. This explains the new sampler
failure, not the historical recovery rejection.


## Candidate-specific results and next work

The initial [sanitized manifest](witness-qualification-batch-manifest.json) retains all six
attempts, original transcript/archive hashes, local reviews and teardown readback.
Four reduced diagnostic attempts failed and two passed. These are distinct source
boundaries; **none of those six initial attempts passed both pinned releases**.
The subsequent same-source diagnostic continuation is recorded below.

| Source | FreeBSD | Reduced suite | Retained boundary |
| --- | --- | --- | --- |
| `8dc1461` | 15.1 | Failed | Generic sampler rejection on all paths |
| `996fa7e` | 15.1 | Failed | Native one-column `ps` output; original inputs retained |
| `aefd67d` | 15.1 | Passed | Corrected column collection; reduced diagnostics only |
| `aefd67d` | 14.5 | Failed | Connector usage rejected during recovery; projection command rejected |
| `caf9ce4` | 14.5 | Passed | Additional rejection-original retention, not a proven causal fix |
| `caf9ce4` | 15.1 | Failed | Small mode passed; projection sampling command rejected |

`usage-command` conflates unsuccessful command exit, the existing two-second command
bound and oversized output. The cause of these command rejections is unresolved;
no relaxed deadline, cadence or budget is used to turn them into success. A passing
15.1 projection diagnostic also retained a 2.04-second maximum sampling gap against
the nominal 100 ms cadence. It cannot establish complete scheduled sampling.
The new connector-usage rejection and older `6525631` recovery failure each remain
open; a diagnostic-only addition and subsequent pass do not explain either failure.

Each individually passing release/source retained ten original HTML responses per
mode. A separate host driver reopened all **40** original bodies, source-bound helper
outputs and actual publication metadata after transfer, and reapplied the independent
HTML predicates successfully. The driver copied the reviewed source into a private
package, with its source hash recorded in the manifest; this is manual local evidence,
not a new canonical offline verification tool. Its initial missing embedded fixture
preparation failed before replay and was corrected by copying the unchanged testdata.
These different source/release passes do not satisfy a shared-candidate requirement.

Both individually passing sources observed three publication generations with 65
metadata reads in each direct/broker 65-second window. Usage samples cover the wider
monitor lifetime, so their totals cannot be compared as identical calibration windows.
All six completed attempts retain original guest cleanup markers, independently
verified host teardown, matching preflight observations and preserved pinned inputs.
The normal 256 MiB archive cap remained unchanged; the separate physical fault reached
ENOSPC at 1 MiB and rejected its unindexed capture.

The next unblocked work is to retain the exact bounded command failure reason and
resolve connector accounting without dropping unknown usage, then repeat the reduced
suite on both releases at one reviewed source. Complete native 005/006 controls and
sampling/calibration must precede full resource-runner activation. This draft is one
coherent support batch for review; issue 382 and every wider gate stay open.


## Continuing diagnostics in PR 390

The next candidate retains bounded command stdout (64 KiB), stderr (4 KiB),
original wait status, elapsed time and fixed exit/deadline/cancellation/overflow/
start/I/O classifications even when stdout is empty. Real subprocess regressions
cover successful output, nonzero exit, both output ceilings, deadline, cancellation
and failed start. Bounded writes prevent allocation of an oversized process census;
the existing two-second command deadline and sampling cadence are unchanged.
Connector rejection originals also retain the connection stage, wait failure and
context cancellation/deadline flags. These diagnostics supply missing evidence;
they do not themselves fix the historical command or connector rejection.
The [continuation manifest](witness-command-diagnostics-manifest.json) records
behavioral source `7a5490e`, unchanged base `0745c96` and the same PR title. Its
signed focused review executed cmd race/original-wire fuzz and fresh required
check/lint/test/secrets. Both reduced suites passed at this **same source**: 15.1
326.47 seconds, 14.5 285.79 seconds. The separately reopened original metadata,
raw helper outputs and all 40 HTML responses also passed independent validation
with their own actual source/compiler/artifact/probe bindings. Guest/host teardown,
pinned preservation and matching preflight observations passed on each release.

Neither command nor connector rejection recurred in these two new runs. A successful
diagnostic run supplies current evidence, not the missing cause of historical
rejections. The original six-attempt manifest remains byte-identical, together
with the other 156 historical manifests and seven primary operator files.

### Observed sampling coverage remains incomplete

A separate audit partitions the original relative timestamps into nominal 100 ms
intervals, through the last retained sample. It includes the original initial
origin slot and excludes the closing tail, so this is a bounded diagnostic census,
not a complete scheduled-window acceptance calculation. Multiple observations in
a later interval never fill an earlier unobserved interval.

| Release / mode | Nominal intervals | Observed distinct intervals | Unobserved | Additional observations sharing an interval | Largest gap |
| --- | --- | --- | --- | --- | --- |
| 15.1 / native small | 720 | 720 | 0 | 0 | 122.64 ms |
| 15.1 / projection | 591 | 508 | 83 | 23 | 1,624.14 ms |
| 14.5 / native small | 704 | 704 | 0 | 0 | 156.81 ms |
| 14.5 / projection | 528 | 482 | 46 | 15 | 1,933.07 ms |

The largest 15.1 projection gap includes zero additional product-pair CPU,
1.49 seconds of controller CPU and 1.52 seconds of observer CPU across its
1.62-second wall interval. That suggests measurement overhead under the pinned
two-CPU TCG profile; it is correlation rather than a causal proof. The largest
14.5 gap has different CPU deltas, so the same explanation cannot be assumed.
[FreeBSD's statistics-clock source](https://github.com/freebsd/freebsd-src/blob/stable/14/sys/kern/kern_clock.c)
updates maximum RSS, which is relevant to short-lived processes, but no retained
failing connector measurement establishes this as the historical rejection cause.

Next work must distinguish command duration from controller scheduling and preserve
complete independent sampling while retaining every original custody check,
measurement cost, deadline and budget. Complete native 005/006 controls and full
resource qualification remain required. PR 390 stays draft; no broader gate closes.


### Focused projection profiling observation

A subsequent [profile-only record](witness-projection-profile-manifest.json) runs
just the projection subtest at documentation head `17a9f3e`, with Go's existing
CPU-profile option. It **failed** `usage-command-deadline` and remains separate
from qualification: profiling changes controller cost. Original bounded output
has four fields per row, no stdout/stderr overflow, a zero child exit and an expired
context, with 2.42039 seconds elapsed for the complete command step. This establishes
a new deadline-envelope rejection; it does not establish when the child exited
or whether command execution, output handling or controller scheduling caused
the lateness. The earlier generic command rejections remain unexplained.

The retained controller CPU profile spans startup and the projection case. Of
8.26 sampled CPU seconds, generic SHA-256 accounts for 7.33 seconds (88.74% flat);
artifact-digest paths account for 6.84 seconds (82.81% cumulative). These totals
include startup checks outside the sampling window. They cannot prove the cause
of a particular missing interval, nor justify removing original artifact checks.
The pinned bundled `go tool pprof` was unavailable; analysis used the same pinned
Go runtime's `cmd/pprof` sources, adding no repository dependency or service.
An initial private launcher syntax error failed before guest work and is retained
as preparation failure, separate from the actual failed native run.

The focused failure retains original stdout/wait/context/profile/transcript/archive
hashes, matching preflight, unchanged pins, guest cleanup and independently verified
host teardown. Next diagnostics should retain the original Run result and command
duration for successful samples, isolating controller scheduling from command work.
A sampling correction must preserve independent lifetime/custody checks and every
measurement cost. Existing deadlines, cadence, budgets and acceptance gates remain.
