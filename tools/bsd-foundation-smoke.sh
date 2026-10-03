#!/bin/sh
# SPDX-License-Identifier: BSD-3-Clause
set -eu
[ "$(uname -s)" = "$1" ]
cd "$(dirname "$0")/.."
for script in tools/check-repo.sh .githooks/commit-msg .githooks/pre-commit; do
    sh -n "$script"
done
export GIT_AUTHOR_NAME='BSD smoke'
export GIT_AUTHOR_EMAIL='bsd-smoke@example.invalid'
message=$(mktemp)
trap 'rm -f "$message"' EXIT HUP INT TERM
printf '%s\n' 'docs: exercise native commit hook' > "$message"
if sh .githooks/commit-msg "$message"; then
    echo "Missing DCO was accepted" >&2
    exit 1
fi
printf '\n%s\n' 'Signed-off-by: BSD smoke <bsd-smoke@example.invalid>' >> "$message"
sh .githooks/commit-msg "$message"
echo "Native shell and DCO hook checks passed on $1"
