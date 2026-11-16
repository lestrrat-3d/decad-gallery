#!/usr/bin/env python3
"""Measure the Gallery siren CI case against two local Decad checkouts."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import time
from pathlib import Path


TIME = "/usr/bin/time"
METRICS = re.compile(r"PERF_WALL_SECONDS=([0-9.]+)\s+PERF_MAX_RSS_KB=(\d+)")


def run(command: list[str], *, cwd: Path, env: dict[str, str], log: Path) -> float:
    started = time.monotonic()
    with log.open("w") as stream:
        result = subprocess.run(command, cwd=cwd, env=env, stdout=stream,
                                stderr=subprocess.STDOUT, check=False)
    elapsed = time.monotonic() - started
    if result.returncode:
        raise RuntimeError(f"command failed with status {result.returncode}: {log}")
    return elapsed


def make_workspace(gallery: Path, decad: Path, path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.unlink(missing_ok=True)
    subprocess.run(["go", "work", "init", str(gallery), str(decad)],
                   cwd=path.parent, env={**os.environ, "GOWORK": "off"}, check=True)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--gallery", type=Path, required=True)
    parser.add_argument("--baseline", type=Path, required=True)
    parser.add_argument("--candidate", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--pairs", type=int, default=1)
    args = parser.parse_args()

    gallery = args.gallery.resolve()
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    variants = {"baseline": args.baseline.resolve(), "candidate": args.candidate.resolve()}
    workspaces = {name: output / name / "go.work" for name in variants}
    for name, decad in variants.items():
        make_workspace(gallery, decad, workspaces[name])

    common_env = {**os.environ, "GOMAXPROCS": "4"}
    for name in variants:
        warm_env = {**common_env, "GOWORK": str(workspaces[name])}
        run(["go", "test", "./models/handcranksiren", "-run", "^$", "-count=1"],
            cwd=gallery, env=warm_env, log=output / f"warm-{name}.log")

    rows = []
    for pair in range(1, args.pairs + 1):
        order = ("baseline", "candidate") if pair % 2 else ("candidate", "baseline")
        for variant in order:
            log = output / f"pair-{pair}-{variant}.log"
            env = {**common_env, "GOWORK": str(workspaces[variant])}
            command = [TIME, "-f", "\\nPERF_WALL_SECONDS=%e\\nPERF_MAX_RSS_KB=%M",
                       "python3", "scripts/ci/run_tests.py", "siren", "default"]
            elapsed = run(command, cwd=gallery, env=env, log=log)
            content = log.read_text()
            match = METRICS.search(content)
            if match is None:
                raise RuntimeError(f"timing metrics were absent: {log}")
            row = {
                "pair": pair,
                "variant": variant,
                "wall_seconds": float(match.group(1)),
                "max_rss_kb": int(match.group(2)),
                "elapsed_seconds": round(elapsed, 2),
                "log": log.name,
            }
            rows.append(row)
            print(json.dumps(row, sort_keys=True), flush=True)

    (output / "results.json").write_text(json.dumps(rows, indent=2) + "\n")


if __name__ == "__main__":
    main()
