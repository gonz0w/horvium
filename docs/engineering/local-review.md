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

Ordinary product and documentation review runs `make check lint test secrets`:
content, links/configuration, ShellCheck, formatting/vet, ordinary tooling/Go tests
and Gitleaks. The finite [Temper and validation-tooling route](#scope-reconciliation-for-424)
below uses its own mandatory checks. A final identity check rejects changes to
HEAD, tree or working files.

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

Disposable foundation Git repositories disable automatic maintenance and garbage
collection in their local fixture configuration. This keeps commit commands from
launching detached work that can race temporary-directory teardown; normal
operator Git configuration is unchanged. A trace regression checks that two
commits under an inherited aggressive maintenance policy start no automatic
maintenance child. Cleanup errors remain failures, without deletion retries.


## Scope reconciliation for #424

Cam's direct 2026-10-25 decision for [#424](https://github.com/gonz0w/horvium/issues/424)
replaces blanket product qualification for process/configuration-only changes.
[PROJECT](../../PROJECT.md#verification-and-review) records the local old decision,
replacement, authority and limits. Existing immutable upstream pins are unchanged.
Classified portable checks may run once on a permitted development host under
the selected [host and target policy](../policy/verification.md#portable-checks-and-target-verification). Horvium runtime
support and actual target-specific kernel/process behavior remain FreeBSD.

The implementation distinguishes pure Temper metadata from affected validation
tooling. The original #424 transition was accepted before this repair. The
[#433 verifier repair](https://github.com/gonz0w/horvium/issues/433) updates its stale
source admission and selector for the separately reviewed #434 policy adoption.
Source preparation and mocked dispatch results do not establish execution of a
new candidate or renew the historical transition authority. Historical failures
retain their original scope and qualification. Pre-commit remains the unchanged
`make check secrets`; signature, actual-author DCO, title, clean range, source
identity and applicable custody/teardown requirements remain mandatory.

### Required mapping

| Impact | Applicable checks |
| --- | --- |
| Reviewed retained Temper bundle and selected local roles, optionally with eligible prose | `make check secrets`, then `tools/check-temper-source.py`; no product Go/compiler query or deep reuse. |
| Separately reviewed exact range of the named validation tooling and affected local policy below | The same metadata commands, then `tools/temper-regressions.py`; no product Go/compiler query or deep reuse. |
| Unbound, stale, ambiguous, additional or unknown policy/tooling input; product/shared execution/hooks/modules/packaging | Existing conservative full/focused behavior and applicable native checks. |
| Explicit `--full` / `make review-full` | Existing fresh complete portable baseline and deep checks, regardless of a supplied scope. |

Pure Temper selection derives the complete actual bundle and role inventory from
the same admitted reader used by the source verifier. Candidate inventories do
not enlarge that scope before admission; invalid admission keeps conservative
full review. Unknown `.codex` paths, root `policy.toml` and every `docs/policy/`
path stay outside prose and pure Temper selection, including mixed role/policy
changes. Policy bytes remain in behavioral fingerprints.

`REVIEWED_ADMISSION_SHA256` in the trusted source verifier binds the exact reviewed
#434 input at consumer `02b54adf162658749a09718f05ff8b3fee8e58cd`. Its canonical
JSON framing has a domain tag and sorted relative path/type/SHA-256 records for
complete `.agent-temper` and `.codex` inventories and contents, root `policy.toml`,
all selected module bytes and `AGENTS.md`. The reader rejects empty directories,
links, executable metadata and special files before content reads in the known
inventories. It structurally validates root selection and checks normalized,
contained module paths and their ancestors before opening them. Nothing declared
in policy executes.

Only after aggregate admission does it validate the retained source revision/URL,
complete actual manifest and retained license, role TOML fields and requested
settings/instructions, and module identity/path/prerequisite consistency. The
single anchor binds the exact reviewed values without duplicating upstream pins,
model tables, instruction digests or a role count. Candidate manifest checksums
alone cannot admit new bytes. There is no CLI/environment override, Git-object
requirement or automatic digest refresh. The verifier itself stays outside its
own aggregate; review must preserve trust in its code and anchor.

Every covered change, including a tiny prose edit in selected policy, requires
explicit reviewed readmission. This conservative maintenance cost prevents a
candidate from expanding its own source or policy boundary. Separate semantic
policy review remains required. Parsing and hashes do not prove actual client
role discovery, model execution or native product behavior.

### Source-bound validation-tooling route

A separate reviewer examines the complete candidate and supplies an external
JSON binding to the lead. Pass its path with `REVIEW_SCOPE` when using `make review`,
or `--scope` when invoking the gate directly with `.tools/venv/bin/python3`
from the candidate root. The file must be outside the
candidate with no symlink components. It is bounded to one MiB, strict standard
JSON, and duplicate keys are rejected. No file inside a candidate can declare
its own exemption. Possessing this developer-controlled JSON is source binding,
not a cryptographic independent-review attestation; inspect the linked actual
review and exact inputs before relying on it.

The object admits exactly these fields:

| Field | Required value |
| --- | --- |
| `schema` | Integer `1` |
| `route` | `temper-validation-tooling-v1` |
| `base_commit`, `source_commit`, `source_tree` | Actual complete forty-hex source identities, matching the clean signed range |
| `changed_entries` | Complete path-sorted raw Git diff entries, with `path`, `old_mode`, `new_mode`, `old_blob`, `new_blob`, `status`; no duplicates or omitted additions/deletions |
| `commands` | Exactly `["metadata", "temper-source", "review-regressions"]` |
| `review_reference` | The separate review's `https://github.com/gonz0w/horvium/issues/424#issuecomment-` or `https://github.com/gonz0w/horvium/issues/433#issuecomment-` URL and numeric comment ID |

`git diff --raw --no-abbrev --no-renames -z BASE HEAD` supplies the entry identity;
renames remain deletion plus addition. Only ordinary nonexecutable file modes
or absent entries are eligible. The closed inventory never executes argv from
JSON. Command identifiers map in repository code to `make check secrets`, the
named source verifier and the named affected regression runner, in that order.

Eligible paths are the admitted pure Temper/prose paths plus AGENTS, PROJECT, CONTRIBUTING,
root `policy.toml`, and the exact adopted `docs/policy/` modules `work.md`,
`product.md`, `verification.md`, `authority.md`, `documentation.md`, `routing.md`
and `history.md`,
`docs/engineering/github-workflow.md`, this review contract,
`docs/compatibility/matrix.md`, `tools/review-plan.py`, `tools/review-gate.py`,
`tools/check-temper-source.py`, `tools/temper-regressions.py`, the exact manual
checker source record `tools/temper/SOURCE.md`, and these seven exact
regression modules: `test_review_plan`, `test_review_gate`,
`test_temper_review_scope`, `test_temper_gate_dispatch`,
`test_temper_scope_binding`, `test_temper_bound_dispatch`, `test_temper_source`.
Other policy descendants are ineligible. A #424 review URL does not authenticate
new bytes, revive expired budgets or waive stale/incomplete binding checks.
The runner loads each named module explicitly and rejects missing, empty, failed
or skipped tests. The gate requires one complete successful result with exactly
that module inventory before qualification. Tools outside that list, especially
`owned-process.py`, hooks, Makefile, content checker, bootstrap and OS adapters,
cannot enter through a binding. Unknown or invalid initial binding keeps the
ordinary conservative route. A bound scope that changes during review rejects
qualification; it does not switch routes mid-run.

Both metadata routes require that exact candidate venv interpreter, its executable
baseline and matching venv prefix. Parent-directory aliases of that candidate
are equivalent; the exact `python3` entry name and canonical venv directory
remain required. Another link to the same base Python outside that venv fails.
This keeps `make check` on the same interpreter
and dependency search path as the gate; a different direct interpreter fails.
The fingerprint includes venv configuration, actual Python/Git/make/shell/Gitleaks bytes, PyYAML
package bytes, relevant interpreter/environment settings, source checker/test
inputs and scope digest. Final checks reject changed source, scope, tests, tools
or verification inputs. Their records identify the narrower qualification scope
and do not reuse product deep evidence. Source/tool hashes detect drift; exact
tool admission is still a prerequisite. No tool installation or new execution
authority follows from supplying a binding.

### Verify the transition without self-selection

Choose the seven affected regression modules explicitly from this reviewed
contract and frozen source, independently of the new selector. Preserve the
original ten selector assertions, real Git identity negatives and unbound
policy/tooling/shared/product/full fallbacks. The existing gate regressions
preserve clean source, signature, actual-author DCO, title, ancestor and
nonempty-range checks. Added tests cover exact bindings, conservative refusal,
fixed dispatch, skipped/missing/failing regression results, command/teardown
failure and final source/input/scope drift. Use the same gate-entry assertions
against preserved old and new source for meaningful causal RED/GREEN; missing
new helper modules are not causal RED. Record actual source/test/tool identities,
raw outputs and status distinctly in #424. Tests written in a candidate are not
an execution claim.

Pure selector/data tests need Python. Real Git identity fixtures additionally
need admitted Git. Content/configuration and unchanged hooks/secrets need pinned
PyYAML/Gitleaks. The source verifier and seven affected regressions require a permitted
unprivileged POSIX development host and pinned Python 3.14.7 after exact effects
inspection under the lasting October 31 policy. The real gate still requires its
exact candidate venv; using another admitted interpreter for direct source/data
tests does not pass that gate identity check. Actual unchanged OwnedProcess lease/kernel tests require
their own finite admission on the relevant target; mocked dispatch cannot
qualify real control sockets, sessions, process groups, cleanup or escalation.
OpenBSD observations do not establish FreeBSD target-specific behavior. Preserve
FreeBSD runtime/native gates, both declared release targets and milestone full
review. No new executor, retry, credential, permission or resource authority is
granted by this contract.
