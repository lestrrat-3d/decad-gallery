import contextlib
import io
import json
import re
import unittest
from unittest import mock

import run_tests


class RunPatternTest(unittest.TestCase):
    def test_single_test(self):
        self.assertEqual(run_tests.run_pattern(["TestA"]), "^TestA$")

    def test_subtests_anchor_each_level(self):
        self.assertEqual(run_tests.run_pattern(["TestA/sub_b/c"]), "^TestA$/^sub_b$/^c$")

    def test_alternatives(self):
        self.assertEqual(run_tests.run_pattern(["TestA", "TestB/x"]), "^TestA$|^TestB$/^x$")

    def test_bevel_patterns_have_no_group(self):
        for mode, shards in run_tests.BEVEL_TESTS.items():
            for shard, tests in shards.items():
                with self.subTest(mode=mode, shard=shard):
                    self.assertNotIn("(", run_tests.run_pattern(tests))


class BevelJobGroupsTest(unittest.TestCase):
    def test_remainder_includes_new_tests_and_partially_selected_tests(self):
        available = ["TestRealEightToothFullPatternJoin", "TestSpiralInputsMembers", "TestNew"]
        shards = run_tests.bevel_remainder_shards(available)
        self.assertEqual({test for shard in shards for test in shard}, {"TestSpiralInputsMembers", "TestNew"})
        self.assertEqual(sum(map(len, shards)), 2)
        self.assertEqual(shards, run_tests.bevel_remainder_shards(list(reversed(available))))

    def test_remainder_skips_already_selected_subtest(self):
        listing = mock.Mock(stdout="TestSpiralInputsMembers\n")
        with mock.patch.object(run_tests.subprocess, "run", return_value=listing), \
             mock.patch.object(run_tests, "go_test") as go_test:
            run_tests.run_bevel("pr", "rest-0")
        command, expected = go_test.call_args.args
        self.assertEqual(expected, ["TestSpiralInputsMembers"])
        self.assertEqual(command[command.index("-skip") + 1], "^TestSpiralInputsMembers$/^module_0_5$")

    def test_empty_remainder_shard_succeeds_without_running_tests(self):
        with mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout="TestNew\n")), \
             mock.patch.object(run_tests, "go_test") as go_test:
            run_tests.run_bevel("pr", "rest-3")
        go_test.assert_not_called()

    def test_groups_partition_all_shards_once(self):
        shards = [shard for group in run_tests.BEVEL_JOB_GROUPS.values() for shard in group]
        self.assertEqual(sorted(shards), list("abcdefgh"))

    def test_group_runs_original_shards_in_order(self):
        with mock.patch.object(run_tests, "go_test") as go_test:
            run_tests.run_bevel("pr", "abgh")
        self.assertEqual(
            go_test.call_args_list,
            [
                mock.call(
                    [
                        "./models/bevelgear",
                        "-run",
                        run_tests.run_pattern(run_tests.BEVEL_TESTS["pr"][shard]),
                        "-timeout",
                        "15m",
                    ],
                    run_tests.BEVEL_TESTS["pr"][shard],
                )
                for shard in ("a", "b", "g", "h")
            ],
        )


