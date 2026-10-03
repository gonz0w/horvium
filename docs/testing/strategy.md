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
