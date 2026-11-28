import contextlib
import copy
import hashlib
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import case_inventory
import reconcile_records


class CasePlanTest(unittest.TestCase):
    def test_cycloidal_pins_match_the_checked_in_sources(self):
        # A stale pin turns the cycloidal case split off without any failure. Update the pins only after
        # reviewing the source change (see CYCLOIDAL_REVIEWED_GO).
        self.assertEqual(case_inventory.stale_pins("cycloidal"), [])

    def test_stale_pins_name_the_mismatched_pin(self):
        with mock.patch.object(case_inventory, "CYCLOIDAL_REVIEWED_GO", "0" * 64):
            problems = case_inventory.stale_pins("cycloidal")
        self.assertEqual(len(problems), 1)
        self.assertIn("CYCLOIDAL_REVIEWED_GO", problems[0])

    def test_plan_fails_instead_of_writing_a_null_plan(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory, "out")
            with mock.patch.dict(os.environ, {"GITHUB_OUTPUT": str(output)}), \
                    mock.patch.object(case_inventory, "discover", return_value=None), \
                    mock.patch.object(case_inventory, "stale_pins", return_value=["pin is stale"]), \
                    self.assertRaisesRegex(SystemExit, "pin is stale"):
                case_inventory.plan_main()
            self.assertFalse(output.exists())

    def test_retry_compares_the_original_plan_with_the_current_checkout(self):
        first = {"package": "github.com/lestrrat-3d/decad-gallery/models/cycloidal",
                 "identity": {"revision": "a", "run_id": "bob", "attempt": "1", "profile": {}}}
        current = copy.deepcopy(first)
        current["identity"]["attempt"] = "2"
        with mock.patch.object(case_inventory, "discover", side_effect=lambda name: copy.deepcopy(current)):
            case_inventory.validate_manifest(first)
            for field, value in (("run_id", "different"), ("revision", "b"), ("profile", {"GOFLAGS": "-race"}),
                                 ("attempt", "3")):
                changed = copy.deepcopy(first)
                changed["identity"][field] = value
                with self.subTest(field=field), self.assertRaises(ValueError):
                    case_inventory.validate_manifest(changed)

    def test_plan_assigns_every_case_once(self):
        manifest = {"parents": {"TestA": ["three", "one", "two"], "TestB": ["single"]}}
        with mock.patch.object(case_inventory, "validate_manifest"):
            plan = case_inventory.build_plan(manifest)
        groups = [job for job in plan["jobs"] if job["kind"] == "group"]
        pairs = [(job["parent"], name) for job in groups for name in job["names"]]
        self.assertCountEqual(pairs, [("TestA", "one"), ("TestA", "two"), ("TestA", "three"),
                                      ("TestB", "single")])
        self.assertEqual(len(pairs), len(set(pairs)))
        self.assertEqual([job["kind"] for job in plan["jobs"][:2]], ["audit", "remainder"])

    def test_altered_or_missing_assignment_never_executes(self):
        manifest = {"parents": {"TestA": ["one", "two"]}}
        with mock.patch.object(case_inventory, "validate_manifest"), \
                mock.patch.object(case_inventory, "run_group") as execute:
            plan = case_inventory.build_plan(manifest)
            altered = copy.deepcopy(plan)
            altered["jobs"].pop()
            with self.assertRaisesRegex(ValueError, "incomplete or altered"):
                case_inventory.run_planned(altered, "TestA-0")
            with self.assertRaisesRegex(ValueError, "uniquely assigned"):
                case_inventory.run_planned(plan, "missing")
        execute.assert_not_called()

    def test_strict_worker_rejects_missing_duplicate_and_skipped_children(self):
        package = "example.com/model"
        def event(action, test=None):
            return json.dumps({"Action": action, "Package": package, **({"Test": test} if test else {})}) + "\n"
        valid = [event("start"), event("run", "TestA"), event("run", "TestA/one"),
                 event("pass", "TestA/one"), event("pass", "TestA"), event("pass")]
        process = mock.Mock(stdout=valid)
        process.wait.return_value = 0
        with mock.patch.object(case_inventory.subprocess, "Popen", return_value=process), \
                contextlib.redirect_stdout(io.StringIO()):
            case_inventory.strict_run(["go"], {}, package, ["TestA"], ["TestA", "TestA/one"])
        streams = [valid[:-3] + valid[-2:], valid[:3] + [event("run", "TestA/one")] + valid[3:],
                   [line.replace('"pass", "Package": "example.com/model", "Test": "TestA/one"',
                                 '"skip", "Package": "example.com/model", "Test": "TestA/one"')
                    for line in valid]]
        for stream in streams:
            process = mock.Mock(stdout=stream)
            process.wait.return_value = 0
            with self.subTest(stream=stream), mock.patch.object(case_inventory.subprocess, "Popen", return_value=process), \
                    contextlib.redirect_stdout(io.StringIO()), self.assertRaises(ValueError):
                case_inventory.strict_run(["go"], {}, package, ["TestA"], ["TestA", "TestA/one"])

    def test_offline_cli_accepts_same_run_producers_from_two_attempts(self):
        package = reconcile_records.full_partition.CYCLOIDAL
        parents = {"TestBuildOptions": ["one", "two"], "TestCornersVariants": ["one", "two", "three"],
                   "TestMeasuredFits": ["one", "two"]}
        manifest = {"package": package, "parents": parents, "tests": sorted([*parents, "TestOther"]),
                    "identity": {"revision": "abc", "run_id": "123", "attempt": "1", "profile": {}}}
        plan = {"version": 1, "manifest": manifest, "jobs": case_inventory.assignments(manifest)}
        digest = hashlib.sha256(json.dumps(plan, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
        with tempfile.TemporaryDirectory(dir=".tmp") as directory:
            root = Path(directory)
            artifacts = root / "artifacts"
            inventory = root / "inventory.json"
            inventory.write_text(json.dumps({"version": 1, "case_plan": plan, "required_variants": [],
                "modules": {"root": {package: manifest["tests"]}, "renderer": {"example.com/render": []}}}))
            for index, job in enumerate(plan["jobs"]):
                case = job["case"]
                folder = artifacts / ("ci-records-case-workers-" + case)
                folder.mkdir(parents=True)
                attempt = str(1 + index % 2)
                receipt = dict(version=2, case=case, kind=job["kind"], producer_attempt=attempt,
                               plan_sha256=digest, identity=manifest["identity"])
                if job["kind"] != "audit":
                    tests = (["TestOther"] if job["kind"] == "remainder" else
                             [job["parent"], *(job["parent"] + "/" + name for name in job["names"])])
                    meta = dict(version=1, invocation=case, module=".", job="case-workers", case=case,
                                revision="abc", run_id="123", run_attempt=attempt, package=package)
                    rows = [dict(meta, kind="start", test=name) for name in [None, *tests]]
                    rows += [dict(meta, kind="test", test=name, status="pass") for name in tests]
                    rows += [dict(meta, kind="package", status="pass"), dict(meta, kind="complete")]
                    path = folder / "execution.jsonl"
                    path.write_text("".join(json.dumps(row) + "\n" for row in rows))
                    receipt["record_sha256"] = case_inventory.record_digest(path)
                (folder / "receipt.json").write_text(json.dumps(receipt))
            folder = artifacts / "ci-records-renderer-full"
            folder.mkdir()
            meta = dict(version=1, invocation="renderer", module="tools/render", job="renderer-full", case="",
                        revision="abc", run_id="123", run_attempt="1", package="example.com/render")
            (folder / "renderer-full.jsonl").write_text(json.dumps(dict(meta, kind="package", status="pass")) +
                                                       "\n" + json.dumps(dict(meta, kind="complete")) + "\n")
            argv = ["reconcile_records.py", "--execution-dir", str(artifacts), "--inventory", str(inventory),
                    "--revision", "abc", "--run-id", "123", "--run-attempt", "2"]
            with mock.patch.object(reconcile_records.sys, "argv", argv), mock.patch.dict(os.environ, {}, clear=True), \
                    mock.patch.object(reconcile_records, "FULL_JOBS", {"package-workers": (), "renderer-full": ("",)}), \
                    mock.patch.object(reconcile_records.full_partition, "select", return_value=([package], set(), [])), \
                    mock.patch.object(case_inventory, "discover", side_effect=AssertionError("requires checkout")), \
                    contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(reconcile_records.main(), 0)

    def test_execution_gate_rejects_missing_or_wrong_case_evidence(self):
        identity = {"revision": "a" * 40, "run_id": "123", "attempt": "1", "profile": {}}
        manifest = {"package": "example.com/model", "parents": {"TestA": ["one", "two"]},
                    "tests": ["TestA", "TestOther"], "identity": identity}
        with mock.patch.object(case_inventory, "validate_manifest"):
            plan = case_inventory.build_plan(manifest)
            digest = hashlib.sha256(json.dumps(plan, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
            Path(".tmp").mkdir(exist_ok=True)
            with tempfile.TemporaryDirectory(dir=".tmp") as directory:
                receipts, records = {}, {}
                for job in plan["jobs"]:
                    case = job["case"]
                    receipt = Path(directory) / (case + ".receipt")
                    receipt.write_text(json.dumps({"version": 1, "case": case, "kind": job["kind"],
                                                   "plan_sha256": digest, "identity": identity}))
                    receipts[case] = receipt
                    if job["kind"] == "audit":
                        continue
                    tests = ([job["parent"]] + [job["parent"] + "/" + name for name in job["names"]]
                             if job["kind"] == "group" else ["TestOther"])
                    meta = {"version": 1, "invocation": case, "module": ".", "job": "case-workers",
                            "case": case, "revision": identity["revision"], "run_id": "123", "run_attempt": "1"}
                    rows = [{**meta, "kind": "start", "package": manifest["package"], "test": test}
                            for test in [None, *tests]]
                    rows += [{**meta, "kind": "test", "package": manifest["package"], "test": name, "status": "pass"}
                            for name in tests]
                    rows.extend([{**meta, "kind": "package", "package": manifest["package"], "status": "pass"},
                                 {**meta, "kind": "complete"}])
                    path = Path(directory) / (case + ".jsonl")
                    path.write_text("".join(json.dumps(row) + "\n" for row in rows))
                    data = json.loads(receipt.read_text())
                    data["record_sha256"] = case_inventory.record_digest(path)
                    receipt.write_text(json.dumps(data))
                    records[case] = path
                self.assertTrue(case_inventory.verify_execution(plan, receipts, records))
                case = "TestA-1"
                path, receipt = records[case], receipts[case]
                old_records, old_receipt = path.read_text(), receipt.read_text()
                rows = [dict(json.loads(line), run_attempt="2") for line in old_records.splitlines()]
                path.write_text("".join(json.dumps(row) + "\n" for row in rows))
                data = dict(json.loads(old_receipt), version=2, producer_attempt="2",
                            record_sha256=case_inventory.record_digest(path))
                receipt.write_text(json.dumps(data))
                with mock.patch.object(case_inventory, "discover", side_effect=AssertionError("offline checkout")):
                    self.assertTrue(case_inventory.verify_execution(plan, receipts, records, "2"))
                with self.assertRaisesRegex(ValueError, "attempt"):
                    case_inventory.verify_execution(plan, receipts, records, "1")
                receipt.write_text(json.dumps(dict(data, producer_attempt="1")))
                with self.assertRaisesRegex(ValueError, "records"):
                    case_inventory.verify_execution(plan, receipts, records, "2")
                path.write_text(old_records)
                receipt.write_text(old_receipt)
                incomplete = dict(receipts)
                incomplete.pop("audit")
                with self.assertRaisesRegex(ValueError, "artifacts"):
                    case_inventory.verify_execution(plan, incomplete, records)
                path = records["remainder"]
                original = path.read_text()
                rows = [json.loads(line) for line in original.splitlines()]
                bad_rows = [[row for row in rows if row["kind"] != "test"],
                            [{**row, "status": "skip"} if row["kind"] == "package" else row for row in rows],
                            rows[:-1], [rows[0], *rows]]
                original_receipt = receipts["remainder"].read_text()
                for invalid in bad_rows:
                    path.write_text("".join(json.dumps(row) + "\n" for row in invalid))
                    with self.assertRaisesRegex(ValueError, "records"):
                        case_inventory.verify_execution(plan, receipts, records)
                    data = json.loads(original_receipt)
                    data["record_sha256"] = case_inventory.record_digest(path)
                    receipts["remainder"].write_text(json.dumps(data))
                    with self.assertRaisesRegex(ValueError, "records"):
                        case_inventory.verify_execution(plan, receipts, records)
                    receipts["remainder"].write_text(original_receipt)
                path.write_text(original)
                records["TestA-0"].write_text(records["TestA-1"].read_text())
                with self.assertRaisesRegex(ValueError, "records"):
                    case_inventory.verify_execution(plan, receipts, records)


if __name__ == "__main__":
    unittest.main()
