import contextlib
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import reconcile_records
import report_timings
import run_tests


class TimingReportTest(unittest.TestCase):
    def test_producer_report_and_reconciliation(self):
        events = [
            {"Action": "pass", "Package": "example.com/p", "Test": "TestA/sub", "Elapsed": 1.5},
            {"Action": "pass", "Package": "example.com/p", "Test": "TestA", "Elapsed": 2},
            {"Action": "pass", "Package": "example.com/p", "Test": "TestB"},
            {"Action": "pass", "Package": "example.com/p", "Elapsed": 3},
        ]
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "records.jsonl"
            with mock.patch.dict(os.environ, {"CI_TEST_RECORDS": str(path), "GITHUB_JOB": "job"}), \
                    contextlib.redirect_stdout(io.StringIO()):
                run_tests.finished_tests([json.dumps(event) for event in events])
            result = report_timings.report([path], 1)
            self.assertEqual(result["counts"], {"test": 2, "package": 1})
            self.assertEqual(result["missing_or_invalid_elapsed"], 1)
            self.assertEqual(result["slowest"]["test"][0]["test"], "TestA")
            self.assertEqual(result["slowest"]["test"][0]["job"], "job")
            self.assertEqual(result["slowest"]["package"][0]["elapsed"], 3)
            self.assertTrue(reconcile_records.reconcile([path], [path])["ok"])

    def test_invalid_elapsed_is_not_a_sample(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "records.jsonl"
            path.write_text("".join(json.dumps({"kind": "test", "elapsed": value}) + "\n"
                                    for value in (-1, True, None, "2", float("inf"))))
            result = report_timings.report([path], 2)
            self.assertEqual(result["missing_or_invalid_elapsed"], 5)
            self.assertEqual(result["counts"]["test"], 0)
