# M2 native disk correlation

Date: 2026-10-05. Status: all twelve native assertions and complete host teardown passed on both
FreeBSD targets; accepted in [PR 31](https://github.com/gonz0w/horvium/pull/31) at
merged source `ef60c33489a2ddd30c99e2a8e865a76272473f2b`.
[Approved EDD](../../milestones/M2/EDD.md), [Test Design](../../milestones/M2/TDD.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Read-only contract

`inventory.ParseDiskObservation` correlates one complete successful `diskinfo`
projection and its separate `diskinfo -s` serial capture with the complete,
fresh `kern.geom.confxml` capture. It performs no I/O and has no helper, HTTP,
CLI, policy or runtime collector wiring. The three byte inputs share one aggregate
byte budget; UTF-8, fields, delimiters and graph structure/records are bounded
before projection. Errors contain fixed diagnostics and return a zero observation.

The numeric output is exactly one newline-terminated tab row with six columns,
or nine when optional firmware geometry is reported. Each unsigned decimal is
canonical, without signs, units, padding or overflow. Sector size is a positive
native uint32, capacity a positive signed-off_t-compatible value, and byte
capacity, exact sector divisibility and sector count must agree. Firmware columns
are syntax/type checked and never used as layout or alignment policy.

An exact `/dev/<single-component-provider>` address must resolve once in the
whole graph to a direct DISK provider without backing parents. Its serial must
be present, unique in that supplied collection, consistent across GEOM sources,
and exactly match the independently captured serial. Duplicate/missing/conflicting
serials, lunid-only identity, logical/PART/MD/multipath providers, parents,
ambiguous addresses and serial/capacity/sector disagreement fail correlation.
Known GEOM serial/lunid provenance is preserved in the returned provider.
Lunid observation is not independent lunid qualification.

The returned sector count, stripe size and stripe offset retain the literal
native values. A zero stripe is not replaced by logical sector size or presented
as physical sector size. The inspected utility emits zero when stripe ioctls fail
as well as for native zero values, so even a correlated row cannot establish
physical alignment. Sector policy remains a separate reviewed gate.

Callers must supply the complete successful source captures. Syntactically valid
XML cannot prove that a caller did not remove a provider or that independent
captures are atomic. IDs/addresses are transient; collection uniqueness is not
durable enrollment, hardware identity, local-only provenance or multi-host fencing.
Incoming consumers are not interpreted as absent use. In particular a DISK with
partition children can correlate without being unused. No result grants write
eligibility: alias collapsing/overlap, boot/swap/mount/signature/pool use,
reservations, fresh drift and physical/locality proof remain open.

## Tests and native procedure

Failing behavioral tests preceded implementation. Tests cover six/nine columns,
literal sector/stripe variants, zero stripe preservation, conflicting/absent/duplicate
serials, provenance and direct backing, repeated addresses, malformed/oversized
rows, aggregate/field/graph limits, UTF-8, canonical numbers and geometry drift.
`FuzzDiskObservation` adds a thirteenth bounded target to the local review gate;
failures must expose no partial observation.

The [owned five-disk fixture](devices.md), accepted in
[PR 28](https://github.com/gonz0w/horvium/pull/28) at merged source
`27dafaba7a5d04a4896f27776ef79f4b1f4c2bbc`, now feeds actual full GEOM/serial/geometry
captures into the parser. Independent expected observations remain in the test.
Both distinct serial/sector variants must correlate; both duplicate providers
and the missing serial must be refused. Changing a native geometry projection
must fail. Eight original fixture assertions plus four exact correlation
assertions, the executed native test and host preservation/teardown are mandatory.

```sh
python3.14 tools/local-qualification.py devices 15.1 --run
python3.14 tools/local-qualification.py devices 14.5 --run
make lab-status
```

Use clean signed source and run guests sequentially on the Mac. All disks are
new fixture-owned 64 MiB snapshot virtio backings; no operator devices are used.
Source `031dbe715c0c0d6d94b92c19689f4fd78123c23b` passed all twelve assertions on FreeBSD
15.1-RELEASE-p3 and 14.5-RELEASE through Mac arm64/QEMU 11.1.1 TCG (two CPUs,
6144 MiB). Both runs preserved the complete raw backings, base image and
QEMU/firmware and removed all disks/image/key/variables with verified owned-group
teardown. The [sanitized execution manifest](disk-correlation-manifest.json)
records exact input hashes, actual versions/utility/image/runtime hashes and
outcomes. Final PR-head portable evidence is recorded separately in the PR;
fixture input changes require fresh qualification. Older eight-assertion manifests
retain their original tested source and cannot qualify this parser. No complete
M2-DEVICE-001, product mutation, physical controller or production claim follows.

## Primary-source provenance

The inspected `diskinfo.c` is byte-identical at the two retained release revisions,
SHA-256 `0dd9050ee91a0cd0c8a7a286bc3b078002438c5a5d9817c5a2eb695a6b4ab27a`.
[14.5 source](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/usr.sbin/diskinfo/diskinfo.c),
[15.1 source](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/usr.sbin/diskinfo/diskinfo.c).
The [original source hash record](devices-manifest.json) remains unchanged;
source inspection does not prove which source built an installed binary.
Actual utility/version/image/runtime hashes must be recorded with native results.
