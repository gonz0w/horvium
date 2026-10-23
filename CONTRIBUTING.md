# Contributing

Follow the authoritative [Suite-Standards delivery agreement](https://github.com/gonz0w/suite-standards/blob/4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22/docs/backlog-delivery.md)
and [rollout](https://github.com/gonz0w/suite-standards/blob/4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22/docs/backlog-delivery-rollout.md)
at immutable revision `4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22`.
[Adoption #419](https://github.com/gonz0w/horvium/issues/419) owns this prospective
consumer change; the existing advisory profile manifest remains independently pinned.
Use [AGENTS.md](AGENTS.md), the [local command/WIP index](docs/engineering/agent-tasks.md),
[GitHub workflow mapping](docs/engineering/github-workflow.md) and relevant approved
milestone packet for Horvium-specific constraints. Central Ready/Done and coherent
batching apply; local native/security/merge gates remain mandatory.

## Development and review

Use short-lived topic branches from `main`. Add a failing test, implement the
smallest correct change, then refactor with passing tests. Bug fixes add regression
coverage. Keep a PR focused on one outcome and update its docs alongside code.
Include related adapters, tests, compatibility notes, runbook and source-bound
evidence in that PR. Use incremental signed commits for internal prerequisites.
Split delivery only for unresolved design/authorization, a distinct safety boundary
or scope too large to review coherently; explain the reason in the owning issue.

Run `make tools` and `make hooks`. After signing/signing off the changes, run
`PR_TITLE='type: actual proposed PR title' make review` from a clean checkout.
It verifies commit signatures/DCO/title and runs `make check lint test secrets`
plus race/fuzz checks for affected packages and their consumers. Use
`make review-full` for fresh complete portable milestone/release validation; see
the [local review gate](docs/engineering/local-review.md).
Report the tested source/base/title and results in the PR. GitHub Actions are
disabled: maintainers review the evidence before merging. Native evidence still
needs both declared release candidates and disposable fixtures; see
[local qualification and teardown](docs/testing/local-qualification.md).
Use `make lab-status` and `make lab-stop` to verify/stop owned development sessions.
Native FreeBSD, client, recovery and release gates remain separate. Developer
tests may use VMs; Horvium's runtime is a dedicated file server.

Use Conventional-Commit-style titles and DCO trailers (`git commit -s`). Protected
history and release tags require signatures. See [Git workflow](docs/engineering/git-workflow.md).
Agents leave PRs open for the maintainer. Independent approval becomes required
when a distinct reviewer joins; the solo maintainer cannot approve their own PR.

A PR explains the problem, resulting behavior, meaningful validation, compatibility,
security implications, and migration/rollback where relevant. Do not include
credentials or identifiable operator data in test evidence.

## Dependencies and documentation

Apply the [dependency policy](docs/engineering/dependency-policy.md). Documents
follow the [lifecycle](docs/process/document-lifecycle.md). Use stable requirement
and test IDs so acceptance evidence can be traced back to the design.

## Issue references and acceptance

Use `Refs gonz0w/horvium#number` for broader gates. Use `Closes` only if the PR
satisfies every criterion of that bounded issue. A merged component does not
accept an entire milestone. Link candidate receipts and preserve failed/skipped
limits. Manually reconcile issue labels/state, Project fields and dependencies
at pickup, PR review, merge and acceptance; tracking adds no automation.
