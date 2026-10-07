# M2 explicit property member enrollment

Date: 2026-10-07. Status: both native targets and complete portable review passed; accepted in [PR 65](https://github.com/gonz0w/horvium/pull/65).
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [issue 19](https://github.com/gonz0w/horvium/issues/19).

The complete property safety source now requires an explicit root-recorded member
baseline for every supported native leaf. A scope-only policy does not prove
member enrollment. Observing a pool, reading a marker or encoding metadata never
grants ownership or permission. No enrollment command, provisioner, adoption UI,
native submission backend or executor is introduced.

## Record and compatibility contract

`EncodeOwnershipRecord` retains the exact version 1 wire format/checksum when
members are absent. Nonempty members select version 2: the same `version`, `kind`,
`policy`, `checksum` envelope and existing policy `revision`, `pool_guid`,
`pool_name`, `scopes`, plus required nonempty `members`. Each member has all seven
required fields: `vdev_guid`, `serial`, `serial_source`, `media_size`,
`sector_size`, `stripe_size`, `stripe_offset`. Both schemas reject unknown,
duplicate, case-variant, missing or null fields, wrong types and unknown versions.
Dispatch uses two fixed-depth strict schema walks, never arbitrary recursive JSON.
Errors return zero policy; no guessed migration or partial member set is returned.

Members sort by canonical nonzero decimal vdev GUID; GUIDs and serials are unique.
Serials are nonempty valid UTF-8, have no control characters or edge whitespace,
and retain exact `geom.config.ident` or `provider.config.ident` provenance.
Media/logical-sector sizes must be positive, within native numeric ranges and
exactly divisible. Stripe size/offset retain bounded native observations, including
zero. These are logical geometry facts, not proof of physical sector size, local
controller ownership or suitability for shared SAN. No hardware count or sector
size is hardcoded. Device paths and transient GEOM IDs are not persisted.

All records remain at most 64 KiB. Conservative shared scope/member field budgets
are checked before copying/sorting; the existing field and count ceilings remain.
Exhaustion refuses the entire record. Version 2 hashes compact canonical
`version/kind/policy` content with the `horvium:ownership:v2` domain followed by
newline; checksums detect damage and do not authenticate root metadata.
Canonical member facts also bind the existing policy hash, so same-revision
member drift invalidates volatile plans and persisted property intents. The
property-intent wire format is unchanged.

Version 1 remains readable by descriptor readers and catalogs, with no invented
members. It cannot satisfy complete property safety. No automatic upgrade occurs;
an authorized future root enrollment workflow must establish actual member facts
and publish a new revision. Explicit typed policies without members encode legacy
records; the publisher still enforces revision/namespace/capacity/backing rules.
Older code refuses version 2 without rewriting it. There is no unattended downgrade
or member removal, and a file's checksum is not enrollment authority.

## Native safety composition

After root policy/property binding, the source denies a missing member baseline.
Fresh independent provider collection must match every recorded GUID, unique
serial with its exact provenance, media/logical-sector and stripe fields. No
partial/extra/reused path or unsupported provider can substitute for a member.
The existing system exclusions and later complete provider correspondence remain
required. A final canonical root policy reread covers all stages, including
members; every failure returns zero policy and observation. Success still has
`Eligible=false`. Sequential observations are not atomic and do not fence another
root actor. Enrollment provisioning, store/recovery eligibility, fresh atomic
admission, authorization, transport and execution remain separate pending gates.

## Qualification procedure

Failing-first tests cover the new codec and safety gate. Behavioral coverage checks
canonical ordering and input isolation, exact legacy bytes, strict schema and
checksummed semantic refusals, limits, every member's identity/provenance/geometry,
legacy/partial/extra/duplicate sets and final member-policy drift. A persisted
intent roundtrip must retain the member policy hash and reject subsequent drift.
The existing ownership fuzz target includes a version 2 seed.

The focused disposable fixture retains all 37 previous native cases and adds nine:
`enrollment_bound`, `enrollment_legacy_denied`, `enrollment_member_drift_denied`,
`enrollment_newer_denied`, `enrollment_restored`, `enrollment_catalog`,
`enrollment_legacy_read`, `enrollment_publication`,
`enrollment_publication_preserved`. Fixture setup explicitly records independent
observations of its declared 512/4096 logical-sector mirror; this test setup is not
a product enrollment workflow. Version 1/2 root files must read and count correctly.
Kernel `statfs` checks the actual state backing: accepted ZFS/UFS publication must
roundtrip version 2, while softupdates UFS must refuse before changing the record.
A sanitized backing/result line distinguishes the executed paths. No mount/tunefs
settings change. Existing GUID-checked pool/swap/private-file cleanup removes every
owned fixture; guest/host/device/key/image/process teardown remains mandatory.

```sh
make test-focus PKG=./internal/storage TEST='^Test(PropertyEnrollment|PropertyMembers)'
make test-focus PKG=./internal/host/freebsd TEST='^TestPropertySafety'
PR_TITLE='feat(storage): require explicit property member enrollment' make review
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

The shared evidence checker requires one complete portable review. Native
qualification is limited to these 46 focused cases: unchanged metadata file
engines and their historical crash/capacity/ACL evidence keep their original
sources. GitHub Actions stay disabled; no runner or host service is installed.
No throughput, physical hardware, populated commercial scale or M2 acceptance
claim follows.

The initial complete portable review refused the new tests because their broad
ownership names also matched the historical state-suite selector. Enrollment-specific
test names keep its exact inventory unchanged; the tests still run in ordinary
portable/race review. The earlier focused native run passed all 46 cases, but
remains evidence only for its original source. Qualify the corrected source afresh.

## Executed qualification

Signed source `b95f60c873020f632674a2302acb1a4a3a6ca646`, based on PR 64 merge
`759c7de96ed03afa633b688b3d1f462a8e40f5b6`, passed all 46 exact assertions on
FreeBSD 15.1-RELEASE-p3 and 14.5-RELEASE with the same checksum-bound executable.
Kernel statfs observed actual ZFS state backing/publication success on 15.1 and
UFS with soft updates/publication denial on 14.5. Both retained expected record
bytes. Guest workloads including setup/teardown took 84.813/70.110 seconds;
actual tests took 52.38/38.25 seconds. These are Mac/QEMU fixture measurements,
not NAS throughput or hardware qualification. See the
[sanitized execution manifest](property-member-enrollment-manifest.json).

Complete portable review passed `make check lint test secrets`, 112 tooling tests,
all 11 race packages and all 35 compiler-listed bounded fuzz targets in 186.500
seconds. Version 1 golden bytes and all 40 historical M2 manifests remain unchanged.
Native metadata file engines also remain byte-identical; the previous 777-case
state/crash/capacity evidence is historical and was not rerun or relabeled.
All owned pools/mounts/swap/md/private records and guest/image/key/variables/
executable/device fixtures were removed. Base/snapshot/runtime inputs were
preserved, and `make lab-status` verified all 332 owned sessions stopped at this
checkpoint. No Actions, runner or host service was enabled.

The initial failed portable review and earlier successful native source remain
recorded separately. Documentation followups rerun the full cheap baseline and
may reuse only exact successful deep inputs. Enrollment provisioning/adoption,
store/recovery eligibility, runtime admission/execution, operator workflows and
M2 acceptance remain pending. Successful member matching grants no write authority.
