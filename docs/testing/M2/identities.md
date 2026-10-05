# M2 dataset identity binding

Date: 2026-10-04. Status: D2 partial identity projection passed on both pinned release fixtures;
accepted in [PR 23](https://github.com/gonz0w/horvium/pull/23), merged source
`4011e3d9848b2430c7ab30d7d70981a3e0fdd88c`.
Design: [approved M2 Test Design](../../milestones/M2/TDD.md),
[slice sequence](../../milestones/M2/implementation.md).

## Boundary and fixed projection

The property parser cannot detect dropping every row of a filesystem. This slice
adds an independent complete dataset identity/type table, required by the internal
property-review model. The runtime remains read-only; no helper collector,
eligibility proof, enrollment, submission or native executor is connected.

```text
/sbin/zfs list -H -p -r -t filesystem,volume,snapshot,bookmark -o name,guid,type <pool>
```

`ParseIdentities` requires exactly three nonempty tab-separated fields per
newline-terminated record, canonical positive uint64 GUIDs, known types, unique
names, a filesystem pool root and complete native owner/parent relationships.
Filesystem and volume GUIDs must be unique. Snapshot and bookmark GUIDs can
repeat: a received snapshot preserves the source GUID, and a bookmark can refer
to that snapshot. Bind each version to its local owning dataset GUID as well as
its type/name/GUID. Observation never grants snapshot or bookmark mutation scope.

Names use the existing conservative management subset. Snapshot/bookmark names
use exactly `@`/`#` and a 1–64 byte ASCII suffix starting with a letter or digit,
then letters/digits/underscore/hyphen. Unsupported native names deny this planner;
M1 observation remains separate. Input bounds are 64 MiB, 100,000 identity records
and 4,096 bytes per field. Combined policy/property/identity preflight accounting
and the existing 64 KiB review limit apply, without truncation or paging impact.

`VerifyFilesystemIdentities` validates exact filesystem name/GUID agreement with
the independently parsed property table. Missing whole filesystems and reused
names fail. Volumes are classified but conservatively refuse pool qualification:
volume impacts are not represented in this filesystem-only slice. This does not
qualify mixed pools or claim volume management support.

A future trusted collector must check bounded capture, successful command exit,
full type coverage and independent identity enumerations before and after
property reads. `VerifyIdentityStability` compares normalized complete tables,
including version membership and ownership. It detects observed drift; it cannot
provide an atomic fence or detect a change that appears and disappears between
reads. Final native checks, exclusive writer rules, eligibility and external-root
race handling remain unimplemented obligations.

## Primary sources

The pinned 14.5 [list manual](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/man/man8/zfs-list.8)
identifies all four selectable dataset types. The [property manual](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/man/man7/zfsprops.7)
describes lifetime dataset/bookmark GUIDs and preservation of a received snapshot's
GUID. These inspected sources motivate the owner binding; actual native tests
are required separately. Source hashes will accompany executed evidence.

## Review drift and tests

Reviews retain an owned, sorted version slice for target, ancestors and the
entire target subtree. Adding/removing/renaming/reusing a version or changing its
owner invalidates revalidation. Unrelated sibling versions remain outside that
review. The selected filesystem scopes and property permissions remain explicit.

Tests were observed failing against deny-only stubs before implementation.
`TestPropertyIdentityCoverageCannotOmitWholeFilesystem`, version ownership/drift,
volume refusal, ordering and budget tests cover subsets of M2-OWN-001,
M2-DRIFT-001, M2-PLAN-001, M2-PROPERTY-001 and M2-SCALE-001. `FuzzIdentities`
joins the bounded fuzz gate. The evidence checker requires all original property
assertions plus nine new native identity assertions and executed teardown; unit
pass, boot, skip or incomplete markers cannot qualify.

The opt-in owned-fixture test extends the previous property procedure with
snapshot/bookmark correlation, replicated GUIDs, snapshot rename/name reuse,
complete filesystem coverage, volume classification/refusal and stable reads.
It creates a sparse volume with `volmode=none` only inside its declared pool,
then removes that owned fixture. No operator disks, credentials or inventory are
inputs. These setup commands are test code, not product storage operations.

## Qualification

Native run [37244653409](https://github.com/gonz0w/horvium/actions/runs/37244653409)
passed both guests at functional/suite source
`2d4478c8a8e34b0ac40657d7330b4d73f0df6b21`. Each executed all 20 property/identity
assertions and teardown; the evidence checker passed and the checksum-pinned base
image and AnyVM runtime remained unchanged. No native repairs were needed.

| Guest | Native result | Observed tools |
|---|---|---|
| FreeBSD 15.1-RELEASE-p3 amd64 | All 20 assertions passed; owned pool/md removed | ZFS userland/module 2.4.2-1-FreeBSD_g6330a45b0; Go 1.27.1 |
| FreeBSD 14.5-RELEASE amd64 | All 20 assertions passed; owned pool/md removed | ZFS userland/module 2.2.9-FreeBSD_g079ba86d7; Go 1.27.1 |

The [sanitized manifest](identities-manifest.json) records exact job/revision links,
utility and tested input hashes, inspected primary source hashes, guest/image and
runtime pins, and individual assertions. The QEMU guests used qemu64, two vCPUs,
6,144 MiB, snapshot mode and no VNC; host QEMU 8.2.2 ran on Ubuntu 24.04.5.
Fixture commands were confined to declared disposable resources. No host services
were installed or enabled for repository validation.

Local `make check lint test secrets` and `make go-check` passed, including vet,
the complete unit suite, race detector and eight bounded fuzz targets. The FreeBSD
amd64 test-binary cross-build passed before the native run; compilation alone is
not native evidence. Later evidence/documentation edits preserve tested functional
and suite bytes. Prior property results remain separately preserved in
[the property record](properties.md).

This is partial D2 observation evidence, not mutation or release qualification.
Full slice 1, every complete M2 Test Design row, state/recovery/privilege/mutation/
resource/browser gates and hardware/commercial support remain pending. Manual M1
accessibility remains pending by maintainer choice.

The current property qualification gate also requires the eight filesystem-state
assertions in [state qualification](dataset-state.md). The 20-case identity
manifest preserves its original tested source; later checker changes do not
retroactively qualify it.