class SirenCasesTest(unittest.TestCase):
    def test_remainder_runs_unassigned_tests(self):
        listing = mock.Mock(stdout="TestValidate\nTestHandCrankSirenVariant\nTestBuildRejectsNilDocument\nTestNew\n")
        with (
            mock.patch.object(run_tests.subprocess, "run", return_value=listing),
            mock.patch.object(run_tests, "go_test") as go_test,
        ):
            run_tests.run_siren("rest")
        tests = ["TestBuildRejectsNilDocument", "TestNew"]
        go_test.assert_called_once_with(
            ["./models/handcranksiren", "-run", run_tests.run_pattern(tests), "-timeout", "40m", "-count=1"],
            tests,
            package="github.com/lestrrat-3d/decad-gallery/models/handcranksiren",
        )

    def test_siren_cases(self):
        expected = {
            "default": (
                "^(TestHandCrankSirenDefault|TestValidate|TestMotion|TestPitch|TestGearboxParams)$",
                ["TestHandCrankSirenDefault", "TestValidate", "TestMotion", "TestPitch", "TestGearboxParams"],
            ),
            "spike": (
                "^TestGearboxSpike$",
                ["TestGearboxSpike"],
            ),
            "slots12": (
                "^TestHandCrankSirenVariant$/^slots_12$",
                ["TestHandCrankSirenVariant/slots_12"],
            ),
            "slots24": (
                "^TestHandCrankSirenVariant$/^slots_24$",
                ["TestHandCrankSirenVariant/slots_24"],
            ),
        }
        self.assertEqual(run_tests.SIREN_CASES, expected)

    def test_default_and_spike_partition_previous_selection(self):
        previous = {
            "TestHandCrankSirenDefault", "TestValidate", "TestMotion", "TestPitch",
            "TestGearboxParams", "TestGearboxSpike",
        }
        selected = []
        for case in ("default", "spike"):
            pattern, expected = run_tests.SIREN_CASES[case]
            matches = {name for name in previous if re.fullmatch(pattern, name)}
            self.assertEqual(matches, set(expected))
            selected.append(matches)
        self.assertTrue(selected[0].isdisjoint(selected[1]))
        self.assertEqual(selected[0] | selected[1], previous)

    def test_siren_cases_pass_expected_variant(self):
        for case, (pattern, expected) in run_tests.SIREN_CASES.items():
            with self.subTest(case=case):
                with mock.patch.object(run_tests, "go_test") as go_test:
                    run_tests.run_siren(case)
                args, kwargs = go_test.call_args
                self.assertEqual(args[0][0], "./models/handcranksiren")
                self.assertEqual(args[0][args[0].index("-run") + 1], pattern)
                self.assertEqual(args[0][-3:], ["-timeout", "40m", "-count=1"])
                self.assertEqual(args[1], expected)
                self.assertEqual(kwargs, {})


class TurbineFanTest(unittest.TestCase):
    def test_turbinefan_assigns_every_discovered_test(self):
        listing = mock.Mock(stdout="TestBladePrintEdge\nTestBuildBladeMirrored\nTestOther\nok pkg\n")
        with mock.patch.object(run_tests.subprocess, "run", return_value=listing), \
                mock.patch.object(run_tests, "go_test") as go_test:
            for shard in ("print", "mirror", "rest"):
                run_tests.run_turbinefan(shard)
        selected = [call.args[1] for call in go_test.call_args_list]
        self.assertEqual(selected, [["TestBladePrintEdge"], ["TestBuildBladeMirrored"], ["TestOther"]])

    def test_turbinefan_requires_both_heavy_tests(self):
        listing = mock.Mock(stdout="TestBladePrintEdge\n")
        with mock.patch.object(run_tests.subprocess, "run", return_value=listing):
            with self.assertRaisesRegex(SystemExit, "TestBuildBladeMirrored"):
                run_tests.run_turbinefan("print")


class EscapementTest(unittest.TestCase):
    def test_escapement_runs_all_shards(self):
        with mock.patch.object(run_tests, "go_test_shards") as go_test_shards:
            run_tests.run_escapement()
        go_test_shards.assert_called_once_with(
            "./models/escapement", run_tests.ESCAPEMENT_SHARDS, "40m"
        )

    def test_shards_cover_package_without_splitting_shared_build_tests(self):
        shards = run_tests.ESCAPEMENT_SHARDS
        flattened = [test for shard in shards for test in shard]
        self.assertEqual(len(flattened), 39)
        self.assertEqual(len(flattened), len(set(flattened)))
        self.assertEqual(
            shards[0],
            (
                "TestArborCollars",
                "TestCentreLine",
                "TestCorners",
                "TestEdgeFinishing",
                "TestFits",
                "TestInstallability",
                "TestInventory",
                "TestPlacement",
                "TestPressFitWalls",
                "TestSetJoinsEveryPart",
                "TestSketches",
                "TestSplitShafts",
                "TestSpringFits",
            ),
        )
        self.assertIn("TestCoupling", shards[1])
        self.assertIn("TestPoseNoOverlap", shards[1])

    def test_sharded_runner_checks_coverage_and_runs_two_processes(self):
        shards = (("TestA",), ("TestB",))
        list_result = mock.Mock(stdout="TestA\nTestB\n", returncode=0)
        pass_events = [
            json.dumps({"Action": "pass", "Test": test}) + "\n"
            for test in ("TestA", "TestB")
        ]

        class FakeProcess:
            def __init__(self, command, stdout, **kwargs):
                self.stdout = stdout
                self.returncode = 0
                self.stdout.write(pass_events[0 if "TestA" in command[command.index("-run") + 1] else 1])

            def wait(self):
                return self.returncode

            def poll(self):
                return self.returncode

        with (
            mock.patch.object(run_tests.subprocess, "run", return_value=list_result) as listing,
            mock.patch.object(run_tests.subprocess, "Popen", side_effect=FakeProcess) as popen,
            mock.patch("builtins.print"),
        ):
            run_tests.go_test_shards("./pkg", shards, "1m")

        listing.assert_called_once_with(
            ["go", "test", "-list", "^Test", "./pkg"],
            check=True,
            capture_output=True,
            text=True,
        )
        self.assertEqual(popen.call_count, 2)

    def test_sharded_runner_rejects_incomplete_coverage(self):
        with mock.patch.object(
            run_tests.subprocess,
            "run",
            return_value=mock.Mock(stdout="TestA\nTestB\n"),
        ), mock.patch.object(run_tests.subprocess, "Popen") as popen:
            with self.assertRaisesRegex(SystemExit, "unassigned tests: TestB"):
                run_tests.go_test_shards("./pkg", (("TestA",),), "1m")
        popen.assert_not_called()


