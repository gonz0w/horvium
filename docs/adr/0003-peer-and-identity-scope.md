# ADR 0003: Backup peers and local identities in release one

Status: Accepted. Date: 2026-10-03. Authority: maintainer project discussion.

## Context and decision

Release one can register/recognize another NAS and use it as a backup destination.
It includes the agreed replication and restore outcomes. Local users/groups are
the initial identity scope. AD is not required for release one. Fleet management
and automatic failover follow later designs.

## Consequences

Define peer identity/trust, receive capabilities, constrained credentials,
retention, interrupted-transfer recovery, and restoration. A peer does not imply
cluster membership, shared writable storage, automatic takeover, or management
of the destination's entire host. Discovery must not confer trust.

Alternatives considered: backup deferred, mandatory directory integration, and
initial clustering. Revisit when operators demonstrate a requirement for AD,
fleet operations, or failover and the recovery/security prerequisites are tested.
