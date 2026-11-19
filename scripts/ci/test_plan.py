import os
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock
from contextlib import redirect_stdout
from io import StringIO

import gate
import plan


class PlanTest(unittest.TestCase):
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
                needs = {job: {"result": "skipped"} for job in gate.BROAD_JOBS}
                needs["records-validation"] = {"result": "success" if event in ("schedule", "workflow_dispatch")
                                                or "full" in expected else "skipped"}
                if "broad" in expected:
                    full = event in ("schedule", "workflow_dispatch") or "full" in expected
                    for job in gate.BROAD_JOBS:
                        enabled = job not in ("renderer-full", "bevel-full") and job not in gate.PR_RENDER_JOBS
                        enabled |= job in ("renderer-full", "bevel-full") and full
                        enabled |= job in gate.PR_RENDER_JOBS and not full and "renderer-all" not in expected
                        if job == "bevel-pr":
                            enabled = not full
                        needs[job]["result"] = "success" if enabled else "skipped"
                needs.update(plan={"result": "success", "outputs": {
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
        cases = (([], []), (["models/cycloidal/build_test.go"], ["cycloidal-pr"]),
                 ([".github/workflows/unknown.yml"], ["broad"]),
                 (["scripts/ci/unknown.py"], ["broad"]), (["go.mod"], ["broad", "full"]),
                 (["tools/render/gallery.go"], ["broad", "renderer-all"]))
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
                needs = {job: {"result": "skipped"} for job in gate.BROAD_JOBS}
                needs["records-validation"] = {"result": "success" if "full" in expected else "skipped"}
                needs.update(plan={"result": "success", "outputs": {
                    "plan": json.dumps(result), "matrix": json.dumps(matrix),
                    "count": str(len(matrix["include"]))}}, repository={"result": "success"},
                    worker={"result": "success" if matrix["include"] else "skipped"})
                if not extra:
                    self.assertEqual(matrix["include"], [])
                    self.assertEqual(gate.check(needs, "push"), [])
                    needs["repository"]["result"] = "failure"
                    self.assertTrue(gate.check(needs, "push"))

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

    def test_cycloidal_only_and_mixed_changes(self):
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=self.packages()):
            with mock.patch.object(plan.subprocess, "check_output", return_value=b"models/cycloidal/build_test.go\0"):
                self.assertEqual(plan.plan("push", "base", "head")["suites"], ["cycloidal-pr"])
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", return_value=self.packages()):
            with mock.patch.object(plan.subprocess, "check_output", return_value=b"models/cycloidal/build_test.go\0other.go\0"):
                self.assertEqual(plan.plan("push", "base", "head")["suites"], ["broad"])

    def test_unverified_merge_and_missing_refs_fall_back(self):
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base", b"head other parent"]):
            self.assertEqual(plan.plan("pull_request", "base", "head")["suites"], ["broad"])
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
            needs = {job: {"result": "skipped"} for job in gate.BROAD_JOBS}
            needs["records-validation"] = {"result": "skipped"}
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
        for path in ("go.mod", "go.sum", "tools/render/go.mod", "tools/render/go.sum"):
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

    def test_renderer_change_adds_full_renderer_to_broad(self):
        renderer = plan.ROOT + "/tools/render"
        graph = {renderer: {"dir": "tools/render", "imports": set(), "test_imports": set()}}
        for path in (b"tools/render/gallery.go\0", b"tools/render/readme_views_test.go\0"):
            with self.subTest(path=path), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", return_value=graph), \
                    mock.patch.object(plan.subprocess, "check_output", return_value=path):
                result = plan.plan("push", "base", "head")
                self.assertEqual(result["suites"], ["broad", "renderer-all"])
                self.assertEqual(gate.suites_from_plan(json.dumps(result)), result["suites"])
                self.assertEqual(gate.matrix_for(result["suites"]),
                                 {"include": [{"suite": "renderer-all", "case": 0}]})

    def test_reverse_imports_stop_at_test_edges(self):
        graph = {
            "a": {"dir": "a", "imports": set(), "test_imports": set()},
            "b": {"dir": "b", "imports": {"a"}, "test_imports": set()},
            "c": {"dir": "c", "imports": set(), "test_imports": {"b"}},
            "d": {"dir": "d", "imports": {"c"}, "test_imports": set()},
        }
        self.assertEqual(plan.affected_packages(["a/file.go"], graph, graph), {"a", "b", "c"})
        self.assertEqual(plan.affected_packages(["a/file_test.go"], graph, graph), {"a"})

    def test_docs_only_and_unknown_inputs(self):
        for paths, expected in ((b"README.md\0docs/example.md\0", []),
                                (b"models/clamparm/instructions.md\0", []),
                                (b"models/spinningwheel/README.md\0", []),
                                (b"models/spinningwheel/images/parts/a.png\0", ["wheel-parts"]),
                                (b"models/cycloidal/images/hero.png\0", ["broad"]),
                                (b"scripts/modelgen/spurgear.py\0", ["broad"]),
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
        bolt = plan.ROOT + "/models/m5bolt"
        before = {arm: {"dir": "models/clamparm", "imports": set(), "test_imports": set()},
                  bolt: {"dir": "models/m5bolt", "imports": set(), "test_imports": set()}}
        after = dict(before, **{helper: {"dir": "internal/clamparmcapacity", "imports": set(),
                                        "test_imports": set()}})
        after[arm] = {"dir": "models/clamparm", "imports": set(), "test_imports": {helper}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", side_effect=[before, after]), \
                mock.patch.object(plan.subprocess, "check_output",
                                  return_value=b"internal/clamparmcapacity/check.go\0models/m5bolt/bolt_test.go\0"):
            result = plan.plan("push", "base", "head")
        expected = {"package-clamparmcapacity", "clamparm", "model-m5bolt", "clamparm-render",
                    "renderer-compile"}
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
        for after, paths in ((dict(before, **added), b"internal/clamparmcapacity/check.go\0unknown.sh\0"),
                             (added, b"internal/clamparmcapacity/check.go\0")):
            with self.subTest(paths=paths), \
                    mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                    mock.patch.object(plan, "metadata", side_effect=[before, after]), \
                    mock.patch.object(plan.subprocess, "check_output", return_value=paths):
                self.assertEqual(plan.plan("push", "base", "head")["suites"], ["broad"])

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
                    self.assertEqual(plan.plan("push", "base", "head")["suites"], ["broad"])

    def test_test_import_change_and_removed_package(self):
        helper = plan.ROOT + "/internal/cornercheck"
        model = plan.ROOT + "/models/cycloidal"
        graph = {helper: {"dir": "internal/cornercheck", "imports": set(), "test_imports": set()},
                 model: {"dir": "models/cycloidal", "imports": set(), "test_imports": {helper}}}
        self.assertEqual(plan.affected_packages(["internal/cornercheck/check.go"], graph, graph), {helper, model})
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
        bolt = plan.ROOT + "/models/m5bolt"
        arm = plan.ROOT + "/models/clamparm"
        feature = plan.ROOT + "/features/dovetail"
        renderer = plan.ROOT + "/tools/render"
        graph = {
            bolt: {"dir": "models/m5bolt", "imports": set(), "test_imports": set()},
            arm: {"dir": "models/clamparm", "imports": {bolt}, "test_imports": set()},
            feature: {"dir": "features/dovetail", "imports": set(), "test_imports": set()},
            renderer: {"dir": "tools/render", "imports": {arm, feature}, "test_imports": set()},
        }
        cases = ((b"models/m5bolt/bolt.go\0", {"model-m5bolt", "clamparm", "clamparm-render", "renderer-compile"}),
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
                 (b"internal/clamparmcapacity/check.go\0", {"package-clamparmcapacity", "clamparm",
                                                               "clamparm-render", "renderer-compile"}),
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
        needs = {job: {"result": "skipped"} for job in gate.BROAD_JOBS}
        needs["records-validation"] = {"result": "skipped"}
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


