# Governance

Horvium begins as a maintainer-led project. Cam (`@gonz0w`) is the initial maintainer; the selected GitHub repository is
private `gonz0w/horvium`. Commercial and home use share the same documented behavior contracts.

Product scope is recorded in PRDs, engineering choices in EDDs/ADRs, and release
proof in Test Design Documents. Routine changes use reviewed pull requests.
Governance changes require a focused PR. Maintainers approve designs and releases;
agents prepare reviewable work and never merge without explicit authorization.

In the solo phase, required independent approvals are zero. PRs, signed canonical
history, required local qualification and review, and resolved conversations remain mandatory. When a distinct
reviewer joins, require independent and CODEOWNERS approval. Emergency exceptions
must record scope, authorization, follow-up review, and regression coverage.

The [Agent Temper composition](PROJECT.md) records persistent development
choices and the owner's permission for bounded read-only agent reviews. Existing
maintainer merge and consequential-action gates remain. Resolve missing or
conflicting policy through owner Q&A and save the answer before affected work.
