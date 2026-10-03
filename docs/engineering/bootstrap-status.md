# Bootstrap status

Date: 2026-10-03. Phase: M0 foundation configured; awaiting maintainer review.
No production implementation is approved by this status file.

## Confirmed direction

Horvium (`horvium`), BSD-3-Clause, FreeBSD/ZFS, hardware-neutral dedicated NAS,
16 GB MicroServer as one reference profile, Go/HTML5/CSS/HTMX, BSD Classic UI,
local identities initially, backup-peer awareness/replication/restore in release
one, FreeBSD port first and image later. AD, fleet management, and failover are
deferred. Private repository: [gonz0w/horvium](https://github.com/gonz0w/horvium).
Cam (`@gonz0w`) is the initial maintainer and CODEOWNERS owner.

## Implemented and observed

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

- Maintainer reviews/merges the foundation evidence PR and adopts the process.
- M1 research is tracked in [issue 1](https://github.com/gonz0w/horvium/issues/1).
  Approve its PRD/EDD/Test Design before product implementation.
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
