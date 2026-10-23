# Horvium delivery context

Delivery source: `gonz0w/suite-standards @ 4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22`.
Authoritative [backlog delivery](https://github.com/gonz0w/suite-standards/blob/4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22/docs/backlog-delivery.md) and
[rollout](https://github.com/gonz0w/suite-standards/blob/4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22/docs/backlog-delivery-rollout.md); adoption owner @gonz0w,
[#419](https://github.com/gonz0w/horvium/issues/419). Shared Kanban/INVEST/Ready/Done
and batching rules are maintained there, not copied into this repository.
The advisory go-service manifest and stronger local security/acceptance rules
remain independently scoped. No moving-branch fetch is needed during validation.

## Outcome, queue and commands

The first deployable NAS release remains M1-M6; M7 is the later package-based
image. The independent **read-only lab package planning slice** is install,
explicit protected setup, start, authenticated HTML/CLI inventory, stop and restart.
It changes no release denominator or runtime write gate. Its ordered issue chain
is [design #159](https://github.com/gonz0w/horvium/issues/159),
[build/stage #160](https://github.com/gonz0w/horvium/issues/160), then
[native installed lifecycle #161](https://github.com/gonz0w/horvium/issues/161).
GitHub owns current status; #159 is the next independent Ready story once WIP permits.

Run from the repository root with existing pinned tools. Static documentation
checks use `make check` and `make secrets`; the normal complete source command is
`make check lint test secrets`. After a signed clean checkpoint,
`PR_TITLE='type: actual title' make review`; milestones/releases use `make review-full`.
Existing required gates still apply to the current candidate. Process-bearing
checks remain held while shared-host containment is unproved. A static pass permits
no test retry, native run, download, credential/security change or deployment.

## Numeric WIP and inherited exception

Default: at most **2 active batches per repository and 1 per implementer**.
In Progress, Blocked and Review count. Related issues in one coherent outcome
count once; umbrella milestone trackers are not extra implementation batches.

Dated reconciliation, 2026-10-23: four inherited held outcomes remain visible:

- M1 manual/browser acceptance #137 (parent #130).
- Original runtime/startup qualification #138/#139/#305 (parent #131).
- Owned-process cleanup #414 / draft PR415.
- Dashboard resource qualification #382.

Existing owners, evidence and statuses are preserved. Native/manual/containment
holds are not cleared by relabeling, elapsed time or a source merge. The one-shot
shared boot observation established no later boot; restart is deferred and the
approved elevated startup portable retry is unused. Reuse existing observations;
this adoption grants no diagnostic or retry authority.

**Temporary WIP exception:** the Horvium consumer owner records the user-directed
adoption/backlog batch #419 in addition to those four inherited held outcomes
(maximum 5 total), accountable maintainer @gonz0w, with no new product implementation
batch. Reason: central
instructions and a usable Ready queue must be adopted without interrupting held
work. Review at adoption handoff and no later than **2026-10-24**; end the extra
allowance at that handoff. Reconcile inherited excess before pulling #159 or any
other new batch. Never close/reset/relabel old work to manufacture capacity.

## Local mapping and exceptions

The existing board has Backlog/Ready/In progress/Review/Validation/Done.
When work cannot proceed, state `Delivery state: Blocked` with owner, reason and
next action in the issue and preserve native blocker links; the board may retain
Validation for outstanding acceptance. Such work still counts against WIP.
GitHub rejected the new #419 -> #131 native parent edge because #131 already has
100 children. Keep the readable relationship and limit exception in #419; do not
reparent historical issues or manufacture a different parent to evade that limit.
Existing native edges are unchanged. No Project options, protections or automatic
status/merge rules are changed.

Required independent review, signed author DCO, actual native evidence and explicit
operational authority remain effective. No acceptance is inferred from the pointer.
Revert adoption through a reviewed source/pointer change, retaining issue history,
profile pins and all product/failure evidence. This dated queue/WIP context is an
index into GitHub, not a second live status board.
