"""Check record uploads without running the geometry suites."""

from pathlib import Path
import os
import json
import re
import subprocess
import tempfile
import unittest

import gate
import reconcile_records


WORKFLOW = Path(__file__).resolve().parents[2] / ".github/workflows/repository.yml"


def job_section(workflow, job):
    return re.split(r"\n  [a-z][a-z0-9-]*:\n", workflow.split(f"  {job}:\n", 1)[1], maxsplit=1)[0]


class WorkflowRecordsTests(unittest.TestCase):
    def test_sevensegment_cases_match_record_validator_and_dispatch(self):
        section = job_section(WORKFLOW.read_text(), "sevensegment")
        matrix = section.split("        shard:\n", 1)[1].split("    name:", 1)[0]
        cases = re.findall(r"          - ([a-z-]+)", matrix)
        self.assertEqual(tuple(cases), reconcile_records.FULL_JOBS["sevensegment"])
        self.assertEqual(tuple(cases), gate.SEVENSEGMENT_CASES)

    def test_swiss_cases_match_record_validator(self):
        section = job_section(WORKFLOW.read_text(), "swisslever-full")
        cases = re.search(r"case: \[([^]]+)\]", section).group(1).split(", ")
        self.assertEqual(tuple(cases), reconcile_records.FULL_JOBS["swisslever-full"])

    def test_renderer_shard_matrix_matches_record_validator(self):
        section = job_section(WORKFLOW.read_text(), "renderer-full")
        values = re.search(r"shard: \[([^]]+)\]", section).group(1).split(", ")
        expected = [str(index) for index in range(gate.RENDER_FULL_SHARDS)]
        self.assertEqual(values, expected)
        self.assertEqual(reconcile_records.FULL_JOBS["renderer-full"], tuple(expected))

    def test_package_matrix_is_only_created_and_read_for_full_runs(self):
        section = job_section(WORKFLOW.read_text(), "package-workers")
        self.assertIn("if: needs.plan.result == 'success'", section)
        self.assertIn("matrix: ${{ fromJSON(needs.plan.outputs.package_matrix) }}", section)
        plan = job_section(WORKFLOW.read_text(), "plan")
        self.assertIn("package_matrix: ${{ steps.package-matrix.outputs.matrix }}", plan)
        self.assertIn("python3 scripts/ci/full_partition.py package-matrix", plan)
        for expression in ("github.event_name == 'schedule'", "github.event_name == 'workflow_dispatch'",
                           "contains(fromJSON("):
            self.assertIn(expression, section)
            self.assertIn(expression, plan)
        self.assertIn("contains(fromJSON(steps.plan.outputs.plan).suites, 'full')", plan)
        self.assertIn("contains(fromJSON(needs.plan.outputs.plan).suites, 'full')", section)

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
            "package-workers": "package-workers-${{ matrix.case }}",
            "renderer-full": "renderer-full-${{ matrix.shard }}",
            "escapement": "escapement",
            "onewayclutch": "onewayclutch",
            "clamparm": "clamparm",
            "turbinefan": "turbinefan",
            "wheel": "wheel-${{ matrix.case }}",
            "siren": "siren-${{ matrix.case }}",
            "bevel-full": "bevel-full-${{ matrix.shard }}",
            "planetary-full": "planetary-full-${{ matrix.case }}",
            "bevel-remainder": "bevel-remainder",
            "handcrankfan-full": "handcrankfan-full-${{ matrix.case }}",
            "swisslever-full": "swisslever-full-${{ matrix.case }}",
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
                if job in ("wheel", "siren", "bevel-full", "planetary-full", "renderer-full", "handcrankfan-full", "swisslever-full", "package-workers"):
                    case = "shard" if job in ("bevel-full", "renderer-full") else "case"
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
                         "turbinefan", "wheel", "siren", "bevel-full", "planetary-full",
                         "bevel-remainder", "handcrankfan-full", "swisslever-full"):
            self.assertIn(producer, section.split("\n", 1)[0])
        self.assertIn("pattern: ci-records-*", section)
        self.assertIn("python3 scripts/ci/inventory.py > inventory.json", section)
        self.assertIn("--execution-dir ci-artifacts --inventory inventory.json", section)
        self.assertIn('--revision "$GITHUB_SHA" --run-id "$GITHUB_RUN_ID"', section)
        self.assertIn('--run-attempt "$GITHUB_RUN_ATTEMPT"', section)
        self.assertIn("if: always()", section)
        self.assertIn("job[\"result\"] != \"success\"", section)
        self.assertIn("records-validation", job_section(workflow, "model-checks").split("runs-on:", 1)[0])

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
        self.assertIn("timing-summary.json", upload)
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

    def test_timing_report_runs_after_validation_and_reads_downloaded_artifacts(self):
        section = job_section(WORKFLOW.read_text(), "records-validation")
        validation = section.index("      - name: Validate full execution records")
        report = section.index("      - name: Report test timings")
        upload = section.index("      - name: Retain inventory and validation summary")
        self.assertLess(validation, report)
        self.assertLess(report, upload)
        step = section[report:upload]
        self.assertNotIn("if: always()", step)
        command = re.search(r"        run: (python3 scripts/ci/report_timings.py .* > timing-summary.json)", step)
        self.assertIsNotNone(command)
        scratch = WORKFLOW.parents[2] / ".tmp"
        scratch.mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=scratch) as directory:
            root = Path(directory)
            artifact = root / "ci-artifacts/ci-records-example"
            artifact.mkdir(parents=True)
            (artifact / "example.jsonl").write_text(json.dumps({"kind": "test", "elapsed": 1.25,
                                                               "test": "TestExample"}) + "\n" +
                                                    json.dumps({"kind": "package", "elapsed": 2,
                                                                "cached": False}) + "\n")
            script = WORKFLOW.parents[2] / "scripts/ci/report_timings.py"
            command_line = command.group(1).replace("scripts/ci/report_timings.py", str(script))
            result = subprocess.run(command_line, shell=True, cwd=root, check=False)
            self.assertEqual(result.returncode, 0)
            output = json.loads((root / "timing-summary.json").read_text())
            self.assertEqual(output["slowest"]["test"][0]["elapsed"], 1.25)


if __name__ == "__main__":
    unittest.main()
