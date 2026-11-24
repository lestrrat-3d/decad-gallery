import contextlib
import copy
import io
import json
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


if __name__ == "__main__":
    unittest.main()
