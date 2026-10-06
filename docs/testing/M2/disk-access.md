# M2 aggregate disk access

Date: 2026-10-05. Status: implementing; portable behavior tested; native
qualification and maintainer review pending. [Approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [issue 19](https://github.com/gonz0w/horvium/issues/19).

## Observation contract

`inventory.ParseDiskAccess` independently correlates successful complete GEOM,
diskinfo and diskinfo serial captures to one uniquely identified direct disk.
It retains the provider's canonical `r<read>w<write>e<exclusive>` aggregate
access counts, each a nonnegative native signed-int value (at most 2147483647).
Missing/duplicate/negative/overflow/noncanonical/nested modes refuse a zero result.
Every immediate attached consumer must supply recognized counts; their sum must
match the provider without overflow. Counts include access propagated by aliases
and other layers. They do not attribute processes, classify every attachment or
represent an exact number of userspace descriptors. Unrelated provider modes are
outside this selected projection; the complete bounded graph still validates.

`storage.ReviewDeviceUse` now requires this projection from its same GEOM and
independent disk captures. Access counters are value-owned facts; fresh changes
invalidate the immutable review even when no mount, swap or pool changed.
Unavailable/contradictory modes fail the whole review, with no partial observation.
Transient GEOM IDs and ordering remain excluded from semantic drift comparison.
The M1 inventory contract and runtime collector remain unchanged.

These are positive local observations only. Zero counters cannot prove absent
signatures, partitions, exported pools, remote-host use, eligibility or enrollment.
The reads are not atomic; input syntax cannot prove successful execution,
completeness, freshness or provenance. Observation does not reserve the disk and
cannot fence another root writer. There is no product I/O, new command, helper,
HTTP/CLI route, transport/persisted schema or mutation authorization.

## Bounds and portable tests

Aggregate input bytes and scalar/XML fields retain the inventory bounds. Within
the access parser, independent disk/graph parsing and a second bounded XML pass
share disjoint conservative record/element quotas; independent geometry/serial
and selected provider/consumer modes consume their quotas. The combined review
allocates a separate stage quota for access and retains its aggregate byte cap.
No parser borrows unused quota or truncates output to fit.

Tests failed first because the access parser/types and review field were absent.
Behavioral tests cover identity binding, provider/consumer equality, zero/max-int,
missing/duplicate/negative/overflow/unknown modes, zero error results and bounds.
Combined tests prove raw access and close drift without mounts, and missing or
contradictory mode refusal. `FuzzDiskAccess` is the twentieth bounded fuzz target.
Evidence regression failed first because missing/failed/duplicate/invented access
markers were accepted; the checker now requires all nine new cases.

## Disposable native procedure

Extend the existing `TestNativeSystemDeviceUse` on the independently verified
owned 64 MiB HVM-UNIQUE-512 disk. Require exact whole-disk disk-ident alias and
initial zero access. Hash all five owned guest disks before/after. Hold an actual
direct read-only descriptor, then alias read-only and alias read/write descriptors.
No writes are issued through these handles. Collect successful full native inputs
while each is held; require propagated read/write counts and baseline drift.
Recollect while held to require unchanged validation. Return value copies must
not mutate the baseline. Close all owned descriptors in reverse order, require
zero restoration and close drift, preserve disk bytes and independently verify
root mount/GUID/backing. Unsupported-mode controls alter capture copies only.

Close every handle on failure with test cleanup; the complete disposable VM is
always discarded. No operator disk/path can be supplied. After handle tests, the
existing UFS mount, swap, imported/exported pool and metadata fixtures still run.
All sixty-eight assertions in all four executed native tests plus complete host
backing/base/runtime preservation and disk/image/key/variable/process teardown
must pass. No skip, partial assertion set or old fifty-nine-case result qualifies.

```sh
python3.14 tools/local-qualification.py devices 15.1 --run
python3.14 tools/local-qualification.py devices 14.5 --run
make lab-status
```

## Source provenance and limitations

Both pinned FreeBSD releases' GEOM dump/dev/subr, slice and LABEL sources were inspected.
The producer emits provider/consumer access counters under the topology lock;
`g_access` updates both. DEV open/close applies read/write deltas. Native DEV
`O_EXCL` does not request GEOM exclusive access in these sources; do not treat
exclusive counters as userspace exclusive-open enforcement or a product lock.
The slice layer adds an exclusive count while an alias is open; LABEL accesses
propagate through that layer; native tests must establish
actual direct-disk effects. Executed binary hashes and versions are required;
source inspection alone is not installed binary provenance.

- [15.1 producer](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/geom/geom_dump.c)
- [15.1 DEV access](https://github.com/freebsd/freebsd-src/blob/e38a7085f3a8ecd317c947591fe42b5ca9ab317b/sys/geom/geom_dev.c)
- [14.5 producer](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/geom/geom_dump.c)
- [14.5 DEV access](https://github.com/freebsd/freebsd-src/blob/231eee334c098a8398bc959202415c48058f67f0/sys/geom/geom_dev.c)

Physical devices/controllers, multipath/shared-host safety, full signature absence,
reservation/enrollment and last-moment execution checks remain unqualified.
M2-DEVICE-001 and slice 1 remain incomplete. M1 manual Safari/accessibility remains
pending by maintainer choice. GitHub Actions remain disabled; all work runs locally
on the Mac with supervised disposable QEMU and mandatory teardown.
