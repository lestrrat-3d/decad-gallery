import contextlib
import copy
import hashlib
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import case_inventory


class CasePlanTest(unittest.TestCase):
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
