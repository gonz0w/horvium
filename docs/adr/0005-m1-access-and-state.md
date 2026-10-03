# ADR 0005: bounded local administration for M1

Status: Proposed. Date: 2026-10-03. Owner: Cam (`@gonz0w`).
Approval: pending. Design: [M1 EDD](../milestones/M1/EDD.md).

## Context and proposed decision

The first console is a read-only engineering milestone. It still needs real
authentication and hostile-web protection before displaying host inventory.
Bind to loopback, use SSH forwarding for remote lab access, authenticate a locally
generated 256-bit token, and issue bounded expiring in-memory sessions. Apply CSRF,
exact Host/Origin validation and safe rendering to all page/fragment routes.
Persist only strict startup configuration and the token digest on system storage;
observed generations/sessions remain volatile. No database is needed for M1.

## Alternatives and consequences

- Unauthenticated localhost trusts too much: remote websites can target local
  services, and loopback is not proof of an authorized administrator.
- Human passwords require KDF/account/recovery policy; introducing them now would
  prematurely settle the later identity milestone. Random token hashing is a
  distinct design and must never be reused for password storage.
- External identity/SSO would expand scope and runtime dependencies without a
  first-release requirement. AD is explicitly deferred.
- Persisting caches/sessions adds migrations and secret persistence with no M1
  recovery requirement. Restart can safely invalidate them and re-observe the host.

This is a lab access workflow, with a token prompt and SSH tunnel rather than
network onboarding. M1 does not support an externally exposed HTTP listener or
make a production authentication claim. The deployable product needs its remote
HTTPS, local-admin identities, recovery and key ownership design before release.
Inventory/read-only refresh uses the same auth rules in UI and CLI. Root can
rotate credentials; operator config and unrelated host state are preserved.

Revisit before M3 or any network-admin/package release, and if measured cache/job
requirements need durable state. Approval of this ADR does not approve a future
password/database/SSO implementation.
