import contextlib
import io
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
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
        assigned = [test for _, names in run_tests.SIREN_CASES.values() for test in names]
        listing = mock.Mock(stdout="\n".join({test.split("/")[0] for test in assigned} |
                                             {"TestBuildRejectsNilDocument", "TestNew"}))
        with (
            mock.patch.object(run_tests.subprocess, "run", return_value=listing),
            mock.patch.object(run_tests, "go_test") as go_test,
        ):
            run_tests.run_siren("rest")
        tests = ["TestBuildRejectsNilDocument", "TestHandCrankSirenVariant", "TestNew"]
        go_test.assert_called_once_with(
            ["./models/handcranksiren", "-run", "^Test", "-skip", run_tests.run_pattern(assigned),
             "-timeout", "40m", "-count=1"],
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
                self.assertEqual(kwargs, {"package": "github.com/lestrrat-3d/decad-gallery/models/handcranksiren"})


class EscapementTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        Path(".tmp").mkdir(exist_ok=True)

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
            json.dumps({"Action": "pass", "Package": "pkg", "Test": test}) + "\n"
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
            tempfile.TemporaryDirectory(dir=".tmp") as directory,
            mock.patch.dict(os.environ, {"CI_TEST_RECORDS": str(Path(directory) / "records.jsonl")}),
            mock.patch.object(run_tests.subprocess, "run", return_value=list_result) as listing,
            mock.patch.object(run_tests.subprocess, "Popen", side_effect=FakeProcess) as popen,
            mock.patch("builtins.print"),
        ):
            run_tests.go_test_shards("./pkg", shards, "1m")
            records = [json.loads(line) for line in (Path(directory) / "records.jsonl").read_text().splitlines()]

        self.assertEqual({r["test"] for r in records if r["kind"] == "test"}, {"TestA", "TestB"})
        self.assertEqual(len({r["invocation"] for r in records}), 2)

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

    def test_sharded_runner_rejects_skipped_subtest(self):
        class FakeProcess:
            def __init__(self, command, stdout, **kwargs):
                stdout.write(json.dumps({"Action": "skip", "Test": "TestA/child"}) + "\n")
                stdout.write(json.dumps({"Action": "pass", "Test": "TestA"}) + "\n")

            def wait(self):
                return 0

            def poll(self):
                return 0

        with mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout="TestA\n")), \
             mock.patch.object(run_tests.subprocess, "Popen", side_effect=FakeProcess):
            with self.assertRaisesRegex(SystemExit, "TestA/child"):
                run_tests.go_test_shards("./pkg", (("TestA",),), "1m")


