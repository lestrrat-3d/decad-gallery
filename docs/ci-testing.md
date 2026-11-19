# CI test selection

`.github/workflows/repository.yml` runs on pull requests, pushes to `main`, a daily schedule, and manual dispatch.
The `model-checks` job reports the aggregate gate result. Repository layout and CI Python tests run in the
`repository` job even when the selected worker matrix is empty.

## Pull requests and pushes

`scripts/ci/plan.py` compares the tested commit with the event base. It verifies the checkout and, for a pull
request, the merge commit's base parent. For Go changes it reads package and test imports from both revisions,
selects affected owners and production consumers, and stops propagation at test imports. Registered owners map to
model, feature, package, and renderer checks. Production changes can also select renderer compilation and named
renderer checks. The plan records selected suites and reasons.

Markdown-only changes select no worker suite. Changes to spinning wheel part PNGs select `wheel-parts`. New
registered package directories select broad checks and direct tests. Renderer Go changes select broad checks and
the full renderer suite. A change to either module's `go.mod` or `go.sum` selects `broad` and the `full` marker,
which runs the full root and renderer checks. Unknown shared inputs, unregistered owners, invalid revision information, or unavailable
metadata trigger broad fallback where the planner can safely dispatch it. A new Go directory that cannot be
classified when metadata is unavailable fails planning instead. The wheel README part-image inventory is checked
before planning and can also fail the plan.

`gate.py` turns the plan into a worker matrix and checks the plan, matrix, and expected job results. An empty matrix
must skip the worker; a nonempty matrix must succeed. Broad plans run the dedicated broad jobs. The gate rejects
missing, failed, cancelled, or unexpectedly skipped required jobs.

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

The manually dispatched `ci-probe.yml` uses `probe.py` to create an empty plan and a mocked wheel part image change.
It exercises an empty matrix, a selected mock worker, and simulated failed, cancelled, and missing worker results.
The failed worker intentionally exits with failure. The probe does not run geometry or demonstrate an actual changed
file passing through the production workflow.

Test inventories describe discoverable Go tests, not every parameter branch or geometry property. Selected checks
depend on registered mappings and import metadata. Broad fallback does not make an unclassified new package directly
testable. See [CI measurements](ci-measurements.md) for observed run times and their limits.
