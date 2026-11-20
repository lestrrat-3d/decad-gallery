import json
import os
from pathlib import Path
import re
import tempfile
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
            if job in ("bevel-full", "renderer-full", "planetary-full") and not full:
                continue
            needs[job]["result"] = "success"
    needs.update(plan={"result": "success", "outputs": {
        "plan": json.dumps({"version": 1, "suites": suites}),
        "matrix": json.dumps(matrix), "count": str(len(matrix["include"]))}},
        repository={"result": "success"}, worker={"result": "success" if matrix["include"] else "skipped"},
        **{"records-validation": {"result": "success" if full else "skipped"}})
    return needs


RENDER_PACKAGE = "github.com/lestrrat-3d/decad-gallery/tools/render"


class GateTest(unittest.TestCase):
    def test_reused_plan_requires_a_valid_push_proof(self):
        needs = needs_for([])
        plan = json.loads(needs["plan"]["outputs"]["plan"])
        plan["reused"] = {"pull_request": 12, "run": 42, "tested": "a" * 40,
                          "suites": ["model-ballbearing"]}
        needs["plan"]["outputs"]["plan"] = json.dumps(plan)
        self.assertEqual(gate.check(needs, "push"), [])
        self.assertTrue(gate.check(needs, "pull_request"))
        plan["reused"]["tested"] = "not-a-sha"
        needs["plan"]["outputs"]["plan"] = json.dumps(plan)
        self.assertTrue(gate.check(needs, "push"))

    def test_planetary_dispatch_has_every_shard(self):
        matrix = gate.matrix_for(["model-planetarygearbox"])["include"]
        self.assertEqual([item["case"] for item in matrix], list(range(gate.PLANETARY_PR_SHARDS)))
        self.assertEqual(gate.commands("model-planetarygearbox")[0][-3:], ["planetary", "pr", "0"])

    def test_fan_dispatch_has_every_shard(self):
        matrix = gate.matrix_for(["model-handcrankfan"])["include"]
        self.assertEqual([item["case"] for item in matrix], list(range(gate.FAN_PR_SHARDS)))
        self.assertEqual(gate.commands("model-handcrankfan")[0][-3:], ["fan", "pr", "0"])

    @classmethod
    def setUpClass(cls):
        Path(".tmp").mkdir(exist_ok=True)

    def test_isolated_test_dispatch(self):
        suite = "isolated-ballbearing-TestBearingCatalog"
        self.assertEqual(gate.matrix_for([suite]), {"include": [{"suite": suite, "case": 0}]})
        self.assertEqual(gate.check(needs_for([suite]), "push"), [])
        with mock.patch.dict(os.environ, {"SUITE": suite, "CASE": "0"}), \
                mock.patch.object(gate.sys, "argv", ["gate.py", "run"]), \
                mock.patch.object(gate, "go_test") as run:
            gate.main()
        run.assert_called_once_with(
            ["./models/ballbearing", "-run", "^TestBearingCatalog$", "-timeout", "5m", "-count=1"],
            ["TestBearingCatalog"], package="github.com/lestrrat-3d/decad-gallery/models/ballbearing")

    def test_isolated_package_dispatch(self):
        cases = (
            ("isolated-internal-hexnut-TestBuildSizes", "internal/hexnut", None),
            ("isolated-feature-dovetail-TestBuild", "features/dovetail", None),
            ("isolated-render-TestGalleryCache", "tools/render", None),
            ("isolated-tag-coplanar_test-spinningwheel-TestCoplanar", "models/spinningwheel", "coplanar_test"),
        )
        for suite, package, tag in cases:
            with self.subTest(suite=suite):
                command = gate.commands(suite)[0]
                self.assertEqual(command[-2:] if tag else command[-1:], ["-tags", tag] if tag else ["-count=1"])
                with mock.patch.dict(os.environ, {"SUITE": suite, "CASE": "0"}), \
                        mock.patch.object(gate.sys, "argv", ["gate.py", "run"]), \
                        mock.patch.object(gate, "go_test") as run:
                    gate.main()
                self.assertEqual(run.call_args.kwargs["package"], "github.com/lestrrat-3d/decad-gallery/" + package)
                if package == "tools/render":
                    self.assertNotIn("cwd", run.call_args.kwargs)
                    self.assertEqual(run.call_args.kwargs["module"], "tools/render")
                    self.assertIn("./tools/render", command)
        with self.assertRaises(ValueError):
            gate.commands("isolated-tag-coplanar_test-ballbearing-TestBearingCatalog")
        for suite in ("isolated-planetarygearbox-TestKitGearboxes", "isolated-handcrankfan-TestHandCrankFanDefault"):
            command = gate.commands(suite)[0]
            self.assertEqual(command[command.index("-timeout") + 1], "40m")

    def test_renderer_records_module(self):
        event = json.dumps({"Package": RENDER_PACKAGE, "Test": "TestSame", "Action": "pass"})
        proc = mock.Mock(stdout=[event], returncode=0)
        proc.wait.return_value = 0
        with tempfile.TemporaryDirectory(dir=".tmp") as directory:
            path = Path(directory) / "records.jsonl"
            with mock.patch.dict(os.environ, {"CI_TEST_RECORDS": str(path)}), \
                    mock.patch.object(gate.subprocess, "run", side_effect=[
                        mock.Mock(stdout=RENDER_PACKAGE), mock.Mock(stdout="TestSame\n")]), \
                    mock.patch.object(gate.subprocess, "Popen", return_value=proc):
                gate.run_renderer_full()
            self.assertEqual(json.loads(path.read_text().splitlines()[0])["module"], "tools/render")

    def test_gallery_cache_step_runs_only_on_pr_and_push(self):
        workflow = Path(".github/workflows/repository.yml").read_text()
        step = workflow.split("- name: Check gallery tile cache\n", 1)[1].split("- name:", 1)[0]
        self.assertEqual(step.splitlines()[0].strip(),
                         "if: needs.plan.result == 'success' && fromJSON(needs.plan.outputs.plan).check_gallery_cache")
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
                mock.patch.object(gate, "run_renderer_pr") as dispatch:
            gate.main()
        dispatch.assert_called_once_with("0")

    def test_renderer_shards_cover_each_package_test_once(self):
        p1, p2 = RENDER_PACKAGE, RENDER_PACKAGE + "/extra"
        inventory = {p1: ["TestA", "TestB"], p2: ["TestA", "TestC"]}
        def discover(command, **kwargs):
            if command[-2:] == ["list", "./..."]:
                return mock.Mock(stdout=p1 + "\n" + p2 + "\nexample/other\n")
            return mock.Mock(stdout="\n".join(inventory[command[-1]]) + "\nok\n")
        with mock.patch.object(gate.subprocess, "run", side_effect=discover), \
                mock.patch.object(gate, "go_test") as run:
            for shard in range(gate.RENDER_PR_SHARDS):
                gate.run_renderer_pr(str(shard))
        selected = [(call.kwargs["package"], test) for call in run.call_args_list for test in call.args[1]]
        expected = [(package, test) for package, tests in inventory.items() for test in tests]
        self.assertCountEqual(selected, expected)
        self.assertEqual(len(selected), len(set(selected)))
        with self.assertRaisesRegex(SystemExit, "invalid renderer"):
            gate.run_renderer_pr(str(gate.RENDER_PR_SHARDS))

    def test_full_renderer_inventory_requires_every_package_and_subtest(self):
        packages = [RENDER_PACKAGE, RENDER_PACKAGE + "/extra"]
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
        for suite, (timeout, directory, tests) in gate.RENDER_TESTS.items():
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
                    ["go", "test", "-json", "-run", gate.run_pattern(tests),
                     "-timeout", timeout, "./" + directory], stdout=gate.subprocess.PIPE, text=True,
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
                  for test in gate.RENDER_TESTS["swiss-render"][2]]
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
            event = "schedule" if job in ("bevel-full", "renderer-full", "planetary-full") else "push"
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

    def test_record_validation_required_only_for_full_runs(self):
        for suites, event in ((["broad"], "schedule"), (["broad"], "workflow_dispatch"),
                              (["broad", "full"], "push"), (["broad", "full"], "pull_request")):
            for result in ("failure", "cancelled", "skipped", "missing"):
                with self.subTest(event=event, result=result):
                    needs = needs_for(suites, event)
                    needs["records-validation"]["result"] = result
                    self.assertTrue(gate.check(needs, event))
        needs = needs_for([], "pull_request")
        needs["records-validation"]["result"] = "success"
        self.assertTrue(gate.check(needs, "pull_request"))

    def test_full_selection_requires_full_jobs_on_push_and_pull_request(self):
        for event in ("push", "pull_request"):
            suites = ["broad", "full"]
            needs = needs_for(suites, event)
            self.assertEqual(gate.matrix_for(suites), {"include": []})
            self.assertEqual(gate.check(needs, event), [])
            for job in ("renderer-full", "bevel-full", "planetary-full"):
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
        for suites in (["unknown"], ["full"], ["broad", "unknown"], ["cycloidal-pr", "cycloidal-pr"]):
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

    def test_broad_preserves_selected_model_and_renderer(self):
        suites = ["broad", "model-swisslever", "swiss-render"]
        self.assertEqual(gate.matrix_for(suites), {"include": [
            {"suite": "model-swisslever", "case": 0}, {"suite": "swiss-render", "case": 0}]})
        needs = needs_for(suites)
        self.assertEqual(gate.check(needs, "push"), [])
        needs["worker"]["result"] = "failure"
        self.assertTrue(gate.check(needs, "push"))


if __name__ == "__main__":
    unittest.main()
