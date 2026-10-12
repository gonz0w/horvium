# Deadline provenance runner input binding

Owner: [issue 343](https://github.com/gonz0w/horvium/issues/343), under
[acceptance owner 305](https://github.com/gonz0w/horvium/issues/305) and
[parent 139](https://github.com/gonz0w/horvium/issues/139), after merged consumer
PR 342. Authority: the [adopted method/Test Design](../../research/M2/policy-startup-timeout-provenance.md),
approved [M2 PRD](../../milestones/M2/PRD.md), [EDD](../../milestones/M2/EDD.md),
[TDD](../../milestones/M2/TDD.md), retained-policy contract and ADR 0009 linked
there. This implements only runner-input parts of
M2-POLICY-DEADLINE-PROVENANCE-003/004/006; native execution and broader gates
remain separate.

## Offline boundary

`tools/check-policy-provenance-candidate.py` is development tooling for the
future qualification runner. The caller supplies expected source, compiler and
artifact identities from the independently reviewed build. The tool never
derives expected identities from transcript claims. It verifies:

- The exact repository root, expected actual HEAD and clean Git status including
  untracked files, before and after validation. The candidate HEAD must have a
  verified signature and its author's DCO trailer. The full contributor range
  remains subject to the existing signed local review gate.
- Actual opened executable compiler/artifact files and their SHA-256 values.
  Regular files only, no leaf symlinks, with a 128 MiB development input cap.
  Reads are bounded and nonblocking; changed metadata, file length or replacement
  of the pathname during that read is rejected. Neither executable is run.
- One canonical UTF-8 transcript, capped at the consumer's existing 128 KiB,
  hashed from its actual opened bytes. The unchanged strict consumer requires
  externally bound identity, adopted profile, all 50 controls, normally completed
  parent, scalar trace semantics, original samples, journal and guest teardown.
  Unknown lines are rejected; this tool does not extract selected records from a
  full qualification log.

The Python `verify_candidate` function returns the consumer's sanitized record
plus `transcript_sha256` only for complete diagnostic evidence. A well-formed
unattributed, ambiguous, invalid or overflow trace raises `IncompleteDiagnostic`;
its `evidence` retains the validated scalar record with diagnostic completeness
false. Native acceptance stays false in every case. Invalid files, identities,
source or transcript raise `ValueError` without echoing raw input or Git output.

The offline CLI takes a transcript path, `--repo-root`, `--compiler`, `--artifact`,
`--source-commit`, `--compiler-sha256` and `--artifact-sha256`. Complete diagnostic
evidence prints sanitized JSON and exits 0. Well-formed incomplete evidence prints
its sanitized JSON and exits 1. Invalid inputs exit 2 without a result. These are
development arguments, not NAS configuration or runtime interfaces.

## Evidence and integration limits

Failing-first portable tests use disposable text files and synthetic executable
bytes that are never executed. They cover substituted digests, external identity,
unsafe/empty/oversized files, nonblocking FIFO rejection, mutation/path replacement
during a read, source/signature/DCO/cleanliness checks, a changed final source,
encoding/raw evidence and complete/incomplete/invalid CLI outcomes.

Clean signed source and matching files establish input identity. They do not
attest that this compiler produced that binary, provide an atomic snapshot across
all files, or establish actual native child execution. The forthcoming runner
must independently build from reviewed source under the pinned compiler, retain
the existing build receipt and private full log, pass canonical evidence without
dropping unknown parent lines, and verify independent guest/host cleanup. The
tagged fixture, source isolation/cross-build proof and full runner wiring remain
pending. This PR adds no runnable native suite or changes to qualification or
interruption cleanup.

All 25 preceding policy receipts and existing product, fixture, consumer and
qualification-runner sources remain unchanged. No VM, service, dependency,
deadline, native producer or runtime interface is added. The one adopted fresh
15.1 diagnostic remains pending after fixture/runner review. Both complete native
lifecycle gates, worker/backend/global launch, manual browser/client, hardware,
performance and release acceptance remain open; runtime writes stay disabled.
Historical failed, skipped and unverified evidence retains its original scope.

The [candidate receipt](deadline-provenance-candidate-manifest.json) records the
signed red/green source, mandatory actual-title review, immutable prior receipts
and owned-process teardown. Final receipt-only changes require fresh baseline
and verified exact-input deep reuse.

## Recorded candidate

Signed implementation `d61015cffc205047b9ccc893a7712fa177ea3926`, merge base `5599bd241c711c16a350ef0257f129387e420cfb`,
passed ten focused tests. Signed red `acc3f71b8b96cee186df067f18f8b8176ac8b981` reproduces
one missing-CLI failure and 25 absent-verification errors across nine test methods;
its disposable signed archive was removed. Three supplementary CLI cases verified
the real clean signed source and pinned compiler file, with a disposable synthetic
artifact never executed. Complete, incomplete and invalid inputs returned 0, 1
and 2 respectively. Both CLI fixtures and red archive were removed.

The actual-title `make review` passed 249 tooling tests, eleven portable race
packages and 54 compiler-listed fuzz targets under the shared-tooling fallback.
All 25 prior receipts and existing product/fixture/consumer/qualification-runner
sources remain byte-identical. The receipt records exact source and verified
supervisor teardown. Final receipt-only changes require fresh baseline and
verified exact-input deep reuse. Native build/fixture/runner integration and actual
diagnostic execution remain pending.
