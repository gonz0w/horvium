# Captured prefix retention after a partial storage write

Status: inactive test-only custody fixture; candidate execution pending.
Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-007 / DASH-WIT-NATIVE-005.

A file writer can acknowledge a nonempty prefix and return an error from the same
write. The previous capture copy used `io.MultiWriter(file, hash)`, which skips
hashing that write's acknowledged prefix on error or a short write. The subsequent
custody check then rejects the mismatched hash, leaving bytes on disk without a
sealed diagnostic original. Qualification already fails; this correction preserves
its original evidence rather than making failed storage usable.

Capture now shares the existing counted forwarding writer's acknowledgment rules.
The recorded hash includes exactly each acknowledged prefix, including the final
partial write. Remaining raw/archive bounds, 64 KiB buffer, exclusive creation,
root custody, seal/reopen checks, permanent failure and delivery/publication rules
stay unchanged. This is test-only witness code, with no product runtime change.

## Behavioral and native controls

Portable tests cover complete writes, zero-byte ENOSPC, partial first and later
writes with ENOSPC, a short write without error, and the unchanged remaining-byte
read/write bound. The extracted previous copy behavior fails the three partial
prefix cases before correction. Existing cancelled-read and archive-prefix controls
also remain required.

The fixed native selector runs normal and partial-ENOSPC cases on both pinned
FreeBSD releases. Each owns a fresh full-size 320 MiB tmpfs, records its original
identity and the actual fixture actor, and starts with a complete synthetic golden
original whose byte delivery and eligibility are checked. The fault case fills the
actual mount, independently checks that a new one-byte allocation fails, hashes
and reopens the filler, then truncates that filler by exactly 32 KiB. It captures
a larger fixed synthetic byte stream through the actual store and file writer.

Require a positive acknowledged prefix smaller than one 64 KiB relay buffer,
actual ENOSPC, the original copy-stage/actor boundary, sealed raw bytes with the
exact prefix hash, and an incomplete, undelivered diagnostic reference. After
reclaiming the filler, require a real allocation to succeed while old-original
lookup, failed-original forwarding and another capture stay denied. The rejected
source must remain unread and the first failure boundary unchanged. A zero-byte
failure or unindexed raw file does not satisfy this partial-write control.

Export the actual raw originals and bounded case receipt before unmount, including
on assertion failure. Independent host readback must reopen raw bytes, object/
mode/hash bindings, source/build/preflight and actor receipts. Verify original
mount removal, underlying directory identity, no descendants, unchanged artifacts
and host teardown. No seeded archive state, cap reduction, workload retry or
changed product deadline is used.

```sh
cd /root/work/cmd/horvium
env HORVIUM_NATIVE_VM=1 HORVIUM_PERF_VM=1 /root/work/.tools/dashboard-native.test -test.v -test.count=1 -test.timeout=8m -test.failfast -test.run='^TestNativeDashboardWitnessPartialStorage$'
```

The eight-minute limit is an outer diagnostic allowance for two owned fixtures.
It adds no trusted collection allowance. Original source-bound host preflight,
clean signed review and preserved failed attempts remain required.

## Acceptance boundary

These are synthetic custody controls using actual FreeBSD file I/O, not original
producer streams, broker/caller capacity or manager publication. The existing
[broker full-storage control](witness-broker-enospc.md) failed at file creation;
it does not establish this later partial-copy boundary. Neither evidence replaces
the other or upgrades historical failures. Broker raw/archive limits, integrated
admission, remaining cancellation/recovery, cadence/overhead, full resource and
activation review remain open. Browser/manual accessibility, hardware and release
acceptance remain separate. Canonical resource instrumentation stays inactive and
runtime storage writes remain denied.
