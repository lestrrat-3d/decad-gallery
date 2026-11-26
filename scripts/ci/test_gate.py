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
            if job in ("bevel-full", "renderer-full", "planetary-full", "bevel-remainder", "handcrankfan-full", "swisslever-full") and not full:
                continue
            needs[job]["result"] = "success"
    needs.update(plan={"result": "success", "outputs": {
        "plan": json.dumps({"version": 1, "suites": suites}),
        "case_plan": "null", "case_matrix": '{"include":[]}',
        "matrix": json.dumps(matrix), "count": str(len(matrix["include"]))}},
        repository={"result": "success"}, worker={"result": "success" if matrix["include"] else "skipped"},
        **{name: {"result": "success" if full else "skipped"}
           for name in ("records-validation", "package-workers")})
    needs["case-workers"] = {"result": "skipped"}
    return needs


RENDER_PACKAGE = "github.com/lestrrat-3d/decad-gallery/tools/render"


class GateTest(unittest.TestCase):
    def test_package_workers_skip_only_without_full_coverage(self):
        self.assertEqual(gate.check(needs_for([]), "push"), [])
        full = needs_for(["broad", "full"])
        full["package-workers"]["result"] = "skipped"
        self.assertTrue(any("package-workers" in error for error in gate.check(full, "push")))

    def test_case_plan_requires_the_matrix_worker_and_record_validation(self):
        needs = needs_for(["broad", "full"])
        outputs = needs["plan"]["outputs"]
        from case_inventory import assignments
        manifest = {"parents": {"TestBuildOptions": ["first", "second"], "TestMeasuredFits": ["one", "two"]}}
        jobs = assignments(manifest)
        outputs["case_plan"] = json.dumps({"version": 1, "manifest": manifest, "jobs": jobs})
        self.assertTrue(gate.check(needs, "push"))
        outputs["case_matrix"] = json.dumps({"include": [{"case": job["case"]} for job in jobs]})
        self.assertTrue(gate.check(needs, "push"))
        needs["case-workers"]["result"] = "success"
        self.assertEqual(gate.check(needs, "push"), [])
        needs["records-validation"]["result"] = "failure"
        self.assertTrue(gate.check(needs, "push"))

    def test_reused_plan_requires_a_valid_push_proof(self):
        needs = needs_for([])
        plan = json.loads(needs["plan"]["outputs"]["plan"])
        plan["reused"] = {"pull_request": 12, "run": 42, "tested": "a" * 40,
                          "suites": ["model-ballbearing"]}
        needs["plan"]["outputs"]["plan"] = json.dumps(plan)
        self.assertEqual(gate.check(needs, "push"), [])
        self.assertTrue(gate.check(needs, "pull_request"))
        plan["reused"]["suites"] = ["broad", "full"]
        needs["plan"]["outputs"]["plan"] = json.dumps(plan)
        self.assertEqual(gate.check(needs, "push"), [])
        self.assertTrue(gate.check(needs, "pull_request"))
        plan["reused"]["suites"] = ["broad"]
        needs["plan"]["outputs"]["plan"] = json.dumps(plan)
        self.assertTrue(gate.check(needs, "push"))
        plan["reused"]["suites"] = ["broad", "full"]
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

    def test_isolated_sevensegment_race_test_runs_with_race_detector(self):
        race = gate.commands("isolated-sevensegment-TestSevenSegmentStepsRace")[0]
        self.assertEqual(race[:4], ["go", "test", "-race", "./models/sevensegment"])
        plain = gate.commands("isolated-sevensegment-TestSevenSegmentStepsDeterministic")[0]
        self.assertNotIn("-race", plain)

    def test_render_suites_expand_heavy_tests_and_rest(self):
        with mock.patch.object(Path, "is_dir", return_value=True):
            planetary = gate.commands("render-model-planetarygearbox")
            self.assertEqual(len(planetary), 4)
            self.assertEqual([case[2] for case in planetary],
                             list(gate.RENDER_HEAVY["planetarygearbox"]) + ["rest"])
            self.assertEqual(len(gate.commands("render-model-pinring")), 1)
            self.assertEqual(len(gate.commands("render-tool")), 16)
            for suite in ("render-model-planetarygearbox", "render-model-pinring", "render-tool"):
                matrix = gate.matrix_for([suite])
                self.assertEqual(len(matrix["include"]), len(gate.commands(suite)))
                self.assertEqual(gate.suites_from_plan(json.dumps({"version": 1, "suites": [suite]})), [suite])

    def test_render_suite_rejects_unknown_directory(self):
        with mock.patch.object(Path, "is_dir", return_value=False), self.assertRaises(ValueError):
            gate.commands("render-model-nothing")

    def test_registration_model_renderer_does_not_collide_with_registry_checks(self):
        with mock.patch.object(Path, "is_dir", return_value=True):
            self.assertEqual(gate.commands("render-model-registration"),
                             [["render-package", "models/registration/render", "rest"]])
            self.assertNotEqual(gate.commands("render-registration"), gate.commands("render-model-registration"))
            self.assertEqual(len(gate.matrix_for(["render-registration", "render-model-registration"])["include"]), 2)

    def test_feature_renderer_does_not_inherit_same_named_model_heavy_tests(self):
        with mock.patch.object(Path, "is_dir", return_value=True):
            self.assertEqual(gate.commands("render-feature-planetarygearbox"),
                             [["render-package", "features/planetarygearbox/render", "rest"]])
        with mock.patch.object(gate, "go_test") as run, \
                mock.patch.object(gate.subprocess, "run", return_value=mock.Mock(stdout="TestModel\nok\n")):
            gate.run_render_case("features/planetarygearbox/render", "rest")
        self.assertEqual(run.call_args.args[1], ["TestModel"])

    def test_render_package_case_runs_one_heavy_test_or_the_rest(self):
        with mock.patch.object(Path, "is_dir", return_value=True), \
                mock.patch.object(gate, "go_test") as run, \
                mock.patch.object(gate.subprocess, "run", return_value=mock.Mock(
                    stdout="TestModel\nTestSwissExploded\nTestOther\nok\n")):
            gate.run_render_case("models/swisslever/render", "TestSwissExploded")
            self.assertEqual(run.call_args.args[1], ["TestSwissExploded"])
            self.assertEqual(run.call_args.args[0][:3], ["./models/swisslever/render", "-run", "^TestSwissExploded$"])
            gate.run_render_case("models/swisslever/render", "rest")
            self.assertEqual(run.call_args.args[1], ["TestModel", "TestOther"])
            with self.assertRaises(SystemExit):
                gate.run_render_case("models/swisslever/render", "TestModel")

    def test_render_dispatch_runs_the_case_from_the_command(self):
        with mock.patch.object(Path, "is_dir", return_value=True), \
                mock.patch.dict(os.environ, {"SUITE": "render-model-pinring", "CASE": "0"}), \
                mock.patch.object(gate.sys, "argv", ["gate.py", "run"]), \
                mock.patch.object(gate, "run_render_case") as run:
            gate.main()
        run.assert_called_once_with("models/pinring/render", "rest")

    def test_isolated_render_suites_map_to_render_packages(self):
        with mock.patch.object(Path, "is_dir", return_value=True):
            self.assertEqual(gate.isolated_suite("isolated-modelrender-clamparm-TestModel"),
                             ("models", "clamparm/render", "TestModel"))
            self.assertEqual(gate.isolated_suite("isolated-featurerender-dovetail-TestModel"),
                             ("features", "dovetail/render", "TestModel"))
            command = gate.commands("isolated-modelrender-clamparm-TestModel")[0]
            self.assertEqual(command[:3], ["go", "test", "./models/clamparm/render"])

    def test_isolated_package_dispatch(self):
        cases = (
            ("isolated-internal-hexnut-TestBuildSizes", "internal/hexnut"),
            ("isolated-feature-dovetail-TestBuild", "features/dovetail"),
            ("isolated-render-TestGalleryCache", "tools/render"),
        )
        for suite, package in cases:
            with self.subTest(suite=suite):
                command = gate.commands(suite)[0]
                self.assertEqual(command[-1:], ["-count=1"])
                with mock.patch.dict(os.environ, {"SUITE": suite, "CASE": "0"}), \
                        mock.patch.object(gate.sys, "argv", ["gate.py", "run"]), \
                        mock.patch.object(gate, "go_test") as run:
                    gate.main()
                self.assertEqual(run.call_args.kwargs["package"], "github.com/lestrrat-3d/decad-gallery/" + package)
                if package == "tools/render":
                    self.assertNotIn("cwd", run.call_args.kwargs)
                    self.assertEqual(run.call_args.kwargs["module"], "tools/render")
                    self.assertIn("./tools/render", command)
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
                gate.run_renderer_full("0")
            self.assertEqual(json.loads(path.read_text().splitlines()[0])["module"], "tools/render")

    def test_renderer_jobs_do_not_duplicate_full_renderer(self):
        workflow = Path(".github/workflows/repository.yml").read_text()
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

    def test_full_renderer_shards_assign_every_test_once(self):
        inventory = {RENDER_PACKAGE: list(gate.RENDER_FULL_SECONDS) + ["TestOther"],
                     RENDER_PACKAGE + "/extra": ["TestSame"]}
        shards = gate.renderer_full_shards(inventory)
        assigned = [(package, test) for shard in shards for package, tests in shard for test in tests]
        self.assertCountEqual(assigned, [(package, test) for package, tests in inventory.items() for test in tests])
        self.assertEqual(len(assigned), len(set(assigned)))
        self.assertEqual(shards, gate.renderer_full_shards({p: list(reversed(t)) for p, t in inventory.items()}))
        with self.assertRaises(ValueError):
            gate.renderer_full_shards({RENDER_PACKAGE: ["TestSame", "TestSame"]})

    def test_full_renderer_dispatch_checks_assigned_tests(self):
        inventory = {RENDER_PACKAGE: list(gate.RENDER_FULL_SECONDS) + ["TestOther"]}
        with mock.patch.object(gate, "renderer_packages", return_value=list(inventory)), \
                mock.patch.object(gate.subprocess, "run", return_value=mock.Mock(stdout="\n".join(inventory[RENDER_PACKAGE]))), \
                mock.patch.object(gate, "go_test") as run:
            for shard in range(gate.RENDER_FULL_SHARDS):
                gate.run_renderer_full(str(shard))
        assigned = [test for call in run.call_args_list for test in call.args[1]]
        self.assertCountEqual(assigned, inventory[RENDER_PACKAGE])
        self.assertTrue(all(call.kwargs["module"] == "tools/render" for call in run.call_args_list))

    def test_renderer_dispatch_commands(self):
        for suite, (timeout, directory, tests) in gate.RENDER_TESTS.items():
            package = gate.MODULE + "/" + directory
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
            event = "schedule" if job in ("bevel-full", "renderer-full", "planetary-full", "bevel-remainder", "handcrankfan-full", "swisslever-full") else "push"
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
            for job in ("renderer-full", "bevel-full", "planetary-full", "bevel-remainder", "handcrankfan-full", "swisslever-full"):
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
