import io
import contextlib
import unittest
from unittest import mock

import runner_cpu

CPUINFO = """processor\t: 0
vendor_id\t: AuthenticAMD
model name\t: AMD EPYC  7763 64-Core Processor
cpu MHz\t\t: 2445.426

processor\t: 1
model name\t: AMD EPYC 7763 64-Core Processor
"""

MEMINFO = """MemTotal:       16379840 kB
MemFree:         1000000 kB
"""


class RunnerCpuLineTest(unittest.TestCase):
    def test_parses_model_cpus_and_memory(self):
        self.assertEqual(
            runner_cpu.runner_cpu_line(CPUINFO, MEMINFO, 4),
            "runner-cpu: AMD EPYC 7763 64-Core Processor cpus=4 mem_gb=16",
        )

    def test_missing_text_prints_unknown(self):
        self.assertEqual(
            runner_cpu.runner_cpu_line("", "", None),
            "runner-cpu: unknown cpus=unknown mem_gb=unknown",
        )

    def test_missing_files_print_unknown(self):
        out = io.StringIO()
        with mock.patch("builtins.open", side_effect=FileNotFoundError), \
                mock.patch.object(runner_cpu.os, "cpu_count", return_value=None), \
                contextlib.redirect_stdout(out):
            runner_cpu.print_runner_cpu()
        self.assertEqual(out.getvalue(), "runner-cpu: unknown cpus=unknown mem_gb=unknown\n")


if __name__ == "__main__":
    unittest.main()
