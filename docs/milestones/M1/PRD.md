# PRD: native discovery and read-only console

Status: Approved; implementing in reviewed slices. Owner: Cam (`@gonz0w`). Date: 2026-10-03.
Approval: maintainer merged [design PR 3](https://github.com/gonz0w/horvium/pull/3)
on 2026-10-03, source `527a330f4b1069fa0b7a1414f4d3c6dbab54f5a5`.
Work: [issue 1](https://github.com/gonz0w/horvium/issues/1).
Design: [EDD](EDD.md). Acceptance: [Test Design](TDD.md).

## Problem and outcome

An operator needs an accurate view of a FreeBSD storage host before entrusting
Horvium with changes. M1 establishes discovery, a usable console, and the security
boundary on which later storage/sharing/protection work depends. The same build
must handle the small reference server and larger inventories without assuming
four bays, 16 GB memory, one pool, or a particular controller.

The operator starts a development instance explicitly, authenticates through a
loopback connection or SSH tunnel, sees host/pool/dataset/provider inventory, and
can refresh and inspect it. An operator can inspect the same information through
a CLI using the same application contracts. Existing ZFS resources are observed;
Horvium does not adopt or change them in M1.

M1 is an engineering milestone, not a deployable NAS release. There are no pool,
dataset, user, share, service, host-network, or backup mutations. SMB/NFS/SFTP,
backup pairing, restoration, package lifecycle, and remote HTTPS administration
remain required later milestones. The console must state this boundary plainly.

## Requirements and acceptance

| Requirement | Observable behavior | Acceptance tests |
|---|---|---|
| M1-REQ-001 Native inventory | Report actual OS/kernel/userland versions, architecture, memory, GEOM providers, pool/dataset GUIDs and numeric capacity; distinguish logical providers from physical disks | M1-NATIVE-001, M1-PARSE-001 |
| M1-REQ-002 Honest observation | Every section carries collection time/status; missing capability, timeout, stale/partial data, missing/duplicate disk identity and externally changed resources remain visible | M1-FAIL-001, M1-IDENTITY-001 |
| M1-REQ-003 Read-only boundary | Management runs unprivileged; native helper permits only fixed inventory operations and kernel-authorized callers; HTTP never executes shell/native commands | M1-PRIV-001, M1-NOCHANGE-001 |
| M1-REQ-004 Authenticated administration | Bootstrap token, bounded sessions, login throttling, CSRF and Host/Origin checks protect every inventory view and refresh; no public listener in M1 | M1-AUTH-001, M1-HTTP-001 |
| M1-REQ-005 Console and CLI | Same inventory/generation/error semantics in HTML and CLI; raw HTML5/CSS, local pinned HTMX; UI works without JavaScript | M1-CONTRACT-001, M1-UI-001 |
| M1-REQ-006 Bounded scale | Shared cached collection, pagination, deduplicated refresh, explicit resource limits and incomplete-state reporting; no process per dataset/request | M1-SCALE-001, M1-PERF-001 |
| M1-REQ-007 Independent operation | Management restart/helper outage cannot change existing pools/config or stop file-serving daemons; cache/session restart behavior is documented | M1-RESTART-001, M1-NOCHANGE-001 |
| M1-REQ-008 Reproducible delivery | Exact source/dependency/native evidence, meaningful test-first changes, no unpublished-port claim, and an explicit clean install/uninstall plan | M1-BUILD-001 |

## Compatibility and budgets

Approved runtime targets: supported FreeBSD 15.1 primary and 14.5 compatibility,
amd64 first. These are dated targets; refresh at approval and implementation.
Go 1.27.1, HTMX 2.0.11 and the proposed syscall library are pinned in the
[research manifest](../../research/M1/source-manifest.json). Other architectures
require native qualification. Hardware compatibility follows FreeBSD and evidence.

Browser tests prioritize Safari/macOS and Firefox/Linux, with exact versions
recorded when tests run. Linux/BSD CLI/native tests accompany macOS development.
Later file-client testing prioritizes macOS and Linux/BSD: macOS SMB/SFTP, Unix
SMB/NFS/SFTP. No Windows compatibility or Time Machine claim is made from M1.

Proposed budgets, measured on declared fixtures rather than presented as results:

- Small inventory: four provider fixtures, one pool, 100 datasets; combined Go
  management/helper RSS at most 128 MiB after warm-up; idle CPU below 1% of one
  core over five minutes. This excludes OS/ZFS ARC and file-service memory.
- At 10,000 synthetic datasets: cached page latency p95 below 250 ms for ten
  concurrent authenticated readers; combined RSS at most 256 MiB. Synthetic data
  measures application behavior, not physical storage performance.
- Default page size 100, maximum 500. Default collection interval 30 seconds,
  one collection at a time, ten-second deadline per native command. Dataset count,
  output bytes and cache limits are configurable validated safety budgets, not
  hardcoded commercial support limits. Exhaustion reports incomplete inventory.
- WCAG 2.2 AA is the UI target: keyboard navigation, visible focus, status text,
  labels, light/dark and reduced-motion behavior. Manual tests accompany automated
  checks; meeting selected checks alone is not certification.

## Decisions and remaining gates

The approved [native boundary ADR](../../adr/0004-native-discovery-boundary.md)
and [M1 access/state ADR](../../adr/0005-m1-access-and-state.md) are supported by the
[research findings](../../research/M1/findings.md). No unresolved hardware or
client preference blocks this draft: QEMU is authorized, and client priorities
are recorded above. See [implementation status](implementation.md) for actual
completed components and remaining acceptance gates.

Approval adopts the M1 scope and design only. Release-wide password/identity/ACL
rules, durable jobs, adoption/drift resolution, encryption/key recovery, backup
trust/retention and performance qualification remain separate designs before
their implementation. The approval record above identifies maintainer/date/source revision.
