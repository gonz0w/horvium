# Agent instructions

Delivery authority: [Suite-Standards backlog delivery](https://github.com/gonz0w/suite-standards/blob/4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22/docs/backlog-delivery.md)
and [consumer rollout](https://github.com/gonz0w/suite-standards/blob/4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22/docs/backlog-delivery-rollout.md), immutable revision
`4aeb70ae4d7423c4ae7d52c025f94af4fb53ff22`. Adoption owner: @gonz0w, [#419](https://github.com/gonz0w/horvium/issues/419).
Read that shared agreement, README.md, CONTRIBUTING.md and the applicable milestone
packet before implementation. Use the existing owning backlog stories and the
[local command/WIP index](docs/engineering/agent-tasks.md); do not duplicate the
central policy. Local security and acceptance requirements below remain stronger.

1. Product implementation needs an approved PRD, EDD and Test Design or an
   explicitly authorized research spike. Existing user authorization remains valid.
   Research is evidence until adopted; do not invent CLI/API/config/package names.
2. FreeBSD is the runtime target. The MicroServer is a lab profile, not a hardware
   assumption. Prefer FreeBSD base facilities and Go's standard library; new
   frameworks, Linux dependencies, containers or distributed infrastructure need
   an accepted ADR and requirement. Actual support claims require evidence.
3. Use semantic HTML5, project-owned CSS and HTMX; follow docs/ui/baseline.md.
   No SPA/UI/animation library. Keep privilege and shell execution out of HTTP
   handlers/templates; privileged requests require typed validation/authorization.
   Never interpolate user input into a shell command.
4. Storage/native tests use explicitly admitted disposable fixtures, never an
   operator's live disks, datasets or backup history. Portable/source completion
   never waives both-release native, browser, recovery, resource or hardware gates.
   Preserve operational holds, failures, consumed attempts and user work.
5. Keep commits signed, Conventional and author-DCO signed off. Update behavior
   docs/compatibility in the coherent PR; meaningful fixes need regression coverage.
   Run `make check lint test secrets` and clean-candidate `make review` as required
   by CONTRIBUTING; use `make review-full` at milestones/releases. Record held lanes
   and keep the PR open when required checks/review have not passed.
6. Leave PRs open for maintainer review; merge needs explicit authorization for
   that PR. No service installation, deployment or host enablement merely to
   validate source. Never downgrade signatures or weaken protections/checks.
7. Never commit secrets, keys, credentials, customer data or host-inventory dumps.
   Preserve provenance/notices. Do not spawn agents unless explicitly requested.
8. GitHub owns live status/priority/parent/blocker relationships; Git owns designs
   and exact receipts. Use the existing [workflow index](docs/engineering/github-workflow.md)
   and roadmap. Use Refs for partial/native outcomes and close only scoped,
   demonstrated acceptance. Add no tracking automation.
