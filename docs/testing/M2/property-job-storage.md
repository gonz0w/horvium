# M2 native property-job metadata store

Date: 2026-10-06. Status: implementing; native qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

## Native contract

`StateDirectory.ReadPropertyJob(id)` returns the complete bound review/lifecycle
from a canonical `job-<ID>.json` through the existing root-owned held lease.
Validate effective kernel UID, selector, directory/lock/opened file identity,
ACL/mode/owner/link/type, bounded bytes, complete codec/checksum and filename ID.
Any error returns zero metadata. Getters return independent owned copies; typed
reads never project away the baseline or fabricate one from a legacy record.
Platforms other than FreeBSD return `ErrUnavailable` without filesystem work.

`PublishPropertyJob(job, expectedRevision)` uses the unchanged shared native
[publication transaction](job-publication.md). Zero expected revision requires
absence and one initial queued event. Updates read the exact current durable
property-job predecessor and require one valid appended event, unchanged job
selection/history and every reviewed baseline/time/policy/session binding.
Check the current predecessor before staging and again after staging sync/close;
preserve exact bytes/inode, then strict full metadata through final verification.
Root actors are not fenced by an advisory lock or ordinary drift checks.

Exclusively create the private same-directory `.job-<nonce>.tmp`, validate its
opened identity/ACL, write and sync/close, revalidate objects and complete namespace/
capacity, rename to the canonical destination, then sync the directory. Known
pre-rename failures clean only the verified owned temporary and sync cleanup.
Failed cleanup or final sync returns `ErrStateUncertain` and blocks every publisher
on the held lease. Visible bytes never imply acknowledged durability; no rollback
or automatic retry exists. Reading/inspection remain available. Reopening alone
does not establish recovery or quiescence.

`InspectState` accepts both exact complete job formats by independently strict
decoders, exposing the existing bounded lifecycle summaries. Record bytes include
the entire baseline. Both formats share 64 nonterminal/1,000 terminal slots,
64 MiB ownership/job/transient bytes and submission-ID collision checks. Reserve
old plus new bytes before staging and rename; nothing is evicted to fit.
Reconcile-required remains nonterminal. Inspection grants no admission, reservation
release, replay or pruning authority.

Legacy `ReadJob` still refuses bound property-job files; `ReadPropertyJob` refuses
legacy files. Both publishers require their exact typed current predecessor.
Neither can overwrite the other format, even with matching ID/revision. Creating
any job under an existing ID fails. No migration/reset or baseline reconstruction
exists; unknown/newer/corrupt/orphan/unsafe entries preserve bytes and block the
whole namespace. Ownership publication now coexists with both job formats through
the same complete census. The ownership-only inspector stays strict.

These APIs publish/read metadata only. They do not authenticate application
sessions, claim volatile plans, reserve resources, establish independent native
eligibility, run ZFS changes or acknowledge an admitted job. No runtime, helper,
HTTP/CLI or startup caller consumes them.

## Qualification procedure

Require **555 named state assertions**, **fifty parent tests** and **1,804 distinct
codec/model subtests** on both declared FreeBSD targets with no skips/failures.
The new property-job parent contributes 88 assertions: complete canonical/private
bytes, independent/size/ID reads, legacy coexistence and conversion refusal,
immutable baseline/digest/capture/history predecessors, shared bytes/class/submission
budgets, terminal metadata, old descriptors/lock preservation, each publication
phase, shared uncertainty, late unknown and same-revision independently valid
baseline drift, duplicate serialization, unsafe/damaged records and full teardown.

Thirty-one controls extend existing POSIX/NFSv4 ACL fixtures: directory/lock/
inheritance/record denial, unrelated ownership/legacy ACL refusal for publication,
preserved bytes/ACLs, restored reads/publication, actual native NFSv4 byte access
by kernel UID 65534 with inherited descriptors, and product read/publication denial
despite that grant. Four controls extend the existing memory-backed UFS fixture:
actual property-job publication, independently observed ENOSPC, native read-only
refusal and preserved complete record bytes. No operator storage is used.

Portable failing-first tests require the new APIs and mixed catalog; checker
regressions refuse old 432-case evidence, missing/duplicate/failed/skipped cases,
every required parent and each schema family. Existing shared pipeline controls
and portable property codecs remain mandatory. Reduced budgets prove accounting,
not populated commercial resource/scale acceptance. Phase injection qualifies
ordering/uncertainty, not real device EIO or power-loss durability.

```sh
make test-focus PKG=./internal/storage TEST='^Test(PropertyStateCatalog|PropertyJobStorageUnavailable)$'
PR_TITLE='feat(storage): persist reviewed property jobs on FreeBSD' make review
python3.14 tools/local-qualification.py state 15.1 --run
python3.14 tools/local-qualification.py state 14.5 --run
make lab-status
```

Native qualification is pending. Exact source/input/toolchain/image/runtime/log
provenance and full backing/base/runtime/group teardown must be recorded. Prior
manifests retain their actual tested sources; they are not relabeled. Native
storage eligibility/operations, full executable attribution, actual admission/
reservations, retention/tombstones, recovery/restoration/acknowledgement cuts,
M2 acceptance and deferred M1 manual accessibility remain open.
