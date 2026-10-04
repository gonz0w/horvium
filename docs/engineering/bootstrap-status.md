# Bootstrap status

Date: 2026-10-04. Phase: M0 accepted; M1 design approved and implementing.
No production implementation is approved by this status file.

## Confirmed direction

Horvium (`horvium`), BSD-3-Clause, FreeBSD/ZFS, hardware-neutral dedicated NAS,
16 GB MicroServer as one reference profile, Go/HTML5/CSS/HTMX, BSD Classic UI,
local identities initially, backup-peer awareness/replication/restore in release
one, FreeBSD port first and image later. AD, fleet management, and failover are
deferred. Private repository: [gonz0w/horvium](https://github.com/gonz0w/horvium).
Cam (`@gonz0w`) is the initial maintainer and CODEOWNERS owner.

## Implemented and observed

- Maintainer merged [administration security PR 7](https://github.com/gonz0w/horvium/pull/7),
  source `b71b9915653f9b75785ca350991d0730d5ddb4a0`, on 2026-10-04.
  The next slice adds authenticated console/CLI and foreground development commands;
  see [console contracts/evidence](../testing/M1/console.md).


- Maintainer merged [collector/cache PR 6](https://github.com/gonz0w/horvium/pull/6),
  source `65c8457980127f9a04909a550af3f160a613cef0`, on 2026-10-04.
  Administration security is now accepted; see
  [security contracts/evidence](../testing/M1/security.md).

- Maintainer merged [helper transport PR 5](https://github.com/gonz0w/horvium/pull/5),
  source `60d73215279fd33a1a2facbb66dd198319d19bc2`, on 2026-10-03.
  Collector/cache libraries and native outage/recovery are now accepted;
  see [collector evidence](../testing/M1/collector.md).

- Maintainer merged [native discovery PR 4](https://github.com/gonz0w/horvium/pull/4),
  source `4a21aa72b4a2bb516018bb06f21dbcc287a4ab0f`, on 2026-10-03.
  The kernel-authorized helper library is now accepted; see
  [implementation/evidence](../milestones/M1/implementation.md).

- Maintainer merged [M1 design PR 3](https://github.com/gonz0w/horvium/pull/3),
  source `527a330f4b1069fa0b7a1414f4d3c6dbab54f5a5`, on 2026-10-03. Its PRD,
  EDD, Test Design and ADRs 0004/0005 authorize the scoped M1 implementation.
  Component completion and evidence are tracked in
  [M1 implementation status](../milestones/M1/implementation.md).
- Maintainer merged [foundation review PR 2](https://github.com/gonz0w/horvium/pull/2)
  on 2026-10-03. Squash revision `02f7dcb43afdeedd101e94e41ea06e24335422fe`
  has a GitHub-verified signature and a successful
  [main foundation run](https://github.com/gonz0w/horvium/actions/runs/37161247071).
- Charter, architecture/UI baselines, scope ADRs, roadmap, compatibility/lab models,
  operational topics, engineering/security policies, and design templates.
- BSD Cloud foundation tooling/pins adapted with upstream attribution.
- Two SSH-signed, author-DCO-signed-off bootstrap commits, verified locally;
  GitHub verified the initial branch head's signature.
- Full-tree content/config/link checks, ShellCheck, actionlint, nine foundation
  regression tests, and Gitleaks passed locally on macOS arm64/Python 3.14.7.
- Initial [GitHub foundation run](https://github.com/gonz0w/horvium/actions/runs/37159718556)
  passed on source `dc0630973095f51ef5b5508b92ecae83e5a9a993`.
- [FreeBSD 15.1 amd64 foundation smoke](https://github.com/gonz0w/horvium/actions/runs/37159774954)
  passed on that same source. This proves shell/hook behavior only.
- Active main ruleset requires PRs, signed history, linear history, resolved
  conversations, and repo-hygiene/pr-title/dco. Force pushes and deletion are
  blocked; no bypass actors. Solo phase has zero independent required approvals.
- Active `v*` tag rules forbid moving/deleting release tags. Signing tags remains
  a release verification requirement, not a claim of hosted signature enforcement.
- Squash-only merges, merged-branch cleanup, read-only Actions defaults, workflow
  review approval disabled, immutable Action SHAs, and checksum-pinned dev tools.
- Dependabot vulnerability alerts and automatic security fixes enabled; 22 issue
  labels and M0/M1 milestones configured. Workflow-based dependency updates exist.

## Remaining gates and limits

- M1 research is tracked in [issue 1](https://github.com/gonz0w/horvium/issues/1).
  Complete the approved [PRD](../milestones/M1/PRD.md), [EDD](../milestones/M1/EDD.md),
  and [Test Design](../milestones/M1/TDD.md) acceptance gates. The
  maintainer chose macOS and Linux/BSD client priorities and QEMU native fixtures.
- Confirm confidential security/conduct reporting before public community access
  or a deployable release. BSD Cloud's reporting address is not assumed.
- GitHub security-and-analysis capability metadata was unavailable for this
  private repository. Repository-native secret scanning/push protection and
  paid security features are not claimed; Gitleaks runs locally and in CI.
- Native NAS operations, protocol clients, physical controllers/disks, commercial
  scale, restore, upgrade, packages, images, SBOM/provenance, and production support
  require their later milestone evidence. No installable port has been published.

Python/linters/scanners are development-only. Tool installation supports Linux
and macOS; native FreeBSD validation has separate gates. See the
[lab model](../testing/lab-gate.md) and [compatibility matrix](../compatibility/matrix.md).
