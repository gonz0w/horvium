# Work and delivery contracts

These local clauses were extracted from [PROJECT.md](../../PROJECT.md) without changing their
requirements. Source identity is the consumer revision containing this extraction.
Command paths and literal configuration paths remain relative to the repository root.
Selected applicability is recorded only in root `policy.toml`; linked contracts retain
their existing scope. The owner changes required policy through the existing process.

## Work authority and finite delivery

GitHub Issues and the existing shared Project are the sole live-work authority. The [local workflow](../engineering/github-workflow.md) records exact status/priority/phase mappings and native parent/blocker relationships. Git documents hold design, decisions, stable roadmap links and executed evidence, not a second queue.

Select accepted, bounded work before implementation. Name its owner, outcome,
non-goals, acceptance, dependencies, verification and effects. Keep the approach,
discoveries and continuation in that work record. A broad milestone or unresolved
proposal is not permission to implement. Preserve pending broader acceptance when
a component merges. Follow the [existing workflow](../engineering/github-workflow.md) for selection,
review, status reconciliation and completion. Method adoption does not advance
product qualification or erase failed/historical results.

## Shared delivery extension

Cam (@gonz0w) selected the immutable Suite-Standards
[delivery agreement](https://github.com/gonz0w/suite-standards/blob/4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22/docs/backlog-delivery.md)
and [rollout](https://github.com/gonz0w/suite-standards/blob/4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22/docs/backlog-delivery-rollout.md)
at `4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22` in
[#419](https://github.com/gonz0w/horvium/issues/419). Apply their Ready/Done,
coherent-batch and acceptance-mapping requirements through the existing local
workflow. This is a delivery extension, not another Temper profile or tracker.

Ready work names its user/dependent outcome and MVP/release, scope and non-goals,
owner and priority, observable acceptance, dependencies/decisions, proportional
checks and required native/security gates, rollout/rollback, and authority/holds.
Done requires demonstrated acceptance, current required checks/review, linked
evidence/docs and stated rollout acceptance. Source completion alone does not
accept a product outcome. Keep stronger local native, security, command, review
and merge gates effective; preserve any unmet broader acceptance.

The existing WIP cap recorded in #419 is **2 active batches per repository and
1 per implementer**. Started blocked/held and Review work count; related issues
in one coherent batch count once. Keep dated counts, holds, exceptions and next
actions in the existing issues and Project, with #419's reconciliation as the
adoption record. Relabeling or parking started work does not free capacity, and
this extension renews no expired exception or allowance.

The advisory standards profile and its `7a99fb2e6855d9f73752b182118637d9e79dc03c`
source manifest remain independently scoped and unchanged. This pointer grants
no execution authority. Revert it only through the existing reviewed source-change
process, retaining issue history and product evidence.
