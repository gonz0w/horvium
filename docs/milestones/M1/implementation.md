# M1 implementation and acceptance status

Date: 2026-10-03. Design approved in [PR 3](https://github.com/gonz0w/horvium/pull/3).
M1 remains Implementing, not accepted or released. Work proceeds through focused,
maintainer-reviewed slices following the [EDD](EDD.md).

## Slice 1: native discovery models, parsers and reader

Accepted by maintainer merge of [PR 4](https://github.com/gonz0w/horvium/pull/4),
source `4a21aa72b4a2bb516018bb06f21dbcc287a4ab0f`, on 2026-10-03.

Product library, no service/HTTP/CLI installation or data mutations:

- Typed host/GEOM/pool/dataset observations with per-section timestamps and safe
  error codes; partial/failed sections are distinct from successful empty lists.
- Literal TSV parsing with exact fields, uint64 validation, decimal-string GUIDs,
  duplicate identity rejection and no partial rows on failure.
- GEOM graph preserves class/geom/provider/consumer relationships, labels provider
  kinds, and reports missing/conflicting/duplicate disk identity. A value unique
  in one collection is not proven durable or permission to format a device.
- XML has byte, field, depth, element and graph-record budgets; DTD directives,
  contradictory references and malformed structures fail. Unknown bounded inner
  metadata is ignored; no opaque property is interpreted as authority.
- The native string sysctl includes a terminal NUL. Remove exactly one terminal
  byte after accounting for its size; embedded/repeated NULs and trailing junk
  remain errors. Full XML consumption is tested, rather than relying on a decoder
  that returns immediately after the first root element.
- Actual FreeBSD reader uses seven fixed absolute command/argument projections,
  sanitized environment, bounded combined stdout/stderr and process-group
  cancellation. Aggregate collection bytes/graph and ZFS records are bounded.
  Discarded stderr and failed-command output consume the byte budget too.
  Default native command/collection deadlines are ten/45 seconds. The reader uses
  only the Go standard library; the separate helper uses the approved syscall
  module below. Non-FreeBSD constructors fail explicitly.

Native fixture tests require root and explicit disposable-QEMU opt-in. They create
only their own file-backed md/pool, compare captured GUID/size/geometry to
independent native queries, check observed external rename, verify unchanged
mountpoint/readonly/compression, and verify pool/md teardown. No host inventory
dump is printed. Actual product reader tests passed on both selected releases;
see [native discovery evidence](../../testing/M1/discovery.md) for exact source,
versions, procedure, failures/repairs and limits.

## Slice 1 requirement evidence

| Test ID | Implemented portion | Remaining before full M1 gate |
|---|---|---|
| M1-PARSE-001 | Unit hostile/empty/overflow/truncated/limit cases; bounded pool/GEOM fuzz targets | More real hardware fixtures and full integration |
| M1-IDENTITY-001 | Duplicate/missing/conflicting serial/lunid; provider classes/relations; native fixture rename test | Physical controller persistence and future adoption safety |
| M1-FAIL-001 | Source permission/exit/missing/timeout/cancel/byte/record/malformed handling; partial host metadata | Stale cache/UI behavior; helper evidence below |
| M1-NATIVE-001 | Actual native reader and independent fixture assertions | CLI/UI paths; helper evidence below |
| M1-NOCHANGE-001 | Native reader fixture properties preserved; no mutation commands in reader | Independent file-service tests; helper policy below |
| M1-BUILD-001 | Native-constrained Go module, pinned dev/native SDK, no product cgo/Python/Node | Executables/assets/release build provenance |
| M1-SCALE-001 | Reader/parser byte/record bounds and aggregate collection limits | One collector, deduplicated refresh and generation cache |

Local development gates are `make check lint test secrets go-check`. Go checks
include vet, race detection, and short bounded parser fuzzing. Race tests use cgo
as development instrumentation; product builds use CGO_ENABLED=0. Python installs
the reviewed SDK in test/development only. Native test logs record actual versions;
a skipped disposable fixture is never native acceptance.

## Slice 2: kernel-authorized helper transport

Accepted by maintainer merge of [PR 5](https://github.com/gonz0w/horvium/pull/5),
source `60d73215279fd33a1a2facbb66dd198319d19bc2`, on 2026-10-03.
`internal/helper` supplies a FreeBSD-only inventory server/client library:

- Actual `LOCAL_PEERCRED` wrapper verifies credential ABI and exact configured
  management UID before request reads; client verifies root listener UID before
  sending. Shared groups/payload authority and a root-client exemption are denied.
- Root-owned restricted directory/ancestors, new 0660 socket, strict ownership,
  refusal of preexisting files/sockets/symlinks and cleanup of only its owned inode.
- Versioned length-prefixed typed records with frame/aggregate/field/depth/count
  bounds, strict duplicate/unknown/missing-field denial and explicit end/EOF.
  Interrupted/malformed exchanges yield no publishable snapshot. Successful
  partial source observations retain errors and timestamps.
- Bounded admitted clients/deadlines, one in-flight collection and no queue;
  cancellation closes I/O and reaches the native reader. No general RPC operation,
  native command/path argument, HTTP listener or file-service management.
- Checksum-pinned x/sys v0.48.0, full BSD notice, no transitive modules/product cgo.
  Local denial/race/bounded framing fuzz tests; native dropped-UID/ownership/
  actual-reader tests use disposable Unix-socket fixtures only.

See [helper evidence](../../testing/M1/helper.md) for native execution status,
exact versions/source and M1-PRIV/FAIL/NOCHANGE/SCALE/BUILD traceability.
Installed service/process-restart qualification remains pending.

## Slice 3: collector, bounded cache and shared queries

Under review in [PR 6](https://github.com/gonz0w/horvium/pull/6). One worker collects on startup/interval, deduplicates
refreshes and publishes owned immutable observations atomically. At most two
bounded generations preserve latest and last complete context; failures retain
data and safe error state, including while retrying. Per-section success timestamps,
90-second freshness, exact generation cursors and reload-required expiry prevent
mixed or falsely fresh rows. Typed bounded pool/dataset/provider pages and host
overview use cached data only. Shutdown clears volatile inventory.

Shared validation and bounded deep copying protect ownership/record/string/model
storage budgets. Local cancellation/race/concurrency/100,000-item cases and native
unprivileged helper outage/recovery integration passed on both releases; tracked in
[collector evidence](../../testing/M1/collector.md). These libraries expose no
HTTP listener, operator CLI or installable service.

## Remaining slices and acceptance

1. Token setup, strict config/session/CSRF/Host/Origin protections and tests.
2. Authenticated CLI and accessible HTML5/CSS/HTMX console, browser evidence.
3. Independent service/restart tests and measured application budgets.

All PRD requirements remain subject to the full [Test Design](TDD.md). Native
reader success is not NAS readiness, commercial qualification or a working UI.
