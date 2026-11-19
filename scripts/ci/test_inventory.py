import unittest
from unittest import mock
from pathlib import Path

import full_partition
import inventory
import run_tests


class InventoryTests(unittest.TestCase):
    def test_workflow_installs_go_before_ci_python_tests(self):
        workflow = (Path(__file__).resolve().parents[2] / ".github/workflows/repository.yml").read_text()
        repository = workflow.split("  repository:\n", 1)[1].split("\n  renderer-full:", 1)[0]
        self.assertLess(repository.index("actions/setup-go@"), repository.index("test_*.py"))

    def test_discover_only_lists_packages_and_tests(self):
        def run(args, **kwargs):
            self.assertTrue(kwargs["check"])
            if args[-2:] == ["list", "./..."]:
                return mock.Mock(stdout="example/a\nexample/b\n")
            self.assertEqual(args[-4:-1], ["test", "-list", "^Test"])
            return mock.Mock(stdout="TestExample\nok  example/a\n")

        with mock.patch.object(inventory.subprocess, "run", side_effect=run):
            self.assertEqual(inventory.discover(["go"]), {"example/a": ["TestExample"],
                                                           "example/b": ["TestExample"]})

    def test_report_uses_runner_assignments_and_names_unknown_subtests(self):
        root = {package: ["TestDedicated"] for package in full_partition.DEDICATED}
        root[full_partition.BEVEL] = [test for tests in run_tests.BEVEL_TESTS["full"].values() for test in tests]
        root[full_partition.BEVEL].append("TestRemainder")
        root[full_partition.ROOT + "/models/handcrankfan"] = ["TestHandCrankFanVariant"]
        result = inventory.report(root, {"renderer": ["TestRender"]})
        self.assertEqual(result["known_assignments"]["root"][full_partition.BEVEL]["TestRemainder"],
                         ["repository/bevel-remainder"])
        self.assertEqual(result["full_environment"], full_partition.FULL_ENV)
        self.assertIn("discovered at runtime", result["subtests"])
        self.assertEqual(result["modules"]["renderer"]["renderer"], ["TestRender"])
        self.assertEqual(result["known_assignments"]["renderer"]["renderer"]["TestRender"], ["renderer-full"])


if __name__ == "__main__":
    unittest.main()
