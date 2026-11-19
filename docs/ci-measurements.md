# CI measurements

These GitHub Actions runs completed on 19 November 2026. Durations are observations, not forecasts.

| Run | Change | Successful / skipped jobs | Elapsed | Initial wait | Sum of job waits | Runner seconds |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| [1201](https://github.com/lestrrat-3d/decad-gallery/actions/runs/44468505211) | PR #337 | 31 / 3 | 33m 05s | 7m 22s | 2,723s | 8,057s |
| [1203](https://github.com/lestrrat-3d/decad-gallery/actions/runs/44472525165) | Main after #337 | 31 / 3 | 27m 18s | 3s | 3,722s | 8,456s |
| [1204](https://github.com/lestrrat-3d/decad-gallery/actions/runs/44472791618) | PR #338 | 34 / 3 | 17m 37s | 24s | 11,218s | 7,848s |
| [1207](https://github.com/lestrrat-3d/decad-gallery/actions/runs/44474984127) | Main after #338 | 34 / 3 | 13m 09s | 3s | 2,615s | 8,077s |
| [1213](https://github.com/lestrrat-3d/decad-gallery/actions/runs/44481639445) | PR #339, CI-only changes | 4 / 15 | 1m 13s | 3s | 7s | 64s |

Elapsed time is the run's `updatedAt` minus `createdAt`. Initial wait ends when the first successful job starts.
Job wait is `started_at` minus `created_at`; runner time is `completed_at` minus `started_at`. The sums include
successful jobs only. Jobs overlap, so neither sum is elapsed time. Initial wait can include workflow-level waiting
before GitHub creates a job. Runner time is not billed minutes.

In run 1203, `bevel-rest` took 23m 14s. In run 1207, the longest `bevel-rest` shard took 6m 29s. PR #338 split
the remainder into four jobs. PR #337 added selection and inventory checks, so earlier runs do not establish the same
coverage. The runs above use different revisions and conditions; they are not a controlled same-revision comparison.
Successful jobs alone do not prove every intended test or parameter branch ran.

Run 1213's plan classified the changes as presentation or CI-only and selected no model suites. The `plan`,
`repository`, `gate`, and `model-checks` jobs succeeded; GitHub recorded 15 skipped job entries, including unexpanded
matrix jobs. Its 1m 13s elapsed time was 11m 56s less than run 1207's 13m 09s. The runs used different revisions,
selected workloads, and conditions, so this difference is not a controlled estimate of the selection's savings.
