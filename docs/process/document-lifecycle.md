# Documentation lifecycle

Markdown beside the code is canonical. Keep living docs current in the same PR
as behavior changes. Use reviewable Mermaid/ASCII diagrams where useful. Record
important decisions in Git rather than leaving them only in chat.

| Document | Purpose |
|---|---|
| PRD | What/why, operator outcomes, non-goals, acceptance requirements |
| EDD | How, boundaries/state, privilege, failures, upgrade/recovery |
| Test Design Document (TDD) | Requirement-to-test traceability, fixtures, native gates, evidence |
| ADR | One durable decision, alternatives, consequences, revisit trigger |
| Research note | Dated primary evidence and limitations; not adopted architecture |
| Spike | Time-boxed experiment answering a design question |

**TDD in filenames means Test Design Document.** Test-driven development means
writing failing behavioral tests before implementation, then refactoring with
passing tests. The project practices both; do not confuse them.

Milestones progress Proposed -> Researching -> Draft -> Review -> Approved ->
Implementing -> Validating -> Released -> Superseded. Each active milestone pins
reference versions and creates linked PRD/EDD/TDD documents before product work.
Do not write detailed future APIs months in advance. Templates are starting points,
not approved designs. Every accepted requirement has stable IDs and acceptance
evidence. Superseded docs identify successors; Git preserves their history.
