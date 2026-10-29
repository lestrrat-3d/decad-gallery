#!/usr/bin/env python3
"""Check how run_tests.py reads the decad requirement from go.mod."""

import unittest

import run_tests

BLOCK = """module m

require (
\tgithub.com/lestrrat-3d/decad v0.0.0-20261027054635-bacba52e1142
\tgithub.com/lestrrat-3d/decad-gallery v0.0.0
)
"""


class DecadRequirementTest(unittest.TestCase):
    def test_block_requirement(self):
        self.assertEqual(run_tests.decad_requirement(BLOCK), "v0.0.0-20261027054635-bacba52e1142")

    def test_single_line_requirement(self):
        self.assertEqual(run_tests.decad_requirement("require github.com/lestrrat-3d/decad v1.2.3\n"), "v1.2.3")

    def test_gallery_module_is_not_decad(self):
        self.assertIsNone(run_tests.decad_requirement("require github.com/lestrrat-3d/decad-gallery v0.0.0\n"))

    def test_missing_file_text(self):
        self.assertIsNone(run_tests.decad_requirement(""))


if __name__ == "__main__":
    unittest.main()
