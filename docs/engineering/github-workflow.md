# GitHub development workflow

The [shared private Project](https://github.com/users/gonz0w/projects/2) is Horvium's
live queue. The [roadmap index](../roadmap/github-roadmap-index.md) contains stable
gate and milestone links. Specifications, management contracts, PRDs, EDDs,
Test Designs, ADRs and candidate-specific receipts remain authoritative in Git.
GitHub owns live task status, priority and native dependency relationships.
Issue counts and merged components do not establish NAS or release readiness.

## Central delivery and local mapping

Follow [Suite-Standards backlog delivery](https://github.com/gonz0w/suite-standards/blob/4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22/docs/backlog-delivery.md)
and [rollout](https://github.com/gonz0w/suite-standards/blob/4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22/docs/backlog-delivery-rollout.md)
at immutable `4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22`.
[Local commands, MVP queue, numeric WIP and inherited exception](agent-tasks.md)
are owned by [adoption #419](https://github.com/gonz0w/horvium/issues/419).
Central policy defines Ready/Done, INVEST refinement, flow and coherent batching.
This file keeps only Horvium's existing tracking and acceptance mapping.

Horvium currently has one repository and the private Project2. It owns management,
FreeBSD storage, console/CLI, packaging and acceptance. Independent file daemons
and hardware qualification remain separate. Preserve M0-M7 milestone identities;
first deployable release remains M1-M6. Proposals authorize no new runtime behavior.

| Project Status | Exactly one status label |
| --- | --- |
| Backlog | `status:backlog` |
| Ready | `status:ready` |
| In progress | `status:in-progress` |
| Review | `status:review` |
| Validation | `status:validation` |
| Done | `status:done` |

Blocked acceptance retains its actual board state and a named `Delivery state:
Blocked` reason/owner/next action in the issue, plus native dependencies; it still
counts as WIP. Do not revive retired status/blocked or status/needs-design labels.
Priority labels P0/P1/P2 mirror Project Priority; Phase mirrors M0-M7. Maintain
native parent/sub-issue and blocked-by links, keeping readable body links and
historical accepted evidence. Text links do not replace supported native edges.

At filing, pickup, PR review, merge and acceptance, manually reconcile issue state,
one status label, Project Status/Phase/Priority, milestone and native relationships.
Existing work-item fields may be extended in place; avoid recreating trackers for
format alone. No tracking/status/merge automation is added.

Use short-lived `codex/` branches from current main and signed author-DCO commits.
Run the existing required source/review commands in CONTRIBUTING; native gates
still need their actual disposable both-release/client/browser/recovery evidence.
Use Refs for partial/qualification work. A component merge does not accept the
parent or milestone; closing requires every scoped criterion and explicit entire-
phase acceptance. Independent source review does not substitute for held checks.
Agents leave PRs open for the maintainer; current Actions-disabled/local-review
and protection requirements are preserved. No publication may trigger a held lane.

## Recoverable migration and reconciliation

The migration uses stable `<!-- horvium-roadmap:KEY -->` issue-body markers and
checkpointed key-to-issue mappings. Preserve existing issue numbers and original
text, milestone numbers and useful labels. Snapshot issues, milestones, labels,
PR evidence and Project settings privately before writes; checkpoint each returned
ID before the next mutation. An unknown create outcome requires searching markers
and re-reading GitHub before retrying. Never blindly recreate issues or Projects.
Do not rerun an initial desired-status snapshot after maintainers have changed
live state; reconcile deliberately against current GitHub instead.

Inventory all pages, compare markers/counts, confirm one status per issue, check
closure evidence, milestone dates/states, native parents/blockers and absence of
cycles, and read back every Project item's Status/Phase/Priority, saved view filters
and grouping, linked repositories and private visibility. Existing Git evidence
is not rewritten. The [migration record](../roadmap/github-migration.md) records
what was observed; it is a dated receipt, not a second task-status board.

Never post credentials, private keys, host inventories, customer data or exploitable
security details in ordinary issues. Follow [SECURITY.md](../../SECURITY.md) for a
private reporting route. Tracking authorization does not authorize publication,
installation, services, deployment or changes to live storage.
