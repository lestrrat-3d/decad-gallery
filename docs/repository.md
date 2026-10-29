# Repository layout and checks

| Path | Purpose |
| --- | --- |
| `models/<name>/` | One shape, pattern, or assembly per directory. |
| `models/README.md` | The gallery index with one row per model. |
| `docs/` | Developer documentation: [adding a model](adding-models.md), the [build pipeline](model-pipeline.md), the [renderer](rendering.md), and one page per model package under `docs/models/`. |
| `tools/render/` | Builds model images from Decad bodies. |
| `AGENTS.md` | Instructions for coding agents working on models. |
| `.github/` | Repository checks. |

The repository owns each item's human-readable construction instructions, Decad construction steps, Go
code, and geometry checks. The [model build pipeline](model-pipeline.md) describes this path.
[`fusion360-gear-generator`](https://github.com/lestrrat-3d/fusion360-gear-generator) is a downstream
consumer that translates the Go construction into Python code for Fusion add-ins.

The repository check runs `python3 scripts/check_repository.py`. It checks the foundation files and
requires a README, instructions, steps, hero image, Go source and tests for each item directory. Run
`go test ./...` for the models' geometry checks; CI runs both commands.

## Test selection in CI

Pull requests and pushes to `main` run `python3 scripts/ci/run_tests.py models pr`, a fast subset of the model
tests. The nightly schedule and manual runs use `models full` instead, which runs everything that the fast subset
and the bevel shards skip.

`TestRibbonDefaultIsOneVerifiedLoft` (the 681-section screw gear ribbon loft, about 15 seconds) is in the full run
only. A pull request that changes the `github.com/lestrrat-3d/decad` requirement in `go.mod` or
`tools/render/go.mod` also runs it, through `python3 scripts/ci/run_tests.py models decad-bump HEAD^1`. The
command compares the requirement against the pull request's base (`HEAD^1` in the merge checkout, which is why the
workflow fetches two commits) and does nothing when it is unchanged. Run the same command with any base ref to
check a bump locally. `python3 scripts/ci/test_run_tests.py` tests the requirement parsing.
