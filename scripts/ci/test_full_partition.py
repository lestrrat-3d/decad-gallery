import unittest
from unittest import mock

import full_partition as partition


class FullPartitionTests(unittest.TestCase):
    def inventory(self):
        assigned = {test for tests in partition.run_tests.BEVEL_TESTS["full"].values() for test in tests}
        return {**{package: {"TestDedicated"} for package in partition.DEDICATED},
                partition.BEVEL: assigned | {"TestNewBevel"},
                partition.ROOT + "/models/handcrankfan": {"TestHandCrankFanDefault", "TestHandCrankFanVariant", "TestOther"},
                partition.ROOT + "/models/swisslever": {"TestForkAction", "TestOther"},
                partition.ROOT + "/models/newmodel": {"TestNewModel"}}

    def test_selection_keeps_new_packages_and_bevel_remainder(self):
        inventory = self.inventory()
        packages, expected, remainder = partition.select(inventory)
        self.assertFalse(set(packages) & (partition.DEDICATED | {partition.BEVEL}))
        self.assertEqual(set(packages) | partition.DEDICATED | {partition.BEVEL}, set(inventory))
        self.assertIn((partition.ROOT + "/models/newmodel", "TestNewModel"), expected)
        self.assertEqual(remainder, ["TestNewBevel"])

    def test_missing_dedicated_package_fails(self):
        inventory = self.inventory()
        del inventory[next(iter(partition.DEDICATED))]
        with self.assertRaises(ValueError):
            partition.select(inventory)

    def test_unknown_or_duplicate_bevel_assignment_fails(self):
        inventory = self.inventory()
        with mock.patch.object(partition.run_tests, "BEVEL_TESTS", {"full": {"a": ["TestMissing"]}}):
            with self.assertRaises(ValueError):
                partition.select(inventory)
        with mock.patch.object(partition.run_tests, "BEVEL_TESTS", {"full": {"a": ["TestNewBevel"] * 2}}):
            with self.assertRaises(ValueError):
                partition.select(inventory)

    def test_dispatch_sets_full_variants_and_passes_remainder(self):
        inventory = self.inventory()

        renderer = partition.ROOT + "/tools/render"

        def command(args, **_kwargs):
            if args[1] == "list":
                return mock.Mock(stdout="\n".join([*inventory, renderer]))
            return mock.Mock(stdout="\n".join(inventory[args[-1]]))

        with mock.patch.object(partition.subprocess, "run", side_effect=command), \
                mock.patch.object(partition.run_tests, "go_test") as go_test:
            partition.main()
            partition.main("bevel-remainder")
            partition.main("handcrankfan", "default")
            partition.main("handcrankfan", "flat-three")
            partition.main("handcrankfan", "steep-three")
            partition.main("handcrankfan", "rest")
            partition.main("swisslever", "fork")
            partition.main("swisslever", "rest")
        self.assertEqual(go_test.call_count, 8)
        ordinary, bevel, default, flat, steep, rest, swiss_fork, swiss_rest = go_test.call_args_list
        self.assertEqual(ordinary.kwargs["env"]["HANDCRANKFAN_FULL"], "1")
        self.assertEqual(ordinary.kwargs["env"]["PLANETARY_ALL_KITS"], "1")
        self.assertNotIn(partition.ROOT + "/models/planetarygearbox", ordinary.args[0])
        self.assertNotIn(renderer, ordinary.args[0])
        self.assertEqual(bevel.args[1], ["TestNewBevel"])
        self.assertIn("TestHandCrankFanVariant/flat-three", flat.args[1])
        self.assertIn("TestHandCrankFanVariant/steep-three", steep.args[1])
        self.assertEqual(flat.args[0][2], "^TestHandCrankFanVariant$/^flat\\-three$")
        self.assertEqual(flat.kwargs["env"]["HANDCRANKFAN_VARIANTS"], "flat-three,steep-three")
        self.assertEqual(default.args[1], ["TestHandCrankFanDefault"])
        self.assertEqual(rest.args[1], ["TestOther"])
        self.assertEqual(flat.kwargs["env"]["HANDCRANKFAN_FULL"], "1")
        self.assertEqual(swiss_fork.args[1], ["TestForkAction"])
        self.assertEqual(swiss_rest.args[1], ["TestOther"])
        self.assertNotIn(partition.ROOT + "/models/handcrankfan", ordinary.args[0])
        self.assertNotIn(partition.ROOT + "/models/swisslever", ordinary.args[0])

    def test_fan_partition_rejects_missing_full_tests(self):
        with self.assertRaises(ValueError):
            partition.fan_tests({"TestHandCrankFanDefault"}, "rest")

    def test_swiss_partitions_cover_new_tests_and_reject_invalid_input(self):
        tests = {"TestForkAction", "TestOther", "TestNew"}
        fork, rest = (partition.swiss_tests(tests, case) for case in ("fork", "rest"))
        self.assertFalse(set(fork) & set(rest))
        self.assertEqual(set(fork) | set(rest), tests)
        for inventory, case in (({"TestOther"}, "rest"), (tests, "other")):
            with self.assertRaises(ValueError):
                partition.swiss_tests(inventory, case)


if __name__ == "__main__":
    unittest.main()
