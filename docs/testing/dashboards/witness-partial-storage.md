# Captured prefix retention after a partial storage write

Status: inactive test-only custody fixture; focused results below.
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

The fixed native selector runs three cases on both pinned releases. Each owns
a fresh full-size 320 MiB tmpfs, records its identity and actual fixture actor,
and starts with an eligible, delivered synthetic golden original:

- Normal capture preserves complete bytes and delivery.
- ENOSPC fills the actual mount, independently checks that a new one-byte
  allocation fails, hashes/reopens the filler, and truncates it by 32 KiB. The
  observed tmpfs behavior rejects the next 64 KiB write with zero acknowledged
  bytes. Retain that empty diagnostic original and actual ENOSPC separately.
- Positive partial write runs in a disposable child with the original fixture
  executable and bounded inherited bootstrap. Only that child sets its soft
  `RLIMIT_FSIZE` to 8,193 bytes and ignores SIGXFSZ; hard/store/archive/mount bounds
  stay unchanged. The actual file writer must acknowledge exactly 8,193 bytes and
  return EFBIG. Record before/during/restored limits, actual child actor and wait.
  Restore the limit before exporting or checking post-failure allocation.

Both fault cases require the copy-stage/actor/error boundary and a sealed,
incomplete, undelivered reference with the exact acknowledged prefix hash. After
removing the actual filler or restoring the child limit, require allocation to
succeed while old-original lookup, failed-original forwarding and another capture
stay denied. The rejected source remains unread and the first boundary unchanged.
Zero-byte ENOSPC does not satisfy the positive partial-write criterion, and EFBIG
is not reported as physical ENOSPC. No process limit is changed in the parent.

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

The eight-minute limit is an outer diagnostic allowance for three owned fixtures.
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

## Preserved failed fixture assumption

Source `dcb70e0374311386bbf0358151395c10b9b26983` passed normal custody
on 15.1, then failed the positive-prefix predicate: the full-size tmpfs returned
ENOSPC with zero acknowledged bytes after 32 KiB was reclaimed. An empty sealed
`capture-io` original and its actual copy/errno boundary were retained. The whole
run failed in 27.76 seconds; 14.5 was not started. Guest/host teardown passed.
That attempt remains failed. It does not demonstrate a partial filesystem write
or disprove the portable regression. The new child file-limit control exercises
that distinct boundary without relabeling the earlier ENOSPC result.

## Source-bound component results

Signed clean reviewed source `75622461d7238e151f2713885b8ddc070beac7d3`, base `f36d4b0a2ca93c27a4197291f55f84934621af41`,
passed all three cases on each pinned FreeBSD/amd64 profile:

| Case | FreeBSD 15.1 | FreeBSD 14.5 |
| --- | --- | --- |
| Normal synthetic capture, delivery and eligibility | Passed | Passed |
| Actual tmpfs ENOSPC, zero-byte sealed diagnostic | Passed | Passed |
| Child-only file-size limit, actual 8,193-byte EFBIG prefix | Passed | Passed |

The [immutable manifest](witness-partial-storage-manifest.json) records exact
source/base/title, compiler/artifacts, source archive, command/transcript/archive/
probe/readback hashes and both failed and passing source boundaries. Independent
readback reopened ten raw originals across the passing runs: six complete and
delivered golden originals plus two empty ENOSPC and two positive EFBIG diagnostic
originals. These are synthetic custody values; no original establishes publication.
Both fault cases on each release denied old eligibility, failed forwarding and
new source reads after actual resource restoration/allocation. Actual child waits
exited zero within the 45-second child-control budget; child limits were restored and parent
limits were never changed. All original mount/directory, guest/host artifact,
descendant and runtime teardown checks passed. All 173 earlier manifests and
seven operator-owned files remained byte-identical.

The passing native source completed fresh check/lint/test/secrets, affected
`cmd/horvium` race checks and its one compiler-listed portable fuzz target.
A documentation-only followup may reuse that executed exact-input deep evidence
while repeating the required baseline and final lab-stop/status checks. This is
component evidence under DASH-WIT-NATIVE-005, not complete native/resource or
release acceptance. The preserved 15.1 predecessor remains failed and its 14.5
execution remains absent.