class WheelTest(unittest.TestCase):
    def test_each_named_case_requires_its_expected_tests(self):
        for case in run_tests.WHEEL_CASES:
            with self.subTest(case=case), mock.patch.object(run_tests, "go_test") as go_test:
                run_tests.run_wheel(case)
                self.assertEqual(go_test.call_args.args[1], run_tests.WHEEL_EXPECTED[case])
                self.assertTrue(go_test.call_args.args[1])

    def test_remainder_includes_tests_without_wheel_prefix(self):
        with mock.patch.object(run_tests, "go_test") as go_test:
            run_tests.run_wheel("rest")
        command = go_test.call_args.args[0]
        self.assertEqual(command[command.index("-run") + 1], "^Test")
        self.assertEqual(command[command.index("-skip") + 1], run_tests.WHEEL_HEAVY)


class FanVariantsTest(unittest.TestCase):
    def test_fan_variant_patterns_select_each_full_subtest(self):
        expected = {
            "flat-three": "^TestHandCrankFanVariant$/^flat-three$",
            "steep-three": "^TestHandCrankFanVariant$/^steep-three$",
        }
        self.assertEqual(run_tests.FAN_VARIANTS, expected)
        for variant, pattern in expected.items():
            with self.subTest(variant=variant):
                with mock.patch.dict(run_tests.os.environ, {}, clear=True):
                    with mock.patch.object(run_tests, "go_test") as go_test:
                        run_tests.run_fan_variant(variant)
                args, kwargs = go_test.call_args
                self.assertEqual(args[0][0], "./models/handcrankfan")
                self.assertEqual(args[0][args[0].index("-run") + 1], pattern)
                self.assertEqual(args[1], [f"TestHandCrankFanVariant/{variant}"])
                self.assertEqual(args[2]["HANDCRANKFAN_FULL"], "1")
                self.assertEqual(kwargs, {})


