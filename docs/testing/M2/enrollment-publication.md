# M2 reviewed enrollment metadata publication

Date: 2026-10-07. Status: implementation; qualification pending.
Authority: [approved EDD](../../milestones/M2/EDD.md),
[Test Design](../../milestones/M2/TDD.md), [ADR 0006](../../adr/0006-storage-ownership-and-plans.md),
[ADR 0007](../../adr/0007-durable-storage-jobs.md),
[issue 19](https://github.com/gonz0w/horvium/issues/19).

`PropertyEnrollmentState.Publish` adds explicit one-attempt root ownership-record
publication to the accepted sealed review/binding. It changes metadata only;
no ZFS command, management consumer, operator CLI/HTTP transport, restored-state
clearance or native write eligibility is added. The proposal, predecessor and
original lease come from the binding, not publication arguments. Publication
arguments are independently fresh complete trusted native inputs, never a browser
assertion. A context token authenticates no root actor; actual kernel effective
UID 0 is required again by the native publisher.

## Contract

A plan owns a shared volatile claim. Plan/binding copies and rebinding the same
plan retain that same claim; they cannot mint a second attempt. Before claiming,
validate context, actual clock, root-context binding and the full proposed/native/
provider/system baseline. Waiting for the claim mutex must not bypass cancellation
or expiry. Immediately before invoking publication, consume the claim. Every
publication attempt, including failure, is consumed. There is no replay or stored
acknowledgement: reconcile actual metadata and explicitly review a new proposal.
The claim creates no background work or growing lease-wide claim registry.

Hold the original root lease/mutex across complete metadata reinspection, staging,
verification and publication. Repeat the accepted complete census, exact canonical
predecessor and entire byte/name/backing digest, proposed-owner system exclusion,
projected whole-temporary capacity and retained job holds. Only the transaction's
independently inode-verified temporary may be excluded from subsequent censuses.
Compare all metadata again after staging and immediately before rename; unrelated
records/jobs are part of the baseline. Unknown/malformed/newer/unsafe/full state,
closed/changed leases, mismatched predecessors or orphan uncertainty deny.
Distinct reviews of the same predecessor serialize; only one can publish.

Repeat actual clock/context checks after waiting for the root mutex, before/after
each full census, predecessor reads, final verification and rename. The accepted
file sync, close, object checks, descriptor-relative rename, published inode/byte
verification and directory sync engine remains unchanged. Preserve cancellation/
expiry and store denial reasons when pre-rename cleanup succeeds. Cleanup or
post-rename uncertainty wins, latches the original lease and never returns success
or removes/rolls back a visible destination. Successful directory sync returns
its actual acknowledgement even if cancellation occurs afterward; it cannot
encourage replay. Other root mutations remain unfenced; filesystem calls/mutex
waits are not preempted by request cancellation.

Ownership versions 1/2 and job schemas remain unchanged. Explicit first enrollment
requires verified absence/revision one; successor or legacy adoption requires the
exact predecessor and next revision. No automatic adoption, inferred members,
reservation release, eviction, durable claim format or restoration validation is
added. Review/Validate APIs stay read-only. A metadata publication is not permission
for an unprivileged management process to launch native work. Runtime root/session
authorization, final native/recovery eligibility, full transport/backend/executor,
restart/reboot/power-cut and M2 acceptance remain required.

## Qualification

Portable failing-first tests cover concurrent copies, preflight denial without
claiming, consumed failed/successful attempts, actual late acknowledgement and
non-FreeBSD refusal without filesystem work. The native metadata suite retains
all historical 777 assertions/66 parents/1,876 codec children and adds 41 controls
in `TestNativeEnrollmentPublication`: actual first/successor/explicit legacy
publication, canonical private records, old inode/stable lock preservation, copied/
rebound/distinct-review claims, exact/unrelated policy drift, queued/terminal holds,
unknown/orphan state, whole-temporary capacity, late staging and before-rename
drift, cancellation/expiry/backwards clocks, file-sync/rename/cleanup/directory-sync
failures, visible uncertainty and write blocks, closed/zero bindings, actual
nonroot kernel UID denial and fixture teardown. Require all 818 assertions, 67
parents, unchanged 1,876 children and complete backing/guest/process/host teardown
on both releases. Re-run the historical safety suite because the census was
refactored for caller-held publication inspection. Do not relabel prior evidence.

That suite uses synthetic trusted pool/provider/system fixture inputs and actual
native metadata filesystems. FreeBSD 15.1 uses root ZFS; 14.5 uses its existing
owned non-softupdates UFS memory filesystem under the system namespace. These
results do not establish runtime MD provider eligibility or full native adoption.
The focused pool suite separately retains all 84 assertions and adds four actual
native first/successor/repeat-denial/preservation controls. Root ZFS must publish
reviews built with independently collected real mirror/provider/system inputs;
softupdates UFS must return zero-review refusal, zero-binding publication refusal
and absent records. Its repeat control is continued zero-binding denial, not a
successful claim/publication. Require all 88 cases on both releases.

```sh
make test-focus PKG=./internal/storage TEST='^TestEnrollmentPublication'
PR_TITLE='feat(storage): publish reviewed enrollment metadata' make review
.tools/venv/bin/python3 tools/local-qualification.py state 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py state 14.5 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 15.1 --run
.tools/venv/bin/python3 tools/local-qualification.py pool-properties 14.5 --run
make lab-status
```

Shared checker/selector changes require complete portable review. All 45 historical
manifests and the shared descriptor-relative publication engine must remain
byte-identical. No Actions, persistent runner or host service is installed.
