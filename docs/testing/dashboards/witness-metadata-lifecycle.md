# Native witness publication lifecycle and passive idle

Status: inactive test-only batch under [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md), accepted
[native EDD](../../design/dashboards/native-witness-binding-EDD.md) and
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-004 / DASH-WIT-006 / DASH-WIT-008 / DASH-WIT-009.

Run `TestNativeDashboardWitnessMetadataLifecycle` on both pinned disposable
FreeBSD releases, sequentially, from clean signed review and original build/ABI
preflight. One batch covers publication coalescing, retention/eviction, failed
confirmation, passive native source cadence and the complete small-mode idle
window. It extends the [authenticated refresh control](witness-refresh-overlap.md)
without changing product artifacts, transport caps, 30-second scheduling or the
original 45-second collection deadline. No workload retries or cap overrides.

For native-small and projection-10,000, observe three distinct complete manager
publications using two one-shot refresh requests (202 or genuine busy joins).
Three concurrent registry callers per publication must independently recheck the
same original and share exactly one authenticated metadata GET. Repeated confirmed
lookups must issue no extra metadata requests. Metadata expectations use only the
original independently decoded source; HTTP contributes ID/status/count facts.

After three complete publications, a fresh registry must reject the first
archived-but-evicted ID using the real API's 409 response. Subsequent lookups,
including a current ID, remain denied without another metadata request. The
previously confirmed original remains immutable evidence for its historical
sample; this never asserts that its evicted API generation is still retained.
A separate fresh registry receives a successful original HTTP response, then its
caller cancels before confirmation returns. Cancellation permanently denies that
registry. Another fresh registry denies a new confirmation while marked idle,
performs zero metadata requests, and stays denied after leaving idle.

For native-small, retain the full 300-second request-free idle window. An owned
client transport counter and registry operation count must remain unchanged;
there are no HTTP requests, refreshes or observer controls inside that window.
The unchanged scheduled source continues collecting into original sealed files.
A separate existing sampler records original kernel usage at nominal 100ms
without HTTP or observer-control calls. Its original file, owner/child identities,
readback, sample schedule, missing/late slots, process usage and wait receipts are
retained. Its measurement window brackets idle and may include setup/stop cost;
it is not an exact idle CPU budget result. Retain tmpfs space snapshots before and
after, rather than claiming a measured peak. After idle ends, observe and confirm
a newly current background publication through the same coalescing path.

Preserve every bounded original HTTP response in a generated exclusive root-only
file, including expired and cancelled controls. The successful-only metadata
capture interface retains its original behavior. Oversized/failed reads cannot
become successful observations. Preserve every raw original and sealed index,
all source IDs and independently decoded begin/end times, complete delivery/EOF,
original deadline, helper/configuration/token preservation and guest/host teardown.
Do not label every delivered background original as publication-confirmed.

Portable test-first controls cover raw refused-response capture, strict bounded
reads, no retry, observation failure privacy, legacy successful-only capture,
three-way confirmation and failure draining, and request counting. A separate
independently declared partial source exercises the real manager: latest partials
retain the last complete generation, overwrite previous partials, then expire the
obsolete complete after a new complete publication. This is portable partial
publication evidence, not a native partial producer claim.

This batch supplies related short integration controls and passive diagnostics.
It does not activate the canonical resource runner, measure direct-path calibration,
qualify full metadata/sample integration, arbitrary cancellation/recovery, all
admission ordering, numeric/latency/resource budgets, browser/manual, hardware or
release acceptance. Failed/skipped historical limitations remain authoritative.
Runtime writes stay denied. Preserve failed candidates; corrections require new
signed source, clean review and relevant both-release execution. Final lab-stop
and lab-status must show no owned development process left running.

## Candidate evidence and unfinished qualification

Implementation source `538ac966025d14224e2b3733ffdc2603f942bb0a` passed fresh signed
portable review. The [immutable manifest](witness-metadata-lifecycle-manifest.json)
binds actual source/base/title, compiler/artifacts/ABI, original controller/command,
preflight and independently reopened raw/index/metadata/sampler files.

| FreeBSD | Actual result | Publication controls | Native-small idle | Sampler |
| --- | --- | --- | --- | --- |
| 15.1 | Native suite PASS, 444.33s | 7 confirmed IDs, 21 concurrent callers, 7 healthy metadata GETs; 2 expired 409s, 2 cancellation-after-200 controls | 300.003s; zero owned HTTP or observer/metadata control requests | 3,001 samples; zero missing, duplicate or late slots; CPU counters non-monotonic |
| 14.5 | Preparation FAILED before native execution | Not executed | Not executed | No guest sampler/cleanup receipt or original archive |

On 15.1, every new publication shared one metadata operation across three callers;
repeated and previously confirmed historical originals caused no extra metadata.
Expiry/cancellation registries stayed denied; idle guards performed no metadata
operation. Four native-small and three projection IDs are explicitly confirmed.
All fourteen native-small and three projection raw originals/indexes were reopened;
ten background originals do not acquire publication confirmation from delivery.
Actual frames, IDs/end counts, section/dataset counts, exact hashes/forward EOF,
sealed objects and actor/artifact lineage match. Original deadlines, product
artifacts and configuration/token bytes were preserved; guest/host cleanup passed.

The full idle window retained ten additional source originals and all actual
source start/finish times and gaps. Kernel sampling retained nominal 100ms cadence,
raw file/object/hash, original owner/child records and wait usage. Its sampled CPU
counters decrease at some slots: PairCPU twice, ObserverCPU once, ControllerCPU
twice and SamplerCPU once. The controller's first-to-last delta is -15,559,000ns.
The exact values/drop positions remain in the manifest; cause is unknown. An
additional offline nonnegative-delta assumption exposed this limitation and was
replaced with explicit diagnostic reporting, not clamping or resource acceptance.
CPU accounting, direct-path equivalence and resource budgets remain unqualified.
The sampler window brackets idle, and tmpfs snapshots do not establish peak use.

14.5 completed artifact transfers, then its original preflight SSH command exited
255 with `Connection timed out during banner exchange`. The native test never
started; there is no native archive or guest cleanup receipt. Independent host
process/listener/key/image/variables teardown and pinned runtime preservation
passed. That failed preparation stays failed, separate from 15.1's native evidence.
Automatic approval review rejected a proposed fresh same-source preparation; no
additional guest was launched. Human approval is required before another attempt.
Any future setup correction must retain the original preflight verification,
workload, limits and trusted collection deadline, with all attempts preserved.

Five portable behavioral methods passed, including real-manager partial/latest/
last-complete retention. Final signed documentation repeats baseline checks and
may reuse only exact-input deep evidence from the executed source review. All 179
historical manifests and seven operator files remain unchanged. Both-release,
CPU/accounting, full metadata/sample/resource and wider gates above remain open.
