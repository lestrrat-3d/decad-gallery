import os
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import gate
import plan


class PlanTest(unittest.TestCase):
    def packages(self):
        return {"github.com/lestrrat-3d/decad-gallery/models/cycloidal": {
            "dir": "models/cycloidal", "imports": set(), "test_imports": set()}}

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
        self.assertEqual(set(suites), {"turbinefan", "model-handcrankfan", "fan-motion", "renderer-compile"})
        dispatch = mock.Mock()
        for suite in suites:
            dispatch(suite)
        self.assertEqual(dispatch.call_count, 4)
        self.assertNotIn("wheel", suites)

    def test_deleted_or_new_package_falls_back(self):
        with self.assertRaises(ValueError):
            plan.affected_packages(["models/new/build.go"], {}, self.packages())

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
        self.assertEqual(set(suites), {"wheel", "wheel-motion", "wheel-parts", "renderer-compile"})

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

    def test_new_feature_gets_direct_test(self):
        feature = plan.ROOT + "/features/dovetail"
        graph = {feature: {"dir": "features/dovetail", "imports": set(), "test_imports": set()}}
        with mock.patch.object(plan, "git", side_effect=[b"head", b"head", b"base"]), \
                mock.patch.object(plan, "metadata", side_effect=[{}, graph]), \
                mock.patch.object(plan.subprocess, "check_output", return_value=b"features/dovetail/demo.go\0"):
            self.assertEqual(plan.plan("push", "base", "head")["suites"], ["broad", "new-feature-dovetail"])