class WheelTest(unittest.TestCase):
    def test_remainder_rejects_missing_or_overlapping_assignments(self):
        with mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout="TestA\n")):
            with self.assertRaisesRegex(SystemExit, "missing from inventory"):
                run_tests.run_remainder("./models/spinningwheel", ["TestMissing/child"])
            with self.assertRaisesRegex(SystemExit, "overlap"):
                run_tests.run_remainder("./models/spinningwheel", ["TestA", "TestA/child"])

    def test_each_named_case_requires_its_expected_tests(self):
        for case in run_tests.WHEEL_CASES:
            with self.subTest(case=case), mock.patch.object(run_tests, "go_test") as go_test:
                run_tests.run_wheel(case)
                self.assertEqual(go_test.call_args.args[1], run_tests.WHEEL_EXPECTED[case])
                self.assertTrue(go_test.call_args.args[1])

    def test_remainder_includes_tests_without_wheel_prefix(self):
        assigned = [test for tests in run_tests.WHEEL_EXPECTED.values() for test in tests]
        listing = "\n".join({test.split("/")[0] for test in assigned} | {"TestNew"})
        with mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout=listing)), \
                mock.patch.object(run_tests, "go_test") as go_test:
            run_tests.run_wheel("rest")
        command = go_test.call_args.args[0]
        self.assertEqual(command[command.index("-run") + 1], "^Test")
        self.assertEqual(command[command.index("-skip") + 1], run_tests.run_pattern(assigned))
        self.assertIn("TestNew", go_test.call_args.args[1])

    def test_remainder_selects_new_nested_subtests_with_go(self):
        Path(".tmp").mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=".tmp") as directory:
            root = Path(directory)
            (root / "go.mod").write_text("module fixture\n\ngo 1.23\n")
            (root / "selection_test.go").write_text('''package fixture_test
import "testing"
func TestVariant(t *testing.T) {
    t.Run("small", func(t *testing.T) {
        t.Run("old", func(t *testing.T) {})
        t.Run("new", func(t *testing.T) {})
    })
    t.Run("new-variant", func(t *testing.T) {})
}
func TestWhole(t *testing.T) { t.Fatal("assigned elsewhere") }
func TestNew(t *testing.T) {}
''')
            assigned = ["TestVariant/small/old", "TestWhole"]
            result = subprocess.run(["go", "test", "-json", "-run", "^Test", "-skip",
                                     run_tests.run_pattern(assigned), "./..."], cwd=root,
                                    check=True, capture_output=True, text=True)
        events = [json.loads(line) for line in result.stdout.splitlines()]
        passed = {event["Test"] for event in events if event.get("Action") == "pass" and "Test" in event}
        self.assertTrue({"TestVariant/small/new", "TestVariant/new-variant", "TestNew"} <= passed)
        started = {event["Test"] for event in events if event.get("Action") == "run" and "Test" in event}
        self.assertFalse({"TestVariant/small/old", "TestWhole"} & started)
        self.assertFalse([event for event in events if event.get("Action") == "skip"])


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


class FanShardTest(unittest.TestCase):
    def test_shards_partition_tests_and_allow_only_known_skips(self):
        tests = ["TestValidate", "TestHandCrankFanDefault", "TestHandCrankFanVariant"] + [
            f"TestCase{i:02d}" for i in range(20)]
        with mock.patch.dict(run_tests.os.environ, {}, clear=True), \
                mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout="\n".join(tests))), \
                mock.patch.object(run_tests, "go_test") as go_test:
            for shard in range(run_tests.FAN_PR_SHARDS):
                run_tests.run_fan_pr(str(shard))
        selected = [test for call in go_test.call_args_list for test in call.args[1]]
        self.assertCountEqual(selected, tests)
        self.assertEqual(len(selected), len(set(selected)))
        for call in go_test.call_args_list:
            self.assertEqual(call.args[0][2], run_tests.run_pattern(call.args[1]))
            self.assertEqual(call.kwargs["allowed_skips"],
                             frozenset({"TestHandCrankFanDefault", "TestHandCrankFanVariant"}))

    def test_shards_reject_invalid_inventory_and_shard(self):
        for inventory, shard in (("TestOther\n", "0"), ("", "0"), ("TestValidate\n", "10")):
            with self.subTest(shard=shard, inventory=inventory), \
                    mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout=inventory)), \
                    self.assertRaises(SystemExit):
                run_tests.run_fan_pr(shard)

    def test_full_fan_does_not_allow_skips(self):
        with mock.patch.dict(run_tests.os.environ, {"HANDCRANKFAN_FULL": "1"}), \
                mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout="TestValidate\n")), \
                mock.patch.object(run_tests, "go_test") as go_test:
            run_tests.run_fan_pr("0")
        self.assertEqual(go_test.call_args.kwargs["allowed_skips"], frozenset())


