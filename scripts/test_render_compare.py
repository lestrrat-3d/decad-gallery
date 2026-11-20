"""Tests for the render comparison helpers; no Go builds."""
from pathlib import Path
import tempfile
import unittest

import render_compare as rc


class ParseJobsTest(unittest.TestCase):
    def test_parses_kit_and_blank_kit(self):
        jobs = rc.parse_jobs("a\treadme\t-\tmodels/a/images\nb\treadme\tk1\tmodels/b/images/kit-k1\n\n")
        self.assertEqual(jobs, [rc.Job("a", "readme", "", "models/a/images"),
                                rc.Job("b", "readme", "k1", "models/b/images/kit-k1")])

    def test_rejects_malformed_line(self):
        with self.assertRaises(ValueError):
            rc.parse_jobs("a\treadme\tmodels/a")


class CompareTreesTest(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory(dir=rc.ROOT / ".tmp") if (rc.ROOT / ".tmp").is_dir() \
            else tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.left, self.right = Path(temp.name) / "l", Path(temp.name) / "r"
        self.left.mkdir()
        self.right.mkdir()

    def put(self, side, name, data):
        (side / name).write_bytes(data)

    def test_equal(self):
        for side in (self.left, self.right):
            self.put(side, "a.png", b"x")
        self.assertEqual(rc.compare_trees(self.left, self.right), ([], []))

    def test_missing_file(self):
        self.put(self.left, "a.png", b"x")
        self.assertEqual(rc.compare_trees(self.left, self.right), (["a.png"], []))

    def test_differing_bytes(self):
        self.put(self.left, "a.png", b"x")
        self.put(self.right, "a.png", b"y")
        self.assertEqual(rc.compare_trees(self.left, self.right), ([], ["a.png"]))

    def test_single_file_outputs(self):
        self.put(self.left, "m.gif", b"x")
        self.put(self.right, "m.gif", b"x")
        self.assertEqual(rc.compare_trees(self.left / "m.gif", self.right / "m.gif"), ([], []))


class JudgeTest(unittest.TestCase):
    def test_both_fail_passes(self):
        status, detail = rc.judge("boom", "bang", ([], []))
        self.assertEqual(status, "both-fail")
        self.assertIn("boom", detail)
        self.assertIn("bang", detail)

    def test_one_fails_is_mismatch(self):
        self.assertEqual(rc.judge("boom", "", ([], []))[0], "mismatch")
        self.assertEqual(rc.judge("", "bang", ([], []))[0], "mismatch")

    def test_same(self):
        self.assertEqual(rc.judge("", "", ([], []))[0], "same")

    def test_file_set_difference_is_mismatch(self):
        self.assertEqual(rc.judge("", "", (["a.png"], []))[0], "mismatch")

    def test_difference_stable_in_base_is_mismatch(self):
        self.assertEqual(rc.judge("", "", ([], ["a.png"]), ([], []))[0], "mismatch")

    def test_difference_that_base_repeats_is_nondeterministic(self):
        self.assertEqual(rc.judge("", "", ([], ["a.gif"]), ([], ["a.gif"]))[0], "nondeterministic")

    def test_unrelated_file_still_mismatches(self):
        status, detail = rc.judge("", "", ([], ["a.gif", "b.png"]), ([], ["a.gif"]))
        self.assertEqual(status, "mismatch")
        self.assertIn("b.png", detail)


if __name__ == "__main__":
    unittest.main()
