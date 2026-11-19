import contextlib
import io
import json
import os
import subprocess
import unittest
from unittest import mock

import select_models as selector


def diff(status, path, old="100644", new="100644"):
    return f":{old} {new} abc def {status}\0{path}\0".encode()


class SelectionTests(unittest.TestCase):
    def test_model_paths_and_gallery_image(self):
        paths = diff("M", "models/onewayclutch/build.go") + diff("M", "docs/images/gallery.png")
        self.assertEqual(selector.changed_models(paths, {"onewayclutch"}), {"onewayclutch"})

    def test_repository_only_paths_do_not_force_broad_selection(self):
        paths = b"".join(
            diff("M", path)
            for path in (
                "README.md",
                "models/README.md",
                "models/plaquestand/README.md",
                "scripts/check_repository.py",
            )
        )
        self.assertEqual(selector.changed_models(paths, {"plaquestand"}), {"plaquestand"})

    def test_uncertain_paths_and_modes_fail_closed(self):
        for data in (
            diff("A", "models/onewayclutch/new.go", old="000000"),
            diff("D", "models/onewayclutch/old.go", new="000000"),
            diff("M", "models/onewayclutch/build.go", new="100755"),
            diff("M", "models/unknown/build.go"),
            diff("M", "tools/render/main.go"),
            diff("M", "models/onewayclutch/build.go")[:-1],
        ):
            with self.subTest(data=data), self.assertRaises(ValueError):
                selector.changed_models(data, {"onewayclutch"})

    def test_reverse_dependencies_include_tests_and_internal_packages(self):
        prefix = selector.MODULE + "/"
        packages = [{"ImportPath": prefix + "models/" + model} for model in selector.MODELS]
        packages += [
            {"ImportPath": prefix + "models/source"},
            {"ImportPath": prefix + "internal/helper", "TestImports": [prefix + "models/source"]},
            {"ImportPath": prefix + "models/consumer", "XTestImports": [prefix + "internal/helper"]},
        ]
        affected = selector.affected_models(packages, {"source"})
        self.assertIn("consumer", affected)
        self.assertNotIn("spinningwheel", affected)

    def test_missing_local_dependency_fails_closed(self):
        packages = [{"ImportPath": selector.MODULE + "/models/" + model} for model in selector.MODELS]
        packages[0]["Imports"] = [selector.MODULE + "/internal/missing"]
        with self.assertRaises(ValueError):
            selector.affected_models(packages, {next(iter(selector.MODELS))})

    def test_parent_mismatch_and_git_errors_run_all(self):
        for error in (ValueError("wrong merge"), subprocess.CalledProcessError(1, "git")):
            output = io.StringIO()
            with self.subTest(error=error), mock.patch.object(selector, "select", side_effect=error), \
                    mock.patch.object(selector.Path, "open", return_value=contextlib.nullcontext(output)), \
                    mock.patch.dict(os.environ, {"GITHUB_OUTPUT": "unused"}), \
                    contextlib.redirect_stdout(io.StringIO()):
                selector.main()
            selected = json.loads(output.getvalue().splitlines()[0].split("=", 1)[1])
            self.assertEqual(set(selected), selector.MODELS)

    def test_pr_parent_mismatch_is_rejected(self):
        with mock.patch.dict(os.environ, {"GITHUB_EVENT_NAME": "pull_request", "PR_BASE_SHA": "base",
                                         "PR_HEAD_SHA": "head"}), \
                mock.patch.object(selector, "run", return_value=b"merge base other\n"):
            with self.assertRaises(ValueError):
                selector.select()

    def test_summary_rejects_missing_selected_job(self):
        needs = {job: {"result": "skipped"} for job in selector.CHECKS}
        needs.update({"select": {"result": "success", "outputs": {"models": '["escapement"]'}},
                      "repository": {"result": "success"}, "bevel-full": {"result": "skipped"}})
        with mock.patch.dict(os.environ, {"JOB_RESULTS": json.dumps(needs), "GITHUB_EVENT_NAME": "pull_request"}), \
                contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit):
                selector.summarize()


if __name__ == "__main__":
    unittest.main()
