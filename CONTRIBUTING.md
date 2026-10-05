# Contributing

Read [agent instructions](AGENTS.md), the [architecture baseline](docs/architecture/baseline.md),
and the active milestone before changing behavior. Link nontrivial work to an
issue. Product work requires the relevant PRD, EDD, and Test Design Document;
a research spike must identify its question and time box.

## Development and review

Use short-lived topic branches from `main`. Add a failing test, implement the
smallest correct change, then refactor with passing tests. Bug fixes add regression
coverage. Keep a PR focused on one outcome and update its docs alongside code.

Run `make tools`, `make hooks`, then `make check lint test secrets` and
`make go-check` before requesting review; report the tested commit/results in
the PR. Race/fuzz checks normally run locally to reduce hosted cost. Native
evidence still needs both declared release candidates and disposable fixtures;
see [local qualification](docs/testing/local-qualification.md). Product gates
will add Go checks, native FreeBSD integration, client tests, and recovery tests
when their components exist. Developer tests may use VMs; Horvium's runtime scope
is a dedicated file server.

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
