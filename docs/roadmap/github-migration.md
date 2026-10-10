# GitHub roadmap migration receipt

Date: 2026-10-10. This is a dated audit, not a second live task-status board.
Live work is in the [private Project](https://github.com/users/gonz0w/projects/2)
and [stable roadmap index](github-roadmap-index.md). Designs, contracts, ADRs,
Test Designs and candidate-specific evidence remain authoritative in Git.
The [canonical workflow](../engineering/github-workflow.md) defines manual triage.

## Scope and preserved authority

Inspected repository instructions, contributor/Git/security/release standards,
M0 acceptance, M1/M2 approved packets and amendments, roadmap, compatibility,
merged PRs and source-bound receipts, and the Arxwall workflow/index/forms/template.
Horvium's architecture currently uses one private repository; no split was invented.
The Project is ready for related repositories with matching phases and native
cross-repository sub-issues when they exist. FreeBSD, ZFS, hardware-neutral scope,
package-first distribution, independent file services and existing release gates remain.

Existing issues #1, #19, #29 and #44 retain their numbers, useful labels and original
text in collapsible history. #1/#19 now close their fulfilled research/design scope,
with broad M1/M2 acceptance tracked separately. The original M0/M1 milestones retain
numbers 1/2 and their closed/open state. M2–M7 were added without due dates.
The maintainer's local dashboard/security proposals and associated documentation
were carried into this reviewable branch; they remain planning, not approved runtime
implementation or new compatibility evidence. The concurrent primary checkout and
its drafts were preserved.

## Read-back verification

| Observed item | Result |
|---|---|
| Repository | `gonz0w/horvium`, private |
| Shared Project | Horvium development, private, linked to Horvium |
| Issues | 179 total: 175 created, 4 existing trackers preserved |
| Issue states | 124 closed scoped outcomes; 55 open work/gates |
| Repository milestones | 8 total: 2 retained, 6 created; zero due dates |
| Workflow labels | Exactly one `status:*` label per issue |
| Consistent labels | 14 new type/status/priority labels; existing definitions retained |
| Native parent links | 171 verified against intended owning parents |
| Native blocked-by links | 78 verified; dependency graph has no cycles |
| Project items | 179, each with verified Status, Phase and Priority |
| Saved views | All work; Current phase (M2); Ready next; Acceptance gates; Completed evidence |
| View configuration | Filters/layouts read back; Phase/Repository groups and Status columns verified |
| Actions | Still disabled; no workflow, runner or synchronization service added |

Closure is limited to 123 bounded merged PR outcomes plus the original accepted M0
gate. 122 PR merge commits are ancestors of the inspected main revision; PRs 15/17
were accepted through the PR 14/16 branches and are scoped to their distinct outcomes.
Historical failed, skipped, unstarted and unsupported native/browser/resource cases
remain in linked receipts. All 106 existing evidence/source-manifest blobs inspected
are unchanged from main. A merged component never closes the broader capability.
M1's manual browser/accessibility gate remains open by maintainer choice. M2 runtime,
integration/recovery/browser/resource and later packaging/release gates remain open.

## Repository validation and recovery

`make check lint test secrets` passed, including 186 tooling tests, ordinary Go tests,
formatting/vet, ShellCheck and Gitleaks. Issue forms have six consistent sections,
unique IDs and one initial Backlog workflow label. The signed clean-candidate
`make review` performed the existing conservative full portable plan because policy
and `.github` inputs changed: 11 race packages and 54 bounded fuzz targets passed.
This is portable review only; no new native NAS qualification is claimed.

Executed deep source: `dbc0624c9d61288be8c66dc8406cd6ed97811972`.
Review base: `f3fc4a34ffa210f9512fde3b88a470943cd2a51f`.
Actual PR title: `docs: migrate Horvium roadmap to GitHub tracking`.
The prose-only receipt followup reruns baseline checks and uses the existing
exact-input evidence-reuse policy; the original executed deep source remains explicit.
Final source/base/title and reuse result are recorded in the PR and private receipts.
Commits are signed and author DCO signed off. Owned review/development teardown
was verified; no host services or disposable NAS fixtures were enabled.

An initial sandboxed baseline attempt was interrupted after Unix-socket permission
restrictions; it is not passing evidence. The complete authorized local run above
passed. A Project-item create returned an already-existing-content result; recovery
re-read its original ID, completed only the remaining items and passed the full audit.
No duplicate issue, Project or milestone was created. Initial API/schema probes and
incomplete reconciliation are not treated as final verification.

Private ignored migration snapshots retain original trackers/settings/PR records,
stable marker mappings, operation checkpoints and read-back audits under
`.tools/roadmap-migration/` in the primary development checkout. Unknown creation
outcomes are recovered by markers and native IDs rather than blindly retried.
The [workflow](../engineering/github-workflow.md) documents idempotent reconciliation,
including avoiding a stale initial status snapshot after maintainer changes.
No secrets, keys, host inventory or customer data are part of committed tracking.

## Next unblocked work

The highest-priority Ready design pickup is
[#288: retained-history startup recovery slice](https://github.com/gonz0w/horvium/issues/288).
Independent planning is available in [#166: dashboard packet](https://github.com/gonz0w/horvium/issues/166)
and [#167: second-release threat model](https://github.com/gonz0w/horvium/issues/167).
Consult GitHub for their current status. These are review boundaries, not permission
to enable an unqualified writer, install a chart dependency or deliver new security controls.