class PlanetaryFullTest(unittest.TestCase):
    def test_pr_shards_partition_every_test_once(self):
        tests = ["TestDefaultDimensionsUnchanged", "TestKitGearboxes"] + [f"TestCase{i:02d}" for i in range(30)]
        with mock.patch.dict(run_tests.os.environ, {}, clear=True), \
                mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout="\n".join(tests))), \
                mock.patch.object(run_tests, "go_test") as go_test:
            for shard in range(run_tests.PLANETARY_PR_SHARDS):
                run_tests.run_planetary_pr(str(shard))
        selected = [test for call in go_test.call_args_list for test in call.args[1]]
        self.assertCountEqual(selected, tests)
        self.assertEqual(len(selected), len(set(selected)))
        for call in go_test.call_args_list:
            self.assertEqual(call.args[0][2], run_tests.run_pattern(call.args[1]))
            self.assertIn("TestKitGearboxes/planet608-shaft605", call.kwargs["allowed_skips"])

    def test_pr_shards_reject_missing_inventory_and_invalid_shard(self):
        for inventory, shard in (("TestOther\n", "0"), ("", "0"), ("TestDefaultDimensionsUnchanged\n", "13")):
            with self.subTest(shard=shard, inventory=inventory), \
                    mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout=inventory)), \
                    self.assertRaises(SystemExit):
                run_tests.run_planetary_pr(shard)

    def test_shards_cover_inventory_and_require_kit_variants(self):
        inventory = "TestKitGearboxes\nTestOther\nTestNew\nok\n"
        with mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout=inventory)), \
                mock.patch.object(run_tests, "go_test") as go_test:
            run_tests.run_planetary_full("kits")
            run_tests.run_planetary_full("rest")
        kits, rest = go_test.call_args_list
        self.assertEqual(kits.args[0][2], run_tests.run_pattern(["TestKitGearboxes"]))
        self.assertEqual(rest.args[0][2], run_tests.run_pattern(["TestNew", "TestOther"]))
        self.assertEqual(len(kits.args[1]), 4)
        self.assertIn("TestKitGearboxes/planet608-shaft608", kits.args[1])
        self.assertEqual(kits.kwargs["env"]["PLANETARY_ALL_KITS"], "1")
        self.assertEqual(rest.args[1], ["TestNew", "TestOther"])

    def test_missing_kit_or_invalid_shard_fails(self):
        for inventory, shard in (("TestOther\n", "kits"), ("TestKitGearboxes\nTestOther\n", "bad")):
            with self.subTest(shard=shard), \
                    mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout=inventory)), \
                    self.assertRaises(SystemExit):
                run_tests.run_planetary_full(shard)


class FinishedTestsTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        Path(".tmp").mkdir(exist_ok=True)

    def test_optional_records_keep_packages_subtests_and_invocations(self):
        events = [json.dumps({"Action": "output", "Output": "large output"}),
                  json.dumps({"Package": "one", "Test": "TestA/child", "Action": "skip"}),
                  json.dumps({"Package": "two", "Test": "TestA/child", "Action": "fail"})]
        with tempfile.TemporaryDirectory(dir=".tmp") as directory:
            path = Path(directory) / "records.jsonl"
            with mock.patch.dict(os.environ, {"CI_TEST_RECORDS": str(path)}), \
                    contextlib.redirect_stdout(io.StringIO()):
                run_tests.finished_tests(events)
                run_tests.finished_tests(events[:2])
            records = [json.loads(line) for line in path.read_text().splitlines() if '"kind":"test"' in line]
        self.assertEqual([(r["module"], r["package"], r["test"], r["status"]) for r in records],
                         [(".", "one", "TestA/child", "skip"), (".", "two", "TestA/child", "fail"),
                          (".", "one", "TestA/child", "skip")])
        self.assertEqual(records[0]["invocation"], records[1]["invocation"])
        self.assertNotEqual(records[0]["invocation"], records[2]["invocation"])

    def test_failure_still_records_terminal_events(self):
        process = mock.Mock(stdout=[json.dumps({"Package": "p", "Test": "TestA", "Action": "fail"})],
                            returncode=1)
        process.wait.return_value = 1
        with tempfile.TemporaryDirectory(dir=".tmp") as directory:
            path = Path(directory) / "records.jsonl"
            with mock.patch.dict(os.environ, {"CI_TEST_RECORDS": str(path)}), \
                    mock.patch.object(run_tests.subprocess, "Popen", return_value=process):
                with self.assertRaises(SystemExit):
                    run_tests.go_test(["./p"])
            self.assertEqual(json.loads(path.read_text().splitlines()[0])["status"], "fail")

    def test_metadata_package_failure_and_invalid_identity(self):
        events = [json.dumps({"Package": "p", "Action": "fail"}),
                  json.dumps({"Test": "TestA", "Action": "pass"}), "bad json"]
        env = {"GITHUB_JOB": "wheel", "CI_TEST_CASE": "rest", "GITHUB_SHA": "abc",
               "GITHUB_RUN_ID": "123", "GITHUB_RUN_ATTEMPT": "2"}
        with tempfile.TemporaryDirectory(dir=".tmp") as directory:
            path = Path(directory) / "records.jsonl"
            with mock.patch.dict(os.environ, {**env, "CI_TEST_RECORDS": str(path)}), \
                    contextlib.redirect_stdout(io.StringIO()):
                run_tests.finished_tests(events)
            records = [json.loads(line) for line in path.read_text().splitlines()]
        self.assertEqual([r["kind"] for r in records], ["package", "malformed", "malformed", "complete"])
        self.assertEqual((records[0]["package"], records[0]["status"]), ("p", "fail"))
        self.assertEqual([r["reason"] for r in records[1:3]], ["invalid_identity", "invalid_json"])
        self.assertEqual([records[0][key] for key in ("job", "case", "revision", "run_id", "run_attempt")],
                         ["wheel", "rest", "abc", "123", "2"])

    def test_interrupted_stream_has_no_complete_marker(self):
        def interrupted():
            yield json.dumps({"Package": "p", "Test": "TestA", "Action": "pass"})
            raise OSError("stream interrupted")
        with tempfile.TemporaryDirectory(dir=".tmp") as directory:
            path = Path(directory) / "records.jsonl"
            with mock.patch.dict(os.environ, {"CI_TEST_RECORDS": str(path)}):
                with self.assertRaises(OSError):
                    run_tests.finished_tests(interrupted())
            records = [json.loads(line) for line in path.read_text().splitlines()]
        self.assertEqual([r["kind"] for r in records], ["test"])

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

    def test_selected_test_may_skip_only_when_explicitly_allowed(self):
        event = json.dumps({"Package": "p", "Test": "TestA", "Action": "skip"})
        process = mock.Mock(stdout=[event], returncode=0)
        process.wait.return_value = 0
        with mock.patch.object(run_tests.subprocess, "Popen", return_value=process):
            run_tests.go_test(["./p"], ["TestA"], package="p", allowed_skips=frozenset({"TestA"}))
            with self.assertRaisesRegex(SystemExit, "p/TestA"):
                run_tests.go_test(["./p"], ["TestA"], package="p")
            with self.assertRaisesRegex(SystemExit, "p/TestB"):
                run_tests.go_test(["./p"], ["TestB"], package="p", allowed_skips=frozenset({"TestA"}))

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


class DedicatedModelInventoryTest(unittest.TestCase):
    def test_dedicated_jobs_inventory_every_package(self):
        cases = ((run_tests.run_turbinefan, ("turbinefan",)),
                 (run_tests.run_onewayclutch, ("onewayclutch",)),
                 (run_tests.run_clamparm, ("clamparm", "m5bolt", "m5nut")))
        prefix = "github.com/lestrrat-3d/decad-gallery/models/"
        for runner, names in cases:
            with self.subTest(names=names), \
                 mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout="TestShared\n")) as listing, \
                 mock.patch.object(run_tests, "go_test") as go_test:
                runner()
                self.assertEqual(listing.call_count, len(names))
                go_test.assert_called_once_with(
                    [*(f"./models/{name}" for name in names), "-timeout", "40m", "-count=1"],
                    qualified_expected={(prefix + name, "TestShared") for name in names},
                )

    def test_empty_package_inventory_fails(self):
        with mock.patch.object(run_tests.subprocess, "run", return_value=mock.Mock(stdout="")), \
             mock.patch.object(run_tests, "go_test") as go_test:
            with self.assertRaisesRegex(SystemExit, "model package m5bolt lists no tests"):
                run_tests.run_model_packages(("m5bolt",))
            go_test.assert_not_called()


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