class FinishedTestsTest(unittest.TestCase):
    def test_multi_package_selection_requires_each_package(self):
        events = [json.dumps({"Package": "one", "Test": "TestA", "Action": "pass"}),
                  json.dumps({"Package": "two", "Test": "TestA", "Action": "skip"})]
        process = mock.Mock(stdout=events, returncode=0)
        process.wait.return_value = 0
        with mock.patch.object(run_tests.subprocess, "Popen", return_value=process):
            with self.assertRaisesRegex(SystemExit, "two/TestA"):
                run_tests.go_test(["./one", "./two"], qualified_expected={
                    ("one", "TestA"), ("two", "TestA"),
                })

    def test_multi_package_selection_rejects_unexpected_skipped_subtest(self):
        events = [json.dumps({"Package": "one", "Test": "TestA", "Action": "pass"}),
                  json.dumps({"Package": "one", "Test": "TestA/child", "Action": "skip"})]
        process = mock.Mock(stdout=events, returncode=0)
        process.wait.return_value = 0
        with mock.patch.object(run_tests.subprocess, "Popen", return_value=process):
            with self.assertRaisesRegex(SystemExit, "one/TestA/child"):
                run_tests.go_test(["./one"], qualified_expected={("one", "TestA")})

    def test_only_explicit_skips_are_allowed(self):
        events = [json.dumps({"Package": "p", "Test": "TestA", "Action": "pass"}),
                  json.dumps({"Package": "p", "Test": "TestB", "Action": "skip"})]
        process = mock.Mock(stdout=events, returncode=0)
        process.wait.return_value = 0
        with mock.patch.object(run_tests.subprocess, "Popen", return_value=process):
            run_tests.go_test(["./p"], ["TestA", "TestB"], package="p", allowed_skips=frozenset({"TestB"}))
            with self.assertRaisesRegex(SystemExit, "p/TestB"):
                run_tests.go_test(["./p"], ["TestA", "TestB"], package="p")

    def test_other_package_cannot_hide_missing_selected_test(self):
        events = [json.dumps({"Package": "other", "Test": "TestA", "Action": "pass"})]
        process = mock.Mock(stdout=events, returncode=0)
        process.wait.return_value = 0
        with mock.patch.object(run_tests.subprocess, "Popen", return_value=process):
            with self.assertRaisesRegex(SystemExit, "TestA"):
                run_tests.go_test(["./p"], ["TestA"], package="p")

    def test_package_qualified_results_do_not_collide(self):
        lines = [json.dumps({"Package": "one", "Test": "TestA", "Action": "pass"}),
                 json.dumps({"Package": "two", "Test": "TestA", "Action": "skip"})]
        self.assertEqual(run_tests.finished_tests(lines, qualified=True),
                         {("one", "TestA"): "pass", ("two", "TestA"): "skip"})

    def test_qualified_runner_rejects_skipped_subtest(self):
        events = [json.dumps({"Package": "p", "Test": "TestA/child", "Action": "skip"}),
                  json.dumps({"Package": "p", "Test": "TestA", "Action": "pass"})]
        process = mock.Mock(stdout=events, returncode=0)
        process.wait.return_value = 0
        with mock.patch.object(run_tests.subprocess, "Popen", return_value=process):
            with self.assertRaisesRegex(SystemExit, "p/TestA/child"):
                run_tests.go_test(["./p"], ["TestA"], package="p")

    def test_summary(self):
        lines = [
            json.dumps({"Action": "run", "Test": "TestA"}) + "\n",
            json.dumps({"Action": "output", "Test": "TestA", "Output": "hello\n"}) + "\n",
            json.dumps({"Action": "pass", "Test": "TestA"}) + "\n",
            json.dumps({"Action": "skip", "Test": "TestB"}) + "\n",
            json.dumps({"Action": "pass", "Package": "p"}) + "\n",
            "not json\n",
        ]
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            result = run_tests.finished_tests(lines)
        self.assertEqual(result, {"TestA": "pass", "TestB": "skip"})
        self.assertEqual(out.getvalue(), "hello\nnot json\n")


class BroadModelSelectionTest(unittest.TestCase):
    def test_selected_names_are_required_in_each_assigned_package(self):
        mapping = {"cycloidal": ("TestShared", "TestParent/child"), "helicalgear": ("TestShared",)}
        with mock.patch.object(run_tests, "MODEL_PR_PACKAGES", mapping), \
             mock.patch.object(run_tests, "go_test") as go_test:
            run_tests.run_models("pr")
        prefix = "github.com/lestrrat-3d/decad-gallery/models/"
        self.assertEqual(go_test.call_args.kwargs["qualified_expected"], {
            (prefix + "cycloidal", "TestShared"),
            (prefix + "helicalgear", "TestShared"),
            (prefix + "cycloidal", "TestParent/child"),
        })

    def test_pattern_and_package_inventory_cover_same_names(self):
        assigned = {test for tests in run_tests.MODEL_PR_PACKAGES.values() for test in tests}
        self.assertEqual(assigned, set(run_tests.MODEL_PR_TESTS))


