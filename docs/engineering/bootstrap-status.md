# Bootstrap status

Date: 2026-10-03. Phase: M0 foundation in progress; no product implementation.

## Confirmed in the project discussion

Horvium (`horvium`), BSD-3-Clause, FreeBSD/ZFS, standard-hardware design with the
16 GB MicroServer as a reference, dedicated file serving, Go/HTML5/CSS/HTMX,
BSD Classic UI, local identities first, backup-peer awareness/replication in the
first release, FreeBSD port first and image later. AD, fleet management, and
failover are deferred. The Git repository uses the selected project name. The maintainer confirmed
private `gonz0w/horvium`; CODEOWNERS names `@gonz0w`.

## Prepared

Charter, architecture/UI baselines, accepted scope ADRs, roadmap, compatibility
and lab models, operational topics, engineering/security policies, document
templates, and versioned CI/ruleset definitions. Foundation tooling/pins are
adapted from BSD Cloud with attribution.

## Validation

Local full-tree checks, ShellCheck, actionlint, nine foundation regression tests,
and Gitleaks passed on macOS arm64 with Python 3.14.7. No remote CI, native FreeBSD runtime,
physical hardware, product, port/package, or restoration evidence is claimed.

## Remaining gates

- Create the confirmed private remote and record actual settings.
- Confirm a confidential security/conduct channel before public/deployable release.
- Verify signed bootstrap history before publishing.
- Apply remote protections and observe successful required CI checks.
- Review foundation policies; approve M1 PRD/EDD/Test Design after research.
- Pin runtime/client versions, define measurable budgets, and arrange native labs.

Tool installation supports Linux/macOS development hosts. Native FreeBSD smoke
validates shell/hooks only. A prospective port is not official Ports inclusion.
