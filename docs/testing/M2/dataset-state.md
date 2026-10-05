# M2 filesystem mount and encryption constraints

Date: 2026-10-04. Status: D2 partial filesystem-state projection passed on both pinned guests;
accepted in [PR 24](https://github.com/gonz0w/horvium/pull/24), merged source
`be69a4ee846749e2d845dda5ce0d1633c14eb794`.
Design: [approved EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[slice sequence](../../milestones/M2/implementation.md). Extends the
[identity binding](identities.md) with filesystem operational constraints.

## Fixed observation and review contract

The property and identity tables cannot prove that a filesystem is unencrypted
or writable, or preserve its mount policy in review. Add a separate required
filesystem-state table to the internal model:

```text
/sbin/zfs get -H -p -r -t filesystem -o name,property,value,received,source guid,mountpoint,canmount,readonly,mounted,encryption,encryptionroot,keystatus <pool>
```

`ParseFilesystemStates` requires exactly eight selected rows per filesystem,
with five nonempty tab-separated columns and a final newline. Reject unknown,
duplicate/missing fields, unsupported values/sources, controls, invalid UTF-8,
noncanonical GUIDs, missing parent/source/encryption-root identities and
contradictory readonly/encryption inheritance. All failures return no partial table.
Source names resolve to supplying GUIDs in the same complete table; read-only
pseudo-properties retain their native no-source/default representation.

Retain effective and received values, source and supplying GUID for mountpoint,
canmount, readonly, mounted, encryption, encryptionroot and keystatus. Shadowed
received values remain visible. Canmount is not inherited. Readonly/encryption
inherited values must agree with their named ancestor's supplying value.
Mountpoint inheritance binds the ancestor, without claiming a complete native
mount path derivation algorithm. Observe `none`, `legacy` or an absolute path;
this string is never a mount target authorization or filesystem operation.

Encrypted observations retain their local encryption-root GUID, including the
owning root for inheriting children, plus loaded/unloaded key status. Supported
cipher observations are `off`, `on` and the AES 128/192/256 CCM/GCM variants;
unknown future values are unavailable to this parser. Unencrypted observations
require no encryption root and a native `none`/`-` key status.

`VerifyFilesystemStates` requires exact name/GUID coverage against the independent
full dataset-type table. Any volume, encrypted filesystem, effective readonly
filesystem or temporary readonly source refuses this filesystem-only planning
boundary. This conservatively refuses the entire pool observation even when the
adverse filesystem is not selected. A loaded key never makes an encrypted dataset
eligible. Passing these checks does not prove pool/system/topology/features/device
eligibility and cannot override the existing unqualified-eligibility refusal.

Reviews retain independent sorted state copies for target, ancestors and the
entire target subtree. Changed mount policy, mount state, readonly/crypto source,
received values or key/root identity requires a new review. Input ordering is
normalized; unrelated sibling state does not expand the review, although adverse
pool-wide constraints still refuse planning. No plan can grant a mount/key operation.

Raw input is capped at 64 MiB, 800,000 property rows / 100,000 filesystems and
4,096 bytes per field. Combined policy/property/identity/state preflight accounting
is capped at 64 MiB before typed validation maps; the existing 64 KiB review cap
counts these rows without truncation. These are safety limits, not scale claims.

## Tests, sources and execution

Behavioral tests first failed against deny-only stubs and the old model, then
passed. Coverage includes missing whole state rows, mismatched identities,
readonly/temporary/encrypted loaded or unloaded refusal, retained constraints,
source/enum/shape/limit denial, immutable copies, mount/source/received-value drift
and order normalization. `FuzzFilesystemStates` joins the standard bounded gate.
These cover partial M2-OWN-001, M2-DRIFT-001, M2-PLAN-001, M2-PROPERTY-001 and
M2-SCALE-001; no complete Test Design acceptance row follows.

The inspected [pinned property manual](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/man/man7/zfsprops.7)
describes the mount/readonly/cipher/key/root fields and canmount's noninheritance.
The [native formatter](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/lib/libzfs/libzfs_dataset.c)
describes mounted `yes`/`no` and temporary mount overrides. Source inspection
cannot qualify actual native projection behavior.

The existing explicit root/disposable QEMU fixture adds an independent native
list correlation, readonly denial, canmount source/drift, encrypted root/child
identity and loaded/unloaded denial. Its pure-model drift check uses a synthetic
policy/eligibility input to compare actual observations; it is not a qualified
native eligibility adapter or a write launch. Encrypted resources and fresh random
raw key material belong only to that declared disposable fixture. Keys are never
logged, committed or exported as evidence. No product key/mount/storage operation
or service installation is introduced.

Eight new evidence markers join the original 20 property/identity assertions.
The checker requires all 28 plus actual native execution and owned-fixture teardown;
boot, skip, unit pass, missing or failed cases cannot qualify.

## Qualification and remaining work

Run [37246040343](https://github.com/gonz0w/horvium/actions/runs/37246040343)
passed both pinned guests at functional/suite source
`6dd07d2b5a52569c97f9b7524476348ecd7faeaf`. Each executed all 28 property/identity/
state assertions and teardown. Both evidence checks passed; the checksum-pinned
base image and AnyVM runtime remained unchanged. No native repairs were needed.

| Guest | Observed result | Tools |
|---|---|---|
| FreeBSD 15.1-RELEASE-p3 amd64 | All 28 assertions and teardown passed | ZFS userland/module 2.4.2-1-FreeBSD_g6330a45b0; Go 1.27.1 |
| FreeBSD 14.5-RELEASE amd64 | All 28 assertions and teardown passed | ZFS userland/module 2.2.9-FreeBSD_g079ba86d7; Go 1.27.1 |

The [sanitized manifest](dataset-state-manifest.json) records source/job links,
individual cases, inspected primary and tested input hashes, native utility
hashes and guest/runtime pins. QEMU 8.2.2 on Ubuntu 24.04.5 used fresh snapshot
qemu64 guests with two vCPUs and 6,144 MiB, with VNC off. No operator resource or
fixture key material is included in the evidence. This fixture qualifies only
these projections and pure-model comparisons, not a native eligibility collector.

Local `make check lint test secrets` and `make go-check` passed, including vet,
complete unit/race suites and nine bounded fuzz targets. FreeBSD amd64 cross-build
passed before native execution; compilation alone is not native evidence.
Subsequent evidence/documentation edits preserve the tested functional/suite bytes.
Earlier [property](properties.md) and [identity](identities.md) manifests retain
their original source and case sets, with no retroactive claims.

The runtime remains read-only with no native state collector, root enrollment,
transport submission, durable jobs or mutation executor. Slice 1 eligibility/
system/provider/topology/feature/device-use and complete-impact collection gates
remain pending, as do every complete M2 acceptance row, physical/commercial
qualification and the maintainer's deferred manual M1 accessibility gate.

The current qualification gate additionally requires eight pool identity/import/
policy assertions in [pool qualification](pool-state.md). This 28-case state
manifest preserves its original tested source; later checks do not retroactively
qualify it.
