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

No native run has completed for this candidate at preparation. Actual results
will be added in this same outcome PR. One release, portable tests, a successful
cross-build or process exit alone cannot establish both-release resource acceptance.
Controller/browser/OS/ARC/file daemons and physical throughput remain outside the
paired-process budget. Large inventories are synthetic; one pool cannot qualify
large pool cardinality or commercial hardware. Browser/accessibility, startup
recovery, release acceptance and hardware support remain open. Runtime storage
writes remain disabled. No operator storage or host services are used.
