## Problem and resulting behavior

Describe the trigger, change, and operator outcome.

## Tracking and design

Refs gonz0w/horvium#number. Link requirement IDs, approved PRD/EDD/Test Design,
contracts and ADRs. Use Closes only when this PR satisfies every criterion of
that specific issue; broader acceptance gates use Refs. Name repository-qualified
issues for cross-repository work. Identify parent gates and native blockers.

## Validation

Record the exact tested source/base IDs, actual PR title and `make review` result.
Identify the selected plan, packages/fuzz targets and original source for any
reused evidence. Milestone/release PRs record fresh `make review-full` evidence.
Attach sanitized native/client manifests where required; link stable test IDs.
Record supervised teardown and `make lab-status`; explain any unverified cleanup.

## Compatibility and operations

Describe client/FreeBSD compatibility, security impact, and migration/rollback
where relevant. Identify changed docs and any limits of the evidence.

## Review

- [ ] Owning issue, requirements/design and evidence are linked; closure scope is accurate.
- [ ] Issue status and Project fields/dependencies are reconciled manually.
- [ ] Scope matches the active design or authorized spike.
- [ ] Meaningful tests and documentation are updated.
- [ ] Local review evidence matches the current PR head/base/title.
- [ ] Relevant native/client gates and their limits are identified.
- [ ] Owned development processes are stopped; teardown is verified.
- [ ] No secrets or operator data are included.
- [ ] Commits carry author DCO sign-offs; canonical history will be signed.

Agents leave this PR open for maintainer review and merge.
