# Full-scale native custody-capacity controls

Status: inactive qualification fixture; broader native 005/resource gates open.
Owner: [issue 382](https://github.com/gonz0w/horvium/issues/382).
Authority: [ADR 0011](../../adr/0011-native-dashboard-witnesses.md),
[EDD](../../design/dashboards/native-witness-binding-EDD.md),
[Test Design](../../design/dashboards/native-witness-binding-TDD.md),
DASH-WIT-003 / DASH-WIT-007 / DASH-WIT-NATIVE-005.

The previous archive test seeded a near-full byte count, and the physical fault
used a reduced 1 MiB mount. Those records retain that scope. This batch exercises
the existing custody component at its actual limits without assigning byte/count
state, raising caps or reducing the physical mount. Controller death remains a
separate surviving-owner/process-lifetime boundary; it is not tested here.

These are explicitly **synthetic custody fixtures** on owned disposable tmpfs.
They are not actual native producer snapshots, management publication evidence,
full broker/resource qualification or a hardware support claim. Existing guards
are unchanged. No product API/configuration, runtime dependency or service changes.
No ZFS/md fixture is created because these controls use only temporary mounts.

## Observable cases

| Case | Actual boundary and denial |
| --- | --- |
| Attempt cap | Capture, deliver through actual Unix endpoints/half-close and independently decode 32 complete literal originals; attempt 33 rejects before reading or creating another original. |
| Single active | Block a real capture after its original prefix; the concurrent capture rejects immediately without reading/queuing. Releasing the first capture cannot clear the failure latch. |
| Raw cap | Preserve exactly 67,508,864 actual bytes, observe one additional source byte and retain `raw-limit` failure. |
| Archive cap | Fill the real archive to exactly 256 MiB. The previous complete original remains eligible at the boundary; the next byte produces a retained zero-byte rejected original and permanent `archive-limit` failure. |
| Physical tmpfs | Fill an actual 320 MiB mount until independently observed ENOSPC while archive accounting is below its limit. Release the owned filler; the failed store must still refuse another source and its earlier complete original. |

Request counters change decimal width at 10. The older attempt fixture replaced
IDs after framing without checking completeness, so the nominal 32 attempts did
not establish 32 complete originals. A strengthened behavioral assertion fails at
10 before the fix. Reframe the literal source after changing IDs; all 32 now must
remain complete. Other historical receipts are not rewritten or upgraded.

All byte growth follows actual bounded reads and file writes. Archive-fill streams
are declared zero-byte fixtures with invalid producer grammar and remain ineligible;
they cannot become numerical expectations. Complete literal control originals use
real unchanged bytes and EOF. Each fault must retain its originals and deny a fresh
management/helper/mode attempt before consuming that new body. A prior complete
original must also be denied after the fault. Restored physical space cannot reset
candidate failure. No original is evicted or silently truncated.

## Custody, transport and cleanup

Keep root-only exclusive namespaces, original file objects/modes, before/after
identity, byte counts, SHA-256 and indexed or unindexed prefixes. Export originals
and receipts before outcome reporting can discard a failed case. Preserve actual
full-size bytes; base `tar` and `gzip -1` compress the deterministic diagnostic
payload for transfer. The compressed transport bound stays unchanged. Independent
readback enforces a separate expanded diagnostic bound covering the archive-cap
and raw-cap cases plus small controls; it never changes any store's 256 MiB limit.
The bounded outer diagnostic allowance covers full-size I/O, not product deadlines.

Temporary mount identity and underlying directory identity must match before
unmount/removal. Close original store/file descriptors first. Verify fixture/product
artifacts, empty original reaper, all five mounts and host image/key/listener/firmware
preservation. No operator disks, datasets, snapshots or backup history are fixtures.
Run both pinned releases at one signed clean reviewed source, then independently
reopen actual exported bytes, object/hash receipts, preflight and teardown.

## Remaining acceptance

Full broker/caller integration, actual controller interruption, control-channel
limits and remaining cancellation/recovery cases stay open. Historical 14.5 gaps,
cadence/overhead/augmented budgets, 300-second idle, original large latency failure,
full resource/browser/hardware/release acceptance remain open. Canonical resource
runner stays inactive and runtime writes remain denied. Run required check/lint/
test/secrets, affected-package review and lab-stop/status; no new Actions or paid
automation. A passing custody component does not close the coordinating gate.

## Retained failed physical-fill predecessor

Source `7287cf0d8af28d42161df9d83376d86dea4d3df6` passed signed clean review
before a failed 15.1 attempt (169.46s). Attempt, single-active, raw-cap and
archive-cap controls passed. The physical filler stopped at 335,478,784 bytes
when its 64 KiB write returned ENOSPC, but another complete 1,328-byte source
still fit. Its original receipt reports the failed assertion and an unfailed
store; this is not physical-exhaustion acceptance. Full original bytes, receipts,
failed exit and verified guest/host cleanup are retained. 14.5 was not executed
for that source.

The corrected filler decreases write granularity down to one byte after each
ENOSPC and independently requires a new one-byte file write to fail before
exercising custody. A failed bulk write alone does not prove no smaller allocation
fits. The mount remains 320 MiB and custody limits/guards remain unchanged. New
review and both-release execution are required; later results do not upgrade the
failed predecessor or infer its unobserved free-block count.

A second 15.1 predecessor, source `5ff6723767f8138deaa67b2b184fdd6ccae0cced`,
failed at the filler-object size assertion after 109.37s; the other four controls
passed. One-byte ENOSPC was observed, but custody fault/resume checks were not
reached. Its receipt retains the reported write count (335,540,224 bytes), no
filler hash/object and only the earlier complete original. These missing
observations remain missing. Guest and host teardown passed; 14.5 was not run.

The next fixture records reported write progress separately from the original
file's observed EOF and resumes from that EOF after an error. FreeBSD tmpfs
[write implementation](https://github.com/freebsd/freebsd-src/blob/releng/15.1/sys/fs/tmpfs/tmpfs_vnops.c#L619-L671)
rolls back file size after a failed data move; summing reported partial progress
is not an independent file-size observation. Actual object size, complete file
hash, one-byte refusal and custody failure remain required.

## Same-source native observations

Signed clean reviewed source `45341bd5e5c69c1d8effe078c141b9df0c290473` passed both pinned disposable
FreeBSD/amd64 targets. The [immutable manifest](witness-custody-capacity-manifest.json)
preserves all four attempts, including both failed 15.1 predecessors.

- FreeBSD 15.1: 111.63 seconds; five controls passed.
- FreeBSD 14.5: 118.97 seconds; five controls passed.

Independent streaming readback reopened every exported byte, frame, object/hash
receipt, source/release context, original ABI probe, unchanged preflight and exact
guest/host cleanup. Each passing release retained 32 complete/delivered attempt
originals, one complete overlapping-capture original, the full raw-cap prefix, the
exact archive-cap originals and the physical-fault original census. All five failed
stores denied a fresh mode/management/helper source without reading its body and
denied the earlier complete original. Physical exhaustion included an independent
one-byte ENOSPC probe and complete filler hash; freeing the original filler did
not reset failure. These observations establish only the synthetic custody scope
above. No acceptance gate closes and no runtime writer or resource runner activates.
