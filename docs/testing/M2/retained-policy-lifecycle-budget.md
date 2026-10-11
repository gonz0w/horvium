# Adopted cumulative lifecycle fixture implementation

Status: failing-first candidate declared; implementation/native execution pending.
Date: 2026-10-11. Owner: Cam (`@gonz0w`).
Refs [issue 305](https://github.com/gonz0w/horvium/issues/305) and
[parent 139](https://github.com/gonz0w/horvium/issues/139).
Authority: [adopted budget amendment](../../milestones/M2/retained-policy-fixture-budgets.md),
[adopted policy-only design](../../milestones/M2/retained-policy-startup.md),
[M2 Test Design](../../milestones/M2/TDD.md) and
[ADR 0009](../../adr/0009-property-executor-recovery-mechanism.md).
PR 319 adopted the amendment at merge `aebfb2c8b85040154f6e70ffb3296bba543a9db9`;
proposal source `0802767509a23ec6309980462773ef8f6f034283`. Design issue 318 is
completed; its satisfied native dependency remains retained, alongside issue 288.

## Declared implementation and evidence

Implement exactly the adopted 330-second lifecycle child, 1,080-second lifecycle
parent work context and 40-minute lifecycle process phase. Keep the original
three selectors, five parents, thirteen waited children, all 283 existing controls
and six added actual budget assertions: **289 controls per release**. Exact
inventory and hierarchy are in the amendment; strict profile marker fields are
`child_seconds=330 parent_work_seconds=1080 phase_minutes=40`. Other suites,
caller/product/collector/command/output/cleanup limits and the actual original
factory/registry/leases/native producers/publication protocol remain unchanged.

Write failing-first consumer/profile tests and portable test-only scoped context/
deadline/cancellation tests. Explicit implementation-absent test-only stubs supply
a compilable fixture selection red; this is not fabricated unchanged product red.
Require real live child deadline/timeout flags and the original shared absolute
parent deadline through the complete fact/metadata callback and cleanup work.
No new product API, timeout configuration, cache, retry or producer wrapper.

Run signed clean-source mandatory `make review`, with shared-tooling full portable
fallback, then serial same-artifact native `policy-startup-lifecycle` qualification
on disposable FreeBSD/amd64 15.1 and 14.5. Preserve all eleven preceding receipts
byte-for-byte. Record each executed source/compiler/artifact/input/log/result hash,
failed/skipped attempt, exact native controls/profiles and independent teardown.
Stop at caller/product or revised fixture failure; no automatic further deadline
increase or CPU tuning. Missing in-guest vectors remain unaccepted even when
independent host teardown passes. Normal product builds exclude fixture code.

No successful native receipt follows from design adoption, portable tests or
partial stages. Runtime writes remain disabled. Initial controlled closed seeds
do not qualify worker shutdown; remaining fault/mixed-history/concurrency/census/
context/capacity, configured startup/workers/backend/global launch and browser/
client/hardware/performance/release acceptance remain open.
