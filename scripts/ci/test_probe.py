import unittest
from unittest import mock
from pathlib import Path
import tempfile

import gate
import probe


class ProbeTest(unittest.TestCase):
    def test_real_planner_and_gate(self):
        empty, selected = probe.selections()
        self.assertEqual(probe.needs_for(selected, "success")["records-validation"]["result"], "skipped")
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

    def test_real_fixture_records_reconcile_and_incomplete_is_rejected(self):
        Path(".tmp").mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=".tmp") as directory:
            reference = Path(directory) / "reference.jsonl"
            candidate = Path(directory) / "candidate.jsonl"
            probe.record_fixture(reference)
            probe.record_fixture(candidate)
            probe.check_records(reference, candidate)
            lines = candidate.read_text(encoding="utf-8").splitlines(keepends=True)
            candidate.write_text("".join(lines[:-1]), encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "incomplete invocation"):
                probe.check_records(reference, candidate)


if __name__ == "__main__":
    unittest.main()
