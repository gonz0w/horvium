# M2 filesystem mount and encryption constraints

Date: 2026-10-04. Status: implemented behavioral subset; pinned native execution pending.
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

Both pinned FreeBSD releases must execute before a native claim. The sanitized
source/run/hash manifest will be added after qualification. The runtime remains
read-only with no native state collector, root enrollment, transport submission,
durable jobs or mutation executor. Slice 1 eligibility/system/provider/topology/
feature/device-use and complete-impact collection gates remain pending, as do
full M2 qualification, physical/commercial evidence and the maintainer's deferred
manual M1 accessibility gate.
