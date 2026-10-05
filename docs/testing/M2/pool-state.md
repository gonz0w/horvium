# M2 pool identity, incarnation and policy observations

Date: 2026-10-04. Status: behavioral and pinned native subset accepted in [PR 25](https://github.com/gonz0w/horvium/pull/25),
merged source `9f6cc07d3c19f84b656e9130283b21eaa7c7ce13`.
Design: [approved EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[slice sequence](../../milestones/M2/implementation.md). Extends the
[filesystem constraints](dataset-state.md); runtime remains read-only.

## Fixed single-pool projection

```text
/sbin/zpool get -H -p -o name,property,value,source guid,load_guid,health,readonly,bootfs,altroot,cachefile,compatibility,multihost <pool>
```

`ParsePoolState` requires exactly nine nonempty, four-column tab-separated records
for the requested pool, with a final newline. Reject unknown/duplicate/missing
properties, mixed pool names, controls, invalid UTF-8, invalid source metadata,
noncanonical GUIDs, unsupported health/readonly/multihost values and failed limits.
Failure returns a zero pool state, never partial evidence. Require canonical
positive uint64 `guid` and `load_guid`. Retain native source metadata; pool sources
are none/default/local, without filesystem inheritance or received values.

`guid` is permanent pool identity; `load_guid` identifies its current import
incarnation. A new import invalidates a property review even if permanent GUID,
filesystem identities and properties are unchanged. This does not replace native
boot/process identity, exclusive-writer or final execution checks.

Observe `bootfs` as unset `-` or a filesystem name in the selected pool's existing
conservative name subset. Retain `altroot` as unset or an absolute path and
`cachefile` as default `-`, `none` or an absolute path. These strings are observations,
never import, mount or file access authorizations. Retain compatibility settings
as bounded text with nonempty comma-separated components; no profile file is read,
hashed or resolved by this model. An `off`, `legacy` or named profile setting alone
cannot prove actual active features, installed compatibility or receiver support.

Recognize ONLINE, DEGRADED, FAULTED, OFFLINE, REMOVED, UNAVAIL, SUSPENDED, SPLIT and
UNKNOWN health representations from the inspected native formatter. Only ONLINE
passes this coarse refusal check. Unknown future tokens fail parsing. A full table
may be unavailable for a faulted/missing pool; missing values never become healthy
or writable defaults. Leaf topology/error counters/scans remain unimplemented here.

`VerifyPoolState` binds native pool name/GUID to the observation used by policy
validation. Missing pool observations, readonly imports, a bootfs selector,
non-ONLINE health or multihost enabled refuse planning. A missing bootfs selector
and multihost off do not prove non-system or direct local storage. System/root/swap
backing, shared storage, device use, recognized topology and feature eligibility
still require their independent native proofs. No implemented adapter can set the
model's eligibility flag from these tables alone.

The review stores an independent scalar copy of the full pool state. Changes to
permanent identity, import incarnation, policy values or sources require a fresh
review. Input order is normalized. The pool projection is capped at 64 KiB, nine
records and 4,096 bytes per field before splitting; combined typed input accounting
remains 64 MiB. Pool metadata counts against the existing 64 KiB review cap, with
no truncation or hardware assumptions.

## Behavioral, source and native gates

Behavioral tests failed against stubs and the previous model before implementation.
They cover permanent/import identity, missing observation, boot/readonly/health/
multihost refusal, metadata/immutable copies, source/policy/incarnation drift,
strict shape/enum/number/limit denial and ordering. `FuzzPoolState` is the tenth
bounded fuzz target. These cover partial M2-OWN-001, M2-DRIFT-001, M2-PLAN-001,
M2-HEALTH-001 and M2-SCALE-001; no complete acceptance row is satisfied.

The inspected [pinned pool manual](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/man/man7/zpoolprops.7)
defines GUID/load GUID and boot/import/cache/compatibility semantics. The
[pool formatter](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/lib/libzfs/libzfs_pool.c)
adds suspended/split/unknown representations and explains missing values/sources.
The [import manual](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/contrib/openzfs/man/man8/zpool-import.8)
allows a specified fixture device/GUID and import without mounts. Source inspection
is not native qualification or a product import implementation.

The existing owned disposable fixture independently correlates all pool values
through `zpool list` and sources through single-property reads. It checks bootfs
refusal, policy-preserving reads, permanent GUID across unmounted export/reimport,
changed load GUID and review drift, readonly import refusal, and a new GUID after
recreating its destroyed owned pool under the same name. These setup/cleanup
commands use only the declared pool GUID and md provider, with no force or mounts.
Cleanup verifies the active GUID before destroy and can reclaim only its known
exported fixture; a failed cleanup fails evidence and requires discarding the guest.
No host service is installed or enabled.

The pure-model native comparison uses synthetic trusted policy/eligibility inputs
and actual native observations; it never launches a product operation or qualifies
an eligibility collector. A bootfs marker in a disposable pool does not qualify
actual guest boot/root detection. ONLINE and readonly behavior are native cases;
adverse health token mappings and multihost-on refusal are behavioral/source
coverage, not native fault/shared-storage qualification.

Eight new markers join the original 28. The evidence checker requires all 36 plus
actual native execution and teardown; skipped/empty/missing/failed cases cannot pass.
No operator inventory, disks, credentials or keys enter the evidence.

## Qualification and remaining work

Run [37247751495](https://github.com/gonz0w/horvium/actions/runs/37247751495)
passed both pinned guests at functional/suite source
`552f203501c87822c96e9b712beea962b0f43d5a`. Each executed all 36 property/identity/
filesystem-state/pool assertions and teardown. Both evidence checks passed; the
checksum-pinned base images and AnyVM runtime remained unchanged. No native
repairs were needed.

| Guest | Observed result | Tools |
|---|---|---|
| FreeBSD 15.1-RELEASE-p3 amd64 | All 36 assertions and teardown passed | ZFS userland/module 2.4.2-1-FreeBSD_g6330a45b0; Go 1.27.1 |
| FreeBSD 14.5-RELEASE amd64 | All 36 assertions and teardown passed | ZFS userland/module 2.2.9-FreeBSD_g079ba86d7; Go 1.27.1 |

The [sanitized manifest](pool-state-manifest.json) records source/job links,
individual cases, inspected primary and tested input hashes, native utility
hashes and guest/runtime pins. QEMU 8.2.2 on Ubuntu 24.04.5 used fresh snapshot
qemu64 guests with two vCPUs and 6,144 MiB, with VNC off. No operator inventory,
credentials or fixture keys are included. This qualifies these projections and
pure-model comparisons only, not an eligibility collector or product import.

Local `make check lint test secrets` and `make go-check` passed, including vet,
complete unit/race suites and ten bounded fuzz targets. FreeBSD amd64 cross-build
passed before native execution; compilation alone is not native evidence.
Subsequent evidence/documentation edits preserve the tested functional/suite bytes.
Earlier projection manifests retain their original tested source and case sets.

Full slice 1 still requires provider/
topology/status/features/system/device-use and assembled complete-impact collection;
later privilege/state/recovery/mutation/client/browser/resource gates remain open.
Manual M1 accessibility and physical/commercial qualification remain pending.

The current native gate adds six [feature/profile assertions](features.md) to
these 36 cases. The pool manifest preserves its original tested input hashes
and results; later feature checks do not retroactively qualify it.
