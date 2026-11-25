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
            self.assertEqual(args[-4:-1], ["test", "-list", "^(Test|Example|Fuzz)"])
            return mock.Mock(stdout="TestExample\nok  example/a\n")

        with mock.patch.object(inventory.subprocess, "run", side_effect=run):
            self.assertEqual(inventory.discover(["go"]), {"example/a": ["TestExample"],
                                                           "example/b": ["TestExample"]})

    def test_discovery_rejects_examples_and_fuzz_tests(self):
        for name in ("ExampleThing", "FuzzThing"):
            with self.subTest(name=name), mock.patch.object(inventory.subprocess, "run", side_effect=[
                    mock.Mock(stdout="example/a\n"), mock.Mock(stdout=name + "\n")]):
                with self.assertRaisesRegex(ValueError, name):
                    inventory.discover(["go"])

    def test_report_uses_runner_assignments_and_names_unknown_subtests(self):
        root = {package: ["TestDedicated"] for package in full_partition.DEDICATED}
        root[full_partition.ROOT + "/models/ordinary"] = ["TestOrdinary"]
        root[full_partition.BEVEL] = [test for tests in run_tests.BEVEL_TESTS["full"].values() for test in tests]
        root[full_partition.BEVEL].append("TestRemainder")
        root[full_partition.ROOT + "/models/handcrankfan"] = ["TestHandCrankFanDefault", "TestHandCrankFanVariant", "TestOther"]
        swiss = full_partition.ROOT + "/models/swisslever"
        root[swiss] = ["TestForkAction", "TestOther"]
        planetary = full_partition.ROOT + "/models/planetarygearbox"
        root[planetary] = ["TestKitGearboxes"] + [f"TestOther{i}" for i in range(5)]
        renderer_package = full_partition.ROOT + "/tools/render"
        result = inventory.report(root, {renderer_package: ["TestRender"]})
        self.assertEqual(result["known_assignments"]["root"][full_partition.BEVEL]["TestRemainder"],
                         ["bevel-remainder"])
        self.assertIsNone(result["case_plan"])
        self.assertEqual(result["full_environment"], full_partition.FULL_ENV)
        self.assertIn("discovered at runtime", result["subtests"])
        self.assertEqual(result["modules"]["renderer"][renderer_package], ["TestRender"])
        self.assertEqual(result["known_assignments"]["renderer"][renderer_package]["TestRender"], ["renderer-full/0"])
        segment = full_partition.ROOT + "/models/sevensegment"
        self.assertEqual(result["known_assignments"]["root"][segment]["TestSevenSegmentPlacement"],
                         ["sevensegment/" + case for case in run_tests.SEVENSEGMENT_PLACEMENTS])
        for _, name in run_tests.SEVENSEGMENT_PLACEMENTS.values():
            self.assertIn({"package": segment, "test": "TestSevenSegmentPlacement/" + name},
                          result["required_variants"])
        self.assertEqual(result["known_assignments"]["root"][swiss]["TestForkAction"], ["swisslever-full/fork"])
        self.assertEqual(result["known_assignments"]["root"][swiss]["TestOther"], ["swisslever-full/rest"])
        self.assertEqual(result["known_assignments"]["root"][planetary]["TestKitGearboxes"],
                         ["planetary-full/" + kit for kit in run_tests.PLANETARY_KITS])
        self.assertTrue(all({"package": planetary, "test": "TestKitGearboxes/" + kit} in
                            result["required_variants"] for kit in run_tests.PLANETARY_KITS))

    def test_split_renderer_separates_render_packages(self):
        root_package = full_partition.ROOT + "/models/ballbearing"
        renderer_package = full_partition.ROOT + "/tools/render"
        root, renderer = inventory.split_renderer({root_package: ["TestA"], renderer_package: ["TestB"]})
        self.assertEqual(root, {root_package: ["TestA"]})
        self.assertEqual(renderer, {renderer_package: ["TestB"]})


if __name__ == "__main__":
    unittest.main()
