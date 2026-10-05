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

The gate always runs `make check lint test secrets go-deep-check`: content,
links/configuration, ShellCheck, formatting/vet, tooling/Go tests, Gitleaks, race
detection and all fourteen bounded fuzz targets. There are no prose-only skips.
A final identity check rejects changes to HEAD, tree or working files during the
run. These portable checks do not establish native FreeBSD or client behavior.

Private evidence is under ignored `.tools/review-runs/<session>/`: `checks.log`
and `result.json`, including the source/tree/base, title, verified contributor
range, command, outcome and process-supervisor teardown record. An interrupted
run remains unqualified. Check the PR head and actual PR title against the report;
rerun after any source/base/title change. Review and sanitize logs before sharing.
Never commit raw inventories, fixture credentials or private configuration.

Attach the tested source/base IDs, title, command and pass/fail summary in the PR.
Supply relevant native/client manifests separately. The maintainer reviews the
complete diff and remaining milestone gates; portable success does not waive
native, restoration, security, accessibility or release acceptance.

## Teardown

`make review` and `make browser-preview` use the same lease supervisor as the
local lab. `make lab-status` checks owned sessions and exits nonzero if a session
is running or its teardown cannot be verified. `make lab-stop` stops active owned
sessions and waits for verified teardown. See [lab ownership and limits](../testing/local-qualification.md#teardown).

The supervisor exists only while its foreground workload runs. Routine exit,
failure, timeout, Ctrl-C, SIGTERM and controller SIGKILL trigger termination of
the owned process group, with bounded escalation and a private result record.
No background service is installed. Processes deliberately escaping their group,
external applications launched by macOS services, loss of the supervisor itself,
or host power loss need separate recovery; unverified records never qualify.
