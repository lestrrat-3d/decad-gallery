import unittest
import json
from unittest import mock

import full_partition as partition


class FullPartitionTests(unittest.TestCase):
    def test_package_shards_are_bounded_deterministic_and_complete(self):
        packages = [f"example.com/model{i:03}" for i in range(100)]
        shards = partition.package_shards(packages)
        self.assertEqual(len(shards), partition.MAX_PACKAGE_SHARDS)
        self.assertEqual(shards, partition.package_shards(list(reversed(packages))))
        self.assertCountEqual([package for shard in shards for package in shard], packages)
        self.assertTrue(all(shards))
        with self.assertRaises(ValueError):
            partition.package_shards(["example.com/a", "example.com/a"])

    def test_cycloidal_leaves_ordinary_workers_only_with_a_case_plan(self):
        ordinary = ["example.com/ordinary", partition.CYCLOIDAL]
        self.assertCountEqual(partition.package_shards(ordinary, None)[0], ordinary)
        plan = {"manifest": {"package": partition.CYCLOIDAL}}
        self.assertEqual(partition.package_shards(ordinary, plan), [["example.com/ordinary"]])
        with self.assertRaises(ValueError):
            partition.package_shards(ordinary, {"manifest": {"package": "example.com/ordinary"}})

    def test_package_shard_allows_only_empty_package_skips(self):
        process = mock.Mock(stdout=[], returncode=0)
        process.wait.return_value = 0
        def finished(lines, qualified, package_outcomes, event_audit):
            package_outcomes["example.com/a"] = "skip"
            event_audit["packages"].add("example.com/a")
            event_audit["outcomes"].append(("example.com/a", "skip"))
            event_audit["no_tests"].add("example.com/a")
            return {}
        with mock.patch.object(partition, "ordinary_packages", return_value=["example.com/a"]), \
                mock.patch.object(partition, "discover", return_value=set()), \
                mock.patch.object(partition.subprocess, "Popen", return_value=process) as popen, \
                mock.patch.object(partition.run_tests, "finished_tests", side_effect=finished):
            partition.run_package_shard("0")
            self.assertIn("-p=1", popen.call_args.args[0])
            self.assertIn("-parallel=1", popen.call_args.args[0])
            with mock.patch.object(partition, "discover", return_value={"TestExpected"}), \
                    self.assertRaisesRegex(ValueError, "missing or failed"):
                partition.run_package_shard("0")

    def test_package_shard_rejects_bad_terminal_streams(self):
        package = "example.com/a"
        def event(action, **extra):
            return json.dumps({"Action": action, "Package": package, **extra})
        marker = event("output", Output="?   \texample.com/a\t[no test files]\n")
        streams = ([event("skip")], [event("run", Test="TestLost"), marker, event("skip")],
                   [marker, event("skip"), event("skip")],
                   [marker, event("skip"), json.dumps({"Action": "output", "Package": "example.com/other"})],
                   [marker, event("skip"), "not json"])
        with mock.patch.object(partition, "ordinary_packages", return_value=[package]), \
                mock.patch.object(partition, "discover", return_value=set()):
            for stream in streams:
                process = mock.Mock(stdout=stream)
                process.wait.return_value = 0
                with self.subTest(stream=stream), \
                        mock.patch.object(partition.subprocess, "Popen", return_value=process), \
                        self.assertRaisesRegex(ValueError, "missing or failed"):
                    partition.run_package_shard("0")

    def test_package_shard_rejects_incomplete_and_duplicate_test_events(self):
        package = "example.com/a"
        def event(action, test=None):
            return json.dumps({"Action": action, "Package": package, **({"Test": test} if test else {})})
        streams = ([event("run", "TestA"), event("run", "TestA/child"), event("pass", "TestA"), event("pass")],
                   [event("run", "TestA"), event("pass", "TestA"), event("pass", "TestA"), event("pass")])
        with mock.patch.object(partition, "ordinary_packages", return_value=[package]), \
                mock.patch.object(partition, "discover", return_value={"TestA"}):
            for stream in streams:
                process = mock.Mock(stdout=stream)
                process.wait.return_value = 0
                with self.subTest(stream=stream), \
                        mock.patch.object(partition.subprocess, "Popen", return_value=process), \
                        self.assertRaisesRegex(ValueError, "missing or failed"):
                    partition.run_package_shard("0")

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

    def test_dedicated_workers_discover_only_their_package(self):
        inventory = self.inventory()
        for mode, case, package in (("bevel-remainder", None, partition.BEVEL),
                                    ("handcrankfan", "rest", partition.ROOT + "/models/handcrankfan"),
                                    ("swisslever", "rest", partition.ROOT + "/models/swisslever")):
            with self.subTest(mode=mode), \
                    mock.patch.object(partition, "discover", return_value=inventory[package]) as discover, \
                    mock.patch.object(partition.run_tests, "go_test"):
                partition.main(mode, case)
                discover.assert_called_once_with(package)

    def test_discover_rejects_duplicate_names(self):
        with mock.patch.object(partition.subprocess, "run", return_value=mock.Mock(stdout="TestA\nTestA\n")):
            with self.assertRaises(ValueError):
                partition.discover("example/package")

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
