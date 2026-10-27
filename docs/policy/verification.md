# Verification and execution contracts

These local clauses were extracted from [PROJECT.md](../../PROJECT.md) without changing their
requirements. Source identity is the consumer revision containing this extraction.
Command paths and literal configuration paths remain relative to the repository root.
Selected applicability is recorded only in root `policy.toml`; linked contracts retain
their existing scope. The owner changes required policy through the existing process.

## Verification and review

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
[local review contract](../engineering/local-review.md#scope-reconciliation-for-424)
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
scope only. The [October 24 routing decision](routing.md#codex-routing-selection-and-policy-change--2026-10-24) additionally permits bounded
implementation delegation. Candidate/native execution, other-resource control
and merging retain their existing authority gates. Self-review remains self-review;
an agent review is not an independent human approval or a GitHub approving review.
