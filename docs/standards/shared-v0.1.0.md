# Shared standards v0.1.0 — advisory adoption

Preserve separate repositories. Common roots include README, CONTRIBUTING,
AGENTS, GOVERNANCE, SECURITY, license, ownership and PR templates; docs cover
requirements, decisions, test design, compatibility and recovery. Language
profiles retain Go cmd/internal, native C src/tests and separate conformance
boundaries. Foundation/research/product/native evidence are distinct.

Use coherent outcome PRs, signed Conventional Commits and author DCO. Git owns
durable design/evidence; existing issues/projects own live work/dependencies.
Inspect existing scope before creating trackers. A component merge does not
close milestone acceptance. Prefer independently specified behavioral tests over
implementation-mirroring assertions; bound parser fuzzing and failure experiments.

The portable quality profile includes formatting, vet/static analysis appropriate
to the language, regression tests, conservative affected-package race/fuzz checks,
secret scanning, license/provenance and pinned dependency/advisory review. Tool
selection follows an owning issue and measured value; no blanket new tool suite.
Native security/recovery and browser/accessibility gates remain project-specific.

## Adoption and drift

Pin a full standards revision, profile, adopted file hashes and justified
exceptions in a consumer manifest. Do not fetch a moving branch during checks.
Start with advisory reports; enforce only explicitly adopted requirements.
Updates are normal reviewed PRs on natural work boundaries. Existing stronger
project rules remain effective. Drift checks do not execute candidate code.

The read-only inspector rejects absolute/traversing/symlink paths and compares
adopted file hashes. An exception identifies its owner, reason and review date.
CI/evidence policy changes must invalidate prior deep evidence.

## Presentation

BSD Classic uses warm red branding, neutral surfaces, fine rules, restrained
corners, readable system sans-serif and monospace technical values. Preserve
HTML/no-JS workflows, visible focus, light/dark and reduced motion. Use pinned
local HTMX where adopted; retain domain navigation. Charts show scope, units,
freshness, coverage and gaps, with exact-value tables. Color is supplemented by
labels. No SPA/UI kit, animation engine or runtime CDN is introduced for polish.

Tokens derive from existing Horvium and Arxwall CSS. They are original project
values, not copied FreeBSD website assets. Shared tokens do not imply complete
visual/accessibility acceptance. No runtime stylesheet is changed by the pilot.
