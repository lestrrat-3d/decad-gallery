import unittest
from unittest import mock

import gate
import probe


class ProbeTest(unittest.TestCase):
    def test_real_planner_and_gate(self):
        empty, selected = probe.selections()
        self.assertEqual(gate.check(probe.needs_for(empty, "skipped"), "push"), [])
        self.assertEqual(gate.check(probe.needs_for(selected, "success"), "push"), [])
        for result in ("failure", "cancelled", None):
            with self.subTest(result=result):
                self.assertTrue(any("worker:" in error for error in
                                    gate.check(probe.needs_for(selected, result), "push")))

    def test_rejects_unexpected_selection(self):
        with mock.patch.object(probe.plan, "plan", return_value={"suites": ["broad"]}):
            with self.assertRaisesRegex(ValueError, "expected probe selections"):
                probe.selections()


if __name__ == "__main__":
    unittest.main()
