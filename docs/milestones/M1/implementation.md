# M1 implementation and acceptance status

Date: 2026-10-03. Design approved in [PR 3](https://github.com/gonz0w/horvium/pull/3).
M1 remains Implementing, not accepted or released. Work proceeds through focused,
maintainer-reviewed slices following the [EDD](EDD.md).

## Slice 1: native discovery models, parsers and reader

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
  Default native command/collection deadlines are ten/45 seconds. No runtime
  dependencies beyond Go standard library; no syscall module is added until the
  helper slice uses it. Non-FreeBSD constructors fail explicitly.

Native fixture tests require root and explicit disposable-QEMU opt-in. They create
only their own file-backed md/pool, compare captured GUID/size/geometry to
independent native queries, check observed external rename, verify unchanged
mountpoint/readonly/compression, and verify pool/md teardown. No host inventory
dump is printed. Actual product reader tests passed on both selected releases;
see [native discovery evidence](../../testing/M1/discovery.md) for exact source,
versions, procedure, failures/repairs and limits.

## Requirement evidence in this slice

| Test ID | Implemented portion | Remaining before full M1 gate |
|---|---|---|
| M1-PARSE-001 | Unit hostile/empty/overflow/truncated/limit cases; bounded pool/GEOM fuzz targets | More real hardware fixtures and full integration |
| M1-IDENTITY-001 | Duplicate/missing/conflicting serial/lunid; provider classes/relations; native fixture rename test | Physical controller persistence and future adoption safety |
| M1-FAIL-001 | Source permission/exit/missing/timeout/cancel/byte/record/malformed handling; partial host metadata | Helper transport interruption and stale cache/UI behavior |
| M1-NATIVE-001 | Actual native reader and independent fixture assertions | Helper, CLI and UI paths |
| M1-NOCHANGE-001 | Native reader fixture properties preserved; no mutation commands in reader | Authenticated helper policy and independent file-service tests |
| M1-BUILD-001 | Native-constrained Go module, pinned dev/native SDK, no product cgo/Python/Node | Executables/assets/helper dependency/build provenance |
| M1-SCALE-001 | Reader/parser byte/record bounds and aggregate collection limits | One collector, deduplicated refresh and generation cache |

Local development gates are `make check lint test secrets go-check`. Go checks
include vet, race detection, and short bounded parser fuzzing. Race tests use cgo
as development instrumentation; product builds use CGO_ENABLED=0. Python installs
the reviewed SDK in test/development only. Native test logs record actual versions;
a skipped disposable fixture is never native acceptance.

## Remaining slices and acceptance

1. Kernel-authorized helper transport, safe socket lifecycle and framed records.
2. Single collector, two bounded generations, stale/error publication and paging.
3. Token setup, strict config/session/CSRF/Host/Origin protections and tests.
4. Authenticated CLI and accessible HTML5/CSS/HTMX console, browser evidence.
5. Independent service/restart tests and measured application budgets.

All PRD requirements remain subject to the full [Test Design](TDD.md). Native
reader success is not NAS readiness, commercial qualification or a working UI.