class FullModelInventoryTest(unittest.TestCase):
    def test_full_inventory_runs_remainder_and_requires_gated_cases(self):
        prefix = "github.com/lestrrat-3d/decad-gallery/models/"
        bevel, fan, planetary = (prefix + name for name in ("bevelgear", "handcrankfan", "planetarygearbox"))

        def listing(command, **kwargs):
            if command[1] == "list":
                return mock.Mock(stdout="\n".join((bevel, fan, planetary)))
            return mock.Mock(stdout={bevel: "TestAssigned\nTestRemainder\n",
                                     fan: "TestHandCrankFanVariant\n",
                                     planetary: "TestKitGearboxes\n"}[command[-1]])

        with mock.patch.object(run_tests, "BEVEL_TESTS", {"full": {"a": ["TestAssigned"]}}), \
             mock.patch.object(run_tests.subprocess, "run", side_effect=listing), \
             mock.patch.object(run_tests, "go_test") as go_test:
            run_tests.run_full_models()
        self.assertEqual(go_test.call_count, 2)
        args, kwargs = go_test.call_args_list[0]
        self.assertIn((fan, "TestHandCrankFanVariant/flat-three"), kwargs["qualified_expected"])
        self.assertIn((planetary, "TestKitGearboxes/planet608-shaft608"), kwargs["qualified_expected"])
        self.assertEqual(kwargs["env"]["HANDCRANKFAN_FULL"], "1")
        self.assertEqual(kwargs["env"]["PLANETARY_ALL_KITS"], "1")
        self.assertEqual(go_test.call_args_list[1].args[1], ["TestRemainder"])

    def test_duplicate_bevel_assignments_fail(self):
        with mock.patch.object(run_tests, "BEVEL_TESTS", {"full": {"a": ["TestA"], "b": ["TestA"]}}), \
             mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout="pkg\n")):
            with self.assertRaisesRegex(SystemExit, "duplicate"):
                run_tests.run_full_models()

    def test_unknown_bevel_assignment_fails(self):
        with mock.patch.object(run_tests, "BEVEL_TESTS", {"full": {"a": ["TestMissing"]}}), \
             mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout="pkg\n")):
            with self.assertRaisesRegex(SystemExit, "unknown full bevel shard tests: TestMissing"):
                run_tests.run_full_models()


class CycloidalPilotTest(unittest.TestCase):
    def test_gated_skips_are_scoped_to_model_and_environment(self):
        cases = {
            "handcrankfan": {"TestHandCrankFanDefault", "TestHandCrankFanVariant"},
            "planetarygearbox": {"TestKitGearboxes/planet608-shaft605", "TestKitGearboxes/planet608-shaft608"},
        }
        for name, expected in cases.items():
            with self.subTest(name=name), mock.patch.dict(run_tests.os.environ, {}, clear=True), \
                 mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout="TestA\n")), \
                 mock.patch.object(run_tests, "go_test") as go_test, mock.patch("builtins.print"):
                run_tests.run_model_pr(name)
                self.assertEqual(go_test.call_args.kwargs["allowed_skips"], expected)

    def test_runs_all_listed_tests_with_package_qualified_results(self):
        listing = mock.Mock(stdout="TestA\nTestB\nok  package\n")
        with (
            mock.patch.object(run_tests.subprocess, "run", return_value=listing),
            mock.patch.object(run_tests, "go_test") as go_test,
            mock.patch("builtins.print"),
        ):
            run_tests.run_cycloidal_pr()
        go_test.assert_called_once_with(
            ["./models/cycloidal", "-timeout", "60m", "-count=1"],
            ["TestA", "TestB"],
            package="github.com/lestrrat-3d/decad-gallery/models/cycloidal",
            allowed_skips=frozenset(),
        )

    def test_rejects_empty_inventory(self):
        with mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout="")):
            with self.assertRaisesRegex(SystemExit, "lists no tests"):
                run_tests.run_cycloidal_pr()

    def test_main_dispatches_pilot(self):
        with (
            mock.patch.object(run_tests.sys, "argv", ["run_tests.py", "cycloidal", "pr"]),
            mock.patch.object(run_tests, "run_cycloidal_pr") as pilot,
        ):
            run_tests.main()
        pilot.assert_called_once_with()

    def test_generic_model_runs_listed_tests_in_its_package(self):
        with (
            mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout="TestA\n")),
            mock.patch.object(run_tests, "go_test") as go_test,
            mock.patch("builtins.print"),
        ):
            run_tests.run_model_pr("spurgear")
        go_test.assert_called_once_with(
            ["./models/spurgear", "-timeout", "60m", "-count=1"],
            ["TestA"],
            package="github.com/lestrrat-3d/decad-gallery/models/spurgear",
            allowed_skips=frozenset(),
        )

    def test_generic_model_rejects_invalid_path(self):
        with mock.patch.object(run_tests.subprocess, "run") as listing:
            with self.assertRaisesRegex(SystemExit, "invalid model package name"):
                run_tests.run_model_pr("../other")
        listing.assert_not_called()

    def test_main_dispatches_generic_model(self):
        with (
            mock.patch.object(run_tests.sys, "argv", ["run_tests.py", "model", "pr", "spurgear"]),
            mock.patch.object(run_tests, "run_model_pr") as model,
        ):
            run_tests.main()
        model.assert_called_once_with("spurgear")


if __name__ == "__main__":
    unittest.main()
