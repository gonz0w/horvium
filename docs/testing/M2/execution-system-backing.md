# Original execution-domain system backing

Status: Accepted observation prerequisite (PR 116); runtime denied.
Date: 2026-10-10. Owner: Cam (`@gonz0w`).
[Issue 19](https://github.com/gonz0w/horvium/issues/19),
[accepted mechanism](../../milestones/M2/property-recovery-mechanism.md),
[joint original owner](joint-execution-leases.md),
[previous system capability](property-system-exclusions.md).

PR 115 accepted joint observation ownership at merged source
`3497481a1aa2ed557811e0111e4567b456633fdc`. The stricter private native bridge
adds complete system/owned-pool backing checks inside that same owner. It selects
only the accepted guarded persistent backing and complete system-use producers.
Candidate paths and installation remain trusted internal metadata inputs; there
is no runtime consumer, new configuration, CLI route, service or commissioning.

## Contract

`ObserveSystem` keeps the existing 90-second total context and journal-before-
ledger mutex order. Both original protected-object observers surround the entire
operation. After the first complete ledger census, it collects the complete
system-use view, checks both actual descriptor mount projections and persistent
backing identities, repeats native domain observations, and collects the system
view again. Both complete views must match. The final exact-byte ledger census
and journal snapshot follow every callback; changes made during the last system
phase therefore cannot escape namespace/record drift checks.

Both domain mounts must match source, target and type in the fresh complete view,
have resolved direct physical backing, use writable supported UFS/ZFS without
softupdates, and satisfy the accepted conservative system target grammar (`/`,
`/boot`, `/etc`, `/usr`, `/var`, `/root`, or beneath non-root system targets).
Persistent UFS identity must match the live mount ID under the already accepted
restricted UFS profile. ZFS must also match the independently enumerated pool
GUID and name. Dataset identity still comes from the guarded descriptor producer.

Every ownership GUID in the complete validated ledger contributes to exclusion.
Owned imported pools cannot contain either domain or share its physical backing.
An owned pool missing from imported enumeration has unresolved physical backing
and denies this path. This deliberately refuses exported/offline ownership rather
than assuming it is disjoint. Later support for such environments requires an
accepted independently qualified resolution mechanism. The full collector still
requires all mount/swap/imported-pool captures; shared system disks and swap are
not themselves forbidden, since metadata may share a boot disk.

Ordinary `Observe` retains its previous semantics. Successful strict observations
carry only private distinguishing metadata; no boolean or view grants launch,
publication or recovery permission. Nil system callbacks deny before borrowing.
Probe errors, invalid views, object drift or census changes after inspection begins
retire the original pair; repair cannot reset that latch. Copied/closed owners and
nonroot callers retain the existing denials. Trusted callbacks cannot retain
borrowed descriptors or re-enter the leases. Only the fixed host bridge supplies
native completeness; synthetic inputs prove the storage transfer contract alone.

Complete system collection retains its accepted aggregate bounds: 64 MiB, 100,000
lines/records, 45 seconds per collection and ten seconds per command. Domain reads
retain their 64 KiB and 45-second limits. Two system collections run sequentially
inside the unchanged pair deadline; large or busy environments may conservatively
deny. Equal observations are not atomic snapshots or a kernel fence. Root edits,
joint rollback, intervening changes, memory resume and async kernel work remain
outside automatic clearance. Restore independence, registered writer ownership,
fresh commissioning, acknowledged transitions and startup/global assembly remain
required. No job history, pending state or hold is cleared.

## Tests and gates

Failing-first portable tests require positive UFS, ZFS and mixed pairs and deny
unsafe journal or ledger projections, missing/foreign mount correspondence,
unsupported/read-only/softupdates backing, physical owned-pool overlap, unresolved
or invalid ownership and foreign persistent identity/pool facts. The native bridge
starts as a denial stub so actual positive assertions must fail before connection.

The focused `execution-system` suite retains all four original/joint parents and
147 assertions. Two new parents exercise direct declared UFS and ZFS metadata
fixtures under `/root`, an independently declared enrolled data mirror, complete
native system collection, both strict/ordinary observations, missing ownership,
permanent retirement, errors in either system phase, same-inode record changes in
the final phase, actual unrelated data-mount drift, Close exclusion, preserved policy/data bytes, and full teardown.
ZFS additionally denies actual metadata-pool enrollment and separately moves each
owned domain to an ordinary data mount. The existing five snapshot disk fixture
validates provenance, geometry and root/mount/swap exclusion before mutation.
The fixture's duplicate serial is declared lab backing only, never product
creation identity. Each new parent owns an eight-minute context. Complete
filesystem/directory/data restoration and host device preservation/removal are
required; skipped, repeated or partial evidence denies.

Run both pinned QEMU guests serially, then both affected unchanged `state` safety
corpora. Shared runner/checker changes require one fresh full portable review;
evidence-only followups rerun baseline and reuse exact deep inputs. Historical
manifests and unrelated source inputs stay byte-identical. GitHub Actions remain
disabled. No host services or operator storage are used.

## Executed observations

The [manifest](execution-system-backing-manifest.json) binds final implementation
and qualification inputs to signed source
`c9a58694138366a184e21ada438799e6d11381b3`. Both pinned FreeBSD/amd64 releases
passed all six parents and 200 identity/system assertions: 147 retained original/
joint cases, 25 new UFS and 28 new ZFS cases. Twelve retained fixture teardown
controls, six new domain/data restoration controls and the separate host snapshot
disk preservation/removal gate passed. The focused suites executed source
`228c553ffec314fc0df9dbf6fa1bca17bda4749c`; the sole subsequent code change
adds failure diagnostics to a storage test. Recompiling the host test package at
the final implementation source produced the identical checksum-bound binary.
Actual timings and image/compiler pins are recorded.
These small disposable fixtures do not qualify production scale or performance.

Both affected state corpora passed all 77 parents, 1,075 required
native assertions and 1,930 codec children at the final implementation source,
with matching executable hashes and verified teardown. Full portable review passed
check, lint, test and secrets, 160 tooling tests, eleven race packages and all 53
bounded fuzz targets. Evidence-only followups reuse only those exact deep inputs
while rerunning baseline checks at the final PR head.

The initial portable checker stub failed the UFS positive before implementation.
The first native bridge-stub attempt also exposed a fixture cleanup assumption
about mount-table whitespace; its guest fixture results were incomplete. The
corrected fixture then failed both native positive assertions at the denial stub
without cleanup errors. The first connected native run passed UFS and retained
lease checks but reached ancestor refusal when its ZFS ordinary-mount negative
fixture used `/tmp`. A secure, owned non-system mount directory fixed that test
setup. All failed records retain their actual sources and remain unqualified;
their host processes, disks, keys, images and compiled artifacts were removed.
No partial attempt is counted as a complete native gate.

All 89 historical M2 manifests and 350 other original Go inputs retained their
exact bytes. Only the joint observation type, FreeBSD owner and off-target owner
were extended. One original storage test adds sanitized field-level failure
diagnostics while retaining its preservation assertion; original host test
sources remain unchanged. Every owned session stopped, and each runner removed its compiled executable, guest fixture,
process group, disposable image and key. All five declared host snapshot disk
backings remained byte-identical and were removed after each guest stopped.
GitHub Actions remain disabled; no persistent host service was installed.

An initial FreeBSD 15.1 full state run failed the historical publication-cut
`after_directory_sync_objects_preserved` comparison. A focused repetition then
failed `after_return_retry_safe` after 56 passing parents. The old comparisons
provided no field diagnostic for those failures. Sanitized diagnostics now cover
both exact snapshot comparisons; their assertions and runtime code are unchanged.
All 300 subsequent focused repetitions passed, but that partial run was deliberately
refused by the complete-corpus verifier and cannot qualify the state suite. Passing
full reruns are recorded separately. The cause remains unresolved: these results
do not establish that either earlier failure was benign or fixed. Publication
preservation remains an open investigation for release acceptance under issue 19.
No automatic retry, metadata relaxation or passing-result rewrite was added.

The focused gates establish strict system/owned-pool observation for the declared
supported subset. Offline/exported ownership remains a denial. Independent restoration,
fresh commissioning, registered writer ownership, acknowledged transitions and
startup/global launch assembly remain required; runtime storage writes stay denied.
M1 manual Safari/VoiceOver acceptance remains pending separately.


The subsequent [publication snapshot correction](publication-preservation.md)
reproduced an allocation-count-only preservation failure and qualified its test
correction on both native targets. The historical manifest above retains the
original open-investigation record and failed attempts unchanged. No missing
individual field difference is invented for the earlier uninstrumented attempts.
This followup changes no production observer or storage-write authority.
