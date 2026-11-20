# CI test selection

`.github/workflows/repository.yml` runs on pull requests, pushes to `main`, a daily schedule, and manual dispatch.
The `model-checks` job runs the aggregate checks. Repository layout and CI Python tests run in the
`repository` job even when the selected worker matrix is empty. That job also checks the clamp-arm capacity table
against model dimensions without building geometry.

## Pull requests and pushes

`scripts/ci/plan.py` compares the tested commit with the event base. It verifies the checkout and, for a pull
request, the merge commit's base parent. For Go changes it reads package and test imports from both revisions,
selects affected owners and production consumers, and stops propagation at test imports. Registered owners map to
model, feature, package, and renderer checks. Production changes can also select renderer compilation and named
renderer checks. The plan records selected suites and reasons.

Markdown-only changes select no worker suite. Changes to spinning wheel part PNGs select `wheel-parts`. New
registered packages select their tests and affected consumers. New unregistered packages select broad checks and
direct tests alongside affected known packages. Renderer Go changes select the full renderer suite. A change to either
module's `go.mod` or `go.sum` selects `broad` and the `full` marker, which runs the full root and renderer checks.
Unknown shared inputs select broad checks alongside affected known packages. Deleted packages, unregistered existing
owners, and planning errors select full coverage. A new Go directory that cannot be classified when metadata is
unavailable fails planning instead. The wheel README part-image inventory is checked before planning and can also fail
the plan. Suites already covered by selected broad or full jobs are not dispatched twice.

Test-only changes in model, internal, feature, and renderer files marked `// CI: isolated-tests` run the changed
top-level tests individually. The spinning wheel's coplanar tests run with their required build tag.
Both revisions must carry the marker. CI checks that marked files contain only imports and test functions, and rejects
references to other test functions inside them. Tests in marked files must not depend on another test's side effects.
New test files in these directories containing tests must carry the marker. Helper-only files do not need it.
A private test predicate can opt in with a `// CI: isolated-predicate` doc comment. Give it built-in scalar
parameters and one boolean return expression using only its parameters, literals, comparisons, signs, and boolean
operators. Its result may only guard a single `testing.T.Errorf` call with scalar diagnostic values. Every test
in the package must follow the isolated-test rule above. A helper must neither store a test callback nor depend
on another test's state. Tests must exercise a predicate through Go calls and must not inspect its source or binary.
Put one marked predicate in a file.

A change confined to one marked predicate body runs every isolated top-level test that calls it through private
test helpers. CI examines both revisions, type-checks the tested package, and follows direct calls across files
and inside `testing.T.Run` callbacks. Other changes in the same Go diff, indirect calls, uses from initializers
or methods, build constraints, and packages with a `TestMain` use the normal package selection.

Changes to imports, other helpers, compiler directives, deleted tests, or unmarked files use the normal
package selection. Production and dependency changes use the normal selection too.

`gate.py` turns the plan into a worker matrix and checks the plan, matrix, and expected job results. An empty matrix
must skip the worker; a nonempty matrix must succeed. Broad plans run the dedicated broad jobs. The gate rejects
missing, failed, cancelled, or unexpectedly skipped required jobs.

The planetary PR suite runs one dimensions check and balances the remaining discovered tests across twenty jobs.
Checks that read the same gearbox share a test process. The three default planet mesh checks run on separate workers.
Cost estimates only affect placement. Each job checks its assigned test results. The kit test keeps its existing
allowed skips unless full kits are enabled.
Renderer PR checks partition the discovered package tests across sixteen jobs. Each job requires its assigned tests
and their subtests to pass. Scheduled and manual runs keep the complete renderer job.
The fan PR suite runs one validation test and partitions the remaining discovered tests across nine jobs.
Its two on-demand tests keep their existing allowed skips unless full fan testing is enabled.

As of 2026-11-19, the active `require-model-checks` ruleset requires the `model-checks` status check on the default
branch, `main`. It allows repository administrators to bypass the rule. The rule does not require the branch to be
up to date before merging. See the [ruleset](https://github.com/lestrrat-3d/decad-gallery/rules/27619075).

## Full runs

Scheduled and manually dispatched runs select broad checks and use the full path. Dependency file changes use the
same full path through the `full` marker. `full_partition.py` lists root-module packages and
top-level tests, runs the packages outside dedicated jobs, and runs unassigned bevel tests. Dedicated jobs cover
their packages and the eight full bevel shards. Wheel and siren jobs run assigned cases plus a remainder that skips
assigned paths; the remainder checks for overlapping assignments and missing top-level tests. The runner requires
listed test results and explicitly checks the configured fan and planetary variants. `renderer-full` lists renderer
packages and tests, then requires their pass events and rejects skipped or failed subtests of listed tests. Full runs
skip the separate PR renderer jobs because `renderer-full` runs those renderer tests.

## Probe and limits

Set the repository Actions variable `CI_BROAD_OVERRIDE` to `true` to select broad PR and push coverage.
Delete the variable or set it to any other value to restore change-aware selection. The override preserves full
coverage for dependency changes and every originally selected suite. Scheduled and dispatched
runs remain full. The `model-checks` gate keeps its name and checks the selected jobs. Variable changes
affect newly started runs; rerun or start a run to apply a change.

The manually dispatched `ci-probe.yml` uses `probe.py` to create an empty plan and a mocked wheel part image change.
It exercises an empty matrix, a selected mock worker, and simulated failed, cancelled, and missing worker results.
The failed worker intentionally exits with failure. The probe does not run geometry or demonstrate an actual changed
file passing through the production workflow.

In [probe run 44481894010](https://github.com/lestrrat-3d/decad-gallery/actions/runs/44481894010), the empty worker
was skipped, the selected mock worker succeeded, and the intentionally failed worker failed. The verification job
passed its gate checks for those actual job results and for simulated cancelled and missing worker results. The
overall run concluded failure because the failed worker intentionally exited with failure.

Test inventories describe discoverable Go tests, not every parameter branch or geometry property. Selected checks
depend on registered mappings and import metadata. Broad fallback does not make an unclassified new package directly
testable. See [CI measurements](ci-measurements.md) for observed run times and their limits.


## Reusing a pull request run on main

After a squash merge, the main push can reuse a successful pull request's selected suites.
The planner checks the merged PR, its successful first workflow attempt, its checkout log, and the plan and
model-checks jobs. It verifies the successful "Check planned results" step or the separate gate job in older runs.
The tested merge must have the push base and PR head as its parents. Its tree, changed paths,
and selected suites must match the main push. The push plan records the PR number, run ID, tested commit and suites.

The repository and aggregate checks still run on main. Direct pushes, broad or full plans, CI changes, and
missing or inconsistent evidence run the usual push checks. Set `CI_BROAD_OVERRIDE` to `true` to force broad coverage.
