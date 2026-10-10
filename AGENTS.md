# Agent instructions

1. Read README.md, CONTRIBUTING.md, and relevant docs before editing.
2. M0 permits repository tooling and documentation only. Product implementation
   needs an approved milestone PRD, EDD, and Test Design Document, or an explicitly
   authorized research spike. Ask only about genuinely unresolved decisions;
   user authorization already in the conversation remains valid.
3. Keep FreeBSD as the runtime target. The MicroServer is one lab profile, not a
   source of hardcoded hardware assumptions. Support claims require evidence.
4. Prefer FreeBSD base facilities and Go's standard library. Do not introduce
   frameworks, Linux dependencies, containers, or distributed infrastructure
   without an accepted ADR and an actual requirement.
5. Use semantic HTML5, project-owned CSS, and HTMX for progressive interaction.
   Follow docs/ui/baseline.md. No SPA, UI kit, or animation library.
6. The UI and CLI use explicit management contracts. Keep shell execution and
   privilege out of HTTP handlers/templates. Never interpolate user input into a
   shell command; privileged requests require typed validation and authorization.
7. Write failing behavioral tests before implementation where practical. Fixes
   need regression coverage or a documented reason it is impossible. Test real
   FreeBSD facilities before claiming native behavior.
8. Test storage changes only on disposable fixtures. Do not use an operator's
   live disks, datasets, or backup history as development fixtures.
9. Update behavior documentation and the compatibility matrix in the same PR.
   Run make check lint test secrets before reporting completion. Use make review
   for affected-package race/fuzz checks; make review-full at milestones/releases.
   Relevant native safety/recovery gates remain required for behavior changes.
10. Keep commits coherent, Conventional-Commit-style, signed, and DCO signed off.
    Never silently downgrade signatures if signing is unavailable.
11. Do not commit secrets, private keys, customer data, host inventory dumps, or
    replication credentials. Preserve provenance and third-party notices.
12. Leave PRs open for maintainer review. Do not merge without explicit user
    authorization for that PR. Do not deploy or enable host services merely to
    validate a repository foundation.
13. Treat research as evidence, not architecture, until adopted in an ADR/EDD.
    Do not invent future CLI flags, API routes, config schemas, or package names
    beyond the agreed project identity.
14. Do not spawn agents unless the user explicitly requests parallel agent work.
15. Follow docs/engineering/github-workflow.md and the stable roadmap index.
    Inspect open PRs and unblocked Ready issues before pickup; select and name
    one bounded owning issue, then read its requirements, design and evidence.
    GitHub owns live status/priority/dependencies; Git owns authoritative designs
    and receipts. Keep one status label aligned with Project Status, Phase and
    Priority, and maintain native parent/blocker relationships manually.
16. Use Refs for broader acceptance gates. Use Closes only when every criterion
    of that specific issue is met. Merged components never waive pending native,
    browser, recovery, hardware or release gates. Break broad backlog work into
    reviewed implementation slices when picked up; add no tracking automation.
