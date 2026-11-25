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
            {"Action": "output", "Package": "example.com/p", "Output": "ok  \texample.com/p\t3.000s\n"},
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
                                    for value in (-1, True, None, "2", float("inf"))) +
                            json.dumps({"kind": "package", "elapsed": 1, "cached": False}) + "\n")
            result = report_timings.report([path], 2)
            self.assertEqual(result["missing_or_invalid_elapsed"], 5)
            self.assertEqual(result["counts"]["test"], 0)

    def test_cached_and_unclassified_invocations_are_excluded(self):
        events = []
        for package, output in (("example.com/cached", "ok  \texample.com/cached\t(cached)\n"),
                                ("example.com/unknown", "other output\n")):
            events.extend([{"Action": "pass", "Package": package, "Test": "TestA", "Elapsed": 4},
                           {"Action": "output", "Package": package, "Output": output},
                           {"Action": "pass", "Package": package, "Elapsed": 5}])
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "records.jsonl"
            with mock.patch.dict(os.environ, {"CI_TEST_RECORDS": str(path)}), \
                    contextlib.redirect_stdout(io.StringIO()):
                run_tests.finished_tests([json.dumps(event) for event in events])
            result = report_timings.report([path], 20)
            self.assertEqual(result["counts"], {"test": 0, "package": 0})
            self.assertEqual(result["excluded"], {"cached": 2, "unknown_cache": 2})
            self.assertTrue(reconcile_records.reconcile([path], [path])["ok"])

    def test_failed_timed_package_is_recorded_but_fails_reconciliation(self):
        events = [
            {"Action": "fail", "Package": "example.com/p", "Test": "TestFailure", "Elapsed": 0.01},
            {"Action": "output", "Package": "example.com/p", "Output": "FAIL\texample.com/p\t0.012s\n"},
            {"Action": "fail", "Package": "example.com/p", "Elapsed": 0.012},
        ]
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "records.jsonl"
            with mock.patch.dict(os.environ, {"CI_TEST_RECORDS": str(path)}), \
                    contextlib.redirect_stdout(io.StringIO()):
                run_tests.finished_tests([json.dumps(event) for event in events])
            result = report_timings.report([path], 20)
            self.assertEqual(result["counts"], {"test": 1, "package": 1})
            self.assertEqual(result["slowest"]["test"][0]["status"], "fail")
            self.assertFalse(reconcile_records.reconcile([path], [path])["ok"])
