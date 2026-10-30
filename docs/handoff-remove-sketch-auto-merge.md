# Remove Sketch PR auto-merge

The user requested removal of Decad's Sketch PR auto-merge workflow on 2026-10-31.
Remove `.github/workflows/auto-merge-sketch-updates.yml` in `lestrrat-3d/decad`.
Keep the separate dependent-module version-sync workflow and its script unchanged.
Use the Decad repository's worktree, validation, PR, and CI-watch rules.
Report the PR and checks. Do not create a GitHub issue or comment for this handoff.

The existing Decad session opened [PR2929](https://github.com/lestrrat-3d/decad/pull/2929)
with only the workflow deletion. Its CI is running as of 2026-10-31; leave the PR unmerged.
