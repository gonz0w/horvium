# ADR 0004: bounded native discovery behind a Unix socket

Status: Accepted. Date: 2026-10-03. Owner: Cam (`@gonz0w`).
Authority: maintainer merged [PR 3](https://github.com/gonz0w/horvium/pull/3),
source `527a330f4b1069fa0b7a1414f4d3c6dbab54f5a5`.
Design: [M1 EDD](../milestones/M1/EDD.md).

## Context and proposed decision

The web process needs native storage observations without general root authority.
Use an unprivileged Go service and a separate narrowly privileged inventory helper,
communicating through a protected filesystem Unix socket. Authenticate kernel
peer UID and ABI version, independently validate typed allowlisted operations,
and bound work/output. M1 exposes only read operations. Future mutation operations
need their own design and denial tests; they do not inherit permission from M1.

Use base sysctl/GEOM XML and explicit literal TSV ZFS projections across selected
FreeBSD releases. Go's standard XML/JSON libraries suffice for parsing; propose
the focused BSD-licensed x/sys/unix credential wrapper rather than handwritten
unsafe ABI layouts. No runtime cgo adapter is required by this proposal.

## Alternatives and consequences

- Root web process simplifies calls but exposes excessive privilege to HTTP bugs.
- A sudo/doas command wrapper can constrain commands, but provides a weaker typed
  contract and complicates request authorization and bounded result handling.
- Direct libgeom/cgo offers the native library API, with extra build/ABI coupling;
  XML from the same kernel source keeps the Go build simpler.
- JSON-only ZFS discovery has richer encoding but is unavailable across older
  bundled ZFS versions. Narrow TSV excludes unsafe free-text properties until an
  encoding is natively verified.

The split adds process/socket lifecycle and native denial tests. A compromised
management process can request inventory; it cannot perform storage changes in M1.
OS root compromise is outside this boundary. Native research validates kernel
facilities, not product transport or complete hardware identity. See
[findings](../research/M1/findings.md) and [spike](../research/M1/native-spike.md).

Revisit when native tests reveal incompatible identity/encoding, when supported
architectures change, or before adding privileged mutations.
