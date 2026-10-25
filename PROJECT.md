# Horvium project policy

Adopt the pinned [Agent Temper method](.agent-temper/METHOD.md) as the portable development
method. The [source record](.agent-temper/SOURCE.md) identifies the immutable revision and retained
bytes. Only the [Codex profile](.agent-temper/profiles/codex.md) is selected. Existing local
verification and GitHub practices remain local extensions; the bundled GitHub
and strict-test-first profiles are inactive.
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

Horvium product runtime and target-specific native acceptance remain **FreeBSD**.
All LAB checks still require an authorized BSD executor; macOS and Linux remain
coordination/source-review hosts. The earlier #422 host permission was finite
and expired. Its saved receipt supplies no current host-check exception.

On 2026-10-25, Cam explicitly approved validation by component and change impact
for [Horvium #424](https://github.com/gonz0w/horvium/issues/424) and directed
this adoption lead to apply it. The old blanket requirement made every process,
metadata and routing change take the full product review. The replacement uses
applicable checks once on a designated authorized BSD for portable source,
metadata, provenance, prose and agent-routing configuration. Preserve FreeBSD
coverage for Horvium runtime, OS adapters and target-specific process/kernel
behavior; broaden for affected shared code, execution or uncertain impact.
The reason is to verify the changed contract without unrelated product
qualification. This is a direct owner-approved local policy clarification, recorded
in the owning issue. Existing immutable upstream pins remain unchanged; no
external executable selector or unverified execution result is adopted.

Temper-only evidence covers exact source/inventory/hash/license provenance,
policy and link consistency, configuration syntax and affected role/routing
contracts, with separate review. Unobserved runtime client/model discovery stays
unverified. Changing validation selection or execution is validation-tooling
impact, not pure Temper prose; verify the affected selector/gate behavior and
its failure paths before relying on a narrower mapping. The
[local review contract](docs/engineering/local-review.md#scope-reconciliation-for-424)
records the finite transition and conservative fallback requirements.

Existing executable hooks and review commands remain effective until the
narrower mapping is implemented, independently reviewed and actually verified.
The finite mapping in the local review contract requires its actual affected
regressions and separate review before acceptance. Ordinary/full review retains
`make check lint test secrets` and conservative deep selection; a source/policy
edit alone cannot establish a narrower PASS.
Pre-commit remains `make check secrets`. Signatures, actual-author DCO, clean
source identity, applicable custody/teardown and protected checks remain required.
Actions stay disabled. Missing executor/tool admission blocks affected execution
while source preparation and read-only review may continue. No new worker,
guest, retry, budget, resource, credential, permission, release or deployment
authority is granted. Preserve all historical failures and unqualified attempts.

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
scope only. The October 24 routing decision below additionally permits bounded
implementation delegation. Candidate/native execution, other-resource control
and merging retain their existing authority gates. Self-review remains self-review;
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

### Codex routing selection and policy change — 2026-10-24

The owner explicitly selected model-routed subagent implementation and directed
all existing Temper adopters to be ready for it in
[issue #424](https://github.com/gonz0w/horvium/issues/424). This replaces the
previous read-only-review-only delegation allowance with bounded implementation
delegation within already owner-accepted task scope. The lead must supply explicit
file ownership, acceptance, policy, evidence, stop conditions and resource limits.
The reason is that owner-selected workflow; separate review and existing native,
operational and per-PR maintainer merge approval remain binding. This does not
select a new product/platform/toolchain or expand other-resource authority.

| Role | Assignment | Model | Effort |
| --- | --- | --- | --- |
| `temper_workflow` | Bounded read-only inspection and evidence summaries | `gpt-6-luna` | `medium` |
| `temper_implementer` | Accepted implementation, tests and ordinary debugging | `gpt-6.1-sol` | `high` |
| `temper_analyst` | Difficult analysis or separate complex review | `gpt-6-astra` | `high` |
| `temper_reviewer` | Separate final review of work it did not author | `gpt-6.1-sol` | `high` |

The lead owns tracker writes, integration and returned-evidence verification.
Use at most three concurrent delegated agents, excluding the lead, within existing
resource and WIP limits. No nested helpers without an explicit lead assignment.
Keep tiny or tightly coupled work with the lead when appropriate. No role changes
permissions or authorizes native/operational execution by itself.

If a role or model is unavailable, report it. The lead may do the bounded work
with its current model only when adequate and permitted under existing policy
and limits; otherwise resolve the missing choice with the owner. Never silently
substitute a model or claim that requested settings actually ran. The owner
approves model/effort/role/limit changes; routine routing within this table is
delegated to the lead. Preserve independent final review.

Local `.codex/agents/` files are adapted project configuration outside the pinned
bundle. No global/default model, permission or primary-chat setting changes.
Where native role discovery is unavailable but explicit spawn settings are
supported, use this same table with a complete fresh-context brief and record
observed settings, or follow the fallback. Existing chats do not switch models
when files change. Record client/version and observed role/model/effort metadata
in the owning issue; parsing proves syntax only. Native discovery/routing remain
unverified until actually observed in a fresh supporting session. Earlier active
work retains accepted scope and evidence; future delegation follows this policy.

For this exact routing update in issue #424, the owner authorized bounded
validation in an isolated disposable FreeBSD guest on 2026-10-24, preserving
resource, isolation and teardown limits. Coordinate the shared executor before
startup. This permits the required checks/hooks on the admitted executor; it
does not permit Mac/Linux project checks, production, new credentials,
permission/protection changes or unrelated operations. The owner authorized this
bounded issue's gated merge after required checks and separate review; general
future maintainer approval and operational boundaries remain unchanged.
