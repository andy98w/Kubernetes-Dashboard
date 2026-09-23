#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in staging|production) environment="$1";; *) exit 2;; esac
file="platform/releases/${environment}.json"
git add "$file"
if git diff --cached --quiet; then exit 0; fi
branch="codex/promote-${environment}-${GITHUB_RUN_ID}-${GITHUB_RUN_ATTEMPT}"
git switch -c "$branch"
git config user.name 'github-actions[bot]'
git config user.email '41898282+github-actions[bot]@users.noreply.github.com'
git commit -m "Promote ${environment} image digests"
git push origin "$branch"
gh pr create --base main --head "$branch" --title "Promote ${environment} release" --body "Immutable image promotion. Review the source revision and digests. Confirm staging health before production approval. No cluster credentials are used by this job."
# GITHUB_TOKEN-created PRs do not trigger pull_request workflows automatically.
gh workflow run ci.yml --ref "$branch"
