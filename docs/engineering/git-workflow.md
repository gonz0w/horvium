# Git workflow

Trunk-based development: releasable `main`, short-lived branches, reviewed PRs,
squash merge, and merged-branch cleanup. Use Conventional-Commit-style titles:
`<type>(<scope>): <imperative summary>`. Types: feat, fix, docs, test, refactor,
perf, build, ci, chore, revert. Useful scopes: storage, shares, identity, protection,
peer, jobs, host, api, cli, console, ports, docs, release.

Canonical commits and annotated release tags are signed. Contributors include
an author DCO `Signed-off-by` trailer; signatures and sign-offs serve different
purposes. Configure signing per clone, keep private keys outside the repo, and
verify signatures before publishing. Preserve contributor sign-offs in squash
messages. Never silently disable signing to make a commit succeed.

## Remote protections

The versioned main ruleset requires PRs, signed commits, linear history, resolved
conversations. Required Actions status checks are removed under the local review
policy; Actions are disabled for the repository. Block force pushes and deletion. Release tag rules prevent moving/deleting `v*` tags; signed tags remain
a release check. Do not assume merely storing a ruleset JSON applies it remotely.

During the solo phase, independent approval count is zero and mandatory CODEOWNERS
approval is off. The maintainer deliberately reviews/merges; agents leave PRs
open. Require independent approval when a distinct reviewer joins. CODEOWNERS
must name confirmed accounts before remote bootstrap is complete. No auto-merge
or blanket bypass is adopted. Require source-bound [local review evidence](local-review.md)
and relevant native/client manifests before merging; GitHub does not independently
enforce those local results.

Initial bootstrap can seed `main` before protections are activated. Subsequent
changes use PRs. Record remote settings and checks actually observed in bootstrap
status, including subscription-dependent security features that are unavailable.
