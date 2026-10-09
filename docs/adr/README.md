# Architecture decisions

Accepted records below capture explicit maintainer decisions from 2026-10-03.
They approve product direction, not unspecified implementations.

- [0001 FreeBSD and hardware-neutral scope](0001-freebsd-and-hardware-scope.md)
- [0002 Port-first distribution and license](0002-distribution-and-license.md)
- [0003 First-release peer and identity scope](0003-peer-and-identity-scope.md)

M1 implementation decisions accepted by maintainer merge of
[PR 3](https://github.com/gonz0w/horvium/pull/3):

- [0004 Native discovery boundary](0004-native-discovery-boundary.md)
- [0005 M1 access and state](0005-m1-access-and-state.md)

New durable implementation choices use the [ADR template](../templates/ADR_TEMPLATE.md).

M2 decisions accepted by maintainer merge of
[PR 20](https://github.com/gonz0w/horvium/pull/20) on 2026-10-04:

- [0006 Storage ownership and reviewed plans](0006-storage-ownership-and-plans.md)
- [0007 Durable storage job records](0007-durable-storage-jobs.md)

M2 runtime composition accepted by maintainer merge of
[PR 105](https://github.com/gonz0w/horvium/pull/105) on 2026-10-08:

- [0008 Property runtime recovery and launch gates](0008-property-runtime-recovery-gates.md)

Proposed mechanism decision following the accepted PR 106 research:

- [0009 Property executor supervision and recovery provenance](0009-property-executor-recovery-mechanism.md)
