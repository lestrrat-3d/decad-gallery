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


if __name__ == "__main__":
    unittest.main()
