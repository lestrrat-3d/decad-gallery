# Decad dependency updates

Run `python3 scripts/update_decad.py` from a checkout to update both Go modules.
The command asks Go to resolve Decad's `main` branch directly from its VCS source once, then uses that version for the root
and `tools/render` modules. Go updates and tidies both `go.mod`/`go.sum` pairs.
Only the revision lookup uses `GOPROXY=direct`; immutable module downloads retain the caller's proxy configuration.
The command verifies both declared and selected versions, rejects local Decad replacements, and checks both checksums.
If a Go command or verification fails, it restores all four manifests to their previous contents.
Use `--revision <published-commit-or-version>` to reproduce a specific dependency update.

This command also bootstraps an update on an existing feature branch before the automation is on `main`.
It does not require a manually written pseudo-version or an already running bot.

## Automatic pull requests

The `Update Decad modules` workflow runs weekly on Monday at 02:00 UTC and accepts manual workflow dispatches.
It also accepts the repository-dispatch event `decad-main-updated` for an upstream notifier.
The workflow always resolves the published `main` branch; it does not execute or interpolate dispatch payloads.
The local command and workflow use the same updater.

The workflow opens or updates one draft pull request, `chore-update-decad`, containing both module manifest pairs.
It explicitly dispatches repository checks on the generated head because a pull request created with `GITHUB_TOKEN`
does not trigger ordinary pull-request workflows.
It does not enable auto-merge.

The workflow becomes available after its configuration reaches the default branch.
Its next scheduled run, a manual dispatch, or an upstream repository dispatch generates the first dependency pull request.
GitHub Actions must be allowed to create pull requests in repository or organization settings.
The configuration requests write permissions only for its updater job.

On 2026-11-02 the user chose to keep Actions pull-request creation disabled.
The scheduled workflow cannot open pull requests under that policy.
The updater does not change this setting or use alternate credentials to bypass it.
The local command still updates an existing feature branch's manifests.

Module changes invalidate the spur generation record. The draft must receive a fresh model emission through
[the model pipeline](model-pipeline.md#3-emit-go-from-the-checked-steps) and pass the existing checks before merging.
The updater does not rewrite generation hashes or claim that unchanged generated code was emitted again.

## Untagged revisions

Decad currently publishes untagged commits as Go pseudo-versions.
Dependabot recognizes pseudo-version syntax, but its Go version finder enumerates tagged module versions
and keeps the existing version when that list is empty. It therefore cannot discover Decad's newer untagged commits.
The repository-owned updater resolves `@main` through Go instead.

The relevant primary sources are GitHub's
[Dependabot options](https://docs.github.com/en/code-security/reference/supply-chain-security/dependabot-options-reference),
the official [Go module version fetcher](https://github.com/dependabot/dependabot-core/blob/main/go_modules/lib/dependabot/go_modules/package/package_details_fetcher.rb),
and the [pull-request action's workflow guidance](https://github.com/peter-evans/create-pull-request#token).
