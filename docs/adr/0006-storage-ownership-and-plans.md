# ADR 0006: explicit storage ownership and reviewed plans

Status: Accepted. Date: 2026-10-04. Owner: Cam (`@gonz0w`).
Authority: maintainer merge of [PR 20](https://github.com/gonz0w/horvium/pull/20),
source `753f601960529cd17596d0ffc628afe38d041c82`, on 2026-10-04.
See the approved [M2 design](../milestones/M2/EDD.md); native gates remain required.
[Requirements](../milestones/M2/PRD.md), [tests](../milestones/M2/TDD.md),
[evidence](../research/M2/findings.md).

## Context and decision

M1 inventory is observational. A visible name/provider or a ZFS property must not
silently grant write authority. Persist a root-owned registry keyed by native
pool/filesystem GUID and scope changes through root-owned local enrollment policy.
Adoption selects existing imported resources without changing their properties,
mounts, descendants or OS import policy. The helper independently resolves fresh
device identity and applies typed scope checks after application authentication.

Every mutation uses an immutable expiring helper-created plan, explicit review,
single-use submission and fresh launch validation. Names are current addresses,
not identity. Device paths, GEOM graph identifiers and user-supplied actor fields
are not authority. A creation nonce helps reconcile an acknowledgement gap but
does not replace the durable registry or permit automatic adoption.

The initial operation allowlist covers the M2 PRD's creation, limited properties,
health/scrub, redundant replacement and created-pool recovery. Creation uses
the shared source-pinned `openzfs-2.2` profile after installed/native qualification.
Adopted features are preserved. No force/wipe/rewind/upgrade or arbitrary property
channel is included. Keep the MicroServer a profile, with operator-selected mirror
or RAIDZ2 layouts and tested configurable budgets.

## Alternatives and consequences

Automatic adoption by name is easy but confuses renamed/replaced resources and
unrelated operator configuration. ZFS-only ownership markers travel with storage
and can be copied; they cannot be the authority when the data pool is absent.
An unrestricted privileged shell solves fewer validation problems than it creates.
A root web service would broaden the attack surface without a requirement.

This decision adds enrollment and a review step, but preserves existing storage
and creates a clear recovery boundary. An enrolled compromised management UID can
exercise the permitted operations; the helper does not attest a human identity.
Advisory Horvium locks do not fence external root/ZFS commands or a second host.
Require exclusive managed writes and suspend on drift; document unsupported shared
block storage rather than claim race-free native fencing.

Native evidence must qualify use checks, provider mapping, status formats, mounted
namespaces and common features before their operations are exposed. Revisit when
M3 adds identities/ACLs, M4 adds backup/key policy, or a supported new topology/
controller requires different authority. Its adoption is recorded above;
source/manual observations alone do not qualify the implementation.
