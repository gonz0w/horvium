# Original execution-domain system backing

Status: Implementation prerequisite; native qualification pending; runtime denied.
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
the final phase, Close exclusion, preserved policy/data bytes, and full teardown.
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
