# M2 native property projection

Date: 2026-10-04. Status: implementation; native qualification pending.
[Approved design](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[planning model](planning.md), [slice sequence](../../milestones/M2/implementation.md).
The model subset was accepted by maintainer merge of
[PR 21](https://github.com/gonz0w/horvium/pull/21), source
`f334094eae3d3b6d8ca740465ac09cb5be08ba20`. This projection is another portion of
slice 1; it does not complete that slice or any full M2 acceptance row.

## Parser contract

`internal/storage.ParseProperties` accepts only this fixed projection, with a
validated pool name as the final argument:

```text
/sbin/zfs get -H -p -r -t filesystem -o name,property,value,received,source guid,quota,refquota,compression <pool>
```

There is no product command runner, helper operation, endpoint or CLI flag for
this projection yet. The product runtime remains read-only. The parser consumes
bounded bytes and returns filesystem models; it cannot prove command success,
native completeness/freshness, pool eligibility or root enrollment. The future
reader must qualify those separately, including enumeration before/after the
property read. Dropping all four rows of one filesystem can be syntactically
valid; a parser is not a complete-inventory detector.

Each filesystem must have exactly one row for each selected property. Resolve
parent and inherited source names to GUIDs within the same projection, regardless
of row order. Require a root filesystem and consistent direct parent relationships,
unique GUIDs/names, exact sources, and consistent effective inheritance. The
parser uses the model's conservative management-name subset; unsupported native
names remain observable through the unchanged M1 reader.

Literal native quota/refquota `0` becomes the model's explicit `none` value.
Nonzero values remain canonical decimal uint64 strings. Requests still reject
zero as a limit. Retain each property's received column separately: `-` means
absent, while a shadowed received value remains reviewable even when a local or
inherited value is effective. An active received source must agree with that
received value. This extends immutable plan context and drift checking without
introducing inheritance/restoration commands or a persistent schema.

Reject unknown/duplicate/missing fields, sources or properties, controls/invalid
UTF-8, missing parents/suppliers, inconsistent inheritance, malformed/overflow
numbers, partial final lines and failed limits. Failure returns no partial rows.
Input is capped at 64 MiB, 400,000 property rows / 100,000 filesystems, and 4,096
bytes per field. Bound line size and delimiter count before splitting/allocating
fields. The existing 64 KiB plan and combined policy/model budgets still apply;
these safety caps are not large-system performance qualification.

## Primary source and execution gates

The pinned 14.5 implementation documents the exact received column and source
formatting and emits `0` for literal no-limit quota values:

- [zfs get callback](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/cmd/zfs/zfs_main.c)
- [source/received formatting](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/lib/libzfs/libzfs_util.c)
- [literal quota and received-value handling](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/lib/libzfs/libzfs_dataset.c)

Source inspection is not native output qualification. Unit tests were observed
failing before implementation. `FuzzProperties` joins the standard bounded fuzz
gate. Regression tests ensure a shadowed received-value change invalidates a plan.

`TestNativePropertyObservations` requires root plus the existing explicit QEMU
opt-in. It declares its own random pool namespace, new 512 MiB file-backed md,
filesystems and snapshot/stream before use; no operator device argument exists.
Fixture setup creates a source/received pair with test-only send/receive, checks
actual default/local/inherited/received/shadowed values and exact limits, and
independently correlates GUIDs using a different native list projection. A fixture
foreign child stays unenrolled. Before/after checks preserve projected properties
and mounts; rename preserves the GUID, while name reuse gives a new GUID.

Received-property fixture preparation is not product replication or M4 evidence.
The single-md layout does not qualify creation devices, topology, features,
redundancy, adoption, native write eligibility, durable jobs or operation safety.
The parser's output explicitly fails the unqualified-eligibility planning gate.
Teardown must remove the declared pool/md; temporary fixture files belong to Go's
owned temporary directory. Failure or leak prevents the native result record.

The native script executes this test once in a separate evidence log and runs
`tools/check-property-evidence.py`. Its failing-first regressions refuse skip,
empty/unit-only logs, missing/duplicate/unknown assertions, failure and missing
teardown. Each release must emit the complete native record and test pass.

| Environment | Property projection | Remaining evidence |
|---|---|---|
| Local macOS arm64 | Unit/race/fuzz gates pending final run; native unavailable | Native behavior cannot be inferred from these tests |
| FreeBSD 15.1 amd64 disposable QEMU | Pending execution | Exact tested source/run/version/fixture pins recorded after execution |
| FreeBSD 14.5 amd64 disposable QEMU | Pending execution | Same requirement |

This is partial M2-PROPERTY-001 / M2-DRIFT-001 evidence. Real provider/topology/use
checks, root enrollment, helper transport, mutation, durable recovery, assembled
UI/CLI/browser/resource and all remaining Test Design gates are pending. M1 manual
accessibility remains pending by maintainer choice.
