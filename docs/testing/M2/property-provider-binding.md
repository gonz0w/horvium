# M2 native pool-provider binding

Date: 2026-10-06. Status: implementing; native qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [issue 19](https://github.com/gonz0w/horvium/issues/19).

`PoolProviderSource` binds every leaf of a complete supported native pool topology
to an independent direct DISK provider, unique correlated serial, geometry and
qualified DEV/disk-ident attachment graph. Names are current native addresses,
never enrolled identities. Nested partition/label addresses, missing or duplicate
identities, foreign serial/geometry, repeated disks, unsupported attachments and
uncertain graphs return no provider results. Root/system exclusions remain the
separate complete system-use collection. No controller locality, physical-sector
inference, shared SAN or enrollment capability follows from serial correspondence.

Each of two independent passes reads fixed `freebsd-version -kru`, paired
`zpool status -g -p` / `zpool status -L -P -p`, and
`sysctl -b kern.geom.confxml`. Supported topology must exactly match the earlier
property observation. Every selected address must resolve to one direct uniquely
identified DISK in the whole graph before fixed `/usr/sbin/diskinfo /dev/name`
and `diskinfo -s /dev/name` reads. The native dispatcher independently validates
the narrow address syntax; no shell, arbitrary path or property input is added.
The accepted disk correlation and attachment parsers check the entire graph,
including duplicate identities on unselected devices. Every leaf is required;
failed or truncated projections discard the whole result.

Normalized provider observations and exact release lines must match across the
two passes. Even changed transient GEOM IDs conservatively refuse reuse. The
result retains serial provenance, native media/logical-sector and stripe fields;
zero stripe observations remain zero and imply no physical-sector proof. Returned
objects come from fresh parses. No persistent record or client contract changes.

The collection shares 64 MiB combined stdout/stderr and 100,000 raw lines across
both passes. A separate 64 MiB aggregate parse-byte and 100,000 GEOM-ID allowance,
and the default 250,000 XML-element allowance, are divided across both passes'
one initial graph parse plus two graph parses per selected leaf. Existing field,
XML depth and topology bounds remain. Conservative exhaustion denies the result,
never truncates the member set. Commands have ten seconds and collection 45
seconds. These are initial capability ceilings, not commercial performance claims.

`PropertySafetySource` now requires provider collection between its private
policy/property binding and complete system-use collection. A final canonical
root policy read covers all three stages; missing or changed policy discards the
result. The composed context is 135 seconds, with separate bounded budgets for
its three collectors. Every failure returns zero policy/observation. Success
still keeps `Eligible=false`: actual root device enrollment, store/recovery
validation, atomic admission, authorized transport and execution remain pending.
Repeated sequential views do not make an atomic snapshot or fence competing root
actors; future admission must revalidate independently. There is no runtime writer.

## Qualification procedure

Failing-first portable tests require every member, fresh independent reads and
copies, safe pre-I/O selection, duplicate/missing identity refusal, serial/geometry
mismatch, partition and unsupported attachment refusal, exact topology binding,
valid normalized geometry drift, failures, accounting, limits and cancellation.
A separate regression fails without the composed provider gate. Bounded fuzzing
covers hostile GEOM/diskinfo/serial/status projections.

The focused native fixture retains the previous 32 assertions and adds five:
`providers_correlated`, `provider_geometry`, `providers_duplicate_denied`,
`providers_missing_denied`, `provider_fixtures_teardown`. The existing uniquely
identified mirror must bind both its 512-byte and 4096-byte logical sectors.
Two additional separately owned pools use only the already declared unused
snapshot fixtures: a duplicate-serial mirror and a missing-serial single disk.
Independent native property collection succeeds; provider collection must deny
both with zero results. Each pool has GUID-checked cleanup registered before
creation and is destroyed before the success record. No root/existing pools,
operator data, service or host disk is modified. All 37 exact assertions, actual
native test PASS, binary checksum and complete existing guest/host/device teardown
are mandatory on both releases. Historical manifests retain their original source.

```sh
make test-focus PKG=./internal/host/freebsd TEST='^Test(PoolProviders|PropertySafety)'
PR_TITLE='feat(storage): bind native pool providers for property review' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

The evidence checker change requires one complete portable deep review. The
unrelated metadata publication suite remains outside this focused native run.
GitHub Actions stay disabled; no persistent runner or service is installed.
