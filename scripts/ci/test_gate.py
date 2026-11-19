import json
import os
from pathlib import Path
import re
import unittest
from unittest import mock

import gate


def needs_for(suites, event="push"):
    try:
        matrix = gate.matrix_for(suites)
    except ValueError:
        matrix = {"include": []}
    needs = {job: {"result": "skipped"} for job in gate.BROAD_JOBS}
    full = "full" in suites or event in ("schedule", "workflow_dispatch")
    if "broad" in suites:
        for job in gate.BROAD_JOBS:
            if job == "bevel-pr" and full:
                continue
            if job in gate.PR_RENDER_JOBS and (full or "renderer-all" in suites):
                continue
            if job in ("bevel-full", "renderer-full") and not full:
                continue
            needs[job]["result"] = "success"
    needs.update(plan={"result": "success", "outputs": {
        "plan": json.dumps({"version": 1, "suites": suites}),
        "matrix": json.dumps(matrix), "count": str(len(matrix["include"]))}},
        repository={"result": "success"}, worker={"result": "success" if matrix["include"] else "skipped"})
    return needs


class GateTest(unittest.TestCase):
    def test_gallery_cache_step_runs_only_on_pr_and_push(self):
        workflow = Path(".github/workflows/repository.yml").read_text()
        step = workflow.split("- name: Check gallery tile cache\n", 1)[1].split("- name:", 1)[0]
        self.assertIn("github.event_name == 'pull_request' || github.event_name == 'push'", step)
        self.assertIn("!contains(fromJSON(needs.plan.outputs.plan).suites, 'renderer-all')", step)
        self.assertIn("go -C tools/render test -run '^TestGalleryCache'", step)
        renderer = workflow.split("  renderer-full:\n", 1)[1].split("\n  planetary-motion:", 1)[0]
        self.assertIn("python3 scripts/ci/gate.py renderer-full", renderer)
        for job in gate.PR_RENDER_JOBS:
            job_body = re.split(r"\n  [a-z][a-z-]*:\n", workflow.split(f"  {job}:\n", 1)[1], maxsplit=1)[0]
            self.assertIn("!contains(fromJSON(needs.plan.outputs.plan).suites, 'renderer-all')", job_body)

    def test_broad_with_full_renderer_dispatch(self):
        suites = ["broad", "renderer-all"]
        self.assertEqual(gate.check(needs_for(suites), "push"), [])
        needs = needs_for(suites)
        needs["worker"]["result"] = "skipped"
        self.assertTrue(gate.check(needs, "push"))
        for job in gate.PR_RENDER_JOBS:
            needs = needs_for(suites)
            needs[job]["result"] = "success"
            self.assertTrue(gate.check(needs, "push"))
        with mock.patch.dict(os.environ, {"SUITE": "renderer-all", "CASE": "0"}), \
                mock.patch.object(gate.sys, "argv", ["gate.py", "run"]), \
                mock.patch.object(gate, "run_renderer_full") as dispatch:
            gate.main()
        dispatch.assert_called_once_with()

    def test_full_renderer_inventory_requires_every_package_and_subtest(self):
        packages = [gate.RENDER_PACKAGE, gate.RENDER_PACKAGE + "/extra"]
        def run(command, **kwargs):
            if command[-2:] == ["list", "./..."]:
                return mock.Mock(stdout="\n".join(packages))
            return mock.Mock(stdout="TestSame\nok\n")
        def events(missing=None, skip=False):
            entries = [{"Package": package, "Test": "TestSame", "Action": "pass"}
                       for package in packages if package != missing]
            if skip:
                entries.append({"Package": packages[0], "Test": "TestSame/child", "Action": "skip"})
            proc = mock.Mock(stdout=[json.dumps(entry) for entry in entries], returncode=0)
            proc.wait.return_value = 0
            return proc
        for missing, skip in ((None, False), (packages[1], False), (None, True)):
            with self.subTest(missing=missing, skip=skip), \
                    mock.patch.object(gate.subprocess, "run", side_effect=run), \
                    mock.patch.object(gate.subprocess, "Popen", return_value=events(missing, skip)):
                if missing or skip:
                    with self.assertRaisesRegex(SystemExit, "TestSame"):
                        gate.run_renderer_full()
                else:
                    gate.run_renderer_full()

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

    def test_full_events_require_renderer_inventory_and_skip_duplicate_jobs(self):
        for event in ("schedule", "workflow_dispatch"):
            needs = needs_for(["broad"], event)
            self.assertEqual(gate.check(needs, event), [])
            for job in gate.PR_RENDER_JOBS:
                with self.subTest(event=event, job=job):
                    altered = needs_for(["broad"], event)
                    altered[job]["result"] = "success"
                    self.assertTrue(gate.check(altered, event))
            needs["renderer-full"]["result"] = "skipped"
            self.assertTrue(gate.check(needs, event))

    def test_full_selection_requires_full_jobs_on_push_and_pull_request(self):
        for event in ("push", "pull_request"):
            suites = ["broad", "full"]
            needs = needs_for(suites, event)
            self.assertEqual(gate.matrix_for(suites), {"include": []})
            self.assertEqual(gate.check(needs, event), [])
            for job in ("renderer-full", "bevel-full"):
                with self.subTest(event=event, job=job):
                    altered = needs_for(suites, event)
                    altered[job]["result"] = "skipped"
                    self.assertTrue(gate.check(altered, event))
            for job in gate.PR_RENDER_JOBS | {"bevel-pr"}:
                with self.subTest(event=event, job=job):
                    altered = needs_for(suites, event)
                    altered[job]["result"] = "success"
                    self.assertTrue(gate.check(altered, event))

    def test_invalid_plan_or_matrix_fails(self):
        for suites in (["unknown"], ["full"], ["broad", "cycloidal-pr"], ["cycloidal-pr", "cycloidal-pr"]):
            self.assertTrue(gate.check(needs_for(suites), "push"))
        needs = needs_for(["cycloidal-pr"])
        needs["plan"]["outputs"]["matrix"] = '{"include":[]}'
        self.assertTrue(gate.check(needs, "push"))
        self.assertTrue(gate.check(needs_for([]), "workflow_dispatch"))
        del needs["plan"]["outputs"]["plan"]
        self.assertTrue(gate.check(needs, "push"))

    def test_suite_expansion_preserves_all_shards(self):
        matrix = gate.matrix_for(["wheel", "siren", "bevel-pr", "model-cycloidal", "package-cornercheck"])
        self.assertEqual(len(matrix["include"]), 23)
        self.assertEqual([command[-1] for command in gate.commands("bevel-pr")],
                         ["abgh", "cdef", "rest-0", "rest-1", "rest-2", "rest-3"])
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
