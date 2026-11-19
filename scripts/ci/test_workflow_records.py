"""Check record uploads without running the geometry suites."""

from pathlib import Path
import re
import unittest


WORKFLOW = Path(__file__).resolve().parents[2] / ".github/workflows/repository.yml"


def job_section(workflow, job):
    return re.split(r"\n  [a-z][a-z0-9-]*:\n", workflow.split(f"  {job}:\n", 1)[1], maxsplit=1)[0]


class WorkflowRecordsTests(unittest.TestCase):
    def test_full_suite_records_have_unique_paths_and_required_uploads(self):
        workflow = WORKFLOW.read_text()
        jobs = {
            "repository": "repository",
            "renderer-full": "renderer-full",
            "escapement": "escapement",
            "onewayclutch": "onewayclutch",
            "clamparm": "clamparm",
            "turbinefan": "turbinefan",
            "wheel": "wheel-${{ matrix.case }}",
            "siren": "siren-${{ matrix.case }}",
            "bevel-full": "bevel-full-${{ matrix.shard }}",
        }
        for job, record in jobs.items():
            with self.subTest(job=job):
                section = job_section(workflow, job)
                self.assertIn("CI_TEST_RECORDS:", section)
                self.assertIn(f"name: ci-records-{record}", section)
                self.assertIn(f"path: {record}.jsonl", section)
                self.assertIn("if-no-files-found: error", section)
                self.assertIn("overwrite: true", section)
                self.assertIn("if: always()", section)
                if job in ("wheel", "siren", "bevel-full"):
                    case = "shard" if job == "bevel-full" else "case"
                    self.assertIn(f"CI_TEST_CASE: ${{{{ matrix.{case} }}}}", section)

    def test_shared_broad_jobs_only_upload_records_for_full_runs(self):
        workflow = WORKFLOW.read_text()
        for job in ("escapement", "onewayclutch", "clamparm", "turbinefan", "wheel", "siren"):
            with self.subTest(job=job):
                section = job_section(workflow, job)
                self.assertIn("github.event_name == 'schedule'", section)
                self.assertIn("github.event_name == 'workflow_dispatch'", section)
                self.assertIn("contains(fromJSON(needs.plan.outputs.plan).suites, 'full')", section)
                self.assertIn("|| ''", section)

    def test_validation_collects_full_run_records_and_checks_producers(self):
        workflow = WORKFLOW.read_text()
        section = job_section(workflow, "records-validation")
        for producer in ("repository", "renderer-full", "escapement", "onewayclutch", "clamparm",
                         "turbinefan", "wheel", "siren", "bevel-full"):
            self.assertIn(producer, section.split("\n", 1)[0])
        self.assertIn("pattern: ci-records-*", section)
        self.assertIn("python3 scripts/ci/inventory.py > inventory.json", section)
        self.assertIn("--execution-dir ci-artifacts --inventory inventory.json", section)
        self.assertIn('--revision "$GITHUB_SHA" --run-id "$GITHUB_RUN_ID"', section)
        self.assertIn('--run-attempt "$GITHUB_RUN_ATTEMPT"', section)
        self.assertIn("if: always()", section)
        self.assertIn("job[\"result\"] != \"success\"", section)
        self.assertIn("records-validation", job_section(workflow, "gate").split("runs-on:", 1)[0])


if __name__ == "__main__":
    unittest.main()
