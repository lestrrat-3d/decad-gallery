import contextlib
from collections import Counter
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import reconcile_records
import run_tests


class ReconcileRecordsTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        Path(".tmp").mkdir(exist_ok=True)
        cls.temp = tempfile.TemporaryDirectory(dir=".tmp")
        cls.root = Path(cls.temp.name).resolve()
        (cls.root / "go.mod").write_text("module example.com/fixture\n\ngo 1.23\n")
        (cls.root / "empty").mkdir()
        (cls.root / "empty" / "empty.go").write_text("package empty\n")
        (cls.root / "fixture_test.go").write_text('''package fixture
import "testing"
func TestParent(t *testing.T) {
    t.Run("first", func(t *testing.T) {})
    t.Run("second", func(t *testing.T) {})
}
func TestSingle(t *testing.T) {}
''')
        result = subprocess.run(["go", "test", "-json", "./..."], cwd=cls.root,
                                check=True, capture_output=True, text=True)
        cls.events = result.stdout.splitlines()
        cls.empty_events = subprocess.run(["go", "test", "-json", "./empty"], cwd=cls.root,
                                          check=True, capture_output=True, text=True).stdout.splitlines()

    @classmethod
    def tearDownClass(cls):
        cls.temp.cleanup()

    def produce(self, name, events=None, repeat=1):
        path = self.root / name
        path.unlink(missing_ok=True)
        with mock.patch.dict(os.environ, {"CI_TEST_RECORDS": str(path)}), contextlib.redirect_stdout(io.StringIO()):
            for _ in range(repeat):
                run_tests.finished_tests(self.events if events is None else events)
        return path

    def edit(self, path, transform):
        records = [json.loads(line) for line in path.read_text().splitlines()]
        path.write_text("".join(json.dumps(record) + "\n" for record in transform(records)))

    def test_real_producer_counts_leaves_and_duplicates(self):
        reference = self.produce("reference", repeat=2)
        candidate = self.produce("candidate", repeat=2)
        required = [(".", "example.com/fixture", name) for name in ("TestSingle", "TestParent")]
        result = reconcile_records.reconcile([reference], [candidate], required)
        self.assertTrue(result["ok"], result)
        self.assertEqual(result["duplicate_count"], 3)
        self.assertFalse(any(key[2] == "TestParent" for key, _ in result["duplicate_sample"]))

    def test_parent_without_children_is_a_leaf_in_its_invocation(self):
        path = self.produce("records")
        events = subprocess.run(["go", "test", "-json", "-run", "^TestParent$/^absent$", "."],
                                cwd=self.root, check=True, capture_output=True, text=True).stdout.splitlines()
        with mock.patch.dict(os.environ, {"CI_TEST_RECORDS": str(path)}), contextlib.redirect_stdout(io.StringIO()):
            run_tests.finished_tests(events)
        leaves, _, _, errors, _ = reconcile_records.read_records([path])
        self.assertFalse(errors)
        self.assertEqual(leaves[(".", "example.com/fixture", "TestParent", "pass")], 1)

    def test_missing_required_and_missing_leaf(self):
        reference = self.produce("reference")
        candidate = self.produce("candidate")
        self.edit(candidate, lambda rows: [r for r in rows if r.get("test") != "TestParent/second"])
        result = reconcile_records.reconcile([reference], [candidate], [(".", "example.com/fixture", "TestAbsent")])
        self.assertFalse(result["ok"])
        self.assertEqual(result["missing_count"], 1)
        self.assertEqual(result["missing_required_count"], 1)

    def test_invalid_package_and_failed_outcomes(self):
        reference = self.produce("reference")
        for field, value, error in (("package", "../bad", "invalid terminal event"),
                                    ("package", "", "invalid terminal event"),
                                    ("status", "fail", "failed outcome")):
            with self.subTest(field=field, value=value):
                candidate = self.produce("candidate")
                def change(rows):
                    next(r for r in rows if r["kind"] == "test")[field] = value
                    return rows
                self.edit(candidate, change)
                result = reconcile_records.reconcile([reference], [candidate])
                self.assertFalse(result["ok"])
                self.assertIn(error, result["candidate_errors"])

    def test_incomplete_and_missing_package_outcome(self):
        reference = self.produce("reference")
        for omitted, error in (("complete", "incomplete invocation"), ("package", "missing package outcome")):
            with self.subTest(omitted=omitted):
                candidate = self.produce("candidate")
                self.edit(candidate, lambda rows: [r for r in rows if r["kind"] != omitted])
                result = reconcile_records.reconcile([reference], [candidate])
                self.assertFalse(result["ok"])
                self.assertIn(error, result["candidate_errors"])

    def test_matching_test_skips_fail_but_package_skips_are_allowed(self):
        skipped_events = [json.dumps({**json.loads(line), "Action": "skip"})
                          if json.loads(line).get("Action") == "pass" and
                          json.loads(line).get("Test") == "TestSingle" else line for line in self.events]
        reference = self.produce("reference", skipped_events)
        candidate = self.produce("candidate", skipped_events)
        result = reconcile_records.reconcile([reference], [candidate])
        self.assertFalse(result["ok"])
        self.assertEqual(result["candidate_errors"]["skipped test"], 1)
        script = Path(reconcile_records.__file__).resolve()
        cli = subprocess.run(["python3", str(script), "--reference", str(reference),
                              "--candidate", str(candidate)], capture_output=True, text=True)
        self.assertEqual(cli.returncode, 1)

        package_events = [json.dumps({"Action": "skip", "Package": "example.com/empty"})]
        reference = self.produce("reference", package_events)
        candidate = self.produce("candidate", package_events)
        self.assertTrue(reconcile_records.reconcile([reference], [candidate])["ok"])

    def test_malformed_input_has_bounded_examples(self):
        reference = self.produce("reference")
        candidate = self.produce("candidate", self.events + ["not json"] * 25)
        result = reconcile_records.reconcile([reference], [candidate])
        self.assertEqual(result["candidate_errors"]["malformed producer event"], 25)
        self.assertEqual(len(result["candidate_examples"]["malformed producer event"]), 10)

    def test_inventory_requires_tests_and_packages_in_both_executions(self):
        reference = self.produce("reference")
        candidate = self.produce("candidate")
        for path in (reference, candidate):
            with mock.patch.dict(os.environ, {"CI_TEST_RECORDS": str(path)}), \
                    contextlib.redirect_stdout(io.StringIO()):
                run_tests.finished_tests(self.empty_events, module="tools/render")
        inventory = self.root / "inventory.json"
        data = {"version": 1, "modules": {"root": {"example.com/fixture": ["TestParent", "TestSingle"]},
                                          "renderer": {"example.com/fixture/empty": []}},
                "required_variants": [{"package": "example.com/fixture", "test": "TestParent/first"}]}
        inventory.write_text(json.dumps(data))
        self.assertTrue(reconcile_records.reconcile([reference], [candidate], inventory_path=inventory)["ok"])
        self.edit(candidate, lambda rows: [r for r in rows if r.get("test") != "TestSingle"])
        result = reconcile_records.reconcile([reference], [candidate], inventory_path=inventory)
        self.assertEqual(result["candidate_missing_inventory_tests_count"], 1)
        self.assertEqual(result["reference_missing_inventory_tests_count"], 0)
        self.edit(reference, lambda rows: [r for r in rows if r.get("module") != "tools/render"])
        result = reconcile_records.reconcile([reference], [candidate], inventory_path=inventory)
        self.assertEqual(result["reference_missing_inventory_packages_count"], 1)

        data["modules"]["renderer"]["example.com/fixture/empty"] = ["TestMissing"]
        inventory.write_text(json.dumps(data))
        result = reconcile_records.reconcile([candidate], [candidate], inventory_path=inventory)
        self.assertFalse(result["ok"])
        self.assertEqual(result["candidate_missing_inventory_tests_count"], 2)

    def test_required_variant_missing_from_matching_runs_fails(self):
        records = self.produce("records")
        inventory = self.root / "inventory.json"
        data = {"version": 1, "modules": {"root": {"example.com/fixture": ["TestParent", "TestSingle"]},
                                          "renderer": {"example.com/fixture/empty": []}},
                "required_variants": [{"package": "example.com/fixture", "test": "TestParent/missing"}]}
        inventory.write_text(json.dumps(data))
        result = reconcile_records.reconcile([records], [records], inventory_path=inventory)
        self.assertFalse(result["ok"])
        self.assertEqual(result["reference_missing_inventory_tests_count"], 1)
        self.assertEqual(result["candidate_missing_inventory_tests_count"], 1)
        data["required_variants"][0]["package"] = "example.com/fixture/empty"
        inventory.write_text(json.dumps(data))
        result = reconcile_records.reconcile([records], [records], inventory_path=inventory)
        self.assertIn("root inventory", result["inventory_error"])

    def test_invalid_or_missing_inventory_fails(self):
        records = self.produce("records")
        inventory = self.root / "invalid-inventory.json"
        for contents in (None, "{}", '{"version":1,"modules":{"root":{},"renderer":{}}}'):
            with self.subTest(contents=contents):
                inventory.unlink(missing_ok=True)
                if contents is not None:
                    inventory.write_text(contents)
                result = reconcile_records.reconcile([records], [records], inventory_path=inventory)
                self.assertFalse(result["ok"])
                self.assertTrue(result["inventory_error"])

    def test_single_execution_checks_artifacts_metadata_and_inventory(self):
        directory = self.root / "execution"
        directory.mkdir(exist_ok=True)
        inventory = self.root / "execution-inventory.json"
        inventory.write_text(json.dumps({"version": 1,
            "modules": {"root": {"example.com/fixture": ["TestParent", "TestSingle"]},
                        "renderer": {"example.com/fixture/empty": []}},
            "required_variants": [{"package": "example.com/fixture", "test": "TestParent/first"}]}))
        jobs = {"repository": ("",), "renderer-full": ("",), "wheel": ("native",)}
        root_events = [line for line in self.events if json.loads(line).get("Test") != "TestSingle"]
        wheel_events = [line for line in self.events
                        if not json.loads(line).get("Test", "").startswith("TestParent")]
        for job, case, events, module in (("repository", "", root_events, "."),
                                           ("renderer-full", "", self.empty_events, "tools/render"),
                                           ("wheel", "native", wheel_events, ".")):
            name = job + (f"-{case}" if case else "")
            artifact = directory / f"ci-records-{name}"
            artifact.mkdir(exist_ok=True)
            path = artifact / f"{name}.jsonl"
            path.unlink(missing_ok=True)
            env = {"CI_TEST_RECORDS": str(path), "GITHUB_JOB": job, "CI_TEST_CASE": case,
                   "GITHUB_SHA": "abc", "GITHUB_RUN_ID": "123", "GITHUB_RUN_ATTEMPT": "1"}
            with mock.patch.dict(os.environ, env), contextlib.redirect_stdout(io.StringIO()):
                run_tests.finished_tests(events, module=module)
        with mock.patch.object(reconcile_records, "FULL_JOBS", jobs):
            verify = lambda: reconcile_records.verify_execution(directory, inventory, "abc", "123", "1")
            result = verify()
            self.assertTrue(result["ok"], result)
            previous_attempt = reconcile_records.verify_execution(directory, inventory, "abc", "123", "2")
            self.assertTrue(previous_attempt["ok"], previous_attempt)
            self.assertEqual(result["duplicate_count"], 0)
            matrix = directory / "ci-records-wheel-native" / "wheel-native.jsonl"
            original = matrix.read_text()
            with mock.patch.dict(os.environ, {"CI_TEST_RECORDS": str(matrix), "GITHUB_JOB": "wheel",
                                              "CI_TEST_CASE": "native", "GITHUB_SHA": "abc",
                                              "GITHUB_RUN_ID": "123", "GITHUB_RUN_ATTEMPT": "1"}), \
                    contextlib.redirect_stdout(io.StringIO()):
                run_tests.finished_tests(wheel_events)
            duplicate_result = verify()
            self.assertFalse(duplicate_result["ok"])
            self.assertEqual(duplicate_result["duplicate_count"], 1)
            matrix.write_text(original)
            for field in ("job", "case", "module", "revision", "run_id", "run_attempt"):
                with self.subTest(field=field):
                    self.edit(matrix, lambda rows: [{**row, field: "wrong"} for row in rows])
                    self.assertIn("unexpected metadata", verify()["errors"])
                    matrix.write_text(original)
            for attempt in ("0", "3", "01", "-1"):
                with self.subTest(attempt=attempt):
                    self.edit(matrix, lambda rows: [{**row, "run_attempt": attempt} for row in rows])
                    result = reconcile_records.verify_execution(directory, inventory, "abc", "123", "2")
                    self.assertIn("unexpected metadata", result["errors"])
                    matrix.write_text(original)
            self.edit(matrix, lambda rows: [rows[0], *[{**row, "run_attempt": "2"} for row in rows[1:]]])
            result = reconcile_records.verify_execution(directory, inventory, "abc", "123", "2")
            self.assertIn("unexpected metadata", result["errors"])
            matrix.write_text(original)
            self.edit(matrix, lambda rows: [row for row in rows if row.get("test") != "TestSingle"])
            root = directory / "ci-records-repository" / "repository.jsonl"
            self.edit(root, lambda rows: [row for row in rows if row.get("test") != "TestSingle"])
            self.assertEqual(verify()["missing_inventory_tests_count"], 1)
            self.edit(root, lambda rows: [row for row in rows if row["kind"] != "complete"])
            self.assertIn("incomplete invocation", verify()["errors"])
            (directory / "ci-records-unexpected").mkdir(exist_ok=True)
            self.assertEqual(verify()["extra_artifacts_count"], 1)
            (matrix.parent / "extra.jsonl").touch()
            self.assertEqual(verify()["invalid_artifact_contents_count"], 1)
            matrix.unlink()
            self.assertIn("unreadable file", verify()["errors"])

    def test_single_execution_bounds_duplicate_examples(self):
        duplicates = Counter({(".", "example.com/p", f"Test{i}", "pass"): 2 for i in range(25)})
        directory = self.root / "empty-execution"
        directory.mkdir(exist_ok=True)
        with mock.patch.object(reconcile_records, "FULL_JOBS", {}), \
                mock.patch.object(reconcile_records, "read_records", return_value=(duplicates, set(), set(), {}, {})), \
                mock.patch.object(reconcile_records, "load_inventory", return_value=(set(), set())):
            result = reconcile_records.verify_execution(directory, "unused", "abc", "123", "1")
        self.assertFalse(result["ok"])
        self.assertEqual(result["duplicate_count"], 25)
        self.assertEqual(len(result["duplicate_sample"]), 10)


if __name__ == "__main__":
    unittest.main()
