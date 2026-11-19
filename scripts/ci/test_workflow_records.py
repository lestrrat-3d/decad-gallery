"""Check record uploads without running the geometry suites."""

from pathlib import Path
import os
import re
import subprocess
import tempfile
import unittest


WORKFLOW = Path(__file__).resolve().parents[2] / ".github/workflows/repository.yml"


def job_section(workflow, job):
    return re.split(r"\n  [a-z][a-z0-9-]*:\n", workflow.split(f"  {job}:\n", 1)[1], maxsplit=1)[0]


class WorkflowRecordsTests(unittest.TestCase):
    def test_capacity_document_check_only_runs_for_changed_instructions(self):
        section = job_section(WORKFLOW.read_text(), "repository")
        step = section.split("      - name: Check clamp arm capacity documentation\n", 1)[1]
        step = step.split("\n      - ", 1)[0]
        self.assertIn("if: needs.plan.result == 'success' && "
                      "contains(fromJSON(needs.plan.outputs.plan).changed_paths, "
                      "'models/clamparm/instructions.md')", step)
        self.assertIn("run: go test ./internal/clamparmcapacity", step)

    def test_full_suite_records_have_unique_paths_and_required_uploads(self):
        workflow = WORKFLOW.read_text()
        jobs = {
            "repository": "repository",
            "renderer-full": "renderer-full",
            "escapement": "escapement",
            "onewayclutch": "onewayclutch",
            "clamparm": "clamparm",
            "turbinefan": "turbinefan-${{ matrix.shard }}",
            "wheel": "wheel-${{ matrix.case }}",
            "siren": "siren-${{ matrix.case }}",
            "bevel-full": "bevel-full-${{ matrix.shard }}",
            "planetary-full": "planetary-full-${{ matrix.case }}",
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
                if job in ("wheel", "siren", "bevel-full", "planetary-full", "turbinefan"):
                    case = "shard" if job in ("bevel-full", "turbinefan") else "case"
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
                         "turbinefan", "wheel", "siren", "bevel-full", "planetary-full"):
            self.assertIn(producer, section.split("\n", 1)[0])
        self.assertIn("pattern: ci-records-*", section)
        self.assertIn("python3 scripts/ci/inventory.py > inventory.json", section)
        self.assertIn("--execution-dir ci-artifacts --inventory inventory.json", section)
        self.assertIn('--revision "$GITHUB_SHA" --run-id "$GITHUB_RUN_ID"', section)
        self.assertIn('--run-attempt "$GITHUB_RUN_ATTEMPT"', section)
        self.assertIn("if: always()", section)
        self.assertIn("job[\"result\"] != \"success\"", section)
        self.assertIn("records-validation", job_section(workflow, "gate").split("runs-on:", 1)[0])

    def test_summary_upload_and_validator_exit_status(self):
        section = job_section(WORKFLOW.read_text(), "records-validation")
        command = re.search(r"        run: (python3 scripts/ci/reconcile_records.py .* )> validation-summary.json", section)
        self.assertIsNotNone(command)
        upload = section.split("      - name: Retain inventory and validation summary\n", 1)[1]
        upload = upload.split("\n      - ", 1)[0]
        self.assertIn("if: always()", upload)
        self.assertIn("name: ci-validation-summary", upload)
        self.assertIn("inventory.json", upload)
        self.assertIn("validation-summary.json", upload)
        self.assertIn("overwrite: true", upload)
        self.assertIn("if-no-files-found: warn", upload)
        scratch = WORKFLOW.parents[2] / ".tmp"
        scratch.mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=scratch) as directory:
            root = Path(directory)
            executable = root / "python3"
            env = dict(os.environ, PATH=f"{root}:{os.environ['PATH']}", GITHUB_SHA="revision",
                       GITHUB_RUN_ID="123", GITHUB_RUN_ATTEMPT="1")
            for status, value in ((0, "true"), (1, "false")):
                with self.subTest(status=status):
                    executable.write_text(f'#!/bin/sh\nprintf \'{{"ok": {value}}}\\n\'\nexit {status}\n')
                    executable.chmod(0o755)
                    result = subprocess.run(command.group(0).removeprefix("        run: "), shell=True, cwd=root,
                                            env=env, check=False)
                    self.assertEqual(result.returncode, status)
                    self.assertEqual((root / "validation-summary.json").read_text(), f'{{"ok": {value}}}\n')


if __name__ == "__main__":
    unittest.main()
