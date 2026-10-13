# Assembled dashboard native resource qualification

Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382), under
[Capacity 290](https://github.com/gonz0w/horvium/issues/290), with
[Overview 289](https://github.com/gonz0w/horvium/issues/289) and
[M2 131](https://github.com/gonz0w/horvium/issues/131) as broader gates.
Authority: DASH-REQ-006 / DASH-PERF-001 and the adopted
[resource Test Design](../../design/dashboards/resource-qualification.md).

Merged [PR 381](https://github.com/gonz0w/horvium/pull/381) satisfies the
[runner integration](runner-integration.md) review prerequisite. This outcome
selects one clean signed/DCO candidate with a matching completed local review,
then executes the existing performance suite sequentially on pinned FreeBSD 15.1
and 14.5 disposable Mac QEMU targets. The original workload, resource budgets,
artifact provenance, predicate/control requirements and teardown remain unchanged.

A fresh complete portable review supplies original executed deep evidence for this
candidate; documentation-only baseline checks alone cannot qualify its build.
Before any receipt followup changes HEAD, reopen the two explicitly named original
lab receipts with `verify_pair`. Publish sanitized source-bound observations,
including failed candidates and actual cleanup limitations. Fixes require a new
reviewed source and new relevant execution, never a relabelled failed attempt.

## Original failed candidate

The [failure manifest](native-resource-qualification-manifest.json) retains signed
candidate `552f88ab665463461ee1546317e7aee693aa340a`, base
`d171e5d5ddc632cbd6ffd1015b999443f9f24e49`, and its prepared review title
`test(dashboards): qualify the assembled resource candidate`. Its complete local
review passed required checks, 305 tooling tests, all 11 portable race packages
and 54 fuzz targets. Original build, transfer and guest artifact preflight passed.
The running guest reported FreeBSD kernel/userland 15.1-RELEASE-p3, Go 1.27.1,
ZFS 2.4.2 and 6,387,634,176 physical bytes, using pinned two-CPU/6144-MiB/TCG QEMU.
The final failure/evidence PR has a different title/source; this observation stays
bound to its actual prepared candidate, not the later failure-retention fix.

The pre-existing ignored lab directory first failed its private-permission check,
before creating any guest or disposable fixture. Its owned permissions were
restricted to 0700 before the sole actual native attempt. Both the refusal and
actual attempt remain retained; the native failed candidate was not rerun.

The 15.1 suite finished **failed**, with five thousand scheduled observations:
2,000 synthetic-small initial/replacement requests, then 1,000 initial requests in
each remaining mode. Only synthetic-small reached replacements. Its combined peak
RSS was 35,536,896 bytes and idle CPU 0.2166% of one core over the full 300-second
window. This mode's measurements do not accept the failed parent suite.
The 100,000-record boundary, record exhaustion, byte exhaustion and recovery
controls completed outside normal-size RSS budgets.

All synthetic-large initial responses passed predicates, but latency failed:

| Initial case | Successful responses | p95 milliseconds | Adopted large budget |
| --- | ---: | ---: | ---: |
| Overview | 200 | 619.593 | <250 |
| Pools/default | 200 | 635.177 | <250 |
| Pools/maximum | 200 | 427.537 | <250 |
| Datasets/default | 200 | 875.772 | <250 |
| Datasets/maximum | 200 | 1670.440 | <250 |

The native-small fixture declared 115 total datasets, including 100 independently
counted test-owned datasets. Native-small and projection-10,000 each recorded
200 HTTP-200 predicate failures for Overview and each pool case; their two dataset
cases each passed 200 predicates. Their replacement and native crash/recovery
controls were not reached. The original failed samples contain an empty predicate
classification, so this evidence cannot identify the exact mismatch or establish
that mutable pool accounting, rounding or any particular product path caused it.
No raw bodies or inventory dumps are published.

The resource command and outer worker both exited nonzero with completed clean
owned groups and no escalation. The worker then observed QEMU exit, listener
refusal, key/image/variables removal and unchanged base/QEMU/firmware hashes.
Independent readback confirmed listener/artifact absence and preservation.
The failed suite did not produce the complete independent parent cleanup receipts
or post-resource artifact preflight; host teardown does not assert whole guest
acceptance. Strict/legacy resource acceptance and joint verification were not
reached. FreeBSD 14.5 is explicitly **skipped because this candidate already
failed**. A corrected reviewed candidate needs new relevant execution on both
releases; a repeated failed candidate is not a performance retry strategy.

## Failure retention correction and focused followup

The actual failed attempt exposed a fallback bug: after the worker wrote its
specific exception and cleanup record, the outer supervisor replaced the reason
with `supervisor interrupted the run or worker failed`. The original controller
log, failed suite and result remain unchanged. A failing portable regression
reproduced the overwrite. Fallback now preserves a nonempty original reason and
cleanup observations while maintaining `qualified: false`; missing reasons still
receive the existing generic interruption diagnostic. No timeout, signal, removal
scope, runtime behavior or budget changes. This correction has not executed a new
native candidate and cannot qualify the old failed one.

The next work under issue 382 is a focused diagnostic loop: retain bounded,
non-sensitive predicate failure classifications without publishing response bodies,
and profile the actual query/rendering paths before choosing a correction.
Source inspection identifies inventory-wide index construction/sorting in dataset
queries as one candidate for profiling, not a proven cause. Use narrow behavioral
regressions and scoped diagnostic fixtures first. A corrective candidate still
requires original signed review and both unchanged full native resource procedures
before acceptance; manual browser/accessibility and startup recovery remain open.

Controller/browser/OS/ARC/file daemons and physical throughput remain outside the
paired-process budget. Large inventories are synthetic; one pool cannot qualify
large pool cardinality or commercial hardware. Runtime storage writes remain
disabled. No operator storage, host services, new dependencies or paid automation
are introduced. Existing historical manifests retain their original bytes and
limitations. GitHub owns live status and remaining acceptance.
