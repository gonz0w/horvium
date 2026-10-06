# Local review gate

GitHub Actions are disabled for this repository. Review evidence is produced on
an operator-controlled development machine and reviewed with the PR. There is
no hosted fallback, self-hosted Actions runner, scheduled daemon or fabricated
GitHub success status. GitHub cannot independently enforce these local results;
maintainers must inspect the evidence before merging. Local hooks are helpful
but can be bypassed and are not a substitute for that review.

## Run and review

Install the pinned development tools with `make tools` and enable the existing
commit hooks with `make hooks`. Commit with a verified signature and author DCO
trailer, then use a clean checkout:

```sh
PR_TITLE='ci: move qualification to local hosts' make review
```

Set `BASE_REF` when the intended review base differs from `origin/main`. Fetch
main before choosing the base; the gate does not contact GitHub. It resolves the
base to a full commit ID, requires it to be an ancestor, verifies every
contributor commit signature and author DCO trailer, and checks the actual
proposed PR title. Unsigned, empty, dirty or divergent ranges fail before tests.

Every review runs `make check lint test secrets`: content, links/configuration,
ShellCheck, formatting/vet, all ordinary tooling/Go tests and Gitleaks. These
baseline checks stay inexpensive and prevent unrelated regressions from silently
passing. A final identity check rejects changes to HEAD, tree or working files.

`make review` then chooses a conservative plan from the complete base-to-head
diff. Renames include both old and new paths; deleted and FreeBSD-only inputs
are included. Changes in a Go package, its tests, testdata or embedded assets
select that package and every transitive consumer using the union of host and
FreeBSD imports, test imports and external test imports. Selected packages run
race detection plus **every compiler-listed portable fuzz target**, each bounded
at three seconds/two workers. The inventory is discovered rather than manually
maintained. Shared tooling/hooks, modules, policy, packaging and unmapped inputs
fall back to all portable packages/targets. Selection is a cost policy, not proof
that other behavior is unaffected; native safety/recovery needs separate review.

```sh
# Small development loop; fails if no behavioral tests match.
make test-focus PKG=./internal/storage TEST='^TestOwnershipRecordRoundTripAndCanonicalBytes$'
# Fresh complete portable checks at milestones/releases or when requested.
PR_TITLE='ci: focus local testing on affected changes' make review-full
```

### Documentation and exact-input reuse

A range containing only eligible prose/evidence needs baseline checks without a
new race/fuzz run. Eligible paths are README/NOTICE/CHANGELOG and documentation
Markdown or `*-manifest.json`; review policy files are excluded. Nothing else is
classified as prose. Within a behavioral PR, a followup may reuse **previously
executed**, successful schema-2 local deep evidence only when all non-prose tracked
blob IDs/modes, selected mode/packages, resolved base, host/toolchain/compiler
settings and behavioral environment match. The tested commit must be an ancestor
of the new head. The original deep log must still match its recorded SHA-256,
and its supervisor must report successful execution and completed clean teardown.
Missing, failed, interrupted, legacy, altered, different-base or mismatched evidence
triggers execution. Reuse does not chain through another reused report.
`make review-full` always executes fresh deep checks.

Every followup still checks its current title/signatures/DCO and clean source,
and reruns the entire cheap baseline. The report explicitly distinguishes actual
execution, a documentation-only plan and reused evidence with its original source,
log/report hashes and result. This is developer-controlled local evidence, not
independent attestation; maintainers inspect the source and original record.
Changing reviewed policy/verification tools invalidates reuse.

Private evidence is under ignored `.tools/review-runs/<session>/`: `checks.log`,
`deep.log` when executed, and `result.json`. It includes source/tree/base/title,
verified contributor range, changed paths, plan, verification-input digest,
commands/results and process-supervisor teardown records. Interrupted runs stay
unqualified. Check the current PR head/base/title against the new report; earlier
logs remain bound to their actual executed source. Review/sanitize logs before
sharing; never commit inventories, fixture credentials or private configuration.
These portable checks do not establish native FreeBSD or client behavior.

Attach the tested source/base IDs, title, command and pass/fail summary in the PR.
Supply relevant native/client manifests separately. The maintainer reviews the
complete diff and remaining milestone gates; portable success does not waive
native, restoration, security, accessibility or release acceptance.

## Teardown

`make review`, `make review-full`, `make test-focus` and `make browser-preview` use the same lease supervisor as the
local lab. `make lab-status` checks owned sessions and exits nonzero if a session
is running or its teardown cannot be verified. `make lab-stop` stops active owned
sessions and waits for verified teardown. See [lab ownership and limits](../testing/local-qualification.md#teardown).

The supervisor exists only while its foreground workload runs. Routine exit,
failure, timeout, Ctrl-C, SIGTERM and controller SIGKILL trigger termination of
the owned process group, with bounded escalation and a private result record.
No background service is installed. Processes deliberately escaping their group,
external applications launched by macOS services, loss of the supervisor itself,
or host power loss need separate recovery; unverified records never qualify.
