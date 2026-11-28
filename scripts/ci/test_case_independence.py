import contextlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import case_inventory


class CycloidalIndependenceTest(unittest.TestCase):
    def test_exact_reviewed_model_and_parents(self):
        parents = ["TestBuildOptions", "TestCornersVariants", "TestMeasuredFits"]
        digest = case_inventory.file_digest(Path("models/cycloidal").glob("*.go"))
        with mock.patch.object(case_inventory, "CYCLOIDAL_REVIEWED_GO", digest):
            self.assertTrue(case_inventory.reviewed_cases("cycloidal", parents))
        self.assertFalse(case_inventory.reviewed_cases("cycloidal", parents[:1]))
        self.assertFalse(case_inventory.reviewed_cases("cycloidal", parents + ["TestBuildChamfers"]))
        self.assertFalse(case_inventory.reviewed_cases("other", parents))

    def test_changed_parent_or_helper_falls_back_after_inventory_refresh(self):
        examples = [
            ("build_test.go", "func TestBuildOptions(t *testing.T) {",
             "func TestBuildOptions(t *testing.T) {\n"
             "\tseen := 0\n\tt.Cleanup(func() { require.Equal(t, 13, seen) })"),
            ("clearance_test.go", "drive, err := cycloidal.Build(t.Context(), chainDocument(), p)",
             "_, drive, err := defaultFixture(t.Context(), p)"),
            ("build_helpers_test.go", "return decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))",
             "if defaultDriveFixture.doc != nil { return defaultDriveFixture.doc }\n"
             "\treturn decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))"),
        ]
        Path(".tmp").mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=".tmp") as directory:
            with contextlib.chdir(directory):
                model = Path("models/cycloidal")
                model.mkdir(parents=True)
                for filename, before, _ in examples:
                    (model / filename).write_text("package cycloidal_test\n" + before + "\n")
                approved = case_inventory.file_digest(model.glob("*.go"))
                config = {"version": 1, "parents": ["TestBuildOptions", "TestCornersVariants", "TestMeasuredFits"]}
                for filename, before, after in examples:
                    with self.subTest(filename=filename):
                        path = model / filename
                        original = path.read_text()
                        self.assertIn(before, original)
                        path.write_text(original.replace(before, after, 1))
                        config["source_sha256"] = case_inventory.file_digest(model.glob("*_test.go"))
                        (model / "ci_cases.json").write_text(json.dumps(config))
                        with mock.patch.object(case_inventory, "CYCLOIDAL_REVIEWED_GO", approved):
                            self.assertIsNone(case_inventory.discover("cycloidal"))
                        path.write_text(original)
