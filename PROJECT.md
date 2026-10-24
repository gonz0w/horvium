# Horvium project policy

Adopt the pinned [Agent Temper method](.agent-temper/METHOD.md) as the portable development
method. The [source record](.agent-temper/SOURCE.md) identifies the immutable revision and retained
bytes. **No optional profiles are selected.** Existing local verification and
GitHub practices remain local extensions; the bundled profiles are inactive.
The [project charter](docs/project/charter.md) owns product scope. This policy owns development decisions and
authority; it does not replace those product contracts.

### Work authority and finite delivery

GitHub Issues and the existing shared Project are the sole live-work authority. The [local workflow](docs/engineering/github-workflow.md) records exact status/priority/phase mappings and native parent/blocker relationships. Git documents hold design, decisions, stable roadmap links and executed evidence, not a second queue.

Select accepted, bounded work before implementation. Name its owner, outcome,
non-goals, acceptance, dependencies, verification and effects. Keep the approach,
discoveries and continuation in that work record. A broad milestone or unresolved
proposal is not permission to implement. Preserve pending broader acceptance when
a component merges. Follow the [existing workflow](docs/engineering/github-workflow.md) for selection,
review, status reconciliation and completion. Method adoption does not advance
product qualification or erase failed/historical results.

### Local contracts and simplicity

FreeBSD/ZFS, Go management and narrow typed privilege operations, independent file daemons and HTML/CSS/HTMX. The MicroServer remains one lab profile. Preserve both declared FreeBSD targets, disposable storage, restore/client/accessibility and teardown gates. Current storage-control commands remain unregistered/unavailable and runtime writes remain denied; packaging source is not installation qualification.

Keep current languages, tools, dependencies and support decisions at their
canonical local paths. Use the smallest implementation that meets current
acceptance, safety, recovery and accessibility requirements. A new language,
service, dependency or abstraction needs a present benefit and maintenance-cost
justification; significant direction changes need an owner decision. Preserve
existing immutable Suite/tool/source pins and approved exceptions. The consumer
source revision binds these local extensions; external sources retain their own
recorded revisions, hashes and content. Temper imposes no OS, toolchain or hosting.

### Verification and review

`make check lint test secrets` and the signed clean-candidate `make review` remain required; milestone/release or conservative fallback uses the existing full policy. Its selector does not exempt new method/policy paths. Actions stay disabled. Product/native/client recovery checks retain their own required boundaries.

Follow existing behavior-first/regression and PRD/EDD/Test Design obligations
where they apply. Prose changes use source, link, policy and reader checks; do not
invent red/green runtime history. Preserve final required checks and conservative
fallbacks. Record passed, failed, skipped, unavailable and not-applicable results
separately. A stored recipe or component merge is not native, browser, hardware,
performance, operational or release acceptance.

On 2026-10-24 the owner approved lasting bounded read-only agent reviews of
accepted work. The lead may assign a finite source/policy/evidence inspection to
a separate agent and record the reviewer, exact candidate, scope, findings and
resolution. This replaces any previous blanket no-agent rule for that review
scope only. It does not authorize delegated implementation, candidate/native
execution, other-resource control or merging. Self-review remains self-review;
an agent review is not an independent human approval or a GitHub approving review.

### Decisions and action authority

Cam (@gonz0w) remains the owner. Agents may prepare accepted changes, choose
routine implementation/documentation details within scope, run authorized checks
and prepare signed, actual-author-DCO review candidates. Existing Conventional
Commit and PR rules remain. Use only the existing authorized signing identity;
do not create keys, trust files or alter security configuration to force a pass.

Leave PRs open for maintainer review. Current policy requires explicit maintainer
merge authorization; method adoption and review permission do not supply it.
Required checks, review findings and final source/signature/DCO verification must
be resolved before an authorized merge. Release, publication, deployment,
credentials, operational/native effects and changes outside the owning work item
retain their separate authority. The adopted method grants none of these.

If a required decision is missing or applicable rules conflict, record the
question in the owning work item, ask the owner and stop only dependent work.
Continue independent authorized work. Save the old decision, replacement, reason,
scope and authority here or in its canonical local decision record before
proceeding. Apply recorded answers in later sessions without asking again.
Compatible rules apply together; file order, a newer upstream timestamp or an
agent preference does not override them. No Temper core deviation is declared.

### Documentation for readers

Use the [documentation reference](.agent-temper/docs/reference/documentation.md)
for reader-oriented structure and original STE-informed plain-English procedures.
Do not claim formal ASD-STE100 compliance or reproduce its dictionary. Put the
result, prerequisites, working directory and meaningful recovery in task guides;
use consistent terms and synthetic examples. Separate reference from explanation
when readers benefit, without creating empty sections or duplicate documents.

README explains purpose, maturity and navigation; agent instructions provide entry
routes; contribution guidance explains delivery; this policy records decisions.
Product/user documentation owns supported behavior and setup/recovery. Decisions
explain tradeoffs; evidence retains actual source-bound observations. Keep live
work in its authority above. Preserve the existing installed/manual and browser
accessibility requirements, and distinguish proposed setup from executed support.

### Maintenance of the method

Use the existing work/review process for pin or extension updates, exceptions and
removal. Review semantic changes, compatibility, source retention, links/licenses
and a representative work item from pickup through completion. Owner decisions
are needed for authority or required-policy changes. Adoption findings belong in
[the owning item](https://github.com/gonz0w/horvium/issues/422); publish a sanitized upstream proposal only with
appropriate authority. A project-specific requirement is not a universal rule.
