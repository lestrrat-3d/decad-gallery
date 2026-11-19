import json
import os
import unittest
from unittest import mock

import gate


def needs_for(suites, event="push"):
    try:
        matrix = gate.matrix_for(suites)
    except ValueError:
        matrix = {"include": []}
    needs = {job: {"result": "skipped"} for job in gate.BROAD_JOBS}
    if "broad" in suites:
        for job in gate.BROAD_JOBS:
            if job == "bevel-pr" and event not in ("push", "pull_request"):
                continue
            if job in ("bevel-full", "renderer-full") and event not in ("schedule", "workflow_dispatch"):
                continue
            needs[job]["result"] = "success"
    needs.update(plan={"result": "success", "outputs": {
        "plan": json.dumps({"version": 1, "suites": suites}),
        "matrix": json.dumps(matrix), "count": str(len(matrix["include"]))}},
        repository={"result": "success"}, worker={"result": "success" if matrix["include"] else "skipped"})
    return needs


class GateTest(unittest.TestCase):
    def test_renderer_dispatch_commands(self):
        package = "github.com/lestrrat-3d/decad-gallery/tools/render"
        for suite, (timeout, tests) in gate.RENDER_TESTS.items():
            events = [json.dumps({"Package": package, "Test": test, "Action": "pass"}) for test in tests]
            proc = mock.Mock(stdout=events, returncode=0)
            proc.wait.return_value = 0
            with self.subTest(suite=suite), \
                    mock.patch.dict(os.environ, {"SUITE": suite, "CASE": "0"}), \
                    mock.patch.object(gate.sys, "argv", ["gate.py", "run"]), \
                    mock.patch.object(gate.subprocess, "Popen", return_value=proc) as dispatch:
                self.assertEqual(gate.matrix_for([suite]), {"include": [{"suite": suite, "case": 0}]})
                gate.main()
                dispatch.assert_called_once_with(
                    ["go", "-C", "tools/render", "test", "-json", "-run", gate.run_pattern(tests),
                     "-timeout", timeout, "./..."], stdout=gate.subprocess.PIPE, text=True,
                )

    def test_renderer_rejects_missing_skipped_or_failed_test(self):
        package = "github.com/lestrrat-3d/decad-gallery/tools/render"
        for action in (None, "skip", "fail"):
            events = [] if action is None else [json.dumps({"Package": package, "Test": "TestSwissPresets",
                                                            "Action": action})]
            proc = mock.Mock(stdout=events, returncode=0)
            proc.wait.return_value = 0
            with self.subTest(action=action), mock.patch.object(gate.subprocess, "Popen", return_value=proc):
                with self.assertRaisesRegex(SystemExit, "TestSwissPresets"):
                    gate.run_renderer("swiss-render", gate.commands("swiss-render")[0])

    def test_renderer_rejects_skipped_subtest_of_passing_test(self):
        package = "github.com/lestrrat-3d/decad-gallery/tools/render"
        events = [json.dumps({"Package": package, "Test": test, "Action": "pass"})
                  for test in gate.RENDER_TESTS["swiss-render"][1]]
        events.append(json.dumps({"Package": package, "Test": "TestSwissExploded/child", "Action": "skip"}))
        proc = mock.Mock(stdout=events, returncode=0)
        proc.wait.return_value = 0
        with mock.patch.object(gate.subprocess, "Popen", return_value=proc):
            with self.assertRaisesRegex(SystemExit, "TestSwissExploded/child"):
                gate.run_renderer("swiss-render", gate.commands("swiss-render")[0])

    def test_valid_selections(self):
        for suites, event in (([], "push"), (["cycloidal-pr"], "push"), (["broad"], "push"),
                              (["broad"], "schedule"), (["broad"], "workflow_dispatch")):
            with self.subTest(suites=suites, event=event):
                self.assertEqual(gate.check(needs_for(suites, event), event), [])

    def test_required_jobs_fail_closed(self):
        for job in ("plan", "repository", "worker"):
            for result in ("failure", "cancelled", "skipped", "missing"):
                with self.subTest(job=job, result=result):
                    needs = needs_for(["cycloidal-pr"])
                    needs[job]["result"] = result
                    self.assertTrue(gate.check(needs, "push"))

    def test_empty_matrix_must_be_skipped(self):
        needs = needs_for([])
        needs["worker"]["result"] = "success"
        self.assertTrue(gate.check(needs, "push"))

    def test_broad_jobs_are_required_and_unplanned_jobs_skipped(self):
        for job in gate.BROAD_JOBS:
            event = "schedule" if job in ("bevel-full", "renderer-full") else "push"
            needs = needs_for(["broad"], event)
            needs[job]["result"] = "cancelled"
            self.assertTrue(gate.check(needs, event), job)
        needs = needs_for([])
        needs["wheel"]["result"] = "success"
        self.assertTrue(gate.check(needs, "push"))

    def test_invalid_plan_or_matrix_fails(self):
        for suites in (["unknown"], ["broad", "cycloidal-pr"], ["cycloidal-pr", "cycloidal-pr"]):
            self.assertTrue(gate.check(needs_for(suites), "push"))
        needs = needs_for(["cycloidal-pr"])
        needs["plan"]["outputs"]["matrix"] = '{"include":[]}'
        self.assertTrue(gate.check(needs, "push"))
        self.assertTrue(gate.check(needs_for([]), "workflow_dispatch"))
        del needs["plan"]["outputs"]["plan"]
        self.assertTrue(gate.check(needs, "push"))

    def test_suite_expansion_preserves_all_shards(self):
        matrix = gate.matrix_for(["wheel", "siren", "bevel-pr", "model-cycloidal", "package-cornercheck"])
        self.assertEqual(len(matrix["include"]), 20)
        self.assertEqual(gate.commands("bevel-pr")[-1][-1], "rest")
        self.assertEqual(gate.commands("model-cycloidal")[0][-3:], ["model", "pr", "cycloidal"])
        self.assertEqual(gate.commands("package-cornercheck")[0][2], "./internal/cornercheck")
        self.assertTrue(gate.check(needs_for(["wheel", "siren"]), "push") == [])

    def test_broad_with_new_package_dispatch(self):
        suites = ["broad", "new-model-cycloidal", "new-package-cornercheck"]
        self.assertEqual(gate.check(needs_for(suites), "push"), [])
        self.assertEqual(gate.commands("new-model-cycloidal")[0][2], "./models/cycloidal")
        self.assertEqual(gate.commands("new-package-cornercheck")[0][2], "./internal/cornercheck")


if __name__ == "__main__":
    unittest.main()
