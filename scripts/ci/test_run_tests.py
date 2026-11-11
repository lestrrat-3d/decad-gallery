import contextlib
import io
import json
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


class SirenCasesTest(unittest.TestCase):
    def test_siren_cases(self):
        self.assertEqual(set(run_tests.SIREN_CASES), {"default", "variants"})


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
