import unittest
from unittest import mock

import replay


class ReplayTest(unittest.TestCase):
    def test_expectations_detect_missing_paths_and_wrong_selection(self):
        result = {"changed_paths": ["other"], "suites": ["wheel"]}
        errors = replay.validate(result, {"expected"}, {"cycloidal-pr"}, {"wheel"})
        self.assertEqual(len(errors), 3)

    def test_empty_expectation_rejects_any_suite(self):
        self.assertTrue(replay.validate({"changed_paths": [], "suites": ["siren"]}, set(), set(), set()))

    def test_replay_rejects_gate_failure(self):
        result = {"changed_paths": ["models/cycloidal/build_test.go"], "suites": ["cycloidal-pr"]}
        with mock.patch.object(replay.subprocess, "check_output", side_effect=["head", "base"]), \
                mock.patch.object(replay.plan, "plan", return_value=result), \
                mock.patch.object(replay.gate, "check", return_value=["worker failed"]):
            _, errors = replay.replay("44e73e3e5")
        self.assertEqual(errors, ["worker failed"])

    def test_module_change_simulates_full_jobs(self):
        result = {"changed_paths": ["go.mod", "go.sum"], "suites": ["broad", "full"]}
        with mock.patch.object(replay.subprocess, "check_output", side_effect=["head", "base"]), \
                mock.patch.object(replay.plan, "plan", return_value=result), \
                mock.patch.object(replay.gate, "check", return_value=[]) as check:
            _, errors = replay.replay("9f71de90c")
        self.assertEqual(errors, [])
        needs = check.call_args.args[0]
        self.assertEqual(needs["renderer-full"]["result"], "success")
        self.assertEqual(needs["bevel-full"]["result"], "success")
        self.assertEqual(needs["bevel-pr"]["result"], "skipped")
        self.assertTrue(all(needs[job]["result"] == "skipped" for job in replay.gate.PR_RENDER_JOBS))
        self.assertEqual(needs["worker"]["result"], "skipped")
        self.assertEqual(needs["records-validation"]["result"], "success")

    def test_renderer_all_skips_individual_renderer_jobs(self):
        result = {"changed_paths": ["models/plaquestand/plaquestand.go"],
                  "suites": ["broad", "new-model-plaquestand", "renderer-all"]}
        with mock.patch.object(replay.subprocess, "check_output", side_effect=["head", "base"]), \
                mock.patch.object(replay.plan, "plan", return_value=result), \
                mock.patch.object(replay.gate, "check", return_value=[]) as check:
            _, errors = replay.replay("b2200f3eb")
        self.assertEqual(errors, [])
        needs = check.call_args.args[0]
        self.assertTrue(all(needs[job]["result"] == "skipped" for job in replay.gate.PR_RENDER_JOBS))
        self.assertEqual(needs["bevel-pr"]["result"], "success")
        self.assertEqual(needs["worker"]["result"], "success")
        self.assertEqual(needs["records-validation"]["result"], "skipped")


if __name__ == "__main__":
    unittest.main()
