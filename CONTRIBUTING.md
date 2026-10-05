# Contributing

Read [agent instructions](AGENTS.md), the [architecture baseline](docs/architecture/baseline.md),
and the active milestone before changing behavior. Link nontrivial work to an
issue. Product work requires the relevant PRD, EDD, and Test Design Document;
a research spike must identify its question and time box.

## Development and review

Use short-lived topic branches from `main`. Add a failing test, implement the
smallest correct change, then refactor with passing tests. Bug fixes add regression
coverage. Keep a PR focused on one outcome and update its docs alongside code.

Run `make tools` and `make hooks`. After signing/signing off the changes, run
`PR_TITLE='type: actual proposed PR title' make review` from a clean checkout.
It verifies commit signatures/DCO/title and runs `make check lint test secrets`
plus all race/fuzz checks; see the [local review gate](docs/engineering/local-review.md).
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
