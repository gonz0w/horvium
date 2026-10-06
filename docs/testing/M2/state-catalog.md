# M2 mixed state catalog and capacity census

Date: 2026-10-06. Status: implementing; native qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Contract

`StateDirectory.InspectState()` observes all ownership and job records through the
existing root-controlled writer lease. It returns independent sorted ownership/job
summaries, actual published byte totals, and nonterminal/terminal job counts.
Only the kernel root UID may inspect; other platforms return a zero catalog and
`ErrUnavailable` without filesystem access. This internal primitive has no
runtime, HTTP, CLI or startup caller.

Enumerate through a fresh descriptor with an independent offset, bound the entire
name set, and classify every name before reading. Require the stable lock and
canonical `ownership-<GUID>.json` or `job-<ID>.json` filenames. Duplicate names,
missing lock, unknown entries and malformed names return no partial catalog.
Recognized orphan ownership temporaries return `ErrStateUncertain` and remain
untouched; other unsupported files remain unknown. There is no cleanup or repair.

Every record uses the shared descriptor-relative private byte reader: root-owned
0600 single-link regular file, safe opened-object ACL, no symlinks, at most 64 KiB,
exact bounded read and successful close, with descriptor/name metadata checked
before and after. Strict version/schema/checksum/lifecycle validation precedes use;
body identity must match the selected filename. Recheck all record identities and
the held directory/lock and namespace after the scan. Return a zero catalog on
any error. These observations detect ordinary drift; they do not fence another
root actor or create a cross-file transaction.

Count actual bytes of both record kinds against 64 MiB. Retain the existing
100,000 ownership enumeration ceiling; enforce separate 64 nonterminal and
1,000 terminal job ceilings. Queued/applying/observing and reconcile-required are
nonterminal; succeeded/failed/cancelled are terminal only after full lifecycle
validation. Refuse a submission ID occurring in two job records, even if both
records independently validate. This consistency check does not admit or deduplicate
requests. No unknown, uncertain or finished record is evicted to fit. Counts are
capability ceilings, not tested commercial workload or memory promises.

Inspection leaves a previously latched publication uncertainty in place. An
operator must still reconcile it; a successful census cannot make writes safe.
The ownership-only inspector/publisher retain their accepted strict namespace and
refuse job files. Adoption of this census into mixed publication, including old
plus new temporary byte reservation, remains a later gate.

## Qualification procedure

Require **336 named assertions**, thirty-two parents (nine native and twenty-three
unit), and all **176 codec/schema subtests** on both declared FreeBSD targets.
The census adds forty-eight native controls: complete sorted independent mixed
observations; exact total bytes and class counts; reduced ownership/class/byte
bounds and actual populated 64/1,000 job ceilings plus exhaustion; conflicting
submissions; byte/inode/foreign/orphan preservation; unknown namespace;
unsafe job types, owner/mode, size, schema/checksum and body binding; damaged
ownership with a valid job; real record/namespace/directory/lock drift; anchored
rename; retained uncertainty; nil/inert/closed leases and complete teardown.

Thirteen additional assertions extend the owned UFS POSIX/NFSv4 ACL fixtures:
whole-catalog refusal for unsafe directory, lock, ownership/job record and inherited
ACLs; exact restored mixed accounting on both models; and a real UID-65534 child
that refuses inspection despite an effective native NFSv4 read grant and inherited
directory descriptor. Every earlier lease/reader/ACL/publication/capacity control
continues to execute, including actual filesystem ENOSPC and read-only denials.

Portable behavioral tests cover all seven lifecycle classifications, filename/body
binding, sorted independent selectors, strict denials, submission collisions,
exact/overflow byte and class limits. A new bounded selector fuzz target rejects
partial or unsafe results. Tests first fail for missing census APIs. The evidence
regression first proves old 275-case logs could incorrectly qualify, then the
expanded validator refuses those logs and missing/duplicate/failed/skipped cases.

```sh
make test-focus PKG=./internal/storage TEST='^TestStateCatalog'
PR_TITLE='feat(storage): inspect bounded mixed state records' make review
python3.14 tools/local-qualification.py state 15.1 --run
python3.14 tools/local-qualification.py state 14.5 --run
make lab-status
```

Shared verification changes select one complete portable review. Evidence-only
followups may reuse matching executed deep checks while rerunning required
baseline/source/title verification. Cross-compilation is not native evidence.
Traceability: partial M2-STATE-001/M2-JOB-001/M2-PRIV-001 at T1/T3. Actual fixed
record counts qualify refusal/accounting only; M2-SCALE-001 resource testing remains
separate. All native files are newly owned disposable fixtures.

## Remaining boundaries

Job publication and mixed publication capacity reservation, complete executable
intent, admission/deduplication, reservations, tombstones/retention, restoration,
executor quiescence, startup recovery and acknowledgement/power-cut gates remain
open. No native storage write, enrollment, service deployment, release or
physical/commercial/client support claim follows. Preserve earlier manifests at
their actual executed sources. Deferred M1 accessibility and M2 acceptance remain
pending.
