# Native discovery slice evidence

Date: 2026-10-03 (America/Denver). Scope: first M1 implementation slice.
Design: [PRD](../../milestones/M1/PRD.md), [Test Design](../../milestones/M1/TDD.md).
This is partial milestone evidence; it does not accept all M1 requirements.

## Native product execution

[QEMU run 37164798739](https://github.com/gonz0w/horvium/actions/runs/37164798739)
passed both actual product discovery jobs, both historical research jobs and
foundation smoke. Product/suite source: `d70f76af72afc0c5f00328f328d7927aa26a6092`.
The product script used CGO_ENABLED=0, the reviewed Go 1.27.1
archive checksum, Python 3.14.7 as a test/development installer only, and the
actual internal Go reader/parser packages.

| Guest amd64 | Installed/running kernel and userland | ZFS userland/module | Native test result |
|---|---|---|---|
| 15.1 | 15.1-RELEASE-p3 for all three | 2.4.2-1-FreeBSD_g6330a45b0 for both | Passed |
| 14.5 | 14.5-RELEASE for all three | 2.2.9-FreeBSD_g079ba86d7 for both | Passed |

Environment: Ubuntu 24.04 hosted runner, pinned VM Action
`a2f9a41fa97f6848b8c3b791087dfcdaa5b473ff`, builder 2.2.8, snapshot-mode guests,
6,144 MiB memory as shown in the Action invocation, default virtual CPU selection.
Image digests and host hardware are not pinned by this run. This qualifies the
observed contracts in these guests, not an immutable image or performance baseline.

`TestNativeDiscovery` actually ran (not skipped) on both guests. It:

- Created only a fresh 512 MiB file-backed md device and generated-name pool,
  verified exact backing-file ownership, and disabled fixture mountpoints.
- Called the actual `Source.Collect` fixed native reads and typed parsers.
- Compared fixture GUID, pool size and provider geometry/class to independent
  native queries; asserted real nonzero memory and separate OS/ZFS versions.
- Verified discovery preserved mountpoint/readonly/compression; external dataset
  rename appeared in the next observation with the original GUID.
- Destroyed the owned pool/detached the md and verified both disappeared. Any
  teardown failure would fail the test/job. The guest was then discarded.

Coverage contributes to M1-NATIVE-001, M1-PARSE-001, M1-IDENTITY-001,
M1-NOCHANGE-001 and M1-BUILD-001 only for the reader/fixture portions. There is no
helper, authenticated CLI/UI or file-service restart evidence in this slice.

## Local and repository gates

Go 1.27.1 darwin/arm64 development: repository/content/link/config/action checks,
ShellCheck, actionlint, eleven foundation/SDK regression tests, Go vet/unit tests,
race detection and bounded pool/GEOM fuzz runs passed. Gitleaks found no leaks.
Unit tests were observed failing before parser/reader behavior existed; regression
cases were observed failing before the archive/framing/output-budget repairs.
Native host discovery on macOS returns unsupported; fixture tests are not mistaken
for native macOS NAS support.

The 10,000-dataset synthetic projection preserves row count/GUID precision and
reports explicit budget exhaustion rather than truncation. A local illustrative
parser benchmark used the following command on Apple M5 Max,
Go 1.27.1 darwin/arm64, unchanged parser/benchmark code from the native source.
Observed: 100 rows ~24.6 microseconds / 44,000 allocated bytes per operation;
10,000 rows ~2.99 milliseconds / 5,840,661 allocated bytes per operation.
Allocated bytes/op are not peak RSS; this is not request p95, NAS throughput,
MicroServer performance or commercial-host qualification.

```sh
go test ./internal/inventory -run '^$' -bench '^BenchmarkDatasetProjection$' -benchmem -benchtime=200ms
```

## Failures and reproducible repairs

- [Initial native run](https://github.com/gonz0w/horvium/actions/runs/37164064625)
  stopped at SDK setup: the authenticated archive contains a top-level `go`
  directory, which the layout guard incorrectly rejected. A local miniature
  archive regression reproduced it. The fix accepts that directory while retaining
  checksum verification, safe extraction and clean staged SDK replacement.
- [Second run](https://github.com/gonz0w/horvium/actions/runs/37164324803) and
  [safe diagnostic run](https://github.com/gonz0w/horvium/actions/runs/37164574893)
  passed host/ZFS projections but failed GEOM framing. The kernel builds its
  string with SBUF_INCLUDENUL. The strict parser now removes exactly one terminal
  NUL, still accounting for its byte and validating the entire XML. Repeated or
  embedded NULs, second roots and trailing junk remain rejected. The earlier
  research decoder stopped after the first root and did not expose this contract.
- Review found that discarding failed output could bypass aggregate accounting.
  A regression reproduced the gap; consumed stdout/stderr now count even on error,
  and overflow exhausts the remaining budget and cancels the process group.

Inspected immutable [kernel sysctl producer](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/geom/geom_kern.c)
and [XML shape producer](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/geom/geom_dump.c)
are hashed in the [source manifest](../../research/M1/source-manifest.json).
No upstream source or host inventory dump is committed.

## Remaining gates

Physical identity/controller behavior, peer-authenticated helper/framing, cache
freshness/dedup/paging, token/session/web protections, UI/CLI/browser accessibility,
independent file-service operation, complete application budgets and later NAS
protocol/restore/package/upgrade qualification remain required. See
[implementation status](../../milestones/M1/implementation.md).
