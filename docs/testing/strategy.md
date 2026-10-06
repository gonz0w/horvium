# Testing strategy

Test-first behavior changes where practical: failing test, smallest correct code,
refactor. Bugs gain regression tests. Tests assert operator-visible contracts and
failure behavior rather than duplicating implementation details.

| Level | Evidence |
|---|---|
| T0 Repository/static | Formatting, configs, links, dependency/security policy |
| T1 Unit | Validation, parsers, ACL/policy logic, job transitions |
| T2 Component | Handlers, helper contracts, jobs/retries, config validation |
| T3 Native FreeBSD | Actual ZFS, service management, identities, disk discovery |
| T4 Client compatibility | Real SMB/NFS/SFTP clients; permission/metadata/locking |
| T5 End-to-end | Discover -> configure -> share -> snapshot -> replicate -> restore |
| T6 Failure/recovery | Restart, disk replacement, peer loss, full pool, interrupted jobs |
| T7 Upgrade/rollback | Package/state migrations, failed upgrade, preserved data/config |
| T8 Performance | Management budgets, inventory scale, file throughput, background load |

Requirement IDs such as `PROTECT-REQ-001` map to stable test IDs such as
`PROTECT-RESUME-001`. Every test record names source commit, suite revision,
FreeBSD/packages/clients, topology, procedure, result, and limitations.

## Core acceptance behavior

File serving survives management restart. Unauthorized actions fail at both
management and privilege boundaries. Invalid config leaves a usable prior state;
partial apply is surfaced/recovered. Replication resumes correctly, refuses an
unexpected peer identity, handles a full receiver, and does not grant unrestricted
backup-history deletion. Restore verifies file content, ownership, ACLs, and
metadata for supported clients; GUI job success is not restore proof.

Native tests use allowlisted disposable pools/devices. Capability checks skip
unsupported fixtures with an explicit result; skips are never passes. VM tests
prove functionality, not physical controller resilience or actual hardware speed.
Client/OS compatibility claims use [evidence levels](../compatibility/matrix.md).

## Choose checks by change

During development use a named regression, then broaden only when
new failures or unresolved concerns justify it. This loop is not PR qualification:

```sh
make test-focus PKG=./internal/storage TEST='^TestOwnershipRecordRoundTripAndCanonicalBytes$'
```

Every PR still runs `make check lint test secrets`, source/title/signature/DCO
checks and supervised teardown. `make review` adds race detection and all
compiler-listed bounded fuzz targets in changed packages and their transitive
host/FreeBSD/test consumers. Unknown/shared tooling, module, policy and packaging
changes fall back to the complete portable suite. Documentation-only changes
need the baseline; documentation followups in a behavioral PR may reuse valid
local deep evidence only when every verification input and environment matches.

Use `make review-full` for **fresh complete portable** milestone/release validation;
it never reuses deep evidence. Milestone acceptance also runs applicable native,
client, recovery, performance, accessibility and upgrade procedures from the
approved Test Design. Portable full mode does not start those fixtures.

Storage, privilege and recovery behavior still requires its affected native suite
on both declared FreeBSD releases before review. UI/client behavior requires the
relevant browser/client procedures. Unrelated native suites can wait for milestone
acceptance unless shared contracts or the Test Design require them sooner. A
maintainer may always request a broader run. Prior manifests retain their source
and executed cases; reused evidence never becomes a new native execution.

See [selection, evidence reuse and teardown](../engineering/local-review.md)
and [native execution](local-qualification.md).
