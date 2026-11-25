"""Describe the CI runner so job timings can be compared between runs on the same hardware."""

import os


def _read(path: str) -> str:
    try:
        with open(path, encoding="utf-8") as handle:
            return handle.read()
    except OSError:
        return ""


def runner_cpu_line(cpuinfo: str, meminfo: str, cpus: int | None) -> str:
    """Return `runner-cpu: <model> cpus=<n> mem_gb=<n>` from /proc/cpuinfo and /proc/meminfo text."""
    model = "unknown"
    for line in cpuinfo.splitlines():
        key, _, value = line.partition(":")
        if key.strip() == "model name" and value.strip():
            model = " ".join(value.split())
            break
    mem_gb = "unknown"
    for line in meminfo.splitlines():
        fields = line.split()
        if len(fields) >= 2 and fields[0] == "MemTotal:" and fields[1].isdigit():
            mem_gb = str(round(int(fields[1]) / (1024 * 1024)))
            break
    return f"runner-cpu: {model} cpus={cpus if cpus else 'unknown'} mem_gb={mem_gb}"


def print_runner_cpu() -> None:
    """Print the runner-cpu line for this machine; missing files or a non-Linux host print `unknown`."""
    print(runner_cpu_line(_read("/proc/cpuinfo"), _read("/proc/meminfo"), os.cpu_count()), flush=True)
