import os
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock
from contextlib import redirect_stdout
from io import StringIO

import gate
import plan


class PlanTest(unittest.TestCase):
    def test_main_push_reuses_only_a_verified_result(self):
        import reuse_main
        original = {"version": 1, "tested": "a" * 40, "base": "b" * 40, "changed_paths": ["x.go"],
                    "suites": ["model-ballbearing"], "reasons": {"model-ballbearing": ["changed"]}}
        proof = {"pull_request": 12, "run": 42, "tested": "c" * 40}
        with mock.patch.dict(os.environ, {"GITHUB_EVENT_NAME": "push", "GITHUB_ACTIONS": "true"}, clear=True), \
                mock.patch.object(plan, "check_wheel_inventory"), \
                mock.patch.object(plan, "plan", return_value=original), \
                mock.patch.object(reuse_main, "find_reusable_run", return_value=proof), \
                redirect_stdout(StringIO()) as output:
            plan.main()
        result = json.loads(output.getvalue())
        self.assertEqual(result["suites"], [])
        self.assertEqual(result["reused"]["suites"], ["model-ballbearing"])

    def test_override_disables_pull_request_reuse(self):
        import reuse_main
        result = {"version": 1, "tested": "a" * 40, "suites": ["broad", "full"],
                  "reasons": {"broad": ["changed"], "full": ["changed"]}}
        env = {"GITHUB_EVENT_NAME": "push", "GITHUB_ACTIONS": "true", "CI_BROAD_OVERRIDE": "true"}
        with mock.patch.dict(os.environ, env, clear=True), mock.patch.object(plan, "check_wheel_inventory"), \
                mock.patch.object(plan, "plan", return_value=result), \
                mock.patch.object(reuse_main, "find_reusable_run") as find, redirect_stdout(StringIO()) as output:
            plan.main()
        find.assert_not_called()
        self.assertEqual(json.loads(output.getvalue())["suites"], ["broad", "full"])

    def test_isolated_selection_works_without_existing_tmp_directory(self):
        project = Path.cwd()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "scripts/ci").mkdir(parents=True)
            (root / "scripts/ci/testdiff.go").write_bytes((project / "scripts/ci/testdiff.go").read_bytes())
            test_file = root / "models/ballbearing/catalog_test.go"
            test_file.parent.mkdir(parents=True)
            source = 'package ballbearing_test\n\n// CI: isolated-tests\n\nimport "testing"\n\n'
            source += 'func TestBearingCatalog(t *testing.T) { t.Log("before") }\n'
            test_file.write_text(source)
            subprocess.run(["git", "init", "-q"], cwd=root, check=True)
            subprocess.run(["git", "add", str(test_file.relative_to(root))], cwd=root, check=True)
            commit = ["git", "-c", "user.name=CI test", "-c", "user.email=ci@example.invalid",
                      "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "-qm"]
            subprocess.run(commit + ["before"], cwd=root, check=True)
            test_file.write_text(source.replace('"before"', '"after"'))
            subprocess.run(["git", "add", str(test_file.relative_to(root))], cwd=root, check=True)
            subprocess.run(commit + ["after"], cwd=root, check=True)
            self.assertFalse((root / ".tmp").exists())
            code = ('import json, plan; print(json.dumps(plan.isolated_tests('
                    '"HEAD^", "HEAD", ["models/ballbearing/catalog_test.go"])))')
            env = dict(os.environ, PYTHONPATH=str(project / "scripts/ci"))
            output = subprocess.check_output([sys.executable, "-c", code], cwd=root, env=env, text=True)
            self.assertEqual(json.loads(output), ["isolated-ballbearing-TestBearingCatalog"])
            self.assertTrue((root / ".tmp").is_dir())

    def test_single_isolated_test_selects_exact_worker(self):
        path = b"models/ballbearing/catalog_test.go\0"
        suite = "isolated-ballbearing-TestBearingCatalog"
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan.subprocess, "check_output", return_value=path), \
                mock.patch.object(plan, "isolated_tests", return_value=[suite]):
            result = plan.plan("push", "base", "head")
        self.assertEqual(result["suites"], [suite])
        self.assertEqual(gate.matrix_for(result["suites"])["include"], [{"suite": suite, "case": 0}])

    def test_broad_override_and_gate(self):
        cases = (("push", "true", ["cycloidal-pr"], ["broad", "cycloidal-pr"]),
                 ("pull_request", "true", ["model-swisslever", "swiss-render"],
                  ["broad", "model-swisslever", "swiss-render"]),
                 ("pull_request", "true", [], ["broad"]),
                 ("push", "true", ["broad", "new-model-cycloidal", "renderer-all"],
                  ["broad", "new-model-cycloidal", "renderer-all"]),
                 ("push", "true", ["broad", "full"], ["broad", "full"]),
                 ("push", "false", ["cycloidal-pr"], ["cycloidal-pr"]),
                 ("schedule", "true", ["broad"], ["broad"]),
                 ("workflow_dispatch", "true", ["broad"], ["broad"]))
        for event, override, initial, expected in cases:
            with self.subTest(event=event, override=override, initial=initial), \
                    mock.patch.dict(os.environ, {"GITHUB_EVENT_NAME": event, "CI_BROAD_OVERRIDE": override},
                                    clear=True), \
                    mock.patch.object(plan, "check_wheel_inventory"), \
                    mock.patch.object(plan, "plan", return_value={
                        "version": 1, "suites": initial, "reasons": {s: ["original"] for s in initial}}), \
                    redirect_stdout(StringIO()) as output:
                plan.main()
                result = json.loads(output.getvalue())
                self.assertEqual(result["suites"], expected)
                matrix = gate.matrix_for(expected)
                needs = {job: {"result": "skipped"} for job in (*gate.BROAD_JOBS, "case-workers")}
                needs["records-validation"] = {"result": "success" if event in ("schedule", "workflow_dispatch")
                                                or "full" in expected else "skipped"}
                needs["package-workers"] = dict(needs["records-validation"])
                if "broad" in expected:
                    full = event in ("schedule", "workflow_dispatch") or "full" in expected
                    for job in gate.BROAD_JOBS:
                        enabled = job not in ("renderer-full", "bevel-full", "planetary-full", "bevel-remainder",
                                              "handcrankfan-full", "swisslever-full") and job not in gate.PR_RENDER_JOBS
                        enabled |= job in ("renderer-full", "bevel-full", "planetary-full", "bevel-remainder",
                                           "handcrankfan-full", "swisslever-full") and full
                        enabled |= job in gate.PR_RENDER_JOBS and not full and "renderer-all" not in expected
                        if job == "bevel-pr":
                            enabled = not full
                        needs[job]["result"] = "success" if enabled else "skipped"
                needs.update(plan={"result": "success", "outputs": {
                    "case_plan": "null", "case_matrix": '{"include":[]}',
                    "plan": json.dumps(result), "matrix": json.dumps(matrix),
                    "count": str(len(matrix["include"]))}}, repository={"result": "success"},
                    worker={"result": "success" if matrix["include"] else "skipped"})
                self.assertEqual(gate.check(needs, event), [])
                if "broad" in expected:
                    needs["wheel-motion" if not full and "renderer-all" not in expected else "siren"]["result"] = "failure"
                    self.assertTrue(gate.check(needs, event))

    def packages(self):
        return {"github.com/lestrrat-3d/decad-gallery/models/cycloidal": {
            "dir": "models/cycloidal", "imports": set(), "test_imports": set()}}

    def test_ci_changes_select_only_needed_jobs(self):
        # PR #339 changes the CI workflows, documentation and these scripts.
        ci_paths = [".github/workflows/ci-probe.yml", ".github/workflows/repository.yml",
                    ".github/workflows/fan-variants.yml", ".github/workflows/update-decad.yml",
                    "docs/ci-measurements.md", "docs/ci-testing.md"] + sorted(
                        path for path in plan.CI_FILES if path.startswith("scripts/ci/"))
        cases = (([], []), (["scripts/check_repository.py"], []),
                 (["models/cycloidal/build_test.go"], ["cycloidal-pr"]),
                 ([".github/workflows/unknown.yml"], ["broad"]),
                 (["scripts/ci/unknown.py"], ["broad"]), (["go.mod"], ["broad", "full"]),
                 (["tools/render/gallery.go"], ["render-tool", "renderer-compile"]))
        renderer = plan.ROOT + "/tools/render"
        packages = self.packages() | {renderer: {"dir": "tools/render", "imports": set(), "test_imports": set()}}
        for extra, expected in cases:
            with self.subTest(extra=extra), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", return_value=packages), \
                    mock.patch.object(plan.subprocess, "check_output",
                                      return_value=("\0".join(ci_paths + extra) + "\0").encode()):
                result = plan.plan("push", "base", "head")
                self.assertEqual(result["suites"], expected)
                matrix = gate.matrix_for(gate.suites_from_plan(json.dumps(result)))
                needs = {job: {"result": "skipped"} for job in (*gate.BROAD_JOBS, "case-workers")}
                needs["records-validation"] = {"result": "success" if "full" in expected else "skipped"}
                needs["package-workers"] = dict(needs["records-validation"])
                needs.update(plan={"result": "success", "outputs": {
                    "case_plan": "null", "case_matrix": '{"include":[]}',
                    "plan": json.dumps(result), "matrix": json.dumps(matrix),
                    "count": str(len(matrix["include"]))}}, repository={"result": "success"},
                    worker={"result": "success" if matrix["include"] else "skipped"})
                if not extra:
                    self.assertEqual(matrix["include"], [])
                    self.assertEqual(gate.check(needs, "push"), [])
                    needs["repository"]["result"] = "failure"
                    self.assertTrue(gate.check(needs, "push"))

    def test_render_compare_files_select_no_suites(self):
        paths = "scripts/render_compare.py\0scripts/test_render_compare.py\0"
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan.subprocess, "check_output", return_value=paths.encode()):
            self.assertEqual(plan.plan("push", "base", "head")["suites"], [])

    def test_check_repository_only_selects_no_suites(self):
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan.subprocess, "check_output", return_value=b"scripts/check_repository.py\0"):
            result = plan.plan("push", "base", "head")
        self.assertEqual(result["suites"], [])
        self.assertNotIn("broad", result["reasons"])

    def test_check_repository_with_model_or_unknown_input(self):
        cases = ((b"scripts/check_repository.py\0models/cycloidal/build.go\0", ["cycloidal-pr", "renderer-compile"]),
                 (b"scripts/check_repository.py\0unknown.sh\0", ["broad"]))
        for paths, expected in cases:
            with self.subTest(paths=paths), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", return_value=self.packages()), \
                    mock.patch.object(plan.subprocess, "check_output", return_value=paths):
                self.assertEqual(plan.plan("push", "base", "head")["suites"], expected)

    def test_spurgear_generator_only_selects_no_suites(self):
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan.subprocess, "check_output", return_value=b"scripts/modelgen/spurgear.py\0"):
            result = plan.plan("push", "base", "head")
        self.assertEqual(result["suites"], [])
        self.assertNotIn("broad", result["reasons"])

    def test_spurgear_generator_with_unknown_input(self):
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=self.packages()), \
                mock.patch.object(plan.subprocess, "check_output",
                                  return_value=b"scripts/modelgen/spurgear.py\0unknown.sh\0"):
            self.assertEqual(plan.plan("push", "base", "head")["suites"], ["broad"])

    def test_each_existing_workflow_selects_no_geometry(self):
        for name in ("fan-variants.yml", "update-decad.yml"):
            with self.subTest(name=name), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan.subprocess, "check_output",
                                      return_value=(".github/workflows/" + name + "\0").encode()):
                self.assertEqual(plan.plan("push", "base", "head")["suites"], [])

    def test_inventory_only_selects_no_geometry(self):
        paths = (b"scripts/ci/inventory.py\0scripts/ci/test_inventory.py\0"
                 b"scripts/ci/reconcile_records.py\0scripts/ci/test_reconcile_records.py\0"
                 b"scripts/ci/test_workflow_records.py\0")
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan.subprocess, "check_output", return_value=paths):
            self.assertEqual(plan.plan("push", "base", "head")["suites"], [])

    def test_planetary_kit_inventory_helper_selects_no_geometry(self):
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan.subprocess, "check_output", return_value=b"scripts/ci/planetary_kits.go\0"):
            self.assertEqual(plan.plan("push", "base", "head")["suites"], [])

    def test_cycloidal_only_and_mixed_changes(self):
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=self.packages()):
            with mock.patch.object(plan.subprocess, "check_output", return_value=b"models/cycloidal/build_test.go\0"):
                self.assertEqual(plan.plan("push", "base", "head")["suites"], ["cycloidal-pr"])
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=self.packages()):
            with mock.patch.object(plan.subprocess, "check_output", return_value=b"models/cycloidal/build_test.go\0other.go\0"):
                self.assertEqual(plan.plan("push", "base", "head")["suites"], ["broad", "full"])

    def test_unverified_merge_and_missing_refs_fall_back(self):
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base", b"head other parent"]):
            self.assertEqual(plan.plan("pull_request", "base", "head")["suites"], ["broad", "full"])
        with mock.patch.object(plan, "git", side_effect=ValueError("missing")):
            self.assertIn("fallback", plan.plan("push", "base", "head")["reason"])

    def test_real_diff_drives_mocked_jobs_and_gate(self):
        Path(".tmp").mkdir(exist_ok=True)
        original = Path.cwd()
        with tempfile.TemporaryDirectory(dir=".tmp") as directory:
            repo = Path(directory).resolve()
            subprocess.run(["git", "init", "-q", str(repo)], check=True)
            def git(*args):
                return subprocess.check_output(
                    ["git", "-C", str(repo), "-c", "user.name=CI test", "-c",
                     "user.email=ci@example.invalid", *args], stderr=subprocess.DEVNULL
                ).decode().strip()
            test_file = repo / "models/cycloidal/example_test.go"
            test_file.parent.mkdir(parents=True)
            test_file.write_text("before\n")
            git("add", ".")
            git("commit", "-qm", "before")
            base = git("rev-parse", "HEAD")
            test_file.write_text("after\n")
            git("commit", "-qam", "after")
            tested = git("rev-parse", "HEAD")
            try:
                os.chdir(repo)
                with mock.patch.object(plan, "metadata", return_value=self.packages()):
                    result = plan.plan("push", base, tested)
            finally:
                os.chdir(original)
            self.assertEqual(result["changed_paths"], ["models/cycloidal/example_test.go"])
            self.assertEqual(result["suites"], ["cycloidal-pr"])
            dispatch = mock.Mock(return_value="success")
            needs = {job: {"result": "skipped"} for job in (*gate.BROAD_JOBS, "case-workers")}
            needs["records-validation"] = {"result": "skipped"}
            needs["package-workers"] = {"result": "skipped"}
            needs.update(plan={"result": "success", "outputs": {
                "plan": json.dumps(result), "matrix": json.dumps(gate.matrix_for(result["suites"])),
                "count": "1"}}, repository={"result": "success"}, worker={"result": "success"})
            needs["worker"]["result"] = dispatch(result["suites"][0])
            dispatch.assert_called_once_with("cycloidal-pr")
            self.assertEqual(gate.check(needs, "push"), [])
            needs["worker"]["result"] = "cancelled"
            self.assertTrue(gate.check(needs, "push"))

    def test_dispatch_always_broad(self):
        self.assertEqual(plan.plan("workflow_dispatch", "", "head")["suites"], ["broad"])

    def test_module_files_select_full_coverage(self):
        for path in ("go.mod", "go.sum"):
            for event in ("push", "pull_request"):
                with self.subTest(path=path, event=event), \
                     mock.patch.object(plan, "git", side_effect=lambda *args: {
                         ("rev-parse", "HEAD"): b"head",
                         ("rev-parse", "--verify", "head^{commit}"): b"head",
                         ("rev-parse", "--verify", "base^{commit}"): b"base",
                         ("rev-list", "--parents", "-n", "1", "head"): b"head base other",
                     }[args]), \
                     mock.patch.object(plan.subprocess, "check_output", return_value=(path + "\0README.md\0").encode()), \
                     mock.patch.object(plan, "metadata") as metadata:
                    result = plan.plan(event, "base", "head")
                    self.assertEqual(result["suites"], ["broad", "full"])
                    self.assertIn("dependency", result["reason"])
                    metadata.assert_not_called()

    def test_render_tool_source_change_selects_render_tool(self):
        renderer = plan.ROOT + "/tools/render"
        graph = {renderer: {"dir": "tools/render", "imports": set(), "test_imports": set()}}
        for path in (b"tools/render/gallery.go\0", b"tools/render/readme_views_test.go\0"):
            with self.subTest(path=path), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", return_value=graph), \
                    mock.patch.object(plan.subprocess, "check_output", return_value=path):
                result = plan.plan("push", "base", "head")
                expected = ["render-tool", "renderer-compile"] if path.endswith(b"gallery.go\0") else ["render-tool"]
                self.assertEqual(result["suites"], expected)
                with mock.patch.object(Path, "is_dir", return_value=True):
                    self.assertEqual(gate.suites_from_plan(json.dumps(result)), result["suites"])

    def test_reverse_imports_stop_at_test_edges(self):
        graph = {
            "a": {"dir": "a", "imports": set(), "test_imports": set()},
            "b": {"dir": "b", "imports": {"a"}, "test_imports": set()},
            "c": {"dir": "c", "imports": set(), "test_imports": {"b"}},
            "d": {"dir": "d", "imports": {"c"}, "test_imports": set()},
        }
        self.assertEqual(plan.affected_packages(["a/file.go"], graph, graph), ({"a", "b", "c"}, {"a", "b"}))
        self.assertEqual(plan.affected_packages(["a/file_test.go"], graph, graph), ({"a"}, set()))

    def test_docs_only_and_unknown_inputs(self):
        for paths, expected in ((b"README.md\0docs/example.md\0", []),
                                (b"models/clamparm/instructions.md\0", []),
                                (b"models/spinningwheel/README.md\0", []),
                                (b"models/spinningwheel/images/parts/a.png\0", ["wheel-parts"]),
                                (b"models/cycloidal/images/hero.png\0", []),
                                (b"docs/images/gallery.png\0models/cycloidal/README.md\0", []),
                                (b"models/cycloidal/testdata/golden.png\0", ["broad"]),
                                (b"scripts/modelgen/spurgear.py\0", []),
                                (b"README.md\0unknown.sh\0", ["broad"])):
            with self.subTest(paths=paths), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", return_value=self.packages()), \
                    mock.patch.object(plan.subprocess, "check_output", return_value=paths):
                self.assertEqual(plan.plan("push", "base", "head")["suites"], expected)

    def test_model_and_transitive_consumers_dispatch(self):
        prefix = plan.ROOT + "/models/"
        graph = {
            prefix + "turbinefan": {"dir": "models/turbinefan", "imports": set(), "test_imports": set()},
            prefix + "handcrankfan": {"dir": "models/handcrankfan", "imports": {prefix + "turbinefan"},
                                      "test_imports": set()},
        }
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=graph), \
                mock.patch.object(plan.subprocess, "check_output", return_value=b"models/turbinefan/build.go\0"):
            suites = plan.plan("push", "base", "head")["suites"]
        self.assertEqual(set(suites), {"turbinefan", "model-handcrankfan", "fan-motion", "fan-view",
                                       "renderer-compile"})
        dispatch = mock.Mock()
        for suite in suites:
            dispatch(suite)
        self.assertEqual(dispatch.call_count, 5)
        self.assertNotIn("wheel", suites)

    def test_deleted_or_new_package_falls_back(self):
        with self.assertRaises(ValueError):
            plan.affected_packages(["models/new/build.go"], self.packages(), {})

    def test_registered_new_package_and_mixed_model_dispatch(self):
        helper = plan.ROOT + "/internal/clamparmcapacity"
        arm = plan.ROOT + "/models/clamparm"
        bolt = plan.ROOT + "/models/fastener"
        before = {arm: {"dir": "models/clamparm", "imports": set(), "test_imports": set()},
                  bolt: {"dir": "models/fastener", "imports": set(), "test_imports": set()}}
        after = dict(before, **{helper: {"dir": "internal/clamparmcapacity", "imports": set(),
                                        "test_imports": set()}})
        after[arm] = {"dir": "models/clamparm", "imports": set(), "test_imports": {helper}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", side_effect=[before, after]), \
                mock.patch.object(plan.subprocess, "check_output",
                                  return_value=b"internal/clamparmcapacity/check.go\0models/fastener/bolt_test.go\0"):
            result = plan.plan("push", "base", "head")
        expected = {"package-clamparmcapacity", "clamparm", "model-fastener"}
        self.assertEqual(set(result["suites"]), expected)
        matrix = gate.matrix_for(gate.suites_from_plan(json.dumps(result)))
        dispatch = mock.Mock()
        for entry in matrix["include"]:
            dispatch(entry["suite"])
        self.assertEqual({call.args[0] for call in dispatch.call_args_list}, expected)

    def test_registered_new_package_mixed_unknown_or_deleted_falls_back(self):
        helper = plan.ROOT + "/internal/clamparmcapacity"
        arm = plan.ROOT + "/models/clamparm"
        before = {arm: {"dir": "models/clamparm", "imports": set(), "test_imports": set()}}
        added = {helper: {"dir": "internal/clamparmcapacity", "imports": set(), "test_imports": set()}}
        for after, paths, expected in (
                (dict(before, **added), b"internal/clamparmcapacity/check.go\0unknown.sh\0",
                 ["broad", "package-clamparmcapacity"]),
                (added, b"internal/clamparmcapacity/check.go\0", ["broad", "full"])):
            with self.subTest(paths=paths), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", side_effect=[before, after]), \
                    mock.patch.object(plan.subprocess, "check_output", return_value=paths):
                self.assertEqual(plan.plan("push", "base", "head")["suites"], expected)

    def test_new_model_dispatches_broad_and_direct_test(self):
        graph = self.packages()
        added = dict(graph, **{plan.ROOT + "/models/newpart": {
            "dir": "models/newpart", "imports": set(), "test_imports": set()}})
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", side_effect=[graph, added]), \
                mock.patch.object(plan.subprocess, "check_output",
                                  return_value=b"models/newpart/build.go\0unknown.sh\0"):
            result = plan.plan("push", "base", "head")
        self.assertEqual(result["suites"], ["broad", "new-model-newpart"])

    def test_new_package_keeps_existing_affected_suites(self):
        helper = plan.ROOT + "/internal/clamparmcapacity"
        arm = plan.ROOT + "/models/clamparm"
        new = plan.ROOT + "/models/newpart"
        renderer = plan.ROOT + "/tools/render"
        before = {
            helper: {"dir": "internal/clamparmcapacity", "imports": set(), "test_imports": set()},
            arm: {"dir": "models/clamparm", "imports": set(), "test_imports": {helper}},
            renderer: {"dir": "tools/render", "imports": set(), "test_imports": set()},
        }
        after = dict(before, **{new: {"dir": "models/newpart", "imports": set(), "test_imports": set()}})
        for extra in (b"", b"unknown.sh\0", b"tools/render/gallery.go\0"):
            paths = b"models/newpart/build.go\0internal/clamparmcapacity/check.go\0" + extra
            with self.subTest(extra=extra), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", side_effect=[before, after]), \
                    mock.patch.object(plan.subprocess, "check_output", return_value=paths):
                result = plan.plan("push", "base", "head")
            expected = {"new-model-newpart", "package-clamparmcapacity", "clamparm"}
            if extra == b"unknown.sh\0":
                expected.discard("clamparm")
                expected.add("broad")
            if extra.startswith(b"tools/render/"):
                expected |= {"render-tool", "renderer-compile"}
            self.assertEqual(set(result["suites"]), expected)
            self.assertEqual(len(result["suites"]), len(expected))
            with mock.patch.object(Path, "is_dir", return_value=True):
                self.assertEqual(gate.suites_from_plan(json.dumps(result)), result["suites"])

    def render_graph(self, names, extra=None):
        """Graph where tools/render imports every render package and every render package imports the core."""
        core = plan.ROOT + "/internal/renderkit"
        graph = {core: {"dir": "internal/renderkit", "imports": set(), "test_imports": set()}}
        tool = plan.ROOT + "/tools/render"
        graph[tool] = {"dir": "tools/render", "imports": {core}, "test_imports": set()}
        for name in names:
            kind = "features" if name in plan.FEATURES else "models"
            item = f"{plan.ROOT}/{kind}/{name}"
            graph[item] = {"dir": f"{kind}/{name}", "imports": set(), "test_imports": set()}
            graph[item + "/render"] = {"dir": f"{kind}/{name}/render", "imports": {core, item},
                                       "test_imports": set()}
            graph[tool]["imports"] |= {item + "/render"}
        for package, imports in (extra or {}).items():
            graph[plan.ROOT + "/" + package]["imports"] |= {plan.ROOT + "/" + name for name in imports}
        return graph

    def plan_for(self, paths, graph):
        data = "".join(path + "\0" for path in paths).encode()
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=graph), \
                mock.patch.object(plan.subprocess, "check_output", return_value=data):
            return plan.plan("push", "base", "head")

    def test_render_package_change_selects_only_its_suite(self):
        graph = self.render_graph(["clamparm", "pinring"])
        result = self.plan_for(["models/clamparm/render/clamparm.go"], graph)
        self.assertEqual(set(result["suites"]), {"render-model-clamparm", "renderer-compile"})
        self.assertNotIn("renderer-all", result["suites"])

    def test_render_change_reaches_importing_render_packages(self):
        graph = self.render_graph(["onewayclutch", "turbinefan"],
                                  {"models/turbinefan/render": ["models/onewayclutch/render"]})
        result = self.plan_for(["models/onewayclutch/render/onewayclutch.go"], graph)
        self.assertEqual(set(result["suites"]), {"render-model-onewayclutch", "render-model-turbinefan", "renderer-compile"})

    def test_render_test_change_does_not_reach_importers(self):
        graph = self.render_graph(["onewayclutch", "turbinefan"],
                                  {"models/turbinefan/render": ["models/onewayclutch/render"]})
        result = self.plan_for(["models/onewayclutch/render/extra_test.go"], graph)
        self.assertEqual(set(result["suites"]), {"render-model-onewayclutch"})

    def test_model_change_selects_its_render_suite_not_the_named_one(self):
        graph = self.render_graph(["clamparm"])
        graph[plan.ROOT + "/models/clamparm/render"]["imports"] |= {plan.ROOT + "/models/clamparm"}
        graph[plan.ROOT + "/tools/render"]["imports"] |= {plan.ROOT + "/models/clamparm"}
        result = self.plan_for(["models/clamparm/clamparm.go"], graph)
        self.assertEqual(set(result["suites"]), {"clamparm", "render-model-clamparm", "renderer-compile"})

    def test_shared_model_change_selects_affected_render_suites(self):
        graph = self.render_graph(["fastener", "crankhandle", "plaquestand"])
        fastener = plan.ROOT + "/models/fastener"
        crank = plan.ROOT + "/models/crankhandle"
        for name in ("models/fastener/render", "models/crankhandle", "models/crankhandle/render"):
            graph[plan.ROOT + "/" + name]["imports"].add(fastener)
        graph[plan.ROOT + "/models/plaquestand/render"]["imports"].add(crank + "/render")
        result = self.plan_for(["models/fastener/fastener.go"], graph)
        renders = {suite for suite in result["suites"] if suite.startswith("render-")}
        self.assertEqual(renders, {"render-model-fastener", "render-model-crankhandle", "render-model-plaquestand"})
        result = self.plan_for(["models/crankhandle/crankhandle.go"], graph)
        renders = {suite for suite in result["suites"] if suite.startswith("render-")}
        self.assertEqual(renders, {"render-model-crankhandle", "render-model-plaquestand"})

    def test_shared_core_with_unknown_input_selects_full(self):
        graph = self.render_graph(["clamparm"])
        self.assertEqual(self.plan_for(["internal/renderkit/scene.go"], graph)["suites"], ["renderer-all"])
        self.assertEqual(self.plan_for(["internal/renderkit/scene.go", "unknown.sh"], graph)["suites"],
                         ["broad", "full"])

    def test_tool_font_change_selects_render_tool_without_broad(self):
        graph = self.render_graph(["clamparm"])
        self.assertEqual(self.plan_for(["tools/render/fonts/DejaVuSerif.ttf"], graph)["suites"], ["render-tool"])
        result = self.plan_for(["tools/render/fonts/DejaVuSerif.ttf", "unknown.sh"], graph)
        self.assertEqual(set(result["suites"]), {"broad", "render-tool"})

    def registration_files(self, tile=False, models_extra="", gallery_extra=""):
        imp = '\tsevenrender "' + plan.ROOT + '/models/sevensegment/render"\n'
        base = {"models.go": "import (\n\tother \"x\"\n)\nvar m = []M{\n\tother.Model(),\n}\n",
                "gallery.go": "var tiles = []T{\n\t{\"other\", image.Rect(0, 0, 1, 1)},\n}\n",
                "registry_test.go": "func T() {\n\trequire.Len(t, paths, 3)\n}\n"}
        head = {"models.go": base["models.go"].replace("import (\n", "import (\n" + imp).replace(
                    "}\n", "\tsevenrender.Model(),\n}\n") + models_extra,
                "gallery.go": base["gallery.go"] + gallery_extra,
                "registry_test.go": base["registry_test.go"].replace("3)", "4)")}
        if tile:
            head["gallery.go"] = head["gallery.go"].replace(
                "}\n", "\t{\"sevensegment\", image.Rect(0, 0, 2, 2)},\n}\n")
        return base, head

    def registration_result(self, base, head, changed):
        def show(cmd, **kwargs):
            rev, path = cmd[2].split(":", 1)
            return (base if rev == "base" else head)[Path(path).name].encode()

        paths = {"tools/render/" + name for name in changed}
        with mock.patch.object(plan.subprocess, "check_output", side_effect=show):
            return plan.only_new_registration("base", "head", paths, {"sevensegment"})

    def test_registration_without_gallery_tile_is_accepted(self):
        base, head = self.registration_files()
        self.assertTrue(self.registration_result(base, head, ["models.go", "registry_test.go"]))

    def test_registration_with_one_gallery_tile_is_accepted(self):
        base, head = self.registration_files(tile=True)
        self.assertTrue(self.registration_result(base, head, ["models.go", "gallery.go", "registry_test.go"]))

    def test_registration_with_unrelated_models_edit_is_rejected(self):
        base, head = self.registration_files(models_extra="var extra = 1\n")
        self.assertFalse(self.registration_result(base, head, ["models.go", "registry_test.go"]))

    def test_registration_with_gallery_edit_that_is_not_one_tile_is_rejected(self):
        base, head = self.registration_files(gallery_extra="var extra = 1\n")
        self.assertFalse(self.registration_result(base, head, ["models.go", "gallery.go", "registry_test.go"]))
        base, head = self.registration_files(tile=True, gallery_extra="var extra = 1\n")
        self.assertFalse(self.registration_result(base, head, ["models.go", "gallery.go", "registry_test.go"]))

    def test_registration_without_models_or_test_change_is_rejected(self):
        base, head = self.registration_files()
        self.assertFalse(self.registration_result(base, head, ["models.go"]))
        self.assertFalse(self.registration_result(base, head, ["models.go", "gallery.go"]))

    def test_registration_without_tile_selects_registration_suite(self):
        base, head = self.registration_files()
        graph = self.render_graph(["clamparm"])
        after = self.render_graph(["clamparm", "sevensegment"])
        paths = "tools/render/models.go\0tools/render/registry_test.go\0models/sevensegment/render/r.go\0"

        def run(files):
            def show(cmd, **kwargs):
                rev, path = cmd[2].split(":", 1)
                return files[rev == "base"][Path(path).name].encode()
            with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", side_effect=[graph, after]), \
                    mock.patch.object(plan.subprocess, "check_output", side_effect=lambda cmd, **kw: (
                        paths.encode() if cmd[:2] == ["git", "diff"] else show(cmd))):
                return plan.plan("push", "base", "head")

        result = run({True: base, False: head})
        self.assertIn("render-registration", result["suites"])
        self.assertNotIn("render-tool", result["suites"])
        base, head = self.registration_files(models_extra="var extra = 1\n")
        self.assertIn("render-tool", run({True: base, False: head})["suites"])

    def test_new_render_package_selects_its_suite_without_broad(self):
        before = self.render_graph(["clamparm", "pinring"])
        del before[plan.ROOT + "/models/pinring/render"]
        after = self.render_graph(["clamparm", "pinring"])
        data = b"models/pinring/render/pinring.go\0"
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", side_effect=[before, after]), \
                mock.patch.object(plan.subprocess, "check_output", return_value=data):
            result = plan.plan("push", "base", "head")
        self.assertEqual(set(result["suites"]), {"render-model-pinring", "renderer-compile"})

    def test_marked_render_test_selects_isolated_suite(self):
        project = Path.cwd()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "scripts/ci").mkdir(parents=True)
            (root / "scripts/ci/testdiff.go").write_bytes((project / "scripts/ci/testdiff.go").read_bytes())
            test_file = root / "models/clamparm/render/model_test.go"
            test_file.parent.mkdir(parents=True)
            source = 'package render_test\n\n// CI: isolated-tests\n\nimport "testing"\n\n'
            source += 'func TestModel(t *testing.T) { t.Log("before") }\n'
            test_file.write_text(source)
            subprocess.run(["git", "init", "-q"], cwd=root, check=True)
            subprocess.run(["git", "add", str(test_file.relative_to(root))], cwd=root, check=True)
            commit = ["git", "-c", "user.name=CI test", "-c", "user.email=ci@example.invalid",
                      "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "-qm"]
            subprocess.run(commit + ["before"], cwd=root, check=True)
            test_file.write_text(source.replace('"before"', '"after"'))
            subprocess.run(["git", "add", str(test_file.relative_to(root))], cwd=root, check=True)
            subprocess.run(commit + ["after"], cwd=root, check=True)
            code = ('import json, plan; print(json.dumps(plan.isolated_tests('
                    '"HEAD^", "HEAD", ["models/clamparm/render/model_test.go"])))')
            env = dict(os.environ, PYTHONPATH=str(project / "scripts/ci"))
            output = subprocess.check_output([sys.executable, "-c", code], cwd=root, env=env, text=True)
            self.assertEqual(json.loads(output), ["isolated-modelrender-clamparm-TestModel"])

    def test_shared_render_core_change_selects_renderer_all(self):
        core = plan.ROOT + "/internal/renderkit"
        renderer = plan.ROOT + "/tools/render"
        graph = {core: {"dir": "internal/renderkit", "imports": set(), "test_imports": set()},
                 renderer: {"dir": "tools/render", "imports": {core}, "test_imports": set()}}
        cases = ((b"internal/renderkit/renderkit.go\0", ["renderer-all"]),
                 (b"internal/renderkit/renderkit.go\0unknown.sh\0", ["broad", "full"]),
                 (b"internal/renderkit/renderkit_test.go\0", ["package-renderkit"]))
        for paths, expected in cases:
            with self.subTest(paths=paths), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", return_value=graph), \
                    mock.patch.object(plan.subprocess, "check_output", return_value=paths):
                result = plan.plan("push", "base", "head")
            self.assertEqual(result["suites"], expected)

    def test_unknown_input_keeps_model_and_renderer_checks(self):
        model = plan.ROOT + "/models/swisslever"
        graph = {model: {"dir": "models/swisslever", "imports": set(), "test_imports": set()}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=graph), \
                mock.patch.object(plan.subprocess, "check_output",
                                  return_value=b"models/swisslever/build.go\0unknown.sh\0"):
            result = plan.plan("push", "base", "head")
        expected = {"broad", "model-swisslever", "renderer-compile", "swiss-render"}
        self.assertEqual(set(result["suites"]), expected)
        self.assertEqual(len(result["suites"]), len(expected))
        self.assertEqual(gate.suites_from_plan(json.dumps(result)), result["suites"])

    def test_broad_omits_only_suites_run_by_broad_jobs(self):
        helper = plan.ROOT + "/internal/cornercheck"
        wheel = plan.ROOT + "/models/spinningwheel"
        swiss = plan.ROOT + "/models/swisslever"
        graph = {helper: {"dir": "internal/cornercheck", "imports": set(), "test_imports": set()},
                 wheel: {"dir": "models/spinningwheel", "imports": {helper}, "test_imports": set()},
                 swiss: {"dir": "models/swisslever", "imports": {helper}, "test_imports": set()}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=graph), \
                mock.patch.object(plan.subprocess, "check_output",
                                  return_value=b"internal/cornercheck/check.go\0unknown.sh\0"):
            result = plan.plan("push", "base", "head")
        self.assertEqual(set(result["suites"]), {"broad", "model-swisslever", "renderer-compile",
                                                "swiss-render", "wheel-block"})
        self.assertEqual(gate.suites_from_plan(json.dumps(result)), result["suites"])

    def test_renderer_model_and_unknown_input_keep_all_checks(self):
        model = plan.ROOT + "/models/swisslever"
        renderer = plan.ROOT + "/tools/render"
        graph = {model: {"dir": "models/swisslever", "imports": set(), "test_imports": set()},
                 renderer: {"dir": "tools/render", "imports": {model}, "test_imports": set()}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=graph), \
                mock.patch.object(plan.subprocess, "check_output",
                                  return_value=b"tools/render/gallery.go\0models/swisslever/build.go\0unknown.sh\0"):
            result = plan.plan("push", "base", "head")
        expected = {"broad", "model-swisslever", "render-tool", "renderer-compile", "swiss-render"}
        self.assertEqual(set(result["suites"]), expected)
        self.assertEqual(len(result["suites"]), len(expected))
        with mock.patch.object(Path, "is_dir", return_value=True):
            self.assertEqual(gate.suites_from_plan(json.dumps(result)), result["suites"])

    def test_renderer_tool_change_selects_render_tool(self):
        model = plan.ROOT + "/models/swisslever"
        renderer = plan.ROOT + "/tools/render"
        graph = {model: {"dir": "models/swisslever", "imports": set(), "test_imports": set()},
                 renderer: {"dir": "tools/render", "imports": {model}, "test_imports": set()}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=graph), \
                mock.patch.object(plan.subprocess, "check_output",
                                  return_value=b"tools/render/gallery.go\0models/swisslever/build.go\0"):
            result = plan.plan("push", "base", "head")
        self.assertEqual(set(result["suites"]), {"model-swisslever", "render-tool", "renderer-compile", "swiss-render"})
        with mock.patch.object(Path, "is_dir", return_value=True):
            self.assertEqual(gate.suites_from_plan(json.dumps(result)), result["suites"])

    def test_deleted_package_full_covers_new_package_and_renderer(self):
        old = plan.ROOT + "/internal/oldpart"
        new = plan.ROOT + "/models/newpart"
        renderer = plan.ROOT + "/tools/render"
        before = {old: {"dir": "internal/oldpart", "imports": set(), "test_imports": set()},
                  renderer: {"dir": "tools/render", "imports": set(), "test_imports": set()}}
        after = {new: {"dir": "models/newpart", "imports": set(), "test_imports": set()},
                 renderer: before[renderer]}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", side_effect=[before, after]), \
                mock.patch.object(plan.subprocess, "check_output", return_value=(
                    b"internal/oldpart/check.go\0models/newpart/build.go\0tools/render/gallery.go\0")):
            result = plan.plan("push", "base", "head")
        self.assertEqual(result["suites"], ["broad", "full"])
        with mock.patch.object(Path, "is_dir", return_value=True):
            self.assertEqual(gate.suites_from_plan(json.dumps(result)), result["suites"])

    def test_unregistered_existing_package_and_new_package_run_directly(self):
        existing = plan.ROOT + "/internal/unregistered"
        new = plan.ROOT + "/models/newpart"
        before = {existing: {"dir": "internal/unregistered", "imports": set(), "test_imports": set()}}
        after = dict(before, **{new: {"dir": "models/newpart", "imports": set(), "test_imports": set()}})
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", side_effect=[before, after]), \
                mock.patch.object(plan.subprocess, "check_output",
                                  return_value=b"models/newpart/build.go\0internal/unregistered/check.go\0"):
            result = plan.plan("push", "base", "head")
        self.assertEqual(result["suites"], ["new-model-newpart", "new-package-unregistered"])
        with mock.patch.object(Path, "is_dir", return_value=True):
            self.assertEqual(gate.suites_from_plan(json.dumps(result)), result["suites"])

    def test_discovered_package_selects_production_and_test_consumers(self):
        helper = plan.ROOT + "/internal/newhelper"
        feature = plan.ROOT + "/features/newfeature"
        model = plan.ROOT + "/models/newpart"
        graph = {helper: {"dir": "internal/newhelper", "imports": set(), "test_imports": set()},
                 feature: {"dir": "features/newfeature", "imports": {helper}, "test_imports": set()},
                 model: {"dir": "models/newpart", "imports": set(), "test_imports": {feature}}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=graph), \
                mock.patch.object(plan.subprocess, "check_output", return_value=b"internal/newhelper/check.go\0"):
            result = plan.plan("push", "base", "head")
        self.assertEqual(result["suites"], ["new-feature-newfeature", "new-model-newpart",
                                            "new-package-newhelper"])
        with mock.patch.object(Path, "is_dir", return_value=True):
            self.assertEqual(len(gate.matrix_for(result["suites"])["include"]), 3)

    def test_discovered_consumers_select_their_renderers(self):
        helper = plan.ROOT + "/internal/newhelper"
        model = plan.ROOT + "/models/newpart"
        feature = plan.ROOT + "/features/newfeature"
        graph = {helper: {"dir": "internal/newhelper", "imports": set(), "test_imports": set()},
                 model: {"dir": "models/newpart", "imports": {helper}, "test_imports": set()},
                 feature: {"dir": "features/newfeature", "imports": {model}, "test_imports": set()},
                 model + "/render": {"dir": "models/newpart/render", "imports": {model}, "test_imports": set()},
                 feature + "/render": {"dir": "features/newfeature/render", "imports": {feature},
                                       "test_imports": set()}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=graph), \
                mock.patch.object(plan.subprocess, "check_output", return_value=b"internal/newhelper/check.go\0"):
            result = plan.plan("push", "base", "head")
        self.assertEqual(set(result["suites"]), {"new-package-newhelper", "new-model-newpart",
                                                 "new-feature-newfeature", "render-model-newpart",
                                                 "render-feature-newfeature", "renderer-compile"})
        with mock.patch.object(Path, "is_dir", return_value=True):
            self.assertEqual(len(gate.matrix_for(result["suites"])["include"]), 6)

    def test_registered_consumers_without_named_checks_select_renderers(self):
        helper = plan.ROOT + "/internal/cornercheck"
        model = plan.ROOT + "/models/pinring"
        feature = plan.ROOT + "/features/dovetail"
        graph = {helper: {"dir": "internal/cornercheck", "imports": set(), "test_imports": set()},
                 model: {"dir": "models/pinring", "imports": {helper}, "test_imports": set()},
                 feature: {"dir": "features/dovetail", "imports": {helper}, "test_imports": set()},
                 model + "/render": {"dir": "models/pinring/render", "imports": {model}, "test_imports": set()},
                 feature + "/render": {"dir": "features/dovetail/render", "imports": {feature},
                                       "test_imports": set()}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=graph), \
                mock.patch.object(plan.subprocess, "check_output", return_value=b"internal/cornercheck/check.go\0"):
            result = plan.plan("push", "base", "head")
        self.assertEqual(set(result["suites"]), {"package-cornercheck", "model-pinring", "feature-dovetail",
                                                 "render-model-pinring", "render-feature-dovetail", "renderer-compile"})
        with mock.patch.object(Path, "is_dir", return_value=True):
            self.assertEqual(len(gate.matrix_for(result["suites"])["include"]), 6)

    def test_named_renderer_consumer_reaches_dependent_renderer(self):
        helper = plan.ROOT + "/internal/cornercheck"
        model = plan.ROOT + "/models/clamparm"
        renderer = model + "/render"
        dependent = plan.ROOT + "/models/pinring/render"
        graph = {helper: {"dir": "internal/cornercheck", "imports": set(), "test_imports": set()},
                 model: {"dir": "models/clamparm", "imports": {helper}, "test_imports": set()},
                 renderer: {"dir": "models/clamparm/render", "imports": {model}, "test_imports": set()},
                 dependent: {"dir": "models/pinring/render", "imports": {renderer}, "test_imports": set()}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=graph), \
                mock.patch.object(plan.subprocess, "check_output", return_value=b"internal/cornercheck/check.go\0"):
            result = plan.plan("push", "base", "head")
        self.assertEqual(set(result["suites"]), {"package-cornercheck", "clamparm", "clamparm-render",
                                                 "render-model-pinring", "renderer-compile"})

    def test_new_model_and_feature_renderers_keep_distinct_consumers(self):
        model = plan.ROOT + "/models/newpart"
        feature = plan.ROOT + "/features/newpart"
        consumer = plan.ROOT + "/models/pinring/render"
        before = {consumer: {"dir": "models/pinring/render", "imports": set(), "test_imports": set()}}
        after = dict(before)
        for kind, owner in (("models", model), ("features", feature)):
            after[owner] = {"dir": f"{kind}/newpart", "imports": set(), "test_imports": set()}
            after[owner + "/render"] = {"dir": f"{kind}/newpart/render", "imports": {owner},
                                        "test_imports": set()}
        after[consumer] = {"dir": "models/pinring/render", "imports": {feature + "/render"},
                           "test_imports": set()}
        paths = (b"models/newpart/build.go\0models/newpart/render/render.go\0"
                 b"features/newpart/build.go\0features/newpart/render/render.go\0")
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", side_effect=[before, after]), \
                mock.patch.object(plan.subprocess, "check_output", return_value=paths):
            result = plan.plan("push", "base", "head")
        self.assertEqual(set(result["suites"]), {"new-model-newpart", "new-feature-newpart",
                                                 "render-model-newpart", "render-feature-newpart", "render-model-pinring"})
        with mock.patch.object(Path, "is_dir", return_value=True):
            self.assertEqual(gate.commands("render-feature-newpart"),
                             [["render-package", "features/newpart/render", "rest"]])
            self.assertEqual(gate.commands("render-model-newpart"),
                             [["render-package", "models/newpart/render", "rest"]])

    def test_undispatchable_existing_package_selects_full(self):
        package = plan.ROOT + "/internal/nested/helper"
        graph = {package: {"dir": "internal/nested/helper", "imports": set(), "test_imports": set()}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=graph), \
                mock.patch.object(plan.subprocess, "check_output", return_value=b"internal/nested/helper/check.go\0"):
            self.assertEqual(plan.plan("push", "base", "head")["suites"], ["broad", "full"])

    def test_invalid_new_renderer_path_fails_planning(self):
        package = plan.ROOT + "/models/bad-name/render"
        after = {package: {"dir": "models/bad-name/render", "imports": set(), "test_imports": set()}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", side_effect=[{}, after]), \
                mock.patch.object(plan.subprocess, "check_output", return_value=b"models/bad-name/render/x.go\0"):
            with self.assertRaisesRegex(RuntimeError, "new package cannot be dispatched"):
                plan.plan("push", "base", "head")

    def test_model_named_registration_selects_its_renderer(self):
        model = plan.ROOT + "/models/registration"
        graph = {model: {"dir": "models/registration", "imports": set(), "test_imports": set()},
                 model + "/render": {"dir": "models/registration/render", "imports": {model},
                                      "test_imports": set()}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", side_effect=[{}, graph]), \
                mock.patch.object(plan.subprocess, "check_output",
                                  return_value=b"models/registration/build.go\0models/registration/render/x.go\0"):
            result = plan.plan("push", "base", "head")
        self.assertEqual(set(result["suites"]), {"new-model-registration", "render-model-registration"})

    def test_missing_metadata_fails_closed_for_unknown_package(self):
        for path, raises in ((b"models/newpart/build.go\0", True),
                             (b"models/cycloidal/build.go\0", False)):
            with self.subTest(path=path), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", side_effect=OSError("metadata unavailable")), \
                    mock.patch.object(plan.subprocess, "check_output", return_value=path):
                if raises:
                    with self.assertRaises(RuntimeError):
                        plan.plan("push", "base", "head")
                else:
                    self.assertEqual(plan.plan("push", "base", "head")["suites"], ["broad", "full"])

    def test_known_helper_metadata_failure_and_invalid_base_select_full(self):
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", side_effect=OSError("metadata unavailable")), \
                mock.patch.object(plan.subprocess, "check_output",
                                  return_value=b"internal/cornercheck/check.go\0"):
            result = plan.plan("push", "base", "head")
        self.assertEqual(result["suites"], ["broad", "full"])
        self.assertEqual(gate.suites_from_plan(json.dumps(result)), result["suites"])
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", ValueError("invalid base")]):
            result = plan.plan("push", "invalid", "head")
        self.assertEqual(result["suites"], ["broad", "full"])
        self.assertEqual(gate.suites_from_plan(json.dumps(result)), result["suites"])

    def test_test_import_change_and_removed_package(self):
        helper = plan.ROOT + "/internal/cornercheck"
        model = plan.ROOT + "/models/cycloidal"
        graph = {helper: {"dir": "internal/cornercheck", "imports": set(), "test_imports": set()},
                 model: {"dir": "models/cycloidal", "imports": set(), "test_imports": {helper}}}
        self.assertEqual(plan.affected_packages(["internal/cornercheck/check.go"], graph, graph),
                         ({helper, model}, {helper}))
        with self.assertRaises(ValueError):
            plan.affected_packages(["internal/cornercheck/check.go"], graph, {model: graph[model]})

    def test_static_wheel_inventory(self):
        Path(".tmp").mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=".tmp") as directory:
            root = Path(directory)
            wheel = root / "models/spinningwheel"
            (wheel / "images/parts").mkdir(parents=True)
            (wheel / "images/parts/a.png").touch()
            readme = wheel / "README.md"
            row = '<img src="images/parts/a.png">'
            readme.write_text(row)
            plan.check_wheel_inventory(root)
            for contents in ("", row + row, '<img src="images/parts/missing.png">'):
                readme.write_text(contents)
                with self.assertRaises(ValueError):
                    plan.check_wheel_inventory(root)

    def test_wheel_source_checks_built_part_inventory(self):
        wheel = plan.ROOT + "/models/spinningwheel"
        graph = {wheel: {"dir": "models/spinningwheel", "imports": set(), "test_imports": set()}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=graph), \
                mock.patch.object(plan.subprocess, "check_output", return_value=b"models/spinningwheel/build.go\0"):
            suites = plan.plan("push", "base", "head")["suites"]
        self.assertEqual(set(suites), {"wheel", "wheel-motion", "wheel-parts", "wheel-block", "renderer-compile"})

    def test_model_sources_select_renderer_checks(self):
        cases = {"ballbearing": {"bearing-render"}, "bevelgear": {"bevel-render"},
                 "clamparm": {"clamparm-render"}, "escapement": {"escapement-exploded"},
                 "handcrankfan": {"fan-motion", "fan-view"}, "handcranksiren": {"siren-render"},
                 "planetarygearbox": {"planetary-motion", "planetary-render"},
                 "spinningwheel": {"wheel-motion", "wheel-parts", "wheel-block"},
                 "swisslever": {"swiss-render"}}
        for name, expected in cases.items():
            owner = plan.ROOT + "/models/" + name
            graph = {owner: {"dir": "models/" + name, "imports": set(), "test_imports": set()}}
            with self.subTest(model=name), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", return_value=graph), \
                    mock.patch.object(plan.subprocess, "check_output",
                                      return_value=("models/" + name + "/build.go\0").encode()):
                suites = plan.plan("push", "base", "head")["suites"]
                self.assertTrue(set(expected) <= set(suites))
                self.assertIn("renderer-compile", suites)

    def test_clamparm_dependencies_and_feature(self):
        bolt = plan.ROOT + "/models/fastener"
        arm = plan.ROOT + "/models/clamparm"
        feature = plan.ROOT + "/features/dovetail"
        renderer = plan.ROOT + "/tools/render"
        graph = {
            bolt: {"dir": "models/fastener", "imports": set(), "test_imports": set()},
            arm: {"dir": "models/clamparm", "imports": {bolt}, "test_imports": set()},
            feature: {"dir": "features/dovetail", "imports": set(), "test_imports": set()},
            renderer: {"dir": "tools/render", "imports": {arm, feature}, "test_imports": set()},
        }
        cases = ((b"models/fastener/bolt.go\0", {"model-fastener", "clamparm", "clamparm-render", "renderer-compile"}),
                 (b"models/clamparm/arm_test.go\0", {"clamparm"}),
                 (b"features/dovetail/dovetail.go\0", {"feature-dovetail", "renderer-compile"}))
        for path, expected in cases:
            with self.subTest(path=path), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", return_value=graph), \
                    mock.patch.object(plan.subprocess, "check_output", return_value=path):
                self.assertEqual(set(plan.plan("push", "base", "head")["suites"]), expected)

    def test_clamparm_capacity_dispatch(self):
        helper = plan.ROOT + "/internal/clamparmcapacity"
        arm = plan.ROOT + "/models/clamparm"
        graph = {
            helper: {"dir": "internal/clamparmcapacity", "imports": {arm}, "test_imports": set()},
            arm: {"dir": "models/clamparm", "imports": set(), "test_imports": {helper}},
        }
        cases = ((b"models/clamparm/instructions.md\0", set()),
                 (b"internal/clamparmcapacity/check_test.go\0", {"package-clamparmcapacity"}),
                 (b"internal/clamparmcapacity/check.go\0", {"package-clamparmcapacity", "clamparm"}),
                 (b"models/clamparm/capacity_test.go\0", {"clamparm"}))
        for path, expected in cases:
            with self.subTest(path=path), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", return_value=graph), \
                    mock.patch.object(plan.subprocess, "check_output", return_value=path):
                self.assertEqual(set(plan.plan("push", "base", "head")["suites"]), expected)

    def test_new_feature_gets_direct_test(self):
        feature = plan.ROOT + "/features/dovetail"
        graph = {feature: {"dir": "features/dovetail", "imports": set(), "test_imports": set()}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", side_effect=[{}, graph]), \
                mock.patch.object(plan.subprocess, "check_output", return_value=b"features/dovetail/demo.go\0"):
            result = plan.plan("push", "base", "head")
        self.assertEqual(result["suites"], ["feature-dovetail"])
        suites = gate.suites_from_plan(json.dumps(result))
        matrix = gate.matrix_for(suites)
        self.assertEqual({entry["suite"] for entry in matrix["include"]}, set(result["suites"]))
        needs = {job: {"result": "skipped"} for job in (*gate.BROAD_JOBS, "case-workers")}
        needs["records-validation"] = {"result": "skipped"}
        needs["package-workers"] = {"result": "skipped"}
        needs["renderer-full"]["result"] = "skipped"
        needs["bevel-full"]["result"] = "skipped"
        needs.update(plan={"result": "success", "outputs": {
            "plan": json.dumps(result), "matrix": json.dumps(matrix), "count": str(len(matrix["include"]))}},
            repository={"result": "success"}, worker={"result": "success"})
        self.assertEqual(gate.check(needs, "push"), [])
        with mock.patch.dict(os.environ, {"SUITE": "new-feature-dovetail", "CASE": "0"}), \
                mock.patch.object(gate.sys, "argv", ["gate.py", "run"]), \
                mock.patch.object(gate.subprocess, "run") as dispatch:
            gate.main()
        dispatch.assert_called_once_with(["go", "test", "./features/dovetail", "-timeout", "5m"], check=True)


