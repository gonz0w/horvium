# GitHub development workflow

The [shared private Project](https://github.com/users/gonz0w/projects/2) is Horvium's
live queue. The [roadmap index](../roadmap/github-roadmap-index.md) contains stable
gate and milestone links. Specifications, management contracts, PRDs, EDDs,
Test Designs, ADRs and candidate-specific receipts remain authoritative in Git.
GitHub owns live task status, priority and native dependency relationships.
Issue counts and merged components do not establish NAS or release readiness.

## Ownership and work items

Horvium currently has one repository, `gonz0w/horvium`. It owns unprivileged Go
management, the narrow privileged FreeBSD boundary, HTML5/CSS/HTMX console,
CLI, native ZFS integration, packaging and acceptance. Keep file daemons independent
of management. The MicroServer is a lab profile; hardware and commercial scale
support require actual evidence. Do not manufacture additional repositories.
If related repositories are established later, use this same private Project,
matching capability milestones, and one issue in the repository that owns the
outcome. Link cross-repository sub-issues to the coordinating Horvium phase gate.

Use one bounded issue per outcome. Search existing open and closed trackers and
PRs before creating one. Preserve useful labels and accepted evidence. Use the
work-item form; triage assigns type, priority, milestone, parent and blockers.
Broad future outcomes remain backlog until their approved design supports
reviewed implementation slices. Planning issues do not authorize new runtime
behavior or dependencies. M0–M7 are Horvium's existing capability phases with no
promised delivery dates; first-release acceptance spans M1–M6 and the package-first
image is M7. Deferred second-release security proposals do not invent M8 or a tag.

Every triaged issue uses these six sections:

1. Outcome: concrete resulting behavior in the owning repository.
2. Acceptance: observable checklist criteria and evidence needed for closure.
3. Design and requirements: stable IDs and authoritative Git document links.
4. Relationships: parent gate and blocker links, also represented natively.
5. Existing evidence: merged PRs and exact candidate receipts, failures and skips.
6. Tracking scope: boundaries, limitations and remaining work.

Native sub-issue relationships represent ownership/acceptance decomposition.
Native blocked-by relationships represent prerequisites; text links supplement
rather than replace them. Do not close a coordinating gate because a child merged.
Avoid duplicate work across repositories and dependency cycles.

## Status and priority

| Project Status | Exactly one issue status label | Meaning |
|---|---|---|
| Backlog | `status:backlog` | Planned outcome; design or decomposition may remain |
| Ready | `status:ready` | Bounded next action and its dependencies are clear |
| In progress | `status:in-progress` | Active implementation, research or qualification |
| Review | `status:review` | Concrete PR or decision awaiting maintainer review |
| Validation | `status:validation` | Relevant acceptance evidence remains outstanding |
| Done | `status:done` | Scoped criteria met, evidence linked and issue closed |

Use `type:feature`, `type:research`, `type:acceptance`, `type:tracking` or
`type:documentation`. Phase gates use `type:tracking`; qualification tasks use
`type:acceptance`. Preserve useful area, bug, accessibility and historical kind
labels. Retired `status/blocked`, `status/needs-design` and `status/needs-research`
are not workflow state; triage removes them from migrated items while keeping
label definitions available for history. Blocking is a native relationship.

Priority is `priority:P0` for urgent security/release blockers, `priority:P1` for
current phase work and `priority:P2` for later planned work. Project Priority
mirrors that label; Phase mirrors the milestone's M0–M7 prefix. No automatic
import, label synchronization or merge-to-Done automation is assumed.

## Issue-driven pickup and PR review

For a continue request, inspect open PRs and unblocked Ready work first. Select a
coherent highest-priority action within existing authorization, state its issue,
read repository instructions and the applicable design/evidence, and split broad
features into reviewed slices before implementation. M1 manual accessibility
remains pending by maintainer choice; M2 private prerequisites do not enable a
runtime writer. Independent research may proceed while delivery gates are blocked.

Size slices around a complete behavior or integration outcome. Keep its internal
helpers/adapters, tests, docs, compatibility notes and source-bound evidence in
one PR, using incremental signed commits. Internal implementation steps belong
in the owning issue's checklist; they do not each need another issue/merge cycle.
Separate PRs are appropriate for an unresolved design/authorization decision,
distinct safety boundary or a diff too large to review coherently. State that
reason before splitting. Native execution that the adopted Test Design requires
after integration review still follows that review; batching cannot waive it.
Use focused tests while developing and complete the required clean candidate
review before submission. Shared-tool fallback, milestone/release full checks
and relevant native/client safety gates retain their existing policies.

Branch from current merged `main` using a short-lived `codex/` branch for agents.
Use Conventional Commit subjects and `git commit -S -s`; verify signatures and
author DCO trailers. Preserve contributor sign-offs in canonical squash history.
Every change has a PR. Use `Refs gonz0w/horvium#number` for an owning issue or broader
gate, including cross-repository qualifiers. Use `Closes` only when merging that
PR meets every criterion of that specific issue. Acceptance gates normally use
Refs until all native, integration, browser, resource and recovery criteria pass.
Agents leave PRs open; maintainer review and existing protections remain required.

Run `make check lint test secrets` and the signed clean-candidate `make review`.
Use `make review-full` for milestones/releases and the existing conservative
fallback policy. Tracking-only changes need no new NAS qualification; they do not
upgrade old receipts. Relevant behavior changes still require disposable native
fixtures on both declared FreeBSD targets and applicable client/browser/recovery
gates. Identify exact source/base/title, selected scope, reused original evidence,
failed/skipped cases and limits. Verify supervised teardown with `make lab-status`
and `make lab-stop` where needed. Never test on operator disks or backup history.

## Manual triage and acceptance

At filing, pickup, PR creation, merge and acceptance review, manually reconcile:

- One workflow label, Project Status and open/closed issue state.
- Milestone, Project Phase, priority label and Project Priority.
- Native parent/sub-issue and blocked-by links, with readable body links.
- Current PR and source-bound evidence; retained failed/skipped limitations.
- New issues added to the shared Project and filtered views still appropriate.

Merge closes only fully satisfied bounded issues. Otherwise keep the issue open
in Validation, Review or the appropriate active state. Reopening changes Done to
the actual open workflow state in both labels and Project. Close a phase gate and
repository milestone only after explicit maintainer acceptance of the complete
advertised scope. M0's original foundation acceptance is historical; later tooling
items attached there do not replace its original receipts.

The All work view groups by Phase. Current phase is currently M2 with Status
columns and Repository grouping; update its filter deliberately when the active
phase changes. Ready next shows open Ready items; Acceptance gates includes
tracking and acceptance issues; Completed evidence shows closed scoped work.
No paid automation, new Actions workflows, external synchronization or runtime
dependencies are introduced. Actions remain disabled; local results require review.

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

## Portable method and local mapping

[PROJECT.md](../../PROJECT.md) composes the pinned
[Agent Temper core](../../.agent-temper/METHOD.md) with this local workflow.
No optional profile is selected. Keep the existing status/priority/phase mapping,
manual Project synchronization and native parent/blocker relationships. Imported
example labels never rename local states or create a second queue. Record missing
policy questions in the owning issue and persist the owner's answer in project
policy or its canonical decision record before dependent work.
