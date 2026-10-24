# Agent instructions

Read the pinned [Agent Temper method](.agent-temper/METHOD.md),
[local project policy](PROJECT.md) and the selected owning work item first.
Only the Codex profile is selected in local project policy. Product/platform rules below are local
extensions; the method does not choose them. Follow the existing work authority
and required gates. Recorded owner decisions persist across sessions.

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
   Run make check lint test secrets only on an admitted FreeBSD executor.
   The owner's FreeBSD-only LAB rule includes standards/content checks, hooks,
   tests and make review: do not run them on macOS or Linux. The finite #422
   adoption host permission has expired; saved receipts grant no exception.
   Use make review for affected-package race/fuzz checks; make review-full at
   milestones/releases, on the admitted executor. If it is unavailable, preserve
   a source candidate and report the execution gate; do not bypass hooks.
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
14. The owner permits bounded read-only agent reviews and bounded implementation
    delegation within accepted scope and explicit ownership under PROJECT.
    Native execution and merging retain their existing authority gates.
15. Follow docs/engineering/github-workflow.md and the stable roadmap index.
    Inspect open PRs and unblocked Ready issues before pickup; select and name
    one bounded owning issue, then read its requirements, design and evidence.
    Deliver a coherent outcome with its related implementation, tests, docs and
    evidence in one PR; internal helpers are incremental commits, not default PRs.
    GitHub owns live status/priority/dependencies; Git owns authoritative designs
    and receipts. Keep one status label aligned with Project Status, Phase and
    Priority, and maintain native parent/blocker relationships manually.
16. Use Refs for broader acceptance gates. Use Closes only when every criterion
    of that specific issue is met. Merged components never waive pending native,
    browser, recovery, hardware or release gates. Break broad backlog work into
    reviewed outcome-sized slices when picked up. Split only for an unresolved
    design/authorization decision, distinct safety boundary or scope too large
    to review coherently; explain the reason. Add no tracking automation.

Apply the Codex routing profile selected in local project policy. When useful,
delegate bounded inspection to `temper_workflow`, accepted implementation to
`temper_implementer`, difficult analysis to `temper_analyst`, and separate final
review to `temper_reviewer`. Supply the work item/comments, policy, relevant
files, acceptance, checks and edit boundaries. Follow local escalation, fallback,
resource and execution limits; the lead owns integration and verifies evidence.
